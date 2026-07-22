package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"edge-gateway/internal/application"
	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
)

type LineContext struct {
	LineaID      int
	CloudLineaID int
	Gateway      *application.GatewayService
	Commands     *application.CommandService
	CatalogSync  ports.CatalogSyncRepository
}

type HTTPServer struct {
	mux            *http.ServeMux
	server         *http.Server
	db             *sql.DB
	status         *application.StatusService
	broker         ports.SSEBroker
	deviceID       string
	authToken      string
	empresaID      int
	cloudURL       string
	cloudKey       string
	lines          map[int]*LineContext
	cloudLineas    map[int]*LineContext
	defaultLineaID int
	sysMetrics     *SystemMetrics
	sysMetricsMu   sync.RWMutex
}

type HTTPServerConfig struct {
	Port      int
	DeviceID  string
	AuthToken string
	EmpresaID int
	CloudURL  string
	CloudKey  string
}

func NewHTTPServer(
	cfg HTTPServerConfig,
	db *sql.DB,
	lines map[int]*LineContext,
	cloudLineas map[int]*LineContext,
	defaultLineaID int,
	status *application.StatusService,
	broker ports.SSEBroker,
) *HTTPServer {
	s := &HTTPServer{
		mux:            http.NewServeMux(),
		db:             db,
		status:         status,
		broker:         broker,
		deviceID:       cfg.DeviceID,
		authToken:      cfg.AuthToken,
		empresaID:      cfg.EmpresaID,
		cloudURL:       cfg.CloudURL,
		cloudKey:       cfg.CloudKey,
		lines:          lines,
		cloudLineas:    cloudLineas,
		defaultLineaID: defaultLineaID,
	}

	s.registerRoutes()
	s.startMetricsPoller()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.corsMiddleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

func (s *HTTPServer) Start() error {
	log.Printf("[http] edge-gateway listening on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) resolveLineContext(r *http.Request) *LineContext {
	if lid := r.URL.Query().Get("linea_id"); lid != "" {
		if id, err := strconv.Atoi(lid); err == nil {
			// 1. Coincidencia directa por linea_id local
			if lc, ok := s.lines[id]; ok {
				return lc
			}
			// 2. Coincidencia por cloud_linea_id (la tablet manda el ID del cloud)
			if lc, ok := s.cloudLineas[id]; ok {
				return lc
			}
		}
	}
	return s.lines[s.defaultLineaID]
}

func (s *HTTPServer) gatewayFor(r *http.Request) *application.GatewayService {
	return s.resolveLineContext(r).Gateway
}

func (s *HTTPServer) commandsFor(r *http.Request) *application.CommandService {
	return s.resolveLineContext(r).Commands
}

func (s *HTTPServer) catalogSyncFor(r *http.Request) ports.CatalogSyncRepository {
	return s.resolveLineContext(r).CatalogSync
}

func (s *HTTPServer) lineaIDFor(r *http.Request) int {
	return s.resolveLineContext(r).LineaID
}

// cloudLineaIDFor devuelve el cloud_linea_id si está mapeado, o el local linea_id como fallback.
// Los datos de catálogo (productos, variables) llegan del cloud con IDs de nube.
func (s *HTTPServer) cloudLineaIDFor(r *http.Request) int {
	lc := s.resolveLineContext(r)
	if lc.CloudLineaID > 0 {
		return lc.CloudLineaID
	}
	return lc.LineaID
}

func (s *HTTPServer) defaultLine() *LineContext {
	return s.lines[s.defaultLineaID]
}

func (s *HTTPServer) registerRoutes() {
	// Auth
	s.mux.HandleFunc("/edge/auth/operators", s.handleOperators)
	s.mux.HandleFunc("/edge/auth/login", s.handleLogin)

	// Health and status
	s.mux.HandleFunc("/edge/health", s.handleHealth)
	s.mux.HandleFunc("/edge/status", s.handleStatus)
	s.mux.HandleFunc("/edge/cloud/ping", s.handleCloudPing)

	// Config (proxy to edge-config-service)
	s.mux.HandleFunc("/edge/config", s.handleConfig)
	s.mux.HandleFunc("/edge/calibration/start", s.handleCalibration)
	s.mux.HandleFunc("/edge/calibration/status", s.handleCalibrationStatus)

	// Buffer (proxy to resiliencia)
	s.mux.HandleFunc("/edge/buffer/summary", s.handleBufferSummary)
	s.mux.HandleFunc("/edge/events/recent", s.handleRecentEvents)
	s.mux.HandleFunc("/edge/events/pending", s.handlePendingEvents)

	// Stops
	s.mux.HandleFunc("/edge/stops", s.handleStops)
	s.mux.HandleFunc("/edge/stops/", s.handleStopByID)
	s.mux.HandleFunc("/edge/stops/summary", s.handleStopsSummary)
	s.mux.HandleFunc("/edge/stops/duration-by-type", s.handleStopDurationByType)
	s.mux.HandleFunc("/edge/current-turno", s.handleCurrentTurno)

	// Catalogs
	s.mux.HandleFunc("/edge/catalogs/stop-categories", s.handleStopCategories)
	s.mux.HandleFunc("/edge/catalogs/products", s.handleProducts)
	s.mux.HandleFunc("/edge/catalogs/velocidad-nominal", s.handleVelocidadNominal)
	s.mux.HandleFunc("/edge/catalogs/velocidad-nominal/log", s.handleVelocidadNominalLog)
	s.mux.HandleFunc("/edge/catalogs/velocidad-nominal/motivos", s.handleVelocidadNominalMotivos)
	s.mux.HandleFunc("/edge/catalogs/plants-lines", s.handlePlantsLines)
	s.mux.HandleFunc("/edge/catalogs/product-characteristics", s.handleProductCharacteristics)
	s.mux.HandleFunc("/edge/catalogs/sync/categorias", s.handleSyncCategorias)
	s.mux.HandleFunc("/edge/catalogs/sync/productos", s.handleSyncProductos)
	s.mux.HandleFunc("/edge/catalogs/sync/turnos", s.handleSyncTurnos)

	// Variables (OEE vars incluyendo MERMA)
	s.mux.HandleFunc("/edge/variables", s.handleVariables)
	s.mux.HandleFunc("/edge/variables/", s.handleVariableByKey)

	// Production runs
	s.mux.HandleFunc("/edge/production-runs", s.handleProductionRuns)
	s.mux.HandleFunc("/edge/production-runs/", s.handleProductionRunByID)

	// Commands
	s.mux.HandleFunc("/edge/commands", s.handleCommands)
	s.mux.HandleFunc("/edge/commands/", s.handleCommandByID)

	// SSE stream
	s.mux.HandleFunc("/edge/stream", s.handleSSE)

	// Camera health check, snapshot y stream MJPEG
	s.mux.HandleFunc("/edge/camera/health", s.handleCameraHealth)
	s.mux.HandleFunc("/edge/camera/snapshot", s.handleCameraSnapshot)
	s.mux.HandleFunc("/edge/camera/stream", s.handleCameraStream)

	// Métricas de hardware del Jetson (CPU, RAM, disco, GPU, temperaturas)
	s.mux.HandleFunc("/edge/system", s.handleSystemMetrics)

	// Provision — crear schema para una línea
	s.mux.HandleFunc("/edge/provision", s.handleProvisionLine)
	s.mux.HandleFunc("/edge/schemas", s.handleListSchemas)

	// Health (simple alias)
	s.mux.HandleFunc("/health", s.handleSimpleHealth)
}

// ---------- Health ----------

// ---------- Auth ----------

func (s *HTTPServer) handleOperators(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	users, err := s.catalogSyncFor(r).ListUsuarios(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	type operatorDTO struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Nombre   string `json:"nombre"`
		Apellido string `json:"apellido"`
		Rol      string `json:"rol"`
		Activo   bool   `json:"activo"`
	}
	out := make([]operatorDTO, 0, len(users))
	for _, u := range users {
		if u.Activo {
			out = append(out, operatorDTO{ID: u.ID, Username: u.Username, Nombre: u.Nombre, Apellido: u.Apellido, Rol: u.Rol, Activo: true})
		}
	}
	s.jsonResponse(w, http.StatusOK, out)
}

func (s *HTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w)
		return
	}
	var body struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Username == "" {
		s.badRequest(w, "username required")
		return
	}
	users, err := s.catalogSyncFor(r).ListUsuarios(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	for _, u := range users {
		if u.Activo && strings.EqualFold(u.Username, body.Username) {
			s.jsonResponse(w, http.StatusOK, map[string]interface{}{
				"ok":         true,
				"id":         u.ID,
				"username":   u.Username,
				"nombre":     u.Nombre,
				"rol":        u.Rol,
				"empresa_id": u.EmpresaID,
				"device_id":  s.deviceID,
			})
			return
		}
	}
	s.jsonResponse(w, http.StatusUnauthorized, map[string]interface{}{
		"ok":    false,
		"error": "usuario no encontrado",
	})
}

// handlePlantsLines devuelve plantas y líneas.
// Primero intenta la DB local (plantas/lineas, cargadas vía SYNC_PLANTAS_LINEAS).
// Si no hay datos locales, hace proxy al cloud como fallback.
// GET /edge/catalogs/plants-lines
func (s *HTTPServer) handlePlantsLines(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	// ── Intentar desde DB local ─────────────────────────────────────────
	plantas, err := s.catalogSyncFor(r).ListPlantas(r.Context())
	if err == nil && len(plantas) > 0 {
		lineas, _ := s.catalogSyncFor(r).ListLineas(r.Context())
		if lineas == nil {
			lineas = []domain.Linea{}
		}
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"plantas": plantas,
			"lineas":  lineas,
			"source":  "local",
		})
		return
	}

	// ── Fallback: proxy al cloud ────────────────────────────────────────
	if s.cloudURL == "" {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"plantas": []interface{}{},
			"lineas":  []interface{}{},
		})
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	cloudEndpoint := strings.TrimRight(s.cloudURL, "/") + "/api/v1/edge/plants-lines"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, cloudEndpoint, nil)
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"plantas": []interface{}{}, "lineas": []interface{}{}})
		return
	}
	req.Header.Set("Authorization", "Bearer "+s.cloudKey)
	req.Header.Set("X-Device-ID", s.deviceID)

	resp, err := client.Do(req)
	if err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"plantas": []interface{}{}, "lineas": []interface{}{}})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// ---------- Health (aggregated) ----------

func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	health, err := s.gatewayFor(r).AggregatedHealth(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, health)
}

func (s *HTTPServer) handleSimpleHealth(w http.ResponseWriter, r *http.Request) {
	s.jsonResponse(w, http.StatusOK, map[string]string{
		"service": "edge-gateway",
		"status":  "ok",
	})
}

// handleCloudPing prueba la conectividad al servidor cloud y mide la latencia.
// Usa la URL y key configuradas, o acepta override via query params ?url=... &key=...
func (s *HTTPServer) handleCloudPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		targetURL = s.cloudURL
	}
	apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		apiKey = s.cloudKey
	}

	type PingResult struct {
		Reachable  bool   `json:"reachable"`
		LatencyMs  int64  `json:"latency_ms"`
		StatusCode int    `json:"status_code"`
		Error      string `json:"error,omitempty"`
		TestedAt   string `json:"tested_at"`
		TestedURL  string `json:"tested_url"`
	}

	result := PingResult{
		TestedAt: time.Now().Format(time.RFC3339),
	}

	if targetURL == "" {
		result.Error = "cloud URL no configurada"
		result.TestedURL = ""
		s.jsonResponse(w, http.StatusOK, result)
		return
	}

	pingURL := strings.TrimRight(targetURL, "/") + "/health"
	result.TestedURL = pingURL

	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, pingURL, nil)
	if err != nil {
		result.Error = "URL invalida: " + err.Error()
		s.jsonResponse(w, http.StatusOK, result)
		return
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("X-Device-ID", s.deviceID)

	start := time.Now()
	resp, err := client.Do(req)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		s.jsonResponse(w, http.StatusOK, result)
		return
	}
	defer resp.Body.Close()

	result.Reachable = true
	result.StatusCode = resp.StatusCode
	s.jsonResponse(w, http.StatusOK, result)
}

func (s *HTTPServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	st, err := s.status.GetStatus(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	st.LineaID = s.lineaIDFor(r)
	s.jsonResponse(w, http.StatusOK, st)
}

// ---------- Config ----------

func (s *HTTPServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := s.gatewayFor(r).GetConfig(r.Context())
		if err != nil {
			s.internalError(w, err)
			return
		}
		s.jsonResponse(w, http.StatusOK, cfg)

	case http.MethodPut:
		var patch map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
			s.badRequest(w, "invalid JSON body")
			return
		}
		actor := s.extractActor(r)
		result, err := s.gatewayFor(r).UpdateConfig(r.Context(), patch, actor)
		if err != nil {
			s.internalError(w, err)
			return
		}
		s.jsonResponse(w, http.StatusOK, result)

	default:
		s.methodNotAllowed(w)
	}
}

func (s *HTTPServer) handleCalibration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w)
		return
	}
	actor := s.extractActor(r)
	if err := s.gatewayFor(r).StartCalibration(r.Context(), actor); err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusAccepted, map[string]string{"status": "calibration_started"})
}

func (s *HTTPServer) handleCalibrationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	status, err := s.gatewayFor(r).CalibrationStatus(r.Context())
	if err != nil {
		// El detector puede no estar disponible; devolver estado seguro
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{"active": false, "progress": 0.0})
		return
	}
	s.jsonResponse(w, http.StatusOK, status)
}

// ---------- Buffer ----------

func (s *HTTPServer) handleBufferSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	summary, err := s.gatewayFor(r).BufferSummary(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, summary)
}

func (s *HTTPServer) handleRecentEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	limit := s.queryInt(r, "limit", 50)
	var since *time.Time
	if v := r.URL.Query().Get("since"); v != "" {
		var t time.Time
		var err error
		// JS toISOString() incluye milisegundos (RFC3339Nano); intentar ambos formatos
		if t, err = time.Parse(time.RFC3339Nano, v); err != nil {
			t, err = time.Parse(time.RFC3339, v)
		}
		if err == nil {
			since = &t
		}
	}
	events, err := s.gatewayFor(r).RecentEvents(r.Context(), limit, since)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, events)
}

func (s *HTTPServer) handlePendingEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	limit := s.queryInt(r, "limit", 200)
	events, err := s.gatewayFor(r).PendingEvents(r.Context(), limit)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, events)
}

// ---------- Stops ----------

func (s *HTTPServer) handleStops(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := domain.StopFilter{
			Limit:  s.queryInt(r, "limit", 100),
			Offset: s.queryInt(r, "offset", 0),
		}
		// On the edge there is only one physical device so we do NOT filter by
		// device_id — the stops table may contain rows that arrived from the
		// cloud with a different device_id string than what is set in DEVICE_ID.

		if v := r.URL.Query().Get("justified"); v != "" {
			b := v == "true"
			filter.Justified = &b
		}
		if v := r.URL.Query().Get("open"); v != "" {
			b := v == "true"
			filter.Open = &b
		}
		if v := r.URL.Query().Get("stop_type"); v != "" {
			filter.StopType = &v
		}
		if v := r.URL.Query().Get("since"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				filter.Since = &t
			}
		}
		if v := r.URL.Query().Get("until"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				filter.Until = &t
			}
		}

		stops, err := s.gatewayFor(r).ListStops(r.Context(), filter)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if stops == nil {
			stops = []domain.Stop{}
		}
		s.jsonResponse(w, http.StatusOK, stops)

	case http.MethodPost:
		var req domain.CreateStopRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.badRequest(w, "invalid JSON body")
			return
		}
		actor := s.extractActor(r)
		stop, err := s.gatewayFor(r).CreateStop(r.Context(), req, actor)
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		s.jsonResponse(w, http.StatusCreated, stop)

	default:
		s.methodNotAllowed(w)
	}
}

func (s *HTTPServer) handleStopByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/edge/stops/")
	parts := strings.Split(path, "/")
	stopID := parts[0]

	if stopID == "summary" {
		s.handleStopsSummary(w, r)
		return
	}

	if len(parts) > 1 && parts[1] == "justify" {
		s.handleJustifyStop(w, r, stopID)
		return
	}
	if len(parts) > 1 && parts[1] == "close" {
		s.handleCloseStop(w, r, stopID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		stop, err := s.gatewayFor(r).GetStop(r.Context(), stopID)
		if err == domain.ErrStopNotFound {
			s.notFound(w, "stop not found")
			return
		}
		if err != nil {
			s.internalError(w, err)
			return
		}
		s.jsonResponse(w, http.StatusOK, stop)

	case http.MethodDelete:
		actor := s.extractActor(r)
		if err := s.gatewayFor(r).DeleteStop(r.Context(), stopID, actor); err != nil {
			if err == domain.ErrStopNotFound {
				s.notFound(w, "stop not found")
				return
			}
			s.internalError(w, err)
			return
		}
		s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted", "stop_id": stopID})

	default:
		s.methodNotAllowed(w)
	}
}

func (s *HTTPServer) handleJustifyStop(w http.ResponseWriter, r *http.Request, stopID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w)
		return
	}
	var req domain.JustifyStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.badRequest(w, "invalid JSON body")
		return
	}
	req.StopID = stopID
	actor := s.extractActor(r)
	if req.JustifiedBy == "" {
		req.JustifiedBy = actor
	}

	stop, err := s.gatewayFor(r).JustifyStop(r.Context(), req, actor)
	if err == domain.ErrStopNotFound {
		s.notFound(w, "stop not found")
		return
	}
	if err != nil {
		s.badRequest(w, err.Error())
		return
	}
	s.jsonResponse(w, http.StatusOK, stop)
}

func (s *HTTPServer) handleCloseStop(w http.ResponseWriter, r *http.Request, stopID string) {
	if r.Method != http.MethodPost {
		s.methodNotAllowed(w)
		return
	}
	actor := s.extractActor(r)
	stop, err := s.gatewayFor(r).CloseStop(r.Context(), stopID, actor)
	if err == domain.ErrStopNotFound {
		s.notFound(w, "stop not found or already closed")
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, stop)
}

func (s *HTTPServer) handleStopsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	hours := s.queryInt(r, "hours", 24)
	summary, err := s.gatewayFor(r).StopsSummary(r.Context(), hours)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, summary)
}

// GET /edge/stops/duration-by-type?interval_s=60
// Returns total justified stop duration in ms per OEE variable key for the
// given time window. Used by vision-event-detector to enrich OEE snapshots.
func (s *HTTPServer) handleStopDurationByType(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	intervalS := s.queryInt(r, "interval_s", 60)
	data, err := s.gatewayFor(r).StopDurationByType(r.Context(), intervalS)
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, data)
}

// GET /edge/current-turno
// Returns the active turno nombre at the current server time, or "" if none.
func (s *HTTPServer) handleCurrentTurno(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	nombre := s.gatewayFor(r).CurrentTurnoName(r.Context())
	s.jsonResponse(w, http.StatusOK, map[string]string{"nombre": nombre})
}

// ---------- Catalogs ----------

func (s *HTTPServer) handleStopCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	flat, err := s.catalogSyncFor(r).ListCategorias(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	tree := domain.BuildCategoryTree(flat)
	if tree == nil {
		tree = []*domain.CategoryTreeNode{}
	}
	s.jsonResponse(w, http.StatusOK, tree)
}

// handleVelocidadNominal expone los datos de velocidad nominal del catálogo local.
//
// GET /edge/catalogs/velocidad-nominal
//
//	→ { data: [{producto_id, sku, descripcion, velocidad_us, factor_conv}] }
//
// PUT /edge/catalogs/velocidad-nominal
//
//	body: [{producto_id, velocidad_us, factor_conv}]  (o un objeto único)
//	→ Actualiza shared.velocidad_nominal en el edge (efecto inmediato en OEE)
//	→ Propaga best-effort al cloud (PUT /api/velocidad-nominal) si hay conectividad
func (s *HTTPServer) handleVelocidadNominal(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleVelocidadNominalGet(w, r)
	case http.MethodPut:
		s.handleVelocidadNominalPut(w, r)
	default:
		s.methodNotAllowed(w)
	}
}

func (s *HTTPServer) handleVelocidadNominalGet(w http.ResponseWriter, r *http.Request) {
	prods, err := s.catalogSyncFor(r).ListProductos(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	vns, err := s.catalogSyncFor(r).ListVelocidadNominal(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}

	vnMap := make(map[int]domain.VelocidadNominal, len(vns))
	for _, vn := range vns {
		vnMap[vn.ProductoID] = vn
	}

	type Row struct {
		ProductoID  int     `json:"producto_id"`
		SKU         string  `json:"sku"`
		Descripcion string  `json:"descripcion"`
		VelocidadUS float64 `json:"velocidad_us"`
		FactorConv  int     `json:"factor_conv"`
	}

	localLid := s.lineaIDFor(r)
	out := make([]Row, 0, len(prods))
	for _, p := range prods {
		if !p.Activo {
			continue
		}
		if localLid > 0 && (p.LineaID == nil || *p.LineaID != localLid) {
			continue
		}
		vn := vnMap[p.ID]
		fc := vn.FactorConv
		if fc <= 0 {
			fc = 1
		}
		out = append(out, Row{
			ProductoID:  p.ID,
			SKU:         p.Codigo,
			Descripcion: p.Nombre,
			VelocidadUS: vn.VelocidadUs,
			FactorConv:  fc,
		})
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"data": out})
}

func (s *HTTPServer) handleVelocidadNominalPut(w http.ResponseWriter, r *http.Request) {
	type Item struct {
		ProductoID  int     `json:"producto_id"`
		VelocidadUS float64 `json:"velocidad_us"`
		FactorConv  int     `json:"factor_conv"`
		Motivo      string  `json:"motivo,omitempty"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.badRequest(w, "error leyendo body")
		return
	}

	var items []Item
	if json.Unmarshal(body, &items) != nil || len(items) == 0 {
		var single Item
		if json.Unmarshal(body, &single) != nil || single.ProductoID == 0 {
			s.badRequest(w, "payload inválido: se requiere producto_id")
			return
		}
		items = []Item{single}
	}

	repo := s.catalogSyncFor(r)
	actor := s.extractActor(r)

	// Leer valores previos para el log
	prevMap := make(map[int]domain.VelocidadNominal)
	if prev, err := repo.ListVelocidadNominal(r.Context()); err == nil {
		for _, v := range prev {
			prevMap[v.ProductoID] = v
		}
	}

	// Leer SKUs para el log
	skuMap := make(map[int]string)
	if prods, err := repo.ListProductos(r.Context()); err == nil {
		for _, p := range prods {
			skuMap[p.ID] = p.Codigo
		}
	}

	motivoMap := make(map[int]string)
	for _, it := range items {
		if it.Motivo != "" {
			motivoMap[it.ProductoID] = it.Motivo
		}
	}

	records := make([]domain.VelocidadNominal, len(items))
	for i, it := range items {
		fc := it.FactorConv
		if fc <= 0 {
			fc = 1
		}
		records[i] = domain.VelocidadNominal{
			ProductoID:  it.ProductoID,
			VelocidadUs: it.VelocidadUS,
			FactorConv:  fc,
		}
	}

	if err := repo.UpsertVelocidadNominalItems(r.Context(), records); err != nil {
		s.internalError(w, err)
		return
	}

	// Escribir log de cambios (solo filas donde el valor cambió)
	logEntries := make([]domain.VelocidadNominalLog, 0, len(records))
	for _, rec := range records {
		prev := prevMap[rec.ProductoID]
		if prev.VelocidadUs == rec.VelocidadUs && prev.FactorConv == rec.FactorConv {
			continue
		}
		sku := skuMap[rec.ProductoID]
		entry := domain.VelocidadNominalLog{
			ProductoID:       rec.ProductoID,
			SKU:              sku,
			VelocidadUSNueva: rec.VelocidadUs,
			FactorConvNueva:  rec.FactorConv,
			Origen:           "edge",
		}
		if prev.ID > 0 {
			entry.VelocidadUSAnterior = &prev.VelocidadUs
			entry.FactorConvAnterior = &prev.FactorConv
		}
		if actor != "" {
			entry.Usuario = &actor
		}
		if m := motivoMap[rec.ProductoID]; m != "" {
			entry.Motivo = &m
		}
		logEntries = append(logEntries, entry)
	}
	if len(logEntries) > 0 {
		_ = repo.InsertVelocidadNominalLog(r.Context(), logEntries)
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "updated": len(items)})

	// Propagar best-effort al cloud para persistir el cambio.
	if s.cloudURL != "" && s.cloudKey != "" {
		cloudLid := s.cloudLineaIDFor(r)
		go func() {
			type cloudItem struct {
				LineaID     int     `json:"linea_id"`
				ProductoID  int     `json:"producto_id"`
				VelocidadUS float64 `json:"velocidad_us"`
				FactorConv  int     `json:"factor_conv"`
				Motivo      string  `json:"motivo,omitempty"`
			}
			cloudItems := make([]cloudItem, len(records))
			for i, rec := range records {
				cloudItems[i] = cloudItem{
					LineaID:     cloudLid,
					ProductoID:  rec.ProductoID,
					VelocidadUS: rec.VelocidadUs,
					FactorConv:  rec.FactorConv,
					Motivo:      motivoMap[rec.ProductoID],
				}
			}
			payload, _ := json.Marshal(cloudItems)
			cloudEndpoint := strings.TrimRight(s.cloudURL, "/") + "/api/v1/edge/velocidad-nominal"
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPut, cloudEndpoint,
				strings.NewReader(string(payload)),
			)
			if err != nil {
				log.Printf("[velocidad-nominal] cloud request error: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+s.cloudKey)
			req.Header.Set("X-Device-ID", s.deviceID)
			client := &http.Client{Timeout: 8 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("[velocidad-nominal] cloud propagation error: %v", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 300 {
				log.Printf("[velocidad-nominal] cloud propagation status=%d", resp.StatusCode)
			} else {
				log.Printf("[velocidad-nominal] cloud propagation ok linea_id=%d items=%d", cloudLid, len(cloudItems))
			}
		}()
	}
}

// GET /edge/catalogs/velocidad-nominal/log?limit=100
func (s *HTTPServer) handleVelocidadNominalLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	limit := s.queryInt(r, "limit", 100)
	entries, err := s.catalogSyncFor(r).ListVelocidadNominalLog(r.Context(), limit)
	if err != nil {
		s.internalError(w, err)
		return
	}
	if entries == nil {
		entries = []domain.VelocidadNominalLog{}
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"data": entries})
}

func (s *HTTPServer) handleProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	all, err := s.catalogSyncFor(r).ListProductos(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	out := make([]domain.ProductEntry, 0, len(all))
	for _, p := range all {
		if !p.Activo {
			continue
		}
		localLid := s.lineaIDFor(r)
		if localLid > 0 && (p.LineaID == nil || *p.LineaID != localLid) {
			continue
		}
		out = append(out, domain.ProductEntry{SKU: p.Codigo, Description: p.Nombre, Active: p.Activo})
	}
	s.jsonResponse(w, http.StatusOK, out)
}

func (s *HTTPServer) handleProductCharacteristics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	// 1. Columnas (variables de producto para esta línea)
	allCols, err := s.catalogSyncFor(r).ListLineaProductoVars(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	cols := make([]domain.ProductColumn, 0, len(allCols))
	for _, c := range allCols {
		localLid := s.lineaIDFor(r)
		if localLid > 0 && c.LineaID != localLid {
			continue
		}
		cols = append(cols, domain.ProductColumn{VariableID: c.VariableID, Nombre: c.NombreCol, Orden: c.Orden})
	}

	// 2. Características
	allChars, err := s.catalogSyncFor(r).ListProductoCaracteristicas(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}

	// 3. Productos activos de esta línea
	allProds, err := s.catalogSyncFor(r).ListProductos(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	prodMap := make(map[int]domain.Producto)
	for _, p := range allProds {
		localLid := s.lineaIDFor(r)
		if p.Activo && (localLid == 0 || (p.LineaID != nil && *p.LineaID == localLid)) {
			prodMap[p.ID] = p
		}
	}

	// Agrupar valores por producto
	charByProd := make(map[int]map[int]string)
	for _, ch := range allChars {
		localLid := s.lineaIDFor(r)
		if localLid > 0 && ch.LineaID != localLid {
			continue
		}
		if _, ok := prodMap[ch.ProductoID]; !ok {
			continue
		}
		if charByProd[ch.ProductoID] == nil {
			charByProd[ch.ProductoID] = make(map[int]string)
		}
		charByProd[ch.ProductoID][ch.VariableID] = ch.Valor
	}

	prods := make([]domain.ProductWithCharacteristics, 0, len(charByProd))
	for pid, vals := range charByProd {
		p := prodMap[pid]
		prods = append(prods, domain.ProductWithCharacteristics{
			ProductoID: pid,
			Codigo:     p.Codigo,
			Nombre:     p.Nombre,
			Valores:    vals,
		})
	}

	s.jsonResponse(w, http.StatusOK, domain.ProductCharacteristicsResponse{
		Columnas:  cols,
		Productos: prods,
	})
}

func (s *HTTPServer) handleSyncCategorias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	data, err := s.catalogSyncFor(r).ListCategorias(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if data == nil {
		data = []domain.CategoriaParada{}
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (s *HTTPServer) handleSyncProductos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	data, err := s.catalogSyncFor(r).ListProductos(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if data == nil {
		data = []domain.Producto{}
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"data": data})
}

func (s *HTTPServer) handleSyncTurnos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	data, err := s.catalogSyncFor(r).ListTurnos(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if data == nil {
		data = []domain.Turno{}
	}

	// Filtrar por planta_id si se recibe como query param
	if plantaStr := r.URL.Query().Get("planta_id"); plantaStr != "" {
		if plantaID, convErr := strconv.Atoi(plantaStr); convErr == nil {
			filtered := data[:0]
			for _, t := range data {
				if t.PlantaID == nil || *t.PlantaID == plantaID {
					filtered = append(filtered, t)
				}
			}
			data = filtered
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"data": data})
}

// ---------- Commands ----------

func (s *HTTPServer) handleCommands(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		if !s.authRequired(w, r) {
			return
		}
		var req domain.CreateCommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.badRequest(w, "invalid JSON body")
			return
		}

		cmd, err := s.commandsFor(r).ReceiveCommand(r.Context(), req)
		if err == domain.ErrDuplicateCommand {
			s.jsonResponse(w, http.StatusOK, cmd)
			return
		}
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
			"status":     "RECEIVED",
			"command_id": cmd.CommandID,
		})

	case http.MethodGet:
		limit := s.queryInt(r, "limit", 50)
		cmds, err := s.commandsFor(r).ListCommands(r.Context(), s.deviceID, limit)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if cmds == nil {
			cmds = []domain.Command{}
		}
		s.jsonResponse(w, http.StatusOK, cmds)

	default:
		s.methodNotAllowed(w)
	}
}

func (s *HTTPServer) handleCommandByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	commandID := strings.TrimPrefix(r.URL.Path, "/edge/commands/")
	cmd, err := s.commandsFor(r).GetCommand(r.Context(), commandID)
	if err == domain.ErrCommandNotFound {
		s.notFound(w, "command not found")
		return
	}
	if err != nil {
		s.internalError(w, err)
		return
	}
	s.jsonResponse(w, http.StatusOK, cmd)
}

// ---------- SSE Stream ----------

func (s *HTTPServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	clientID := fmt.Sprintf("sse-%d", time.Now().UnixNano())
	events := s.broker.Subscribe(clientID)
	defer s.broker.Unsubscribe(clientID)

	log.Printf("[sse] client connected: %s", clientID)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"client_id\":\"%s\",\"device_id\":\"%s\"}\n\n", clientID, s.deviceID)
	flusher.Flush()

	// Keepalive ticker
	keepalive := time.NewTicker(30 * time.Second)
	defer keepalive.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[sse] client disconnected: %s", clientID)
			return
		case data, ok := <-events:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-keepalive.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// ---------- Middleware ----------

func (s *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Device-ID, X-Actor")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *HTTPServer) authRequired(w http.ResponseWriter, r *http.Request) bool {
	if s.authToken == "" {
		return true
	}

	auth := r.Header.Get("Authorization")
	if auth == "" {
		s.jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "missing Authorization header"})
		return false
	}

	token := strings.TrimPrefix(auth, "Bearer ")
	if token != s.authToken {
		s.jsonResponse(w, http.StatusForbidden, map[string]string{"error": "invalid token"})
		return false
	}
	return true
}

func (s *HTTPServer) extractActor(r *http.Request) string {
	if actor := r.Header.Get("X-Actor"); actor != "" {
		return actor
	}
	if deviceID := r.Header.Get("X-Device-ID"); deviceID != "" {
		return "cloud:" + deviceID
	}
	return "local:operator"
}

// ---------- Helpers ----------

func (s *HTTPServer) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *HTTPServer) badRequest(w http.ResponseWriter, msg string) {
	s.jsonResponse(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func (s *HTTPServer) notFound(w http.ResponseWriter, msg string) {
	s.jsonResponse(w, http.StatusNotFound, map[string]string{"error": msg})
}

func (s *HTTPServer) internalError(w http.ResponseWriter, err error) {
	log.Printf("[http] internal error: %v", err)
	s.jsonResponse(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func (s *HTTPServer) methodNotAllowed(w http.ResponseWriter) {
	s.jsonResponse(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

func (s *HTTPServer) queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultVal
	}
	return n
}

// ---------- Production runs ----------

func (s *HTTPServer) handleProductionRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		filter := domain.ProductionRunFilter{
			Limit: s.queryInt(r, "limit", 200),
		}
		if v := r.URL.Query().Get("since"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				filter.Since = &t
			}
		}
		if v := r.URL.Query().Get("until"); v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				filter.Until = &t
			}
		}
		if v := r.URL.Query().Get("linea_id"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				filter.LineaID = &n
			}
		}
		runs, err := s.gatewayFor(r).ListProductionRuns(r.Context(), filter)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if runs == nil {
			runs = []domain.ProductionRun{}
		}
		s.jsonResponse(w, http.StatusOK, runs)

	case http.MethodPost:
		var req domain.UpsertProductionRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.badRequest(w, "invalid JSON body")
			return
		}
		actor := s.extractActor(r)
		runs, err := s.gatewayFor(r).UpsertProductionRun(r.Context(), req, actor)
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		s.jsonResponse(w, http.StatusOK, runs)

	default:
		s.methodNotAllowed(w)
	}
}

func (s *HTTPServer) handleProductionRunByID(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimPrefix(r.URL.Path, "/edge/production-runs/")
	if runID == "" {
		s.badRequest(w, "missing run_id")
		return
	}
	switch r.Method {
	case http.MethodDelete:
		actor := s.extractActor(r)
		runs, err := s.gatewayFor(r).DeleteProductionRun(r.Context(), runID, actor)
		if err != nil {
			s.internalError(w, err)
			return
		}
		if runs == nil {
			runs = []domain.ProductionRun{}
		}
		s.jsonResponse(w, http.StatusOK, runs)
	default:
		s.methodNotAllowed(w)
	}
}

// GET /edge/variables[?clave=MERMA]
// Retorna las variables de variables, con filtro opcional por clave.
func (s *HTTPServer) handleVariables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	all, err := s.catalogSyncFor(r).ListVariables(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if clave := r.URL.Query().Get("clave"); clave != "" {
		var filtered []domain.Variable
		for _, v := range all {
			if strings.EqualFold(v.Clave, clave) {
				filtered = append(filtered, v)
			}
		}
		if filtered == nil {
			filtered = []domain.Variable{}
		}
		s.jsonResponse(w, http.StatusOK, filtered)
		return
	}
	if all == nil {
		all = []domain.Variable{}
	}
	s.jsonResponse(w, http.StatusOK, all)
}

// PATCH /edge/variables/{clave}
// Body: {"valor":"<nuevo_valor>","dispositivo_id":<int_opcional>}
// Actualiza el campo valor de la variable identificada por clave en variables.
func (s *HTTPServer) handleVariableByKey(w http.ResponseWriter, r *http.Request) {
	clave := strings.TrimPrefix(r.URL.Path, "/edge/variables/")
	if clave == "" {
		s.badRequest(w, "missing variable key")
		return
	}
	if r.Method != http.MethodPatch {
		s.methodNotAllowed(w)
		return
	}

	var body struct {
		Valor         string `json:"valor"`
		DispositivoID int    `json:"dispositivo_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.badRequest(w, "invalid JSON body")
		return
	}
	if body.Valor == "" {
		s.badRequest(w, "campo 'valor' requerido")
		return
	}

	if err := s.catalogSyncFor(r).UpdateVariableValor(r.Context(), strings.ToUpper(clave), body.DispositivoID, body.Valor); err != nil {
		s.internalError(w, err)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "clave": strings.ToUpper(clave), "valor": body.Valor})
}

func (s *HTTPServer) handleVelocidadNominalMotivos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}
	motivos, err := s.catalogSyncFor(r).ListMotivosVelocidad(r.Context())
	if err != nil {
		s.internalError(w, err)
		return
	}
	if motivos == nil {
		motivos = []domain.MotivoVelocidad{}
	}
	// Solo devolver los activos al cliente de tablet
	active := motivos[:0]
	for _, m := range motivos {
		if m.Activo {
			active = append(active, m)
		}
	}
	s.jsonResponse(w, http.StatusOK, map[string]interface{}{"data": active})
}
