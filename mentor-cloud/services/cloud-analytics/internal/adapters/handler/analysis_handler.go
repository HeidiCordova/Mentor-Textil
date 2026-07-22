package handler

import (
	"cloud-analytics/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AnalysisHandler struct {
	repo ports.AnalysisRepository
}

func RegisterAnalysisRoutes(rg *gin.RouterGroup, repo ports.AnalysisRepository, mw gin.HandlerFunc) {
	h := &AnalysisHandler{repo: repo}
	g := rg.Group("/analisis", mw)
	g.GET("/general", h.General)
	g.GET("/energia", h.Energia)
	g.GET("/produccion", h.Produccion)
	g.GET("/pareto", h.Pareto)
	g.GET("/combined", h.Combined)
}

func (h *AnalysisHandler) General(c *gin.Context) {
	f := parseFilter(c)
	data, err := h.repo.GetGeneral(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *AnalysisHandler) Energia(c *gin.Context) {
	f := parseFilter(c)
	data, err := h.repo.GetEnergy(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *AnalysisHandler) Produccion(c *gin.Context) {
	f := parseFilter(c)
	data, err := h.repo.GetProduction(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *AnalysisHandler) Pareto(c *gin.Context) {
	f := parseFilter(c)
	data, err := h.repo.GetPareto(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *AnalysisHandler) Combined(c *gin.Context) {
	f := parseFilter(c)
	data, err := h.repo.GetCombined(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func parseFilter(c *gin.Context) ports.AnalysisFilter {
	f := ports.AnalysisFilter{
		DeviceID: c.Query("device_id"),
		From:     c.Query("from"),
		To:       c.Query("to"),
	}
	if v := c.Query("empresa_id"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.EmpresaID = &i
		}
	}
	if v := c.Query("planta_id"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.PlantaID = &i
		}
	}
	if v := c.Query("linea_id"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.LineaID = &i
		}
	}
	if v := c.Query("limit"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			f.Limit = i
		}
	}
	if v := c.Query("min_interval_s"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			f.MinIntervalS = i
		}
	}
	return f
}
