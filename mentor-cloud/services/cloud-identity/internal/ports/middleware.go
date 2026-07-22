package ports

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type JWTMiddleware struct {
	secret []byte
}

func NewJWTMiddleware(secret string) *JWTMiddleware {
	return &JWTMiddleware{secret: []byte(secret)}
}

func (m *JWTMiddleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token requerido"})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		if tokenStr == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "formato invalido"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return m.secret, nil
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

		userID, _ := claims["sub"].(float64)
		role, _ := claims["role"].(string)

		c.Set("user_id", int(userID))
		c.Set("user_role", role)
		if eid, ok := claims["empresa_id"].(float64); ok {
			c.Set("empresa_id", int64(eid))
		}
		c.Next()
	}
}
