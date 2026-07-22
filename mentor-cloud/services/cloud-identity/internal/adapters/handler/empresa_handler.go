package handler

import (
	"cloud-identity/internal/application"
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EmpresaHandler struct {
	svc *application.EmpresaService
}

func RegisterEmpresaRoutes(r *gin.RouterGroup, svc *application.EmpresaService, mw *ports.JWTMiddleware) {
	h := &EmpresaHandler{svc: svc}
	g := r.Group("/empresas", mw.Auth())
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *EmpresaHandler) List(c *gin.Context) {
	empresas, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": empresas})
}

func (h *EmpresaHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	empresa, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Empresa no encontrada"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": empresa})
}

func (h *EmpresaHandler) Create(c *gin.Context) {
	var e domain.Empresa
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.Estado = true
	if err := h.svc.Create(c.Request.Context(), &e); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": e})
}

func (h *EmpresaHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var e domain.Empresa
	if err := c.ShouldBindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.ID = id

	if err := h.svc.Update(c.Request.Context(), &e); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": e})
}

func (h *EmpresaHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "Empresa eliminada correctamente"}})
}
