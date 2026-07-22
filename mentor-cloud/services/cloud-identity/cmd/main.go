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

	"cloud-identity/internal/adapters"
	"cloud-identity/internal/adapters/handler"
	"cloud-identity/internal/adapters/repository"
	"cloud-identity/internal/application"
	"cloud-identity/internal/ports"

	"github.com/gin-gonic/gin"
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
		port = "8081"
	}
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET not set")
	}

	db, err := repository.NewPgxPool(dbURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepo(db)
	empresaRepo := repository.NewEmpresaRepo(db)
	rolRepo := repository.NewRolRepo(db)
	tokenRepo := repository.NewTokenRepo(db)

	authSvc := application.NewAuthService(userRepo, tokenRepo, jwtSecret)
	userSvc := application.NewUserService(userRepo, rolRepo)
	empresaSvc := application.NewEmpresaService(empresaRepo)

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
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
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
		c.JSON(200, gin.H{"service": "cloud-identity", "status": "ok"})
	})

	api := r.Group("/")
	handler.RegisterAuthRoutes(api, authSvc, mw)
	handler.RegisterUserRoutes(api, userSvc, mw, notifier)
	handler.RegisterEmpresaRoutes(api, empresaSvc, mw)
	handler.RegisterRolRoutes(api, rolRepo, mw)

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go func() {
		log.Printf("cloud-identity listening on :%s", port)
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
	log.Println("cloud-identity shutdown")
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
