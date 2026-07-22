package handler

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type StopHandler struct {
	repo       ports.StopRepository
	dispatcher ports.CommandDispatcher
	notifier   ports.BrowserNotifier
}

func RegisterStopRoutes(rg *gin.RouterGroup, repo ports.StopRepository, dispatcher ports.CommandDispatcher, notifier ports.BrowserNotifier, mw gin.HandlerFunc) {
	h := &StopHandler{repo: repo, dispatcher: dispatcher, notifier: notifier}
	g := rg.Group("/stops", mw)
	g.GET("", h.List)
	g.GET("/summary", h.Summary)
	g.POST("", h.Create)
	g.PUT("/:stopId", h.Update)
	g.POST("/:stopId/justify", h.Justify)
	g.POST("/:stopId/close", h.Close)
	g.DELETE("/:stopId", h.Delete)
}

func (h *StopHandler) List(c *gin.Context) {
	f := ports.StopFilter{
		Limit: 200,
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
	if v := c.Query("device_id"); v != "" {
		f.DeviceID = v
	}
	if v := c.Query("justified"); v != "" {
		b := v == "true"
		f.Justified = &b
	}
	if v := c.Query("open"); v != "" {
		b := v == "true"
		f.Open = &b
	}
	if v := c.Query("stop_type"); v != "" {
		f.StopType = &v
	}
	if v := c.Query("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Since = &t
		}
	}
	if v := c.Query("until"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.Until = &t
		}
	}
	if v := c.Query("limit"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 && i <= 1000 {
			f.Limit = i
		}
	}

	stops, err := h.repo.ListStops(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stops)
}

func (h *StopHandler) Summary(c *gin.Context) {
	f := ports.StopFilter{}

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

	hours := 24
	if v := c.Query("hours"); v != "" {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			hours = i
		}
	}
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	f.Since = &since

	summary, err := h.repo.StopSummary(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// notifyBrowserAsync pushes a stop.changed event to connected browser clients (best-effort).
func (h *StopHandler) notifyBrowserAsync(empresaID int64, lineaID int64) {
	if h.notifier == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.notifier.Notify(ctx, "stop.changed", empresaID, lineaID); err != nil {
			log.Printf("[stops] browser notify falló: %v", err)
		}
	}()
}

// Create creates a new stop in analytics.paradas and dispatches a command to the edge.
func (h *StopHandler) Create(c *gin.Context) {
	var req domain.CreateCloudStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get JWT claims for empresa_id and user_id
	empresaID := extractIntClaim(c, "empresa_id")
	userID := extractIntClaim(c, "user_id")
	if req.EmpresaID == 0 {
		req.EmpresaID = int(empresaID)
	}

	// 1. Resolve device_id before writing so the DB record has it
	if req.DeviceID == "" && req.LineaID != nil {
		if resolved, err := h.repo.ResolveDeviceID(c.Request.Context(), *req.LineaID); err == nil {
			req.DeviceID = resolved
		}
	}

	// 2. Write record in analytics.paradas (with device_id populated)
	stop, err := h.repo.CreateStop(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 3. Dispatch command to edge (async, best-effort via SSE)
	if h.dispatcher != nil && stop.StopID != nil && req.DeviceID != "" {
		payload := map[string]interface{}{
			"stop_id":       *stop.StopID,
			"stop_type":     req.StopType,
			"source":        "cloud",
			"cloud_stop_id": stop.ID,
			"started_at":    stop.Inicio.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if req.Category != nil {
			payload["category"] = *req.Category
		}
		if req.CategoriaID != nil {
			payload["categoria_id"] = *req.CategoriaID
		}
		if req.Reason != nil {
			payload["reason"] = *req.Reason
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.dispatcher.Dispatch(ctx, req.DeviceID,
				"crear_parada", payload, userID, empresaID); err != nil {
				log.Printf("[stops] dispatch crear_parada falló: %v", err)
			}
		}()
	}

	if stop.LineaID != nil {
		h.notifyBrowserAsync(empresaID, int64(*stop.LineaID))
	}
	c.JSON(http.StatusCreated, stop)
}

// Update modifies arbitrary mutable fields of a stop and dispatches MODIFICAR_PARADA to the edge.
func (h *StopHandler) Update(c *gin.Context) {
	stopID := strings.TrimPrefix(c.Param("stopId"), "/")
	if stopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stopId requerido"})
		return
	}

	var req domain.UpdateCloudStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	empresaID := extractIntClaim(c, "empresa_id")
	userID := extractIntClaim(c, "user_id")

	stop, err := h.repo.UpdateStop(c.Request.Context(), stopID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.dispatcher != nil {
		devID := stop.DeviceID
		if devID == "" && stop.LineaID != nil {
			if resolved, err := h.repo.ResolveDeviceID(c.Request.Context(), *stop.LineaID); err == nil {
				devID = resolved
			}
		}
		if devID != "" {
			payload := map[string]interface{}{
				"stop_id": stopID,
			}
			if req.StopType != nil {
				payload["new_type"] = *req.StopType
			}
			if req.Category != nil {
				payload["category"] = *req.Category
			}
			if req.CategoriaID != nil {
				payload["categoria_id"] = *req.CategoriaID
			}
			if req.Reason != nil {
				payload["reason"] = *req.Reason
			}
			if req.StartedAt != nil {
				payload["started_at"] = req.StartedAt.UTC().Format("2006-01-02T15:04:05Z")
			}
			if req.EndedAt != nil {
				payload["ended_at"] = req.EndedAt.UTC().Format("2006-01-02T15:04:05Z")
			}
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := h.dispatcher.Dispatch(ctx, devID,
					"modificar_parada", payload, userID, empresaID); err != nil {
					log.Printf("[stops] dispatch modificar_parada falló: %v", err)
				}
			}()
		}
	}

	if stop.LineaID != nil {
		h.notifyBrowserAsync(empresaID, int64(*stop.LineaID))
	}
	c.JSON(http.StatusOK, stop)
}

// Justify justifies a stop in analytics.paradas and dispatches a command to the edge.
func (h *StopHandler) Justify(c *gin.Context) {
	stopID := c.Param("stopId")
	if stopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stopId requerido"})
		return
	}
	// Handle path prefix if gin adds a leading /
	stopID = strings.TrimPrefix(stopID, "/")

	var req domain.JustifyCloudStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	empresaID := extractIntClaim(c, "empresa_id")
	userID := extractIntClaim(c, "user_id")

	// 1. Update analytics.paradas immediately
	stop, err := h.repo.JustifyStop(c.Request.Context(), stopID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 2. Dispatch justificar_parada to edge (async, best-effort)
	deviceID := stop.DeviceID
	if deviceID == "" && stop.LineaID != nil {
		if resolved, err := h.repo.ResolveDeviceID(c.Request.Context(), *stop.LineaID); err == nil {
			deviceID = resolved
		}
	}
	if h.dispatcher != nil && deviceID != "" {
		payload := map[string]interface{}{
			"stop_id":  stopID,
			"reason":   req.Reason,
			"category": req.Category,
		}
		if req.StopType != "" {
			payload["stop_type"] = req.StopType
		}
		if req.CategoriaID != nil {
			payload["categoria_id"] = *req.CategoriaID
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.dispatcher.Dispatch(ctx, deviceID,
				"justificar_parada", payload, userID, empresaID); err != nil {
				log.Printf("[stops] dispatch justificar_parada falló: %v", err)
			}
		}()
	}

	if stop.LineaID != nil {
		h.notifyBrowserAsync(empresaID, int64(*stop.LineaID))
	}
	c.JSON(http.StatusOK, stop)
}

// Close sets ended_at on the stop and dispatches cerrar_parada to the edge.
func (h *StopHandler) Close(c *gin.Context) {
	stopID := strings.TrimPrefix(c.Param("stopId"), "/")
	if stopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stopId requerido"})
		return
	}

	empresas := extractIntClaim(c, "empresa_id")
	userID := extractIntClaim(c, "user_id")

	now := time.Now().UTC()
	stop, err := h.repo.CloseStop(c.Request.Context(), stopID, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.dispatcher != nil {
		payload := map[string]interface{}{
			"stop_id":  stopID,
			"ended_at": now.Format("2006-01-02T15:04:05Z"),
		}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.dispatcher.Dispatch(ctx, stop.DeviceID, "cerrar_parada", payload, userID, empresas); err != nil {
				log.Printf("[stops] dispatch cerrar_parada falló: %v", err)
			}
		}()
	}

	if stop.LineaID != nil {
		h.notifyBrowserAsync(empresas, int64(*stop.LineaID))
	}
	c.JSON(http.StatusOK, stop)
}

// Delete removes a stop from cloud DB and dispatches eliminar_parada to the edge.
func (h *StopHandler) Delete(c *gin.Context) {
	stopID := strings.TrimPrefix(c.Param("stopId"), "/")
	if stopID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stopId requerido"})
		return
	}

	empresas := extractIntClaim(c, "empresa_id")
	userID := extractIntClaim(c, "user_id")

	// Resolve device_id before deleting
	existing, _ := h.repo.GetStop(c.Request.Context(), stopID)

	if err := h.repo.DeleteStop(c.Request.Context(), stopID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.dispatcher != nil && existing != nil && existing.DeviceID != "" {
		payload := map[string]interface{}{"stop_id": stopID}
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := h.dispatcher.Dispatch(ctx, existing.DeviceID, "eliminar_parada", payload, userID, empresas); err != nil {
				log.Printf("[stops] dispatch eliminar_parada falló: %v", err)
			}
		}()
	}

	if existing != nil && existing.LineaID != nil {
		h.notifyBrowserAsync(empresas, int64(*existing.LineaID))
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true, "stop_id": stopID})
}

func extractIntClaim(c *gin.Context, key string) int64 {
	v, exists := c.Get(key)
	if !exists {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int64:
		return val
	case int:
		return int64(val)
	default:
		return 0
	}
}
