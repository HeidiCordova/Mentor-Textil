package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"resiliencia/internal/adapters"
	"resiliencia/internal/application"
	"resiliencia/internal/domain"
)

func main() {
	dbHost := getEnv("DB_HOST", "postgres-local")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "mentor_edge")
	port := getEnv("PORT", "8002")

	connStr := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser +
		" password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"

	storage, err := adapters.NewPostgresRepo(connStr, "")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Servidor HTTP minimal pre-loop para que el healthcheck de Docker responda
	// mientras se espera la configuración de la primera línea.
	preHealthMux := http.NewServeMux()
	preHealthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"starting","message":"esperando configuracion de linea"}`))
	})
	preHealthSrv := &http.Server{Addr: ":" + port, Handler: preHealthMux}
	go func() {
		if err := preHealthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[resiliencia] pre-health server error: %v", err)
		}
	}()

	// Esperar hasta que haya al menos una línea configurada en DB.
	var lineIDs []int
	for {
		lineIDs = nil
		rows, err := storage.DB().Query(`
			SELECT linea_id FROM config.line_config
			WHERE linea_id > 0
			ORDER BY linea_id`)
		if err != nil {
			log.Printf("[resiliencia] failed to query line configs: %v — reintentando en 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for rows.Next() {
			var lid int
			if err := rows.Scan(&lid); err != nil {
				log.Fatal("[resiliencia] scan line_id: ", err)
			}
			lineIDs = append(lineIDs, lid)
		}
		rows.Close()
		if len(lineIDs) > 0 {
			break
		}
		log.Println("[resiliencia] esperando configuración de línea — configura via UI en :8080")
		time.Sleep(5 * time.Second)
	}

	dedup := domain.NewInMemoryDedup(10000)
	queue := domain.NewDefaultQueuePolicy()

	services := make(map[int]*application.BufferService, len(lineIDs))
	for _, lid := range lineIDs {
		st, err := adapters.NewPostgresRepo(connStr, fmt.Sprintf("linea_%d", lid))
		if err != nil {
			log.Printf("[resiliencia] linea_%d: storage init failed: %v", lid, err)
			continue
		}
		services[lid] = application.NewBufferService(st, dedup, queue)
		log.Printf("[resiliencia] linea_%d initialized", lid)
	}

	server := adapters.NewHTTPServer(services, lineIDs[0], port)

	// Cerrar el servidor pre-loop y ceder el puerto al servidor real.
	shutCtx, shutFn := context.WithTimeout(context.Background(), 3*time.Second)
	preHealthSrv.Shutdown(shutCtx) //nolint
	shutFn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Printf("Starting resiliencia on port %s", port)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	go func() {
		for lid, svc := range services {
			lid, svc := lid, svc
			go func() {
				log.Printf("[resiliencia] linea_%d maintenance started", lid)
				svc.RunMaintenance(ctx, time.Hour)
			}()
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
