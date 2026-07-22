package ports

import (
	"context"
	"net/http"
	"strings"

	"energy-ingest/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// scopeResolverAuth is the minimal interface needed by TokenAuth.
type scopeResolverAuth interface {
	ResolveByAPIKey(ctx context.Context, apiKey string) (deviceID string, scope *domain.DeviceScope, err error)
}

// TokenAuth valida X-API-Key buscando en gateway.device_registry.
// Si el token existe, inyecta device_id y scope en el contexto gin.
// fallbackKey permite compatibilidad con la clave global ENERGY_API_KEY:
// si el token coincide, se acepta sin resolución de scope (el handler usa X-Device-ID).
func TokenAuth(resolver scopeResolverAuth, fallbackKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				key = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "api key requerida"})
			return
		}

		// Resolución por token individual en device_registry (per-meter).
		deviceID, scope, err := resolver.ResolveByAPIKey(c.Request.Context(), key)
		if err == nil {
			c.Set("token_device_id", deviceID)
			c.Set("token_scope", scope)
			c.Next()
			return
		}

		// Fallback: clave global — compatibilidad con dispositivos sin token propio.
		if fallbackKey != "" && key == fallbackKey {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "api key invalida"})
	}
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("claims", claims)
		}
		c.Next()
	}
}
