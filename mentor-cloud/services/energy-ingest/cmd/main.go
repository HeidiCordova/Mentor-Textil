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

	"energy-ingest/internal/adapters/handler"
	"energy-ingest/internal/adapters/repository"
	"energy-ingest/internal/application"
	"energy-ingest/internal/metrics"
	"energy-ingest/internal/ports"

	"mentor.local/shared/cache"
	"mentor.local/shared/multitenancy"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	energyAPIKey := os.Getenv("ENERGY_API_KEY")
	if energyAPIKey == "" {
		log.Fatal("ENERGY_API_KEY requerido")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8086"
	}

	db, err := repository.NewPgxPool(dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	// --- Multitenancy: PlantaPoolManager (opcional) ---
	var plantaRepo *repository.PlantaEnergyRepo
	if encKeyHex := os.Getenv("ENCRYPTION_KEY"); encKeyHex != "" {
		encKey, err := hex.DecodeString(encKeyHex)
		if err != nil {
			log.Fatalf("ENCRYPTION_KEY hex invalido: %v", err)
		}
		creds, err := multitenancy.NewAESCredentialProvider(encKey, db)
		if err != nil {
			log.Fatalf("credential provider: %v", err)
		}
		prov := multitenancy.NewPostgresProvisioner(db, "")
		dsnCfg := multitenancy.DSNConfig{
			SSLMode:     os.Getenv("DB_SSL_MODE"),
			SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
		}
		mgr := multitenancy.NewPlantaPoolManager(db, creds, prov, dsnCfg, cache.NewMemoryCache())
		plantaRepo = repository.NewPlantaEnergyRepo(mgr)
		log.Println("energy-ingest: multitenancy pool manager activo")
	}

	energyRepo := repository.NewEnergyRepo(db)
	scopeResolver := repository.NewScopeResolver(db)
	svc := application.NewEnergyService(energyRepo, plantaRepo, scopeResolver)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	corsOrigins := parseOrigins(os.Getenv("CORS_ORIGINS"))
	c := cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
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
		c.JSON(200, gin.H{"service": "energy-ingest", "status": "ok"})
	})

	metrics.Init()
	metricsHandler := promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})
	r.GET("/metrics", func(c *gin.Context) {
		metricsHandler.ServeHTTP(c.Writer, c.Request)
	})

	api := r.Group("/")
	handler.RegisterRoutes(api, svc, ports.JWTAuth(jwtSecret), energyAPIKey, scopeResolver)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("energy-ingest listening on :%s", port)
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
	log.Println("energy-ingest shutdown")
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
