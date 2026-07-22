package ports

import (
	"net/http"
	"strings"

	"cloud-integration/internal/domain"

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
		empresaID, _ := claims["empresa_id"].(float64)

		c.Set("user_id", int(userID))
		c.Set("user_role", role)
		c.Set("empresa_id", int(empresaID))
		c.Next()
	}
}

func (m *JWTMiddleware) RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != "ADMIN" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acceso restringido"})
			return
		}
		c.Next()
	}
}

type APIKeyMiddleware struct {
	repo APIKeyRepository
}

func NewAPIKeyMiddleware(repo APIKeyRepository) *APIKeyMiddleware {
	return &APIKeyMiddleware{repo: repo}
}

func (m *APIKeyMiddleware) Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-API-Key")
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "X-API-Key requerida"})
			return
		}
		key, err := m.repo.Validate(c.Request.Context(), raw)
		if err != nil || key == nil || !key.Activo {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "clave invalida o inactiva"})
			return
		}
		c.Set("empresa_id", key.EmpresaID)
		c.Set("api_key_id", key.ID)
		c.Set("api_key_scopes", key.Scopes)
		c.Next()
	}
}

func RequireScope(scope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		scopes, _ := c.Get("api_key_scopes")
		if sl, ok := scopes.([]string); ok {
			for _, s := range sl {
				if s == scope {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "scope insuficiente", "required": scope})
	}
}

func HasScope(c *gin.Context, scope string) bool {
	scopes, _ := c.Get("api_key_scopes")
	if sl, ok := scopes.([]string); ok {
		for _, s := range sl {
			if s == scope {
				return true
			}
		}
	}
	return false
}

// EmpresaID extrae de context (funciona tanto para JWT como para API key).
func EmpresaIDFromCtx(c *gin.Context) (int, bool) {
	v, exists := c.Get("empresa_id")
	if !exists {
		return 0, false
	}
	switch id := v.(type) {
	case int:
		return id, true
	case int64:
		return int(id), true
	case domain.APIKey:
		return id.EmpresaID, true
	}
	return 0, false
}
