package handler

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type AlarmaHandler struct {
	repo ports.AlarmaRepository
}

func RegisterAlarmaRoutes(rg *gin.RouterGroup, repo ports.AlarmaRepository, mw gin.HandlerFunc) {
	h := &AlarmaHandler{repo: repo}
	g := rg.Group("/alarmas", mw)
	g.GET("", h.List)
	g.POST("", h.Create)
	g.POST("/:id/reconocer", h.Acknowledge)
}

func (h *AlarmaHandler) List(c *gin.Context) {
	empresaID := queryIntPtr(c, "empresa_id")
	var activa *bool
	if v := c.Query("activa"); v != "" {
		b := v == "true"
		activa = &b
	}

	alarmas, err := h.repo.FindAll(c.Request.Context(), empresaID, activa)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": alarmas})
}

func (h *AlarmaHandler) Create(c *gin.Context) {
	var req struct {
		DeviceID  string `json:"device_id" binding:"required"`
		LineaID   *int   `json:"linea_id"`
		PlantaID  *int   `json:"planta_id"`
		EmpresaID *int   `json:"empresa_id"`
		Tipo      string `json:"tipo" binding:"required"`
		Mensaje   string `json:"mensaje" binding:"required"`
		Severidad string `json:"severidad" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a := domain.Alarma{
		DeviceID:        req.DeviceID,
		LineaID:         req.LineaID,
		PlantaID:        req.PlantaID,
		EmpresaID:       req.EmpresaID,
		Tipo:            req.Tipo,
		Mensaje:         req.Mensaje,
		Severidad:       req.Severidad,
		Activa:          true,
		TimestampEvento: time.Now(),
	}
	if err := h.repo.Create(c.Request.Context(), &a); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (h *AlarmaHandler) Acknowledge(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	if err := h.repo.Acknowledge(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "alarma reconocida"})
}
