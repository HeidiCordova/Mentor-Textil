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

	"cloud-integration/internal/adapters/handler"
	"cloud-integration/internal/adapters/repository"
	"cloud-integration/internal/ports"

	"mentor.local/shared/cache"
	"mentor.local/shared/multitenancy"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
)

func main() {
	dbURL := mustEnv("DATABASE_URL")
	jwtSecret := mustEnv("JWT_SECRET")
	port := env("PORT", "8085")
	corsOrigins := env("CORS_ORIGINS", "")

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

	apiKeyRepo := repository.NewAPIKeyRepo(db)
	queryRepo := repository.NewQueryRepo(pr)

	jwtMw := ports.NewJWTMiddleware(jwtSecret)
	apiKeyMw := ports.NewAPIKeyMiddleware(apiKeyRepo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	r.Use(func(c *gin.Context) {
		if scope, ok := multitenancy.ScopeFromRequest(c.Request); ok {
			c.Request = c.Request.WithContext(multitenancy.WithScope(c.Request.Context(), scope))
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "cloud-integration", "status": "ok"})
	})

	api := r.Group("/api")
	handler.RegisterAPIKeyRoutes(api, apiKeyRepo, jwtMw)

	v1 := r.Group("/api/v1")
	handler.RegisterQueryRoutes(v1, queryRepo, apiKeyMw)

	origins := parseOrigins(corsOrigins)
	c := cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-API-Key"},
		AllowCredentials: true,
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      c.Handler(r),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("cloud-integration listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("cloud-integration stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env: %s", key)
	}
	return v
}

func parseOrigins(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	var origins []string
	for _, o := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
