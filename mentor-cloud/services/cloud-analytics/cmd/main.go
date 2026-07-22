package main

import (
	adapters "cloud-analytics/internal/adapters"
	"cloud-analytics/internal/adapters/handler"
	"cloud-analytics/internal/adapters/repository"
	"cloud-analytics/internal/ports"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mentor.local/shared/cache"
	"mentor.local/shared/multitenancy"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL requerida")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET requerido")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	pool, err := repository.NewPgxPool(dbURL)
	if err != nil {
		log.Fatalf("conexion a base de datos fallida: %v", err)
	}
	defer pool.Close()

	ingestURL := os.Getenv("INGEST_URL")
	if ingestURL == "" {
		ingestURL = "http://cloud-ingest:8082"
	}

	appCache := cache.NewMemoryCache()

	var mgr *multitenancy.PlantaPoolManager
	if encKeyHex := os.Getenv("ENCRYPTION_KEY"); encKeyHex != "" {
		encKey, err := hex.DecodeString(encKeyHex)
		if err != nil {
			log.Fatalf("ENCRYPTION_KEY hex invalido: %v", err)
		}
		creds, err := multitenancy.NewAESCredentialProvider(encKey, pool)
		if err != nil {
			log.Fatalf("credential provider: %v", err)
		}
		tplSQL := os.Getenv("LINEA_TEMPLATE_SQL")
		prov := multitenancy.NewPostgresProvisioner(pool, tplSQL)
		dsnCfg := multitenancy.DSNConfig{
			SSLMode:     os.Getenv("DB_SSL_MODE"),
			SSLRootCert: os.Getenv("DB_SSL_ROOT_CERT"),
		}
		mgr = multitenancy.NewPlantaPoolManager(pool, creds, prov, dsnCfg, appCache)
		log.Println("multitenancy pool manager activo")
	}

	pr := multitenancy.NewPoolResolver(pool, mgr)

	dashRepo := repository.NewDashboardRepo(pr, ingestURL, appCache)
	analysisRepo := repository.NewAnalysisRepository(pr)
	alarmaRepo := repository.NewAlarmaRepository(pr)
	compromisoRepo := repository.NewCompromisoRepository(pr)
	stopRepo := repository.NewStopRepository(pr)
	oeeRepo := repository.NewOEERepository(pr)
	prodRunRepo := repository.NewProductionRunRepository(pr)
	oeeLabRepo := repository.NewOEELabRepository(pr)

	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://cloud-gateway:8888"
	}
	internalKey := os.Getenv("INTERNAL_API_KEY")
	var dispatcher ports.CommandDispatcher
	var notifier ports.BrowserNotifier
	if internalKey != "" {
		dispatcher = adapters.NewGatewayCommandDispatcher(gatewayURL, internalKey)
		notifier = adapters.NewGatewayBrowserNotifier(gatewayURL, internalKey)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     parseOrigins(os.Getenv("CORS_ORIGINS")),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "cloud-analytics"})
	})

	r.Use(func(c *gin.Context) {
		if scope, ok := multitenancy.ScopeFromRequest(c.Request); ok {
			c.Request = c.Request.WithContext(multitenancy.WithScope(c.Request.Context(), scope))
		}
		c.Next()
	})

	mw := jwtMiddleware(jwtSecret)
	api := r.Group("/api")

	handler.RegisterDashboardRoutes(api, dashRepo, mw)
	handler.RegisterAnalysisRoutes(api, analysisRepo, mw)
	handler.RegisterAlarmaRoutes(api, alarmaRepo, mw)
	handler.RegisterCompromisoRoutes(api, compromisoRepo, mw)
	handler.RegisterReportRoutes(api, mw)
	handler.RegisterStopRoutes(api, stopRepo, dispatcher, notifier, mw)
	handler.RegisterOEERoutes(api, oeeRepo, mw)
	handler.RegisterProductionRunRoutes(api, prodRunRepo, pr, mw)
	handler.RegisterOEELabRoutes(api, oeeLabRepo, mw)

	// Internal endpoint — requires X-Internal-Key for service-to-service auth
	internal := r.Group("/internal")
	internal.Use(func(c *gin.Context) {
		if internalKey == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "internal auth not configured"})
			return
		}
		if c.GetHeader("X-Internal-Key") != internalKey {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid internal key"})
			return
		}
		c.Next()
	})
	internal.GET("/stops", func(c *gin.Context) {
		f := ports.StopFilter{Limit: 1000}
		if v := c.Query("empresa_id"); v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				f.EmpresaID = &i
			}
		}
		if v := c.Query("device_id"); v != "" {
			f.DeviceID = v
		}
		stops, err := stopRepo.ListStops(c.Request.Context(), f)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stops)
	})

	srv := &http.Server{Addr: ":" + port, Handler: r}

	go refreshMaterializedViews(pool)

	go func() {
		log.Printf("cloud-analytics escuchando en :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("error en servidor: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("apagando cloud-analytics")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("apagado forzado: %v", err)
	}
}

func jwtMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token requerido"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalido"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "claims invalidos"})
			return
		}
		c.Set("user_id", claims["sub"])
		c.Set("empresa_id", claims["empresa_id"])
		c.Next()
	}
}

func refreshMaterializedViews(pool *pgxpool.Pool) {
	ticker := time.NewTicker(60 * time.Minute)
	defer ticker.Stop()

	refresh := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		rows, err := pool.Query(ctx, `SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'linea_%'`)
		if err != nil {
			log.Printf("mv refresh: error listando schemas: %v", err)
			return
		}
		defer rows.Close()

		var schemas []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err == nil {
				schemas = append(schemas, s)
			}
		}

		for _, s := range schemas {
			q := fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s.mv_produccion_mensual", s)
			if _, err := pool.Exec(ctx, q); err != nil {
				log.Printf("mv refresh: %s: %v", s, err)
			}
		}
		if len(schemas) > 0 {
			log.Printf("mv refresh: %d schemas procesados", len(schemas))
		}
	}

	refresh()
	for range ticker.C {
		refresh()
	}
}

func parseOrigins(raw string) []string {
	if raw == "" {
		return []string{"*"}
	}
	origins := strings.Split(raw, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return origins
}
