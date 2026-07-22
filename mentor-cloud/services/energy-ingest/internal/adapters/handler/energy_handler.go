package handler

import (
	"encoding/json"
	"energy-ingest/internal/application"
	"energy-ingest/internal/domain"
	"energy-ingest/internal/metrics"
	"energy-ingest/internal/ports"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type EnergyHandler struct {
	svc *application.EnergyService
}

func RegisterRoutes(r *gin.RouterGroup, svc *application.EnergyService, jwtMW gin.HandlerFunc, apiKey string, scopeResolver ports.ScopeResolver) {
	h := &EnergyHandler{svc: svc}

	edge := r.Group("/api/v1/energy", ports.TokenAuth(scopeResolver, apiKey))
	edge.POST("/snapshots", h.ReceiveEnergy)

	dash := r.Group("/api/energy", jwtMW)
	dash.GET("/snapshots", h.QuerySnapshots)
	dash.GET("/meters", h.ListMeters)
	dash.GET("/meters/:id", h.GetMeter)
	dash.POST("/meters", h.CreateMeter)
	dash.PUT("/meters/:id", h.UpdateMeter)
	dash.DELETE("/meters/:id", h.DeleteMeter)
}

func (h *EnergyHandler) ReceiveEnergy(c *gin.Context) {
	// Prioridad: device_id resuelto por token en DB; fallback: X-Device-ID header.
	deviceID := ""
	if v, ok := c.Get("token_device_id"); ok {
		deviceID, _ = v.(string)
	}
	if deviceID == "" {
		deviceID = c.GetHeader("X-Device-ID")
	}
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id no pudo resolverse"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("read_body").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var records []domain.EnergyRecord
	if err := json.Unmarshal(body, &records); err != nil {
		metrics.ErrorsTotal.WithLabelValues("parse").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	start := time.Now()
	inserted, err := h.svc.ProcessBatch(c.Request.Context(), deviceID, records)
	if err != nil {
		metrics.ErrorsTotal.WithLabelValues("process").Inc()
		log.Printf("energy ingest error device=%s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.SnapshotsTotal.WithLabelValues(deviceID).Add(float64(inserted))
	metrics.BatchDuration.WithLabelValues(deviceID).Observe(time.Since(start).Seconds())
	c.JSON(http.StatusOK, gin.H{"received": inserted})
}

// empresaIDFromClaims extrae el empresa_id del JWT. Devuelve nil solo para superadmin
// (token sin empresa_id), lo que permite consultas cruzadas de empresas.
func empresaIDFromClaims(c *gin.Context) *int {
	claims, ok := c.MustGet("claims").(map[string]interface{})
	if !ok {
		return nil
	}
	v, ok := claims["empresa_id"].(float64)
	if !ok {
		return nil
	}
	id := int(v)
	return &id
}

func (h *EnergyHandler) QuerySnapshots(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// empresa_id se fuerza desde el JWT — nunca desde el query string.
	// Esto garantiza que cada usuario solo accede a los datos de su empresa.
	filter := ports.SnapshotFilter{
		DeviceID:  c.Query("device_id"),
		MeterID:   c.Query("meter_id"),
		From:      c.Query("from"),
		To:        c.Query("to"),
		EmpresaID: empresaIDFromClaims(c),
		Limit:     limit,
		Offset:    offset,
	}
	if v := c.Query("planta_id"); v != "" {
		id, _ := strconv.Atoi(v)
		filter.PlantaID = &id
	}

	data, total, err := h.svc.GetSnapshots(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": total})
}

func (h *EnergyHandler) ListMeters(c *gin.Context) {
	// empresa_id forzado desde JWT — mismo mecanismo que QuerySnapshots.
	empresaID := empresaIDFromClaims(c)

	meters, err := h.svc.GetMeters(c.Request.Context(), empresaID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": meters})
}

func (h *EnergyHandler) GetMeter(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	meter, err := h.svc.GetMeterByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "meter not found"})
		return
	}
	c.JSON(http.StatusOK, meter)
}

func (h *EnergyHandler) CreateMeter(c *gin.Context) {
	var req domain.MeterCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if req.DeviceID == "" || req.MeterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id and meter_id required"})
		return
	}
	if eid := empresaIDFromClaims(c); eid != nil {
		req.EmpresaID = eid
	}
	meter, err := h.svc.CreateMeter(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, meter)
}

func (h *EnergyHandler) UpdateMeter(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req domain.MeterUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	if err := h.svc.UpdateMeter(c.Request.Context(), id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *EnergyHandler) DeleteMeter(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.DeleteMeter(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
