package handler

import (
	"cloud-analytics/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OEEHandler struct {
	repo ports.OEERepository
}

func RegisterOEERoutes(rg *gin.RouterGroup, repo ports.OEERepository, mw gin.HandlerFunc) {
	h := &OEEHandler{repo: repo}
	g := rg.Group("/oee", mw)
	g.GET("/snapshots", h.Snapshots)
	g.GET("/summary", h.Summary)
	g.GET("/latest", h.Latest)
}

func (h *OEEHandler) Snapshots(c *gin.Context) {
	f := parseOEEFilter(c)
	data, err := h.repo.GetSnapshots(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *OEEHandler) Summary(c *gin.Context) {
	f := parseOEEFilter(c)
	data, err := h.repo.GetSummary(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *OEEHandler) Latest(c *gin.Context) {
	lineaID, err := strconv.Atoi(c.Query("linea_id"))
	if err != nil || lineaID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "linea_id requerido"})
		return
	}
	data, err := h.repo.GetLatest(c.Request.Context(), lineaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func parseOEEFilter(c *gin.Context) ports.OEEFilter {
	f := ports.OEEFilter{
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
