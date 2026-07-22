package handler

import (
	"net/http"
	"strconv"

	"cloud-integration/internal/adapters/repository"
	"cloud-integration/internal/domain"
	"cloud-integration/internal/ports"

	"github.com/gin-gonic/gin"
)

func RegisterAPIKeyRoutes(api *gin.RouterGroup, repo ports.APIKeyRepository, mw *ports.JWTMiddleware) {
	g := api.Group("/integration/keys", mw.Auth())
	g.POST("", createKey(repo))
	g.GET("", listKeys(repo))
	g.DELETE("/:id", revokeKey(repo))
}

func createKey(repo ports.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req domain.CreateKeyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		callerRole, _ := c.Get("user_role")
		callerEmpresa, _ := c.Get("empresa_id")
		roleStr, _ := callerRole.(string)
		if roleStr != "ADMIN" && roleStr != "superadmin" {
			if callerEmpresa.(int) != req.EmpresaID {
				c.JSON(http.StatusForbidden, gin.H{"error": "empresa no permitida"})
				return
			}
		}

		plaintext, prefix, hash, err := repository.GenerateKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error generando clave"})
			return
		}

		key, err := repo.Create(c.Request.Context(), req, hash, prefix)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, domain.CreateKeyResponse{
			APIKey:    *key,
			Plaintext: plaintext,
		})
	}
}

func listKeys(repo ports.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var empresaID int
		role, _ := c.Get("user_role")
		roleStr, _ := role.(string)
		if roleStr == "ADMIN" || roleStr == "superadmin" {
			if v := c.Query("empresa_id"); v != "" {
				id, err := strconv.Atoi(v)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "empresa_id invalido"})
					return
				}
				empresaID = id
			}
		} else {
			v, _ := c.Get("empresa_id")
			empresaID, _ = v.(int)
		}

		if empresaID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "empresa_id requerido"})
			return
		}

		keys, err := repo.List(c.Request.Context(), empresaID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if keys == nil {
			keys = []domain.APIKey{}
		}
		c.JSON(http.StatusOK, keys)
	}
}

func revokeKey(repo ports.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
			return
		}

		var empresaID int
		role, _ := c.Get("user_role")
		roleStr, _ := role.(string)
		if roleStr == "ADMIN" || roleStr == "superadmin" {
			if v := c.Query("empresa_id"); v != "" {
				empresaID, _ = strconv.Atoi(v)
			}
		} else {
			v, _ := c.Get("empresa_id")
			empresaID, _ = v.(int)
		}

		if err := repo.Revoke(c.Request.Context(), id, empresaID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
