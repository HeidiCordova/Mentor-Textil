package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"edge-config-service/internal/adapters"
	"edge-config-service/internal/application"
)

func main() {
	dbHost := getEnv("DB_HOST", "postgres-local")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "mentor_edge")
	port := getEnv("PORT", "8004")

	connStr := "host=" + dbHost + " port=" + dbPort + " user=" + dbUser +
		" password=" + dbPassword + " dbname=" + dbName + " sslmode=disable"

	storage, err := adapters.NewPostgresRepo(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	service := application.NewConfigService(storage)
	server := adapters.NewHTTPServer(service, port)

	go func() {
		log.Printf("Starting edge-config-service on port %s", port)
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
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
