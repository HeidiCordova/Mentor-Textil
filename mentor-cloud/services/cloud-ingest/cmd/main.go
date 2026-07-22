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

	"cloud-ingest/internal/adapters"
	"cloud-ingest/internal/adapters/handler"
	"cloud-ingest/internal/adapters/repository"
	"cloud-ingest/internal/application"
	"cloud-ingest/internal/metrics"
	"cloud-ingest/internal/ports"

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET requerido")
	}

	db, err := repository.NewPgxPool(dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	var mgr *multitenancy.PlantaPoolManager
	if encKeyHex := os.Getenv("ENCRYPTION_KEY"); encKeyHex != "" {
		encKey, err := hex.DecodeString(encKeyHex)
		if err != nil {
			log.Fatalf("ENCRYPTION_KEY hex invalido: %v", err)
		}
		creds, err := multitenancy.NewAESCredentialProvider(encKey, db)
		if err != nil {
			log.Fatalf("credential provider: %v", err)
		}
		tplSQL := os.Getenv("LINEA_TEMPLATE_SQL")
		prov := multitenancy.NewPostgresProvisioner(db, tplSQL)
		dsnCfg := multitenancy.DSNConfig{
			SSLMode:     os.Getenv("DB_SSL_MODE"),
			SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
		}
		mgr = multitenancy.NewPlantaPoolManager(db, creds, prov, dsnCfg, cache.NewMemoryCache())
		log.Println("multitenancy pool manager activo")
	}

	pr := multitenancy.NewPoolResolver(db, mgr)
	eventRepo := repository.NewEventRepo(pr)
	velRepo := repository.NewVelocidadNominalRepo(pr)
	scopeResolver := repository.NewScopeResolver(db)

	gatewayURL := os.Getenv("GATEWAY_URL")
	internalKey := os.Getenv("INTERNAL_API_KEY")
	var notifier ports.Notifier
	if gatewayURL != "" && internalKey != "" {
		notifier = adapters.NewSyncNotifier(gatewayURL, internalKey)
		log.Printf("cloud-ingest: sync notifier enabled (gateway=%s)", gatewayURL)
	}
	ingestSvc := application.NewIngestService(eventRepo, notifier, velRepo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	corsOrigins := parseOrigins(os.Getenv("CORS_ORIGINS"))
	c := cors.New(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-API-Key", "X-Device-ID", "X-Empresa-ID", "X-Planta-ID", "X-Linea-ID"},
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
		c.JSON(200, gin.H{"service": "cloud-ingest", "status": "ok"})
	})
	metricsHandler := promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})
	r.GET("/metrics", func(c *gin.Context) {
		metricsHandler.ServeHTTP(c.Writer, c.Request)
	})
	metrics.Init()

	r.Use(func(c *gin.Context) {
		if scope, ok := multitenancy.ScopeFromRequest(c.Request); ok {
			c.Request = c.Request.WithContext(multitenancy.WithScope(c.Request.Context(), scope))
		}
		c.Next()
	})

	edgeAPIKey := os.Getenv("EDGE_API_KEY")
	if edgeAPIKey == "" {
		log.Fatal("EDGE_API_KEY requerido")
	}
	// internalKey ya fue declarado arriba para el notifier

	api := r.Group("/")
	handler.RegisterIngestRoutes(api, ingestSvc, scopeResolver, ports.JWTAuth(jwtSecret), edgeAPIKey)
	handler.RegisterOEEQueryRoutes(api, eventRepo, internalKey)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("cloud-ingest listening on :%s", port)
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
	log.Println("cloud-ingest shutdown")
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
