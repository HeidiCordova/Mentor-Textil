package main

// Architecture: Edge Gateway (Single Entry Point)
// Cloud never calls internal services directly.
// Tablet never calls internal services directly (in hybrid mode).
// Commands are idempotent and audited.
// SSE is used for real-time UI updates (incremental rendering on tablet).

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"edge-gateway/internal/adapters"
	"edge-gateway/internal/application"
	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("[gateway] starting edge-gateway")

	port := envOrDefault("PORT", "8005")
	authToken := envOrDefault("AUTH_TOKEN", "")

	dbHost := envOrDefault("DB_HOST", "postgres-local")
	dbPort := envOrDefault("DB_PORT", "5432")
	dbUser := envOrDefault("DB_USER", "postgres")
	dbPass := envOrDefault("DB_PASSWORD", "postgres")
	dbName := envOrDefault("DB_NAME", "mentor_edge")

	configURL := envOrDefault("CONFIG_SERVICE_URL", "http://localhost:8004")
	resilienciaURL := envOrDefault("RESILIENCIA_URL", "http://localhost:8002")
	detectorURL := envOrDefault("DETECTOR_URL", "http://localhost:8001")
	enviadorURL := envOrDefault("ENVIADOR_URL", "http://localhost:8003")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName,
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("[gateway] database open error: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("[gateway] database ping error: %v", err)
	}
	log.Println("[gateway] database connected")

	// Servidor pre-loop: sirve /health y /edge/camera/stream ANTES de que haya
	// líneas configuradas. Así nginx nunca recibe 502 por puerto cerrado, y el
	// usuario puede verificar la cámara en Modo Lab durante el primer setup.
	preLoopMux := http.NewServeMux()
	preLoopMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"starting","message":"esperando configuracion de linea"}`))
	})
	preLoopMux.HandleFunc("/edge/camera/stream", adapters.EarlyCameraStreamHandler)
	preLoopMux.HandleFunc("/edge/camera/health", func(w http.ResponseWriter, r *http.Request) {
		// El check de cámara no requiere líneas — reenviar directamente.
		// Necesitamos método GET con url= param.
		adapters.EarlyCameraHealthHandler(w, r)
	})
	preLoopSrv := &http.Server{Addr: ":" + port, Handler: preLoopMux}
	go func() {
		if err := preLoopSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[gateway] pre-loop server error: %v", err)
		}
	}()

	type lineRow struct {
		deviceID     string
		lineaID      int
		empresaID    int
		cloudURL     string
		cloudKey     string
		cloudLineaID int
	}

	// Esperar hasta que haya al menos una línea configurada.
	// Esto permite levantar edge-gateway desde el inicio sin necesidad de
	// reiniciarlo manualmente después de configurar la primera línea.
	var lineRows []lineRow
	for {
		lineRows = nil
		rows, err := db.Query(`
			SELECT device_id, linea_id, empresa_id,
			       COALESCE(cloud->>'cloud_url', ''),
			       COALESCE(cloud->>'cloud_api_key', ''),
			       COALESCE(cloud_linea_id, 0)
			FROM config.line_config
			WHERE linea_id > 0
			ORDER BY linea_id`)
		if err != nil {
			log.Printf("[gateway] failed to query line configs: %v — reintentando en 5s", err)
			time.Sleep(5 * time.Second)
			continue
		}
		for rows.Next() {
			var lr lineRow
			if err := rows.Scan(&lr.deviceID, &lr.lineaID, &lr.empresaID, &lr.cloudURL, &lr.cloudKey, &lr.cloudLineaID); err != nil {
				log.Fatalf("[gateway] scan line config: %v", err)
			}
			lineRows = append(lineRows, lr)
		}
		rows.Close()
		if len(lineRows) > 0 {
			break
		}
		log.Println("[gateway] esperando configuración de línea — configura via UI en :8080")
		time.Sleep(5 * time.Second)
	}

	deviceID := lineRows[0].deviceID
	defaultLineaID := lineRows[0].lineaID
	empresaID := lineRows[0].empresaID
	cloudURL := lineRows[0].cloudURL
	cloudKey := lineRows[0].cloudKey

	bufferClient := adapters.NewHTTPResilienciaClient(resilienciaURL)
	detectorClient := adapters.NewHTTPDetectorClient(detectorURL)
	enviadorClient := adapters.NewHTTPEnviadorClient(enviadorURL)

	broker := adapters.NewSSEBroker(connStr)
	if err := broker.Start(); err != nil {
		log.Fatalf("[gateway] SSE broker start error: %v", err)
	}
	log.Println("[gateway] SSE broker started")

	lines := make(map[int]*adapters.LineContext, len(lineRows))
	for _, lr := range lineRows {
		lineaIDStr := fmt.Sprintf("%d", lr.lineaID)
		lineaSchema := fmt.Sprintf("linea_%d", lr.lineaID)

		stopRepo := adapters.NewPostgresStopRepo(db, lineaSchema)
		commandRepo := adapters.NewPostgresCommandRepo(db, lineaSchema)
		auditRepo := adapters.NewPostgresAuditRepo(db, lineaSchema)
		catalogSyncRepo := adapters.NewPostgresCatalogSyncRepo(db, lineaSchema)
		productionRunRepo := adapters.NewPostgresProductionRunRepo(db, lineaSchema)
		configClient := adapters.NewHTTPConfigClient(configURL, lineaIDStr)

		gatewaySvc := application.NewGatewayService(
			stopRepo, productionRunRepo, catalogSyncRepo, configClient, bufferClient, detectorClient,
			enviadorClient, auditRepo, broker, deviceID,
		)
		commandSvc := application.NewCommandService(
			commandRepo, stopRepo, configClient, auditRepo, broker, catalogSyncRepo, deviceID,
			lr.lineaID, lr.cloudLineaID,
		)

		lines[lr.lineaID] = &adapters.LineContext{
			LineaID:      lr.lineaID,
			CloudLineaID: lr.cloudLineaID,
			Gateway:      gatewaySvc,
			Commands:     commandSvc,
			CatalogSync:  catalogSyncRepo,
		}
		log.Printf("[gateway] linea_%d initialized (device=%s empresa=%d)", lr.lineaID, lr.deviceID, lr.empresaID)
	}

	statusSvc := application.NewStatusService(
		bufferClient, adapters.NewHTTPConfigClient(configURL, fmt.Sprintf("%d", defaultLineaID)),
		enviadorClient, detectorClient,
		adapters.NewPostgresAuditRepo(db, fmt.Sprintf("linea_%d", defaultLineaID)), deviceID,
	)

	httpCfg := adapters.HTTPServerConfig{
		Port:      atoi(port),
		DeviceID:  deviceID,
		AuthToken: authToken,
		EmpresaID: empresaID,
		CloudURL:  cloudURL,
		CloudKey:  cloudKey,
	}
	// Build cloud_linea_id → LineContext secondary map
	cloudLineas := make(map[int]*adapters.LineContext, len(lines))
	for _, lc := range lines {
		if lc.CloudLineaID > 0 {
			cloudLineas[lc.CloudLineaID] = lc
		}
	}

	// Usuarios son globales a la planta: el CommandService del default line necesita
	// replicar SYNC_USUARIOS a todos los schemas de línea.
	{
		extraRepos := make([]ports.CatalogSyncRepository, 0, len(lines)-1)
		for id, lc := range lines {
			if id != defaultLineaID {
				extraRepos = append(extraRepos, lc.CatalogSync)
			}
		}
		if len(extraRepos) > 0 {
			lines[defaultLineaID].Commands.SetExtraCatalogSyncs(extraRepos)
		}
	}

	httpServer := adapters.NewHTTPServer(httpCfg, db, lines, cloudLineas, defaultLineaID, statusSvc, broker)

	// Detener el servidor pre-loop antes de iniciar el completo
	// (ambos usan el mismo puerto).
	preLoopShutCtx, preLoopShutCancel := context.WithTimeout(context.Background(), 3*time.Second)
	preLoopSrv.Shutdown(preLoopShutCtx)
	preLoopShutCancel()

	maintenanceCtx, maintenanceCancel := context.WithCancel(context.Background())
	defer maintenanceCancel()
	for _, lc := range lines {
		lc := lc
		go lc.Gateway.RunMaintenance(maintenanceCtx)
	}

	if cloudURL != "" && cloudKey != "" {
		defaultCmdSvc := lines[defaultLineaID].Commands
		var sseClient *adapters.CloudSSEClient
		sseClient = adapters.NewCloudSSEClient(cloudURL, cloudKey, deviceID,
			func(cmd adapters.CloudCommand) error {
				edgeType := mapCloudCommandType(cmd.Type)
				req := domain.CreateCommandRequest{
					CommandID:      newUUID(),
					DeviceID:       deviceID,
					CommandType:    edgeType,
					Payload:        cmd.Payload,
					IssuedBy:       fmt.Sprintf("cloud:%d", cmd.IssuedBy),
					IdempotencyKey: fmt.Sprintf("cloud-cmd-%d", cmd.ID),
				}
				_, execErr := defaultCmdSvc.ReceiveCommand(context.Background(), req)
				if execErr != nil && execErr != domain.ErrDuplicateCommand {
					log.Printf("[cloud-sse] cmd=%d tipo=%s failed: %v", cmd.ID, cmd.Type, execErr)
				}
				if ackErr := sseClient.AckCloud(context.Background(), cmd.ID); ackErr != nil {
					log.Printf("[cloud-sse] ack failed cmd=%d: %v", cmd.ID, ackErr)
				}
				return nil
			},
		)
		go sseClient.Start(maintenanceCtx)
		log.Printf("[gateway] cloud SSE client started -> %s", cloudURL)

		// Camera push: streams live MJPEG video to cloud for remote monitoring.
		// Runs ffmpeg at 3 fps / 640 px to keep upstream bandwidth low.
		defaultLine := lines[defaultLineaID]
		getCameraURL := func() string {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cfg, err := defaultLine.Gateway.GetConfig(ctx)
			if err != nil || cfg == nil {
				return ""
			}
			if cam, ok := cfg["camera"].(map[string]interface{}); ok {
				if u, ok := cam["url"].(string); ok {
					return u
				}
			}
			return ""
		}
		// getROI colecta ROI de TODAS las líneas, claveado por cloud_linea_id.
		// Incluye source_w/source_h (resolución original de cámara) para que el
		// frontend escale correctamente sobre el stream de 640 px.
		getROI := func() string {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			allLines := make(map[string]interface{})
			for _, lc := range lines {
				if lc.CloudLineaID <= 0 {
					continue
				}
				cfg, err := lc.Gateway.GetConfig(ctx)
				if err != nil || cfg == nil {
					continue
				}
				hasCam := false
				if cam, ok := cfg["camera"].(map[string]interface{}); ok {
					if u, ok := cam["url"].(string); ok && u != "" {
						hasCam = true
					}
				}
				allLines[fmt.Sprintf("%d", lc.CloudLineaID)] = map[string]interface{}{
					"roi":           cfg["roi"],
					"roi_presencia": cfg["roi_presencia"],
					"source_w":      1280,
					"source_h":      720,
					"has_camera":    hasCam,
				}
			}
			if len(allLines) == 0 {
				return ""
			}
			b, err := json.Marshal(map[string]interface{}{"lines": allLines})
			if err != nil {
				return ""
			}
			return string(b)
		}
		cameraPush := adapters.NewCameraPushClient(cloudURL, cloudKey, deviceID,
			getCameraURL, getROI,
		)
		go cameraPush.Start(maintenanceCtx)
		log.Printf("[gateway] camera push client started")
	} else {
		log.Printf("[gateway] CLOUD_URL/CLOUD_API_KEY not configured — cloud SSE disabled")
	}

	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[gateway] http server error: %v", err)
		}
	}()

	log.Printf("[gateway] ready on :%s (device=%s, lines=%d)", port, deviceID, len(lines))

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("[gateway] received signal: %v, shutting down", sig)

	maintenanceCancel()
	broker.Shutdown()
	for _, lc := range lines {
		lc.Commands.Shutdown()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[gateway] http shutdown error: %v", err)
	}

	log.Println("[gateway] edge-gateway stopped")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// mapCloudCommandType convierte el tipo de comando del cloud al tipo del edge.
func mapCloudCommandType(cloudType string) string {
	switch cloudType {
	case "update_config", "actualizar_config":
		return "ACTUALIZAR_CONFIG"
	case "restart_pipeline", "reiniciar_pipeline":
		return "REINICIAR_PIPELINE"
	case "calibrate", "iniciar_calibracion":
		return "INICIAR_CALIBRACION"
	case "modificar_parada":
		return "MODIFICAR_PARADA"
	case "justificar_parada":
		return "JUSTIFICAR_PARADA"
	case "crear_parada":
		return "CREAR_PARADA"
	case "cerrar_parada":
		return "CERRAR_PARADA"
	case "eliminar_parada":
		return "ELIMINAR_PARADA"
	case "SYNC_CATALOG", "sync_catalog":
		return "SYNC_CATALOG"
	case "SYNC_PRODUCTOS", "sync_productos":
		return "SYNC_PRODUCTOS"
	case "SYNC_TURNOS", "sync_turnos":
		return "SYNC_TURNOS"
	case "SYNC_USUARIOS", "sync_usuarios":
		return "SYNC_USUARIOS"
	case "SYNC_VARIABLES", "sync_variables":
		return "SYNC_VARIABLES"
	case "SYNC_LINEA_PRODUCTO_VARS", "sync_linea_producto_vars":
		return "SYNC_LINEA_PRODUCTO_VARS"
	case "SYNC_PRODUCTO_CARACTERISTICAS", "sync_producto_caracteristicas":
		return "SYNC_PRODUCTO_CARACTERISTICAS"
	case "SYNC_PLANTAS_LINEAS", "sync_plantas_lineas":
		return "SYNC_PLANTAS_LINEAS"
	case "SYNC_PARADAS", "sync_paradas":
		return "SYNC_PARADAS"
	case "SYNC_VELOCIDAD_NOMINAL", "sync_velocidad_nominal":
		return "SYNC_VELOCIDAD_NOMINAL"
	case "SYNC_MOTIVOS_VELOCIDAD", "sync_motivos_velocidad":
		return "SYNC_MOTIVOS_VELOCIDAD"
	default:
		return "COMANDO_CUSTOM"
	}
}

// newUUID genera un UUID v4 aleatorio usando crypto/rand.
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant RFC 4122
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
