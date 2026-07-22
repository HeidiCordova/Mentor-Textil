package handler

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CompromisoHandler struct {
	repo ports.CompromisoRepository
}

func RegisterCompromisoRoutes(rg *gin.RouterGroup, repo ports.CompromisoRepository, mw gin.HandlerFunc) {
	h := &CompromisoHandler{repo: repo}
	g := rg.Group("/compromisos", mw)
	g.GET("", h.List)
	g.POST("", h.Create)
	g.PUT("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *CompromisoHandler) List(c *gin.Context) {
	empresaID := queryIntPtr(c, "empresa_id")
	data, err := h.repo.FindAll(c.Request.Context(), empresaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *CompromisoHandler) Create(c *gin.Context) {
	var comp domain.Compromiso
	if err := c.ShouldBindJSON(&comp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if comp.Estado == "" {
		comp.Estado = "pendiente"
	}
	if err := h.repo.Create(c.Request.Context(), &comp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, comp)
}

func (h *CompromisoHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	var comp domain.Compromiso
	if err := c.ShouldBindJSON(&comp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comp.ID = id
	if err := h.repo.Update(c.Request.Context(), &comp); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, comp)
}

func (h *CompromisoHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}
