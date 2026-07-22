package handler

import (
	"cloud-analytics/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	repo ports.DashboardRepository
}

func RegisterDashboardRoutes(rg *gin.RouterGroup, repo ports.DashboardRepository, mw gin.HandlerFunc) {
	h := &DashboardHandler{repo: repo}
	g := rg.Group("/dashboard", mw)
	g.GET("/stats", h.Stats)
	g.GET("/reportes", h.Reportes)
	g.GET("/graficos", h.Graficos)
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	empresaID := queryIntPtr(c, "empresa_id")
	stats, err := h.repo.GetStats(c.Request.Context(), empresaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *DashboardHandler) Reportes(c *gin.Context) {
	empresaID := queryIntPtr(c, "empresa_id")
	from := c.DefaultQuery("from", "")
	to := c.DefaultQuery("to", "")

	reports, err := h.repo.GetReports(c.Request.Context(), empresaID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reports})
}

func (h *DashboardHandler) Graficos(c *gin.Context) {
	empresaID := queryIntPtr(c, "empresa_id")
	charts, err := h.repo.GetCharts(c.Request.Context(), empresaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, charts)
}

func queryIntPtr(c *gin.Context, key string) *int {
	v := c.Query(key)
	if v == "" {
		return nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &i
}
