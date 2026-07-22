package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud-gateway/internal/adapters"
	"cloud-gateway/internal/application"

	"mentor.local/shared/cache"
)

type config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	EdgeAPIKey     string
	InternalAPIKey string
	CORSOrigins    string
	IdentityURL    string
	IngestURL      string
	ConfigURL      string
	AnalyticsURL   string
	IntegrationURL string
	EnergyURL      string
	FrontendURL    string
}

func load() config {
	return config{
		Port:           env("PORT", "8888"),
		DatabaseURL:    must("DATABASE_URL"),
		JWTSecret:      must("JWT_SECRET"),
		EdgeAPIKey:     must("EDGE_API_KEY"),
		InternalAPIKey: env("INTERNAL_API_KEY", ""),
		CORSOrigins:    env("CORS_ORIGINS", ""),
		IdentityURL:    env("IDENTITY_URL", "http://cloud-identity:8081"),
		IngestURL:      env("INGEST_URL", "http://cloud-ingest:8082"),
		ConfigURL:      env("CONFIG_URL", "http://cloud-config:8083"),
		AnalyticsURL:   env("ANALYTICS_URL", "http://cloud-analytics:8084"),
		IntegrationURL: env("INTEGRATION_URL", "http://cloud-integration:8085"),
		EnergyURL:      env("ENERGY_URL", "http://energy-ingest:8086"),
		FrontendURL:    env("FRONTEND_URL", "http://cloud-frontend:80"),
	}
}

func main() {
	cfg := load()

	pool, err := adapters.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	auditRepo := adapters.NewAuditRepo(pool)
	commandRepo := adapters.NewCommandRepo(pool)
	deviceRepo := adapters.NewDeviceRepo(pool, cache.NewMemoryCache())

	mw := adapters.NewMiddleware(cfg.JWTSecret, cfg.EdgeAPIKey, cfg.InternalAPIKey, deviceRepo, parseOrigins(cfg.CORSOrigins))
	svc := application.NewGatewayService(commandRepo, deviceRepo)

	hub := adapters.NewSSEHub()
	browserHub := adapters.NewBrowserSSEHub()
	cameraHub := adapters.NewCameraHub()

	router := adapters.NewRouter(adapters.RouterConfig{
		Identity:       adapters.NewProxy(cfg.IdentityURL),
		Ingest:         adapters.NewProxy(cfg.IngestURL),
		Config:         adapters.NewProxy(cfg.ConfigURL),
		Analytics:      adapters.NewProxy(cfg.AnalyticsURL),
		Integration:    adapters.NewProxy(cfg.IntegrationURL),
		Energy:         adapters.NewProxy(cfg.EnergyURL),
		Frontend:       adapters.NewProxy(cfg.FrontendURL),
		Mw:             mw,
		AuditRepo:      auditRepo,
		Svc:            svc,
		Hub:            hub,
		BrowserHub:     browserHub,
		CameraHub:      cameraHub,
		IdentityURL:    cfg.IdentityURL,
		IngestURL:      cfg.IngestURL,
		ConfigURL:      cfg.ConfigURL,
		AnalyticsURL:   cfg.AnalyticsURL,
		IntegrationURL: cfg.IntegrationURL,
		EnergyURL:      cfg.EnergyURL,
		InternalKey:    cfg.InternalAPIKey,
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // 0 = sin límite — necesario para SSE de larga duración
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("cloud-gateway listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("cloud-gateway stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func must(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env: %s", key)
	}
	return v
}

func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
