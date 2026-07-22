package handler

import (
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RegisterRolRoutes(r *gin.RouterGroup, repo ports.RolRepository, mw *ports.JWTMiddleware) {
	g := r.Group("/roles", mw.Auth())
	g.GET("", func(c *gin.Context) {
		roles, err := repo.FindAll(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, roles)
	})
	g.POST("", func(c *gin.Context) {
		var rol domain.Rol
		if err := c.ShouldBindJSON(&rol); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if rol.Permisos == nil {
			rol.Permisos = []string{}
		}
		if err := repo.Create(c.Request.Context(), &rol); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, rol)
	})
	g.PUT("/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
			return
		}
		var rol domain.Rol
		if err := c.ShouldBindJSON(&rol); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		rol.ID = id
		if rol.Permisos == nil {
			rol.Permisos = []string{}
		}
		if err := repo.Update(c.Request.Context(), &rol); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, rol)
	})
	g.DELETE("/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
			return
		}
		if err := repo.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	})
}
