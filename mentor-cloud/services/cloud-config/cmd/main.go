package main

import (
	"context"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud-config/internal/adapters"
	"cloud-config/internal/adapters/handler"
	"cloud-config/internal/adapters/repository"
	"cloud-config/internal/ports"

	"mentor.local/shared/cache"
	"mentor.local/shared/multitenancy"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL requerido")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET requerido")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	db, err := repository.NewPgxPool(dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	plantaRepo := repository.NewPlantaRepo(db)
	lineaRepo := repository.NewLineaRepo(db)
	dispRepo := repository.NewDispositivoRepo(db)
	provRepo := repository.NewDeviceProvisioningRepo(db)
	varRepo := repository.NewVariableRepo(db)
	dcRepo := repository.NewDeviceConfigRepo(db)
	locacionRepo := repository.NewLocacionRepo(db)

	var plantaMgr *multitenancy.PlantaPoolManager
	var provisioningH *handler.ProvisioningHandler
	if encKeyHex := os.Getenv("ENCRYPTION_KEY"); encKeyHex != "" {
		encKey, err := hex.DecodeString(encKeyHex)
		if err != nil {
			log.Fatalf("ENCRYPTION_KEY hex invalido: %v", err)
		}
		creds, err := multitenancy.NewAESCredentialProvider(encKey, db)
		if err != nil {
			log.Fatalf("credential provider: %v", err)
		}
		dsnCfg := multitenancy.DSNConfig{
			SSLMode:     os.Getenv("DB_SSL_MODE"),
			SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
		}
		templateSQLPath := os.Getenv("LINEA_TEMPLATE_SQL")
		templateSQLBytes, err := os.ReadFile(templateSQLPath)
		if err != nil {
			log.Fatalf("leer template SQL '%s': %v", templateSQLPath, err)
		}
		templateSQL := string(templateSQLBytes)
		var energyTemplateSQL string
		if energyPath := os.Getenv("ENERGY_LINEA_TEMPLATE_SQL"); energyPath != "" {
			energyBytes, err := os.ReadFile(energyPath)
			if err != nil {
				log.Printf("WARN: leer energy template '%s': %v (provisioning de líneas energía deshabilitado)", energyPath, err)
			} else {
				energyTemplateSQL = string(energyBytes)
				log.Println("energy linea template cargado")
			}
		}
		provisioner := multitenancy.NewPostgresProvisioner(db, templateSQL)
		plantaMgr = multitenancy.NewPlantaPoolManager(db, creds, provisioner, dsnCfg, cache.NewMemoryCache())
		provisioningH = handler.NewProvisioningHandler(plantaRepo, lineaRepo, db, provisioner, creds, energyTemplateSQL)
		log.Println("provisioning endpoint habilitado")
	}

	gatewayURL := os.Getenv("GATEWAY_URL")
	internalKey := os.Getenv("INTERNAL_API_KEY")
	var notifier *adapters.SyncNotifier
	if gatewayURL != "" && internalKey != "" {
		notifier = adapters.NewSyncNotifier(gatewayURL, internalKey)
		log.Printf("sync notifier activo -> %s", gatewayURL)
	}

	mw := ports.NewJWTMiddleware(jwtSecret)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	corsOrigins := parseOrigins(os.Getenv("CORS_ORIGINS"))
	c := cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-API-Key", "X-Device-ID"},
		AllowCredentials: true,
		MaxAge:           43200,
	})
	r.Use(func(ctx *gin.Context) {
		c.HandlerFunc(ctx.Writer, ctx.Request)
		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}
		ctx.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"service": "cloud-config", "status": "ok"})
	})

	api := r.Group("/")
	handler.RegisterPlantaRoutes(api, plantaRepo, mw, notifier)
	handler.RegisterLineaRoutes(api, lineaRepo, mw, notifier)
	handler.RegisterDispositivoRoutes(api, dispRepo, provRepo, varRepo, mw)
	handler.RegisterVariableRoutes(api, varRepo, dispRepo, mw, notifier)
	handler.RegisterCatParadasRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterArbolParadasRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterCategoriaParadasRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterTurnosSimpleRoutes(api, db, plantaMgr, mw)
	handler.RegisterLocacionRoutes(api, locacionRepo, mw)
	handler.RegisterDeviceConfigRoutes(api, dcRepo, dispRepo)
	handler.RegisterCanvasOEERoutes(api, db, mw)
	handler.RegisterTurnoDiaRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterProductoCaractRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterVelocidadNominalRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterVelocidadNominalEdgeRoutes(r, db, plantaMgr, notifier)
	handler.RegisterMotivosVelocidadRoutes(api, db, plantaMgr, mw, notifier)
	handler.RegisterLineaConfigRoutes(api, db, mw, notifier)
	handler.RegisterLineaConfigEdgeRoutes(r, db, notifier)

	if provisioningH != nil {
		handler.RegisterProvisioningRoutes(api, provisioningH, mw)
	}

	// Internal service-to-service endpoint for scoped device sync
	if internalKey != "" {
		handler.RegisterInternalSyncRoutes(r, db, plantaMgr, internalKey)
		log.Println("internal device-config endpoint registered")
	}

	// Edge-accessible endpoint: returns all plants and lines (authenticated via API key at gateway)
	r.GET("/plants-lines", func(c *gin.Context) {
		plantas, _ := plantaRepo.FindAll(c.Request.Context(), nil)
		lineas, _ := lineaRepo.FindAll(c.Request.Context(), nil, nil)
		c.JSON(200, gin.H{"plantas": plantas, "lineas": lineas})
	})

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("cloud-config listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Println("cloud-config shutdown")
}

func parseOrigins(raw string) []string {
	if raw == "" || raw == "*" {
		return []string{"*"}
	}
	origins := strings.Split(raw, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return origins
}
