package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"enviador/internal/adapters"
	"enviador/internal/application"
	"enviador/internal/domain"
	"enviador/internal/ports"
)

type lineConfig struct {
	lineaID      int
	cloudLineaID int
	deviceID     string
	cloudURL     string
	apiKey       string
}

func main() {
	dbHost := getEnv("DB_HOST", "postgres-local")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "mentor_edge")
	healthPort := getEnv("HEALTH_PORT", "8003")

	connStr := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser +
		" password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"

	baseStorage, err := adapters.NewPostgresReader(connStr, "")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Levantar health server antes del loop para que el healthcheck de Docker
	// pueda responder aunque aún se esté esperando configuración de líneas.
	healthServer := adapters.NewHealthServer(healthPort)
	go func() {
		log.Printf("[enviador] health server on port %s", healthPort)
		if err := healthServer.Start(); err != nil {
			log.Printf("[enviador] health server error: %v", err)
		}
	}()

	// Esperar hasta que haya al menos una línea configurada en DB.
	var lines []lineConfig
	for {
		lines = nil
		ctx5, cancel5 := context.WithTimeout(context.Background(), 5*time.Second)
		rows, err := baseStorage.DB().QueryContext(ctx5, `
			SELECT linea_id, device_id,
			       COALESCE(cloud->>'cloud_url', ''),
			       COALESCE(cloud->>'cloud_api_key', ''),
			       COALESCE(cloud_linea_id, linea_id)
			FROM config.line_config
			WHERE linea_id > 0
			ORDER BY linea_id`)
		if err != nil {
			cancel5()
			log.Printf("[enviador] failed to query line configs: %v — reintentando en 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for rows.Next() {
			var lc lineConfig
			if err := rows.Scan(&lc.lineaID, &lc.deviceID, &lc.cloudURL, &lc.apiKey, &lc.cloudLineaID); err != nil {
				cancel5()
				log.Fatal("[enviador] failed to scan line config: ", err)
			}
			lines = append(lines, lc)
		}
		rows.Close()
		cancel5()
		if len(lines) > 0 {
			break
		}
		log.Println("[enviador] esperando configuración de línea — configura via UI en :8080")
		time.Sleep(5 * time.Second)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for _, lc := range lines {
		lc := lc
		lineaSchema := fmt.Sprintf("linea_%d", lc.lineaID)

		storage, err := adapters.NewPostgresReader(connStr, lineaSchema)
		if err != nil {
			log.Printf("[enviador] linea_%d: failed to create storage: %v", lc.lineaID, err)
			continue
		}

		// Usar cloud_linea_id para X-Linea-ID (por defecto fallback a linea_id local)
		cloudLineaIDStr := fmt.Sprintf("%d", lc.cloudLineaID)
		cloudClient := adapters.NewHTTPCloudClient(lc.cloudURL, lc.deviceID, lc.apiKey, cloudLineaIDStr, 30*time.Second)
		retryPolicy := domain.NewDefaultRetryPolicy()
		syncPolicy := domain.NewDefaultSyncPolicy()
		service := application.NewSenderService(storage, cloudClient, retryPolicy, syncPolicy, healthServer, lc.lineaID)

		log.Printf("[enviador] linea_%d: device_id=%s cloud_linea_id=%d cloud_url=%s", lc.lineaID, lc.deviceID, lc.cloudLineaID, lc.cloudURL)

		wg.Add(1)
		go func() {
			defer wg.Done()
			service.Run(ctx)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(2 * time.Second)
			hbCtx, hbCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer hbCancel()
			if info, err := cloudClient.Heartbeat(hbCtx); err == nil {
				healthServer.MarkCloudOK()
				log.Printf("[enviador] linea_%d: initial heartbeat OK", lc.lineaID)
				syncDeviceConfig(baseStorage.DB(), lc.lineaID, info)
			} else {
				log.Printf("[enviador] linea_%d: initial heartbeat failed: %v", lc.lineaID, err)
			}

			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					hbCtx, hbCancel := context.WithTimeout(context.Background(), 5*time.Second)
					if _, err := cloudClient.Heartbeat(hbCtx); err == nil {
						healthServer.MarkCloudOK()
					} else {
						healthServer.MarkError()
					}
					hbCancel()
				}
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[enviador] shutting down...")
	cancel()
	wg.Wait()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	healthServer.Shutdown(shutdownCtx)

	log.Println("[enviador] exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// syncDeviceConfig corrige automáticamente device_id y empresa_id en la DB edge
// usando la config canónica que devuelve el cloud en el heartbeat.
// Esto evita que divergencias manuales entre configuraciones produzcan desincronías.
func syncDeviceConfig(db *sql.DB, lineaID int, info *ports.HeartbeatInfo) {
	if info == nil || info.DeviceID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var currentDeviceID string
	var currentEmpresaID int
	err := db.QueryRowContext(ctx,
		`SELECT device_id, empresa_id FROM config.line_config WHERE linea_id = $1`,
		lineaID,
	).Scan(&currentDeviceID, &currentEmpresaID)
	if err != nil {
		log.Printf("[enviador] linea_%d: no se pudo leer config local: %v", lineaID, err)
		return
	}

	needsUpdate := currentDeviceID != info.DeviceID || (info.EmpresaID != 0 && currentEmpresaID != info.EmpresaID)
	if !needsUpdate {
		return
	}

	newEmpresaID := currentEmpresaID
	if info.EmpresaID != 0 {
		newEmpresaID = info.EmpresaID
	}

	_, err = db.ExecContext(ctx,
		`UPDATE config.line_config SET device_id = $1, empresa_id = $2 WHERE linea_id = $3`,
		info.DeviceID, newEmpresaID, lineaID,
	)
	if err != nil {
		log.Printf("[enviador] linea_%d: error auto-corrigiendo config: %v", lineaID, err)
		return
	}
	log.Printf("[enviador] linea_%d: config auto-corregida desde cloud — device_id: %s→%s, empresa_id: %d→%d",
		lineaID, currentDeviceID, info.DeviceID, currentEmpresaID, newEmpresaID)
}
