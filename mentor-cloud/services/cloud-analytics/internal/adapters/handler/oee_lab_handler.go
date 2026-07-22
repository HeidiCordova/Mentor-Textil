package handler

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OEELabHandler struct {
	repo ports.OEELabRepository
}

func RegisterOEELabRoutes(rg *gin.RouterGroup, repo ports.OEELabRepository, mw gin.HandlerFunc) {
	h := &OEELabHandler{repo: repo}
	g := rg.Group("/oee-lab", mw)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.POST("", h.Create)
	g.DELETE("/:id", h.Delete)
}

func (h *OEELabHandler) List(c *gin.Context) {
	sessions, err := h.repo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sessions == nil {
		sessions = []domain.OEELabSession{}
	}
	c.JSON(http.StatusOK, gin.H{"data": sessions})
}

func (h *OEELabHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	s, err := h.repo.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *OEELabHandler) Create(c *gin.Context) {
	var req domain.OEELabSessionCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Nombre == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nombre requerido"})
		return
	}
	if req.Inputs == nil {
		req.Inputs = map[string]interface{}{}
	}
	if req.Results == nil {
		req.Results = map[string]interface{}{}
	}

	userID := extractIntClaim(c, "user_id")
	if userID > 0 {
		req.CreatedBy = strconv.FormatInt(userID, 10)
	}

	s, err := h.repo.Create(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, s)
}

func (h *OEELabHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true, "id": id})
}
