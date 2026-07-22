package handler

import (
	"cloud-ingest/internal/application"
	"cloud-ingest/internal/domain"
	"cloud-ingest/internal/metrics"
	"cloud-ingest/internal/ports"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"mentor.local/shared/multitenancy"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type IngestHandler struct {
	svc      *application.IngestService
	scopes   ports.ScopeResolver
	notifier *edgeControlNotifier
}

type edgeControlNotifier struct {
	mu    sync.RWMutex
	conns map[*websocket.Conn]struct{}
}

func newEdgeControlNotifier() *edgeControlNotifier {
	return &edgeControlNotifier{conns: map[*websocket.Conn]struct{}{}}
}

func (n *edgeControlNotifier) add(c *websocket.Conn) {
	n.mu.Lock()
	n.conns[c] = struct{}{}
	n.mu.Unlock()
}

func (n *edgeControlNotifier) remove(c *websocket.Conn) {
	n.mu.Lock()
	delete(n.conns, c)
	n.mu.Unlock()
}

func (n *edgeControlNotifier) broadcast(v interface{}) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for c := range n.conns {
		_ = c.WriteJSON(v)
	}
}

func RegisterIngestRoutes(r *gin.RouterGroup, svc *application.IngestService, scopes ports.ScopeResolver, jwtMW gin.HandlerFunc, apiKey string) {
	h := &IngestHandler{svc: svc, scopes: scopes, notifier: newEdgeControlNotifier()}

	edge := r.Group("/api/v1/edge", ports.APIKeyAuth(apiKey))
	edge.POST("/oee", h.ReceiveOEE)
	edge.POST("/stops-sync", h.ReceiveStops)
	edge.POST("/production-runs-sync", h.ReceiveProductionRuns)
	edge.POST("/heartbeat", h.Heartbeat)
	edge.GET("/pending-commands", h.GetPendingCommands)
	edge.POST("/pending-commands/ack", h.AckPendingCommands)

	control := r.Group("/api/edge-control", jwtMW)
	control.POST("/energy/config", h.EnqueueEnergyConfig)
	control.POST("/energy/meters/upsert", h.EnqueueEnergyMeterUpsert)
	control.POST("/energy/meters/delete", h.EnqueueEnergyMeterDelete)
	control.POST("/energy/mc60/command", h.EnqueueEnergyMC60Command)
	control.GET("/ws", h.EdgeControlWS)

	r.GET("/datos-recibidos", jwtMW, h.ListRawEvents)
}

func (h *IngestHandler) ReceiveOEE(c *gin.Context) {
	deviceID, lineaID := readEdgeScope(c)
	scope, err := h.resolveScope(c.Request.Context(), deviceID, lineaID)
	if err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("oee").Inc()
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope not resolved: " + err.Error()})
		return
	}

	// Inyectar scope en el contexto para que PoolResolver resuelva el tenant correcto
	ctx := c.Request.Context()
	if scope.PlantaID != nil && scope.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{PlantaID: *scope.PlantaID, LineaID: *scope.LineaID})
	}
	start := time.Now()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("oee").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var records []domain.OEERecord
	if err := json.Unmarshal(body, &records); err != nil {
		var batch domain.IngestBatchRequest
		if err2 := json.Unmarshal(body, &batch); err2 != nil {
			metrics.IngestErrorsTotal.WithLabelValues("oee").Inc()
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		records = batch.Records
	}

	if err := h.svc.ProcessBatch(ctx, deviceID, records, scope); err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("oee").Inc()
		log.Printf("ingest OEE error device=%s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.OEERecordsTotal.WithLabelValues(deviceID).Add(float64(len(records)))
	metrics.OEEBatchDuration.WithLabelValues(deviceID).Observe(time.Since(start).Seconds())
	c.JSON(http.StatusOK, gin.H{"received": len(records)})
}

func (h *IngestHandler) ReceiveStops(c *gin.Context) {
	deviceID, lineaID := readEdgeScope(c)
	scope, err := h.resolveScope(c.Request.Context(), deviceID, lineaID)
	if err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("stops").Inc()
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope not resolved: " + err.Error()})
		return
	}

	// Inyectar scope en el contexto para que PoolResolver resuelva el tenant correcto
	ctx := c.Request.Context()
	if scope.PlantaID != nil && scope.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{PlantaID: *scope.PlantaID, LineaID: *scope.LineaID})
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("stops").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var stops []domain.StopRecord
	if err := json.Unmarshal(body, &stops); err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("stops").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: " + err.Error()})
		return
	}

	inserted, err := h.svc.ProcessStopsBatch(ctx, deviceID, stops, scope)
	if err != nil {
		log.Printf("stops-sync error device=%s: %v", deviceID, err)
		metrics.IngestErrorsTotal.WithLabelValues("stops").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.StopsTotal.WithLabelValues(deviceID).Add(float64(inserted))
	c.JSON(http.StatusOK, gin.H{"received": inserted})
}

func (h *IngestHandler) ReceiveProductionRuns(c *gin.Context) {
	deviceID, lineaID := readEdgeScope(c)
	scope, err := h.resolveScope(c.Request.Context(), deviceID, lineaID)
	if err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("production-runs").Inc()
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope not resolved: " + err.Error()})
		return
	}

	// Inyectar scope en el contexto para que PoolResolver resuelva el tenant correcto
	ctx := c.Request.Context()
	if scope.PlantaID != nil && scope.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{PlantaID: *scope.PlantaID, LineaID: *scope.LineaID})
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("production-runs").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	var runs []domain.ProductionRunRecord
	if err := json.Unmarshal(body, &runs); err != nil {
		metrics.IngestErrorsTotal.WithLabelValues("production-runs").Inc()
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: " + err.Error()})
		return
	}

	upserted, err := h.svc.ProcessProductionRunsBatch(ctx, deviceID, runs, scope)
	if err != nil {
		log.Printf("production-runs-sync error device=%s: %v", deviceID, err)
		metrics.IngestErrorsTotal.WithLabelValues("production-runs").Inc()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.ProductionRunsTotal.WithLabelValues(deviceID).Add(float64(upserted))
	c.JSON(http.StatusOK, gin.H{"received": upserted})
}

func (h *IngestHandler) Heartbeat(c *gin.Context) {
	deviceID, lineaID := readEdgeScope(c)
	if deviceID != "" {
		h.scopes.UpdateLastSeen(c.Request.Context(), deviceID)
	}
	// Resolver scope para devolver la config canónica al edge (auto-corrección)
	scope, err := h.resolveScope(c.Request.Context(), deviceID, lineaID)
	if err != nil || scope == nil {
		// OK aunque no se resuelva scope — el heartbeat no debe fallar
		c.JSON(http.StatusOK, gin.H{"device_id": deviceID})
		return
	}
	resp := gin.H{"device_id": deviceID}
	if scope.EmpresaID != nil {
		resp["empresa_id"] = *scope.EmpresaID
	}
	if scope.PlantaID != nil {
		resp["planta_id"] = *scope.PlantaID
	}
	if scope.LineaID != nil {
		resp["linea_id"] = *scope.LineaID
	}
	c.JSON(http.StatusOK, resp)
}

func (h *IngestHandler) ListRawEvents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := ports.RawEventsFilter{
		DeviceID:  c.Query("device_id"),
		EventType: c.Query("event_type"),
		From:      c.Query("from"),
		To:        c.Query("to"),
		Limit:     limit,
		Offset:    offset,
	}
	if v := c.Query("empresa_id"); v != "" {
		id, _ := strconv.Atoi(v)
		filter.EmpresaID = &id
	}
	if v := c.Query("planta_id"); v != "" {
		id, _ := strconv.Atoi(v)
		filter.PlantaID = &id
	}
	if v := c.Query("linea_id"); v != "" {
		id, _ := strconv.Atoi(v)
		filter.LineaID = &id
	}

	// Inyectar scope multitenancy para resolver el pool y schema correctos
	ctx := c.Request.Context()
	if filter.PlantaID != nil && filter.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{
			PlantaID: *filter.PlantaID,
			LineaID:  *filter.LineaID,
		})
	}

	events, total, err := h.svc.GetRawEvents(ctx, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  events,
		"total": total,
	})
}

func RegisterOEEQueryRoutes(r *gin.RouterGroup, repo ports.OEEQueryRepository, internalKey string) {
	r.GET("/internal/oee", func(c *gin.Context) {
		key := c.GetHeader("X-Internal-Key")
		if internalKey == "" || key != internalKey {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "acceso denegado"})
			return
		}
		filter := ports.OEEFilter{
			DeviceID:  c.Query("device_id"),
			FechaFrom: c.Query("from"),
			FechaTo:   c.Query("to"),
		}
		if v := c.Query("planta_id"); v != "" {
			id, _ := strconv.Atoi(v)
			filter.PlantaID = &id
		}
		if v := c.Query("empresa_id"); v != "" {
			id, _ := strconv.Atoi(v)
			filter.EmpresaID = &id
		}
		if v := c.Query("limit"); v != "" {
			filter.Limit, _ = strconv.Atoi(v)
		}

		snapshots, err := repo.GetSnapshots(c.Request.Context(), filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": snapshots})
	})
}

func readEdgeScope(c *gin.Context) (string, int) {
	deviceID := c.GetHeader("X-Device-ID")
	lineaID, _ := strconv.Atoi(c.GetHeader("X-Linea-ID"))
	return deviceID, lineaID
}

func (h *IngestHandler) resolveScope(ctx context.Context, deviceID string, lineaID int) (*domain.DeviceScope, error) {
	if deviceID != "" {
		go h.scopes.UpdateLastSeen(context.Background(), deviceID)
	}
	if lineaID > 0 {
		scope, err := h.scopes.ResolveByLinea(ctx, lineaID)
		if err != nil {
			return nil, fmt.Errorf("linea=%d: %w", lineaID, err)
		}
		return scope, nil
	}
	scope, err := h.scopes.ResolveByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("device=%s: %w", deviceID, err)
	}
	return scope, nil
}

// GetPendingCommands devuelve los comandos pendientes para que el Enviador los aplique en el edge.
func (h *IngestHandler) GetPendingCommands(c *gin.Context) {
	deviceID, lineaID := readEdgeScope(c)
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Device-ID requerido"})
		return
	}

	// Resolver scope para multitenancy (necesario para localizar el schema correcto)
	scope, err := h.resolveScope(c.Request.Context(), deviceID, lineaID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope not resolved: " + err.Error()})
		return
	}

	// Inyectar scope en el contexto para que PoolResolver use el schema del tenant
	ctx := c.Request.Context()
	if scope.PlantaID != nil && scope.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{PlantaID: *scope.PlantaID, LineaID: *scope.LineaID})
	}

	cmds, err := h.svc.GetPendingCommands(ctx, deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cmds)
}

// AckPendingCommands marca los comandos como aplicados en el edge.
func (h *IngestHandler) AckPendingCommands(c *gin.Context) {
	deviceID, lineaID := readEdgeScope(c)
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Device-ID requerido"})
		return
	}

	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids requeridos"})
		return
	}

	scope, err := h.resolveScope(c.Request.Context(), deviceID, lineaID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope not resolved: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	if scope.PlantaID != nil && scope.LineaID != nil {
		ctx = multitenancy.WithScope(ctx, multitenancy.Scope{PlantaID: *scope.PlantaID, LineaID: *scope.LineaID})
	}

	if err := h.svc.AckPendingCommands(ctx, deviceID, req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.notifier.broadcast(gin.H{
		"event":     "pending_commands_acked",
		"device_id": deviceID,
		"count":     len(req.IDs),
		"ids":       req.IDs,
		"ts":        time.Now().UTC().Format(time.RFC3339),
	})
	c.JSON(http.StatusOK, gin.H{"acked": len(req.IDs)})
}

func (h *IngestHandler) enqueueCommand(c *gin.Context, deviceID string, command string, payload map[string]interface{}) {
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id requerido"})
		return
	}

	scope, err := h.scopes.ResolveByDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope not resolved: " + err.Error()})
		return
	}
	if scope == nil || scope.PlantaID == nil || scope.LineaID == nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "scope incompleto para device_id"})
		return
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload invalido"})
		return
	}

	ctx := multitenancy.WithScope(c.Request.Context(), multitenancy.Scope{PlantaID: *scope.PlantaID, LineaID: *scope.LineaID})
	if err := h.svc.EnqueuePendingCommand(ctx, deviceID, command, payloadBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.notifier.broadcast(gin.H{
		"event":     "pending_command_enqueued",
		"device_id": deviceID,
		"command":   command,
		"payload":   payload,
		"ts":        time.Now().UTC().Format(time.RFC3339),
	})
	c.JSON(http.StatusCreated, gin.H{"status": "queued", "command": command, "device_id": deviceID})
}

func (h *IngestHandler) EnqueueEnergyConfig(c *gin.Context) {
	var req struct {
		DeviceID string            `json:"device_id"`
		Values   map[string]string `json:"values"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalido"})
		return
	}
	allowed := map[string]bool{
		"device_id": true, "cloud_url": true, "energy_api_key": true,
		"send_interval_s": true, "batch_size": true, "config_reload_s": true,
		"write_api_url": true, "command_poll_s": true,
	}
	clean := map[string]string{}
	for k, v := range req.Values {
		if allowed[k] {
			clean[k] = v
		}
	}
	if len(clean) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "values vacio o sin keys permitidas"})
		return
	}
	h.enqueueCommand(c, req.DeviceID, "energy_config_upsert", map[string]interface{}{"values": clean})
}

func (h *IngestHandler) EnqueueEnergyMeterUpsert(c *gin.Context) {
	var req struct {
		DeviceID  string  `json:"device_id"`
		MeterID   string  `json:"meter_id"`
		UnitID    int     `json:"unit_id"`
		Ubicacion *string `json:"ubicacion"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalido"})
		return
	}
	if req.MeterID == "" || req.UnitID < 1 || req.UnitID > 247 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "meter_id y unit_id (1-247) requeridos"})
		return
	}
	h.enqueueCommand(c, req.DeviceID, "energy_meter_upsert", map[string]interface{}{
		"meter_id": req.MeterID, "unit_id": req.UnitID, "ubicacion": req.Ubicacion,
	})
}

func (h *IngestHandler) EnqueueEnergyMeterDelete(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id"`
		MeterID  string `json:"meter_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalido"})
		return
	}
	if req.MeterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "meter_id requerido"})
		return
	}
	h.enqueueCommand(c, req.DeviceID, "energy_meter_delete", map[string]interface{}{"meter_id": req.MeterID})
}

func (h *IngestHandler) EnqueueEnergyMC60Command(c *gin.Context) {
	var req struct {
		DeviceID string                 `json:"device_id"`
		UnitID   int                    `json:"unit_id"`
		Command  string                 `json:"command"`
		Params   map[string]interface{} `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalido"})
		return
	}
	allowed := map[string]bool{"set-ct": true, "set-sys": true, "set-dir": true, "set-time": true, "reset": true, "scan": true}
	if !allowed[req.Command] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "command no permitido"})
		return
	}
	if req.Command != "scan" && (req.UnitID < 1 || req.UnitID > 247) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unit_id (1-247) requerido"})
		return
	}
	h.enqueueCommand(c, req.DeviceID, "energy_mc60_command", map[string]interface{}{
		"unit_id": req.UnitID, "command": req.Command, "params": req.Params,
	})
}

func (h *IngestHandler) EdgeControlWS(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	h.notifier.add(conn)
	defer func() {
		h.notifier.remove(conn)
		_ = conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
