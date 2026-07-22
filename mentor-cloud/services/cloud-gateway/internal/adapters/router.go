package adapters

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud-gateway/internal/application"
	"cloud-gateway/internal/domain"
	"cloud-gateway/internal/ports"
)

type authType int

const (
	authNone      authType = iota
	authJWT       authType = iota
	authAPIKey    authType = iota
	authInternal  authType = iota
	authJWTStream authType = iota // JWT que acepta ?token= para EventSource del browser
)

type routeEntry struct {
	prefix   string
	auth     authType
	strip    string
	upstream http.Handler
}

type Router struct {
	routes         []routeEntry
	catchAll       http.Handler
	mw             *Middleware
	audit          ports.AuditRepository
	svc            *application.GatewayService
	hub            *SSEHub
	browserHub     *BrowserSSEHub
	cameraHub      *CameraHub
	identityURL    string
	ingestURL      string
	configURL      string
	analyticsURL   string
	integrationURL string
	internalKey    string
	hc             *http.Client
}

type RouterConfig struct {
	Identity       *Proxy
	Ingest         *Proxy
	Config         *Proxy
	Analytics      *Proxy
	Integration    *Proxy
	Energy         *Proxy
	Frontend       *Proxy
	Mw             *Middleware
	AuditRepo      ports.AuditRepository
	Svc            *application.GatewayService
	Hub            *SSEHub
	BrowserHub     *BrowserSSEHub
	CameraHub      *CameraHub
	IdentityURL    string
	IngestURL      string
	ConfigURL      string
	AnalyticsURL   string
	IntegrationURL string
	EnergyURL      string
	InternalKey    string
}

func NewRouter(cfg RouterConfig) http.Handler {
	r := &Router{
		mw:             cfg.Mw,
		audit:          cfg.AuditRepo,
		svc:            cfg.Svc,
		catchAll:       cfg.Frontend,
		hub:            cfg.Hub,
		browserHub:     cfg.BrowserHub,
		internalKey:    cfg.InternalKey,
		identityURL:    cfg.IdentityURL,
		ingestURL:      cfg.IngestURL,
		configURL:      cfg.ConfigURL,
		analyticsURL:   cfg.AnalyticsURL,
		integrationURL: cfg.IntegrationURL,
		cameraHub:      cfg.CameraHub,
		hc:             &http.Client{Timeout: 5 * time.Second},
	}

	r.routes = []routeEntry{
		// Internal service-to-service — Internal Key
		{"/internal/broadcast", authInternal, "", http.HandlerFunc(r.handleBroadcast)},
		{"/internal/dispatch", authInternal, "", http.HandlerFunc(r.handleInternalDispatch)},
		{"/internal/notify", authInternal, "", http.HandlerFunc(r.handleInternalNotify)},

		// Edge device endpoints — API Key, longest prefixes first
		{"/api/v1/edge/commands/", authAPIKey, "", http.HandlerFunc(r.handleCommandAck)},
		{"/api/v1/edge/commands", authAPIKey, "", http.HandlerFunc(r.handlePollCommands)},
		{"/api/v1/edge/stream", authAPIKey, "", http.HandlerFunc(r.handleSSEStream)},
		{"/api/v1/edge/plants-lines", authAPIKey, "/api/v1/edge", cfg.Config},
		{"/api/v1/edge/velocidad-nominal", authAPIKey, "/api/v1", cfg.Config},
		{"/api/v1/edge/linea-config", authAPIKey, "/api/v1", cfg.Config},
		{"/api/v1/edge/", authAPIKey, "", cfg.Ingest},

		// Energy endpoints — API Key for edge devices, JWT for dashboard
		{"/api/v1/energy/", authAPIKey, "", cfg.Energy},
		{"/api/energy/", authJWT, "", cfg.Energy},

		// Cloud Edge control (energy) — JWT for REST and JWTStream for WebSocket (?token=)
		{"/api/edge-control/ws", authJWTStream, "", cfg.Ingest},
		{"/api/edge-control/", authJWT, "", cfg.Ingest},

		// Cloud → Edge command dispatch — JWT
		{"/api/v1/cloud/commands", authJWT, "", http.HandlerFunc(r.handleDispatch)},
		// Browser SSE — JWTStream para soportar ?token= (EventSource no puede poner headers)
		{"/api/v1/cloud/stream", authJWTStream, "", http.HandlerFunc(r.handleBrowserSSE)},

		// Config internal path — JWT, no strip
		{"/api/v1/config/", authJWT, "", cfg.Config},

		// Auth — open, strip /api before proxying
		{"/api/auth/", authNone, "/api", cfg.Identity},

		// Identity resources — JWT, strip /api
		{"/api/usuarios", authJWT, "/api", cfg.Identity},
		{"/api/empresas", authJWT, "/api", cfg.Identity},
		{"/api/roles", authJWT, "/api", cfg.Identity},

		// Config resources — JWT, strip /api
		{"/api/plantas", authJWT, "/api", cfg.Config},
		{"/api/lineas", authJWT, "/api", cfg.Config},
		{"/api/dispositivos", authJWT, "/api", cfg.Config},
		{"/api/variables", authJWT, "/api", cfg.Config},
		{"/api/turnos", authJWT, "/api", cfg.Config},
		{"/api/productos", authJWT, "/api", cfg.Config},
		{"/api/cat-paradas", authJWT, "/api", cfg.Config},
		{"/api/locaciones", authJWT, "/api", cfg.Config},
		{"/api/canvas-oee", authJWT, "/api", cfg.Config},
		{"/api/turno-dias", authJWT, "/api", cfg.Config},
		{"/api/variables-linea", authJWT, "/api", cfg.Config},
		{"/api/producto-caracteristicas", authJWT, "/api", cfg.Config},
		{"/api/linea-producto-vars", authJWT, "/api", cfg.Config},
		{"/api/linea-var-catalogo", authJWT, "/api", cfg.Config},
		{"/api/productos-crear", authJWT, "/api", cfg.Config},
		{"/api/velocidad-nominal", authJWT, "/api", cfg.Config},
		{"/api/motivos-velocidad", authJWT, "/api", cfg.Config},
		{"/api/arbol-paradas", authJWT, "/api", cfg.Config},
		{"/api/linea-config", authJWT, "/api", cfg.Config},
		{"/api/admin/", authJWT, "/api", cfg.Config},

		// Ingest resources — JWT, strip /api
		{"/api/datos-recibidos", authJWT, "/api", cfg.Ingest},

		// Analytics — JWT, no strip (service registers under /api group)
		{"/api/oee/", authJWT, "", cfg.Analytics},
		{"/api/stops", authJWT, "", cfg.Analytics},
		{"/api/production-runs", authJWT, "", cfg.Analytics},
		{"/api/dashboard/", authJWT, "", cfg.Analytics},
		{"/api/analisis/", authJWT, "", cfg.Analytics},
		{"/api/alarmas", authJWT, "", cfg.Analytics},
		{"/api/compromisos", authJWT, "", cfg.Analytics},
		{"/api/reportes", authJWT, "", cfg.Analytics},
		{"/api/excel/", authJWT, "", cfg.Analytics},

		// Integration management — JWT
		{"/api/integration/", authJWT, "", cfg.Integration},

		// Integration external API — no gateway auth, service valida X-API-Key
		{"/api/v1/integration/", authNone, "", cfg.Integration},
	}

	sort.Slice(r.routes, func(i, j int) bool {
		return len(r.routes[i].prefix) > len(r.routes[j].prefix)
	})

	return cfg.Mw.CORS(r)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/health" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"cloud-gateway","status":"ok"}`))
		return
	}

	if req.URL.Path == "/api/health" {
		r.handleAggregatedHealth(w, req)
		return
	}

	if req.URL.Path == "/api/server-metrics" {
		r.handleServerMetrics(w, req)
		return
	}

	if req.URL.Path == "/api/notifications" {
		r.handleNotifications(w, req)
		return
	}

	// Camera relay — edge push (APIKey) and browser MJPEG stream (JWT or ?token=)
	if r.cameraHub != nil {
		if req.URL.Path == "/api/v1/camera/push" {
			r.mw.SanitizeHeaders(r.mw.APIKey(http.HandlerFunc(r.cameraHub.HandlePush))).ServeHTTP(w, req)
			return
		}
		if req.URL.Path == "/api/v1/camera/stream" {
			r.mw.SanitizeHeaders(r.mw.JWTStream(http.HandlerFunc(r.cameraHub.HandleStream))).ServeHTTP(w, req)
			return
		}
		if req.URL.Path == "/api/v1/camera/roi" {
			r.mw.SanitizeHeaders(r.mw.JWTStream(http.HandlerFunc(r.cameraHub.HandleROI))).ServeHTTP(w, req)
			return
		}
	}

	start := time.Now()
	cw := &codeWriter{ResponseWriter: w, code: http.StatusOK}
	meta := &RequestMeta{}
	req = req.WithContext(context.WithValue(req.Context(), ctxMeta, meta))

	entry := r.findRoute(req.URL.Path)
	if entry == nil {
		r.catchAll.ServeHTTP(cw, req)
		go r.logAudit(req, cw.code, time.Since(start).Milliseconds(), meta)
		return
	}

	if entry.strip != "" {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, entry.strip)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		if req.URL.RawPath != "" {
			req.URL.RawPath = strings.TrimPrefix(req.URL.RawPath, entry.strip)
		}
	}

	switch entry.auth {
	case authJWT:
		r.mw.SanitizeHeaders(r.mw.JWT(r.mw.ScopeEnforcer(r.mw.ValidateInput(entry.upstream)))).ServeHTTP(cw, req)
	case authJWTStream:
		r.mw.SanitizeHeaders(r.mw.JWTStream(r.mw.ScopeEnforcer(entry.upstream))).ServeHTTP(cw, req)
	case authAPIKey:
		r.mw.SanitizeHeaders(r.mw.APIKey(r.mw.ValidateInput(entry.upstream))).ServeHTTP(cw, req)
	case authInternal:
		r.mw.InternalKey(entry.upstream).ServeHTTP(cw, req)
	default:
		r.mw.SanitizeHeaders(entry.upstream).ServeHTTP(cw, req)
	}

	go r.logAudit(req, cw.code, time.Since(start).Milliseconds(), meta)
}

func (r *Router) findRoute(path string) *routeEntry {
	for i := range r.routes {
		prefix := r.routes[i].prefix
		if strings.HasSuffix(prefix, "/") {
			if strings.HasPrefix(path, prefix) {
				return &r.routes[i]
			}
		} else {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return &r.routes[i]
			}
		}
	}
	return nil
}

func (r *Router) logAudit(req *http.Request, status int, ms int64, meta *RequestMeta) {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)
	if fwd := req.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.SplitN(fwd, ",", 2)[0]
	}
	_ = r.audit.Insert(context.Background(), &domain.AuditEntry{
		Method:    req.Method,
		Path:      req.URL.Path,
		Status:    status,
		LatencyMs: ms,
		IP:        strings.TrimSpace(ip),
		UserID:    meta.UserID,
		EmpresaID: meta.EmpresaID,
		DeviceID:  meta.DeviceID,
		Timestamp: time.Now(),
	})
}

func (r *Router) handlePollCommands(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	deviceID := req.Header.Get("X-Device-ID")
	if deviceID == "" {
		jsonError(w, "X-Device-ID requerido", http.StatusBadRequest)
		return
	}
	cmds, err := r.svc.PendingCommands(req.Context(), deviceID)
	if err != nil {
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	if cmds == nil {
		cmds = []domain.Command{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"data": cmds})
}

func (r *Router) handleCommandAck(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPut {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path: /api/v1/edge/commands/{id}/ack
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	if len(parts) < 2 {
		jsonError(w, "id requerido", http.StatusBadRequest)
		return
	}
	id, err := strconv.ParseInt(parts[len(parts)-2], 10, 64)
	if err != nil {
		jsonError(w, "id invalido", http.StatusBadRequest)
		return
	}
	deviceID := req.Header.Get("X-Device-ID")
	if err := r.svc.AcknowledgeCommand(req.Context(), id, deviceID); err != nil {
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (r *Router) handleDispatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DeviceID string          `json:"device_id"`
		Type     string          `json:"type"`
		Payload  json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		jsonError(w, "payload invalido", http.StatusBadRequest)
		return
	}
	if body.DeviceID == "" || body.Type == "" {
		jsonError(w, "device_id y type requeridos", http.StatusBadRequest)
		return
	}
	meta, _ := req.Context().Value(ctxMeta).(*RequestMeta)

	// Si el tipo corresponde a un recurso SYNC, populamos el payload con los
	// datos reales del cloud-config (scoped por empresa_id del JWT) y creamos
	// los comandos a través de dispatchScopedSync para que el edge reciba
	// la información lista para persistir localmente.
	if _, isSyncResource := syncResourceMapping[body.Type]; isSyncResource {
		created, err := r.dispatchScopedSync(req.Context(), body.DeviceID, meta.EmpresaID, 0, 0, []string{body.Type})
		if err != nil {
			log.Printf("[dispatch] scoped sync error device=%s type=%s: %v", body.DeviceID, body.Type, err)
			jsonError(w, "error al obtener datos de sincronización", http.StatusInternalServerError)
			return
		}
		if len(created) == 0 {
			jsonError(w, "no hay datos para sincronizar con los filtros indicados", http.StatusUnprocessableEntity)
			return
		}
		for _, cmd := range created {
			if r.hub != nil {
				r.hub.Notify(*cmd)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(created[0])
		return
	}

	cmd := &domain.Command{
		DeviceID:  body.DeviceID,
		EmpresaID: meta.EmpresaID,
		Type:      body.Type,
		Payload:   body.Payload,
		IssuedBy:  meta.UserID,
	}
	created, err := r.svc.DispatchCommand(req.Context(), cmd)
	if err != nil {
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	// Notificar en tiempo real si el dispositivo tiene SSE abierto
	if r.hub != nil {
		r.hub.Notify(*created)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// handleInternalDispatch handles service-to-service command dispatch (Internal Key auth).
func (r *Router) handleInternalDispatch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		DeviceID  string          `json:"device_id"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
		UserID    int64           `json:"user_id"`
		EmpresaID int64           `json:"empresa_id"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		jsonError(w, "payload invalido", http.StatusBadRequest)
		return
	}
	if body.DeviceID == "" || body.Type == "" {
		jsonError(w, "device_id y type requeridos", http.StatusBadRequest)
		return
	}
	cmd := &domain.Command{
		DeviceID:  body.DeviceID,
		EmpresaID: body.EmpresaID,
		Type:      body.Type,
		Payload:   body.Payload,
		IssuedBy:  body.UserID,
	}
	created, err := r.svc.DispatchCommand(req.Context(), cmd)
	if err != nil {
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}
	if r.hub != nil {
		r.hub.Notify(*created)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

func (r *Router) handleSSEStream(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	deviceID := req.Header.Get("X-Device-ID")
	if deviceID == "" {
		jsonError(w, "X-Device-ID requerido", http.StatusBadRequest)
		return
	}

	// Read device scope from headers (injected by APIKey middleware)
	empresaID, _ := strconv.ParseInt(req.Header.Get("X-Empresa-ID"), 10, 64)
	plantaID, _ := strconv.ParseInt(req.Header.Get("X-Planta-ID"), 10, 64)
	lineaID, _ := strconv.ParseInt(req.Header.Get("X-Linea-ID"), 10, 64)
	log.Printf("[sse] device=%s connected empresa=%d planta=%d linea=%d", deviceID, empresaID, plantaID, lineaID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // desactiva buffer en nginx

	// Enviar comentario inicial para confirmar la conexión
	fmt.Fprintf(w, ": connected device=%s\n\n", deviceID)
	flusher.Flush()

	ch, unsub := r.hub.Subscribe(deviceID)
	defer unsub()

	// Trigger initial sync in background — delivers all scoped config via SSE
	if empresaID > 0 && r.internalKey != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			created, err := r.dispatchScopedSync(ctx, deviceID, empresaID, plantaID, lineaID, nil)
			if err != nil {
				log.Printf("[sse-init] initial sync error device=%s: %v", deviceID, err)
				return
			}
			if r.hub != nil {
				for _, cmd := range created {
					r.hub.Notify(*cmd)
				}
			}
			log.Printf("[sse-init] initial sync device=%s dispatched=%d", deviceID, len(created))
		}()
	}

	// Replay pending operational commands (crear_parada, etc.) that may have
	// been dispatched while the device was offline.
	go func() {
		time.Sleep(300 * time.Millisecond) // let SSE channel settle
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pending, err := r.svc.PendingCommands(ctx, deviceID)
		if err != nil {
			log.Printf("[sse-init] replay pending error device=%s: %v", deviceID, err)
			return
		}
		replayed := 0
		for _, cmd := range pending {
			// SYNC commands are already re-sent by dispatchScopedSync above
			if strings.HasPrefix(cmd.Type, "SYNC_") || cmd.Type == "SYNC_CATALOG" {
				continue
			}
			if r.hub != nil {
				r.hub.Notify(cmd)
				replayed++
			}
		}
		if replayed > 0 {
			log.Printf("[sse-init] replayed %d pending commands device=%s", replayed, deviceID)
		}
	}()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case cmd, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(cmd)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: command\ndata: %s\n\n", data)
			flusher.Flush()

		case <-heartbeat.C:
			// Mantiene la conexión viva a través de proxies y load balancers
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()

		case <-req.Context().Done():
			return
		}
	}
}

func (r *Router) handleBroadcast(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		EmpresaID int64           `json:"empresa_id"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		jsonError(w, "payload invalido", http.StatusBadRequest)
		return
	}
	if body.EmpresaID == 0 || body.Type == "" {
		jsonError(w, "empresa_id y type requeridos", http.StatusBadRequest)
		return
	}

	// Determine which resource is being synced
	resource := mapTypeToResource(body.Type)

	// Get all devices for this empresa
	devices, err := r.svc.ListDevicesByEmpresa(req.Context(), body.EmpresaID)
	if err != nil {
		log.Printf("[broadcast] list devices error: %v", err)
		jsonError(w, "error interno", http.StatusInternalServerError)
		return
	}

	// Group devices by unique scope (planta_id, linea_id) to minimize config calls
	type scope struct {
		PlantaID int64
		LineaID  int64
	}
	scopeGroups := make(map[scope][]domain.Device)
	for _, d := range devices {
		s := scope{PlantaID: d.PlantaID, LineaID: d.LineaID}
		scopeGroups[s] = append(scopeGroups[s], d)
	}

	var allCreated []*domain.Command
	resources := []string{}
	if resource != "" {
		resources = []string{resource}
	}

	for s, devs := range scopeGroups {
		// Fetch filtered config for this scope
		configData, err := r.fetchScopedConfig(req.Context(), body.EmpresaID, s.PlantaID, s.LineaID)
		if err != nil {
			log.Printf("[broadcast] fetch scoped config scope=(%d,%d) error: %v", s.PlantaID, s.LineaID, err)
			continue
		}

		// Pick only the target resource (or all if resource is empty)
		targetResources := syncResourceMapping
		if resource != "" {
			targetResources = map[string]string{resource: body.Type}
		}

		for res, cmdType := range targetResources {
			if len(resources) > 0 && res != resource {
				continue
			}
			records, ok := configData[res]
			if !ok {
				continue
			}

			payload, _ := json.Marshal(map[string]interface{}{
				"resource":   res,
				"empresa_id": body.EmpresaID,
				"records":    json.RawMessage(records),
			})

			for _, d := range devs {
				cmd, err := r.svc.CreateCommand(req.Context(), &domain.Command{
					DeviceID:  d.DeviceID,
					EmpresaID: body.EmpresaID,
					Type:      cmdType,
					Payload:   payload,
				})
				if err != nil {
					log.Printf("[broadcast] create command error device=%s: %v", d.DeviceID, err)
					continue
				}
				allCreated = append(allCreated, cmd)
			}
		}
	}

	// Notify SSE hub
	if r.hub != nil {
		for _, cmd := range allCreated {
			r.hub.Notify(*cmd)
		}
	}

	log.Printf("[broadcast] dispatched %d scoped commands for empresa=%d type=%s", len(allCreated), body.EmpresaID, body.Type)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"dispatched": len(allCreated)})
}

func (r *Router) handleAggregatedHealth(w http.ResponseWriter, _ *http.Request) {
	type svcResult struct {
		Service   string `json:"service"`
		Status    string `json:"status"`
		LatencyMs int64  `json:"latency_ms"`
	}
	type result struct {
		key  string
		data svcResult
	}

	targets := map[string]string{
		"identity":    r.identityURL,
		"config":      r.configURL,
		"ingest":      r.ingestURL,
		"analytics":   r.analyticsURL,
		"integration": r.integrationURL,
	}

	ch := make(chan result, len(targets))
	for k, u := range targets {
		go func(key, url string) {
			start := time.Now()
			resp, err := r.hc.Get(url + "/health")
			ms := time.Since(start).Milliseconds()
			if err != nil {
				ch <- result{key, svcResult{Service: key, Status: "error", LatencyMs: ms}}
				return
			}
			defer resp.Body.Close()
			var s svcResult
			if decErr := json.NewDecoder(resp.Body).Decode(&s); decErr != nil {
				ch <- result{key, svcResult{Service: key, Status: "error", LatencyMs: ms}}
				return
			}
			s.LatencyMs = ms
			ch <- result{key, s}
		}(k, u)
	}

	svcs := make(map[string]svcResult)
	for range targets {
		res := <-ch
		svcs[res.key] = res.data
	}

	overall := "ok"
	for _, s := range svcs {
		if s.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"gateway":    svcResult{Service: "cloud-gateway", Status: overall, LatencyMs: 0},
		"services":   svcs,
		"checked_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (r *Router) handleServerMetrics(w http.ResponseWriter, _ *http.Request) {
	type promResult struct {
		Value float64 `json:"-"`
		Raw   any
	}

	queries := map[string]string{
		"cpu_usage_pct":  `100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[1m])) * 100)`,
		"ram_usage_pct":  `(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100`,
		"ram_used_gb":    `(node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes) / 1073741824`,
		"ram_total_gb":   `node_memory_MemTotal_bytes / 1073741824`,
		"disk_usage_pct": `(1 - (node_filesystem_avail_bytes{mountpoint="/",fstype!="tmpfs"} / node_filesystem_size_bytes{mountpoint="/",fstype!="tmpfs"})) * 100`,
		"disk_used_gb":   `(node_filesystem_size_bytes{mountpoint="/",fstype!="tmpfs"} - node_filesystem_avail_bytes{mountpoint="/",fstype!="tmpfs"}) / 1073741824`,
		"disk_total_gb":  `node_filesystem_size_bytes{mountpoint="/",fstype!="tmpfs"} / 1073741824`,
		"uptime_seconds": `time() - node_boot_time_seconds`,
		"load1":          `node_load1`,
		"net_rx_bps":     `rate(node_network_receive_bytes_total{device!~"lo|docker.*|br.*"}[1m])`,
		"net_tx_bps":     `rate(node_network_transmit_bytes_total{device!~"lo|docker.*|br.*"}[1m])`,
	}

	type chanResult struct {
		key string
		val float64
		ok  bool
	}

	prometheusURL := "http://prometheus:9090"
	ch := make(chan chanResult, len(queries))
	for k, q := range queries {
		go func(key, query string) {
			req, _ := http.NewRequest(http.MethodGet, prometheusURL+"/api/v1/query", nil)
			v := req.URL.Query()
			v.Set("query", query)
			req.URL.RawQuery = v.Encode()
			resp, err := r.hc.Do(req)
			if err != nil {
				ch <- chanResult{key: key}
				return
			}
			defer resp.Body.Close()
			var envelope struct {
				Data struct {
					Result []struct {
						Value [2]any `json:"value"`
					} `json:"result"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil || len(envelope.Data.Result) == 0 {
				ch <- chanResult{key: key}
				return
			}
			raw, _ := envelope.Data.Result[0].Value[1].(string)
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				ch <- chanResult{key: key}
				return
			}
			ch <- chanResult{key: key, val: val, ok: true}
		}(k, q)
	}

	metrics := make(map[string]float64, len(queries))
	for range queries {
		r := <-ch
		if r.ok {
			metrics[r.key] = r.val
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metrics)
}

// handleNotifications proxies email recipient management to Grafana's contact point API.
// GET  /api/notifications → returns current list of email addresses
// POST /api/notifications → body: {"emails":["a@b.com","c@d.com"]} → updates contact point
func (r *Router) handleNotifications(w http.ResponseWriter, req *http.Request) {
	grafanaURL := "http://grafana:3000"
	grafanaAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:Mentor2026"))
	contactUID := "fffmzf4p9xwjkf"

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	switch req.Method {
	case http.MethodOptions:
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.WriteHeader(http.StatusNoContent)
		return

	case http.MethodGet:
		gReq, _ := http.NewRequestWithContext(req.Context(), http.MethodGet,
			grafanaURL+"/api/v1/provisioning/contact-points", nil)
		gReq.Header.Set("Authorization", grafanaAuth)
		resp, err := r.hc.Do(gReq)
		if err != nil {
			http.Error(w, `{"error":"grafana unavailable"}`, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		var cps []struct {
			UID      string         `json:"uid"`
			Name     string         `json:"name"`
			Type     string         `json:"type"`
			Settings map[string]any `json:"settings"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&cps); err != nil {
			http.Error(w, `{"error":"decode failed"}`, http.StatusInternalServerError)
			return
		}
		var emails []string
		for _, cp := range cps {
			if cp.UID == contactUID {
				if addr, ok := cp.Settings["addresses"].(string); ok && addr != "" {
					for _, e := range strings.Split(addr, ";") {
						e = strings.TrimSpace(e)
						if e != "" {
							emails = append(emails, e)
						}
					}
				}
				break
			}
		}
		if emails == nil {
			emails = []string{}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"emails": emails})

	case http.MethodPost:
		var body struct {
			Emails []string `json:"emails"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || len(body.Emails) == 0 {
			http.Error(w, `{"error":"invalid body, need {emails:[...]}"}`, http.StatusBadRequest)
			return
		}
		// Validate basic email format
		for _, e := range body.Emails {
			if !strings.Contains(e, "@") {
				http.Error(w, `{"error":"invalid email: `+e+`"}`, http.StatusBadRequest)
				return
			}
		}
		addresses := strings.Join(body.Emails, ";")
		payload, _ := json.Marshal(map[string]any{
			"name": "Email Mentor",
			"type": "email",
			"settings": map[string]any{
				"addresses":   addresses,
				"singleEmail": false,
			},
		})
		gReq, _ := http.NewRequestWithContext(req.Context(), http.MethodPut,
			grafanaURL+"/api/v1/provisioning/contact-points/"+contactUID,
			bytes.NewReader(payload))
		gReq.Header.Set("Authorization", grafanaAuth)
		gReq.Header.Set("Content-Type", "application/json")
		resp, err := r.hc.Do(gReq)
		if err != nil {
			http.Error(w, `{"error":"grafana unavailable"}`, http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			body2, _ := io.ReadAll(resp.Body)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write(body2)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "emails": body.Emails})

	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

type codeWriter struct {
	http.ResponseWriter
	code int
}

func (cw *codeWriter) WriteHeader(code int) {
	cw.code = code
	cw.ResponseWriter.WriteHeader(code)
}

// Flush delega al ResponseWriter subyacente para soportar SSE (http.Flusher).
func (cw *codeWriter) Flush() {
	if f, ok := cw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// ── Scope-aware sync helpers ─────────────────────────────────────────────────

// fetchScopedConfig calls cloud-config's /internal/device-config endpoint
// and returns the filtered configuration as a map of resource → records.
func (r *Router) fetchScopedConfig(ctx context.Context, empresaID, plantaID, lineaID int64) (map[string]json.RawMessage, error) {
	body, _ := json.Marshal(map[string]int64{
		"empresa_id": empresaID,
		"planta_id":  plantaID,
		"linea_id":   lineaID,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		r.configURL+"/internal/device-config", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", r.internalKey)

	resp, err := r.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config returned status %d", resp.StatusCode)
	}

	var result map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode config response: %w", err)
	}
	return result, nil
}

// syncResourceMapping maps cloud resource keys to SYNC command types.
var syncResourceMapping = map[string]string{
	"categoria_paradas":        "SYNC_CATALOG",
	"productos":                "SYNC_PRODUCTOS",
	"turnos":                   "SYNC_TURNOS",
	"variables":                "SYNC_VARIABLES",
	"linea_producto_vars":      "SYNC_LINEA_PRODUCTO_VARS",
	"producto_caracteristicas": "SYNC_PRODUCTO_CARACTERISTICAS",
	// Usuarios — credenciales scoped por empresa para autenticación local en edge
	"usuarios": "SYNC_USUARIOS",
	// Plantas y líneas — catálogo de organización scoped por empresa
	"plantas_lineas": "SYNC_PLANTAS_LINEAS",
	// Velocidad nominal — configuración de velocidad por producto+línea
	"velocidad_nominal": "SYNC_VELOCIDAD_NOMINAL",
	// Motivos de velocidad — lista de motivos predefinidos para cambios
	"motivos_velocidad": "SYNC_MOTIVOS_VELOCIDAD",
}

// fetchScopedStops calls cloud-analytics's /internal/stops?empresa_id=X
// and returns the raw JSON array of stop records.
func (r *Router) fetchScopedStops(ctx context.Context, empresaID int64) (json.RawMessage, error) {
	url := fmt.Sprintf("%s/internal/stops?empresa_id=%d", r.analyticsURL, empresaID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Key", r.internalKey)
	resp, err := r.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("analytics returned %d", resp.StatusCode)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// mapTypeToResource maps a SYNC command type back to a config resource key.
func mapTypeToResource(cmdType string) string {
	switch cmdType {
	case "SYNC_CATALOG":
		return "categoria_paradas"
	case "SYNC_PRODUCTOS":
		return "productos"
	case "SYNC_TURNOS":
		return "turnos"
	case "SYNC_VARIABLES":
		return "variables"
	case "SYNC_LINEA_PRODUCTO_VARS":
		return "linea_producto_vars"
	case "SYNC_PRODUCTO_CARACTERISTICAS":
		return "producto_caracteristicas"
	case "SYNC_USUARIOS":
		return "usuarios"
	case "SYNC_PLANTAS_LINEAS":
		return "plantas_lineas"
	case "SYNC_VELOCIDAD_NOMINAL":
		return "velocidad_nominal"
	case "SYNC_MOTIVOS_VELOCIDAD":
		return "motivos_velocidad"
	default:
		return ""
	}
}

// dispatchScopedSync fetches filtered config for the given scope, creates
// SYNC commands for the specified resources, and pushes them to the SSE hub.
// If resources is nil/empty, ALL resource types are synced.
func (r *Router) dispatchScopedSync(ctx context.Context, deviceID string, empresaID, plantaID, lineaID int64, resources []string) ([]*domain.Command, error) {
	configData, err := r.fetchScopedConfig(ctx, empresaID, plantaID, lineaID)
	if err != nil {
		return nil, err
	}

	var created []*domain.Command
	for resource, cmdType := range syncResourceMapping {
		// If a specific list was requested, skip others
		if len(resources) > 0 {
			found := false
			for _, r := range resources {
				if r == resource {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		records, ok := configData[resource]
		if !ok {
			continue
		}

		payload, _ := json.Marshal(map[string]interface{}{
			"resource":   resource,
			"empresa_id": empresaID,
			"records":    json.RawMessage(records),
		})

		cmd, err := r.svc.CreateCommand(ctx, &domain.Command{
			DeviceID:  deviceID,
			EmpresaID: empresaID,
			Type:      cmdType,
			Payload:   payload,
		})
		if err != nil {
			log.Printf("[scoped-sync] create command error device=%s resource=%s: %v", deviceID, resource, err)
			continue
		}
		created = append(created, cmd)
	}

	// ── SYNC_PARADAS: fetch stops from cloud-analytics and dispatch ──
	syncParadas := len(resources) == 0 // on initial connect, sync all including paradas
	if !syncParadas {
		for _, r := range resources {
			if r == "paradas" || r == "SYNC_PARADAS" {
				syncParadas = true
				break
			}
		}
	}
	log.Printf("[scoped-sync] SYNC_PARADAS check: syncParadas=%v empresa=%d", syncParadas, empresaID)
	if syncParadas {
		stopRecords, err := r.fetchScopedStops(ctx, empresaID)
		if err != nil {
			log.Printf("[scoped-sync] fetch stops error empresa=%d: %v", empresaID, err)
		} else if len(stopRecords) > 2 { // not empty JSON array "[]"
			payload, _ := json.Marshal(map[string]interface{}{
				"resource":   "paradas",
				"empresa_id": empresaID,
				"records":    json.RawMessage(stopRecords),
			})
			cmd, err := r.svc.CreateCommand(ctx, &domain.Command{
				DeviceID:  deviceID,
				EmpresaID: empresaID,
				Type:      "SYNC_PARADAS",
				Payload:   payload,
			})
			if err != nil {
				log.Printf("[scoped-sync] create SYNC_PARADAS error device=%s: %v", deviceID, err)
			} else {
				log.Printf("[scoped-sync] SYNC_PARADAS created cmd=%d device=%s records=%d bytes", cmd.ID, deviceID, len(stopRecords))
				created = append(created, cmd)
			}
		}
	}

	return created, nil
}

// handleBrowserSSE streams real-time events to browser clients authenticated via JWT.
// The client subscribes scoped to its empresa_id (from JWT claims).
func (r *Router) handleBrowserSSE(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		jsonError(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	empresaID, _ := strconv.ParseInt(req.Header.Get("X-Empresa-ID"), 10, 64)
	if empresaID == 0 {
		// Superadmin users have no empresa_id in JWT; accept from query param
		empresaID, _ = strconv.ParseInt(req.URL.Query().Get("empresa_id"), 10, 64)
	}
	if empresaID == 0 {
		jsonError(w, "empresa_id requerido", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	fmt.Fprintf(w, ": connected empresa=%d\n\n", empresaID)
	flusher.Flush()

	if r.browserHub == nil {
		// Hub not configured, keep connection open silently
		<-req.Context().Done()
		return
	}

	ch, unsub := r.browserHub.Subscribe(empresaID)
	defer unsub()

	for {
		select {
		case <-req.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// handleInternalNotify receives a notification from an internal service and
// forwards it to connected browser clients via BrowserSSEHub.
func (r *Router) handleInternalNotify(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var event BrowserEvent
	if err := json.NewDecoder(req.Body).Decode(&event); err != nil {
		jsonError(w, "invalid json", http.StatusBadRequest)
		return
	}
	if r.browserHub != nil {
		r.browserHub.Notify(event)
	}
	w.WriteHeader(http.StatusNoContent)
}
