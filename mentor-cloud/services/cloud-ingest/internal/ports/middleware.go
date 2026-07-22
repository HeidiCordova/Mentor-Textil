package ports

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func APIKeyAuth(validKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		deviceID := c.GetHeader("X-Device-ID")

		if key == "" {
			auth := c.GetHeader("Authorization")
			if len(auth) > 7 {
				key = auth[7:]
			}
		}

		if key == "" || deviceID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "api key requerida"})
			return
		}

		if validKey == "" || key != validKey {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api key invalida"})
			return
		}

		c.Set("device_id_header", deviceID)
		// Propagate org scope from gateway
		if v := c.GetHeader("X-Empresa-ID"); v != "" && v != "0" {
			c.Set("empresa_id_header", v)
		}
		if v := c.GetHeader("X-Planta-ID"); v != "" && v != "0" {
			c.Set("planta_id_header", v)
		}
		if v := c.GetHeader("X-Linea-ID"); v != "" && v != "0" {
			c.Set("linea_id_header", v)
		}
		c.Next()
	}
}

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenStr := ""
		if strings.HasPrefix(header, "Bearer ") {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		} else {
			// Compatibilidad para WebSocket desde navegador (no permite setear Authorization).
			tokenStr = strings.TrimSpace(c.Query("token"))
			if tokenStr == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token requerido"})
				return
			}
		}
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
		c.Next()
	}
}
