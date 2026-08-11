package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"edge-config-service/internal/application"
	"edge-config-service/internal/domain"
)

type HTTPServer struct {
	service *application.ConfigService
	server  *http.Server
}

func NewHTTPServer(service *application.ConfigService, port string) *HTTPServer {
	h := &HTTPServer{service: service}

	mux := http.NewServeMux()
	mux.HandleFunc("/config", h.handleConfig)
	mux.HandleFunc("/config/system", h.handleSystem)
	mux.HandleFunc("/config/version", h.handleConfigVersion)
	mux.HandleFunc("/config/lines", h.handleLines)
	mux.HandleFunc("/config/device-id", h.handleDeviceID)
	mux.HandleFunc("/calibration/start", h.handleCalibration)
	mux.HandleFunc("/health", h.handleHealth)

	h.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return h
}

func (h *HTTPServer) Start() error {
	return h.server.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

// handleConfig — GET/PUT by linea_id (integer)
func (h *HTTPServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	lineaIDStr := r.URL.Query().Get("linea_id")
	if lineaIDStr == "" {
		http.Error(w, `{"error":"linea_id query param required"}`, http.StatusBadRequest)
		return
	}
	lineaID, err := strconv.Atoi(lineaIDStr)
	if err != nil {
		http.Error(w, `{"error":"linea_id must be an integer"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getConfig(w, r, lineaID)
	case http.MethodPut:
		h.updateConfig(w, r, lineaID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPServer) getConfig(w http.ResponseWriter, r *http.Request, lineaID int) {
	config, err := h.service.GetConfig(r.Context(), lineaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"device_id":      config.DeviceID,
		"linea_id":       config.LineaID,
		"empresa_id":     config.EmpresaID,
		"roi":            []int{config.ROI.X, config.ROI.Y, config.ROI.Width, config.ROI.Height, config.ROI.BottomMargin},
		"thresholds":     config.Thresholds,
		"fsm":            config.FSM,
		"mode":           config.Mode,
		"config_version": config.ConfigVersion,
		"oee":            config.OEE,
		"cloud":          config.Cloud,
		"tablet":         config.Tablet,
	}

	if config.Camera != nil {
		response["camera"] = config.Camera
	}

	if config.ROIPresencia != nil {
		response["roi_presencia"] = config.ROIPresencia
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *HTTPServer) updateConfig(w http.ResponseWriter, r *http.Request, lineaID int) {
	var updateReq struct {
		ROI          *[]int               `json:"roi"`
		ROIPresencia *[]int               `json:"roi_presencia"`
		Thresholds   *domain.Thresholds   `json:"thresholds"`
		FSM          *domain.FSMConfig    `json:"fsm"`
		Mode         *string              `json:"mode"`
		Camera       *domain.Camera       `json:"camera"`
		OEE          *domain.OEEConfig    `json:"oee"`
		Cloud        *domain.CloudConfig  `json:"cloud"`
		Tablet       *domain.TabletConfig `json:"tablet"`
		EmpresaID    *int                 `json:"empresa_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	config, err := h.service.GetConfig(r.Context(), lineaID)
	if err != nil {
		// Auto-detect device_id from existing lines
		deviceID, _ := h.service.GetDeviceID(r.Context())
		if deviceID == "" {
			deviceID = "1"
		}
		defaultMode := "textil"
		if updateReq.Mode != nil {
			defaultMode = *updateReq.Mode
		}
		defaultOEE := domain.ModeDefaultOEE(defaultMode)
		config = &domain.LineConfig{
			DeviceID:   deviceID,
			LineaID:    lineaID,
			Mode:       defaultMode,
			ROI:        domain.ROIData{X: 120, Y: 60, Width: 320, Height: 200},
			Thresholds: domain.Thresholds{Edge: 0.4, Color: 0.6, Flow: 0.5, DY: 5.0, Beige: 0.35, High: 0.7, Low: 0.3},
			FSM:        domain.FSMConfig{NFrames: 3, Cooldown: 8, ExitFrames: 5, MaxWaitExitFrames: 750},
			OEE:        defaultOEE,
		}
	}
	config.LineaID = lineaID

	if updateReq.ROI != nil && len(*updateReq.ROI) >= 4 {
		roi := *updateReq.ROI
		config.ROI = domain.ROIData{
			X:      roi[0],
			Y:      roi[1],
			Width:  roi[2],
			Height: roi[3],
		}
		if len(roi) >= 5 {
			config.ROI.BottomMargin = roi[4]
		}
	}

	if updateReq.ROIPresencia != nil {
		rp := *updateReq.ROIPresencia
		if len(rp) >= 4 && rp[2] > 0 && rp[3] > 0 {
			config.ROIPresencia = &domain.ROIData{
				X:      rp[0],
				Y:      rp[1],
				Width:  rp[2],
				Height: rp[3],
			}
		} else {
			config.ROIPresencia = nil
		}
	}

	if updateReq.Thresholds != nil {
		config.Thresholds = *updateReq.Thresholds
	}

	if updateReq.FSM != nil {
		config.FSM = *updateReq.FSM
	}

	if updateReq.Mode != nil && config.Mode != *updateReq.Mode {
		config.Mode = *updateReq.Mode
		// Si no se envía OEE explícito, aplicar los defaults textiles.
		if updateReq.OEE == nil {
			modeOEE := domain.ModeDefaultOEE(config.Mode)
			config.OEE.MicroStopMaxS = modeOEE.MicroStopMaxS
			config.OEE.StopMaxS = modeOEE.StopMaxS
			config.OEE.SnapshotInterS = modeOEE.SnapshotInterS
			config.OEE.VelUnit = modeOEE.VelUnit
			config.OEE.VelNominalUS = modeOEE.VelNominalUS
		}
	} else if updateReq.Mode != nil {
		config.Mode = *updateReq.Mode
	}

	if updateReq.Camera != nil {
		config.Camera = updateReq.Camera
	}

	if updateReq.OEE != nil {
		config.OEE = *updateReq.OEE
	}

	if updateReq.Cloud != nil {
		config.Cloud = *updateReq.Cloud
	}

	if updateReq.Tablet != nil {
		config.Tablet = updateReq.Tablet
	}

	if updateReq.EmpresaID != nil {
		config.EmpresaID = *updateReq.EmpresaID
	}

	if err := h.service.UpdateConfig(r.Context(), config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// handleSystem — GET/PUT system defaults (linea_id=0, device_id='_system')
func (h *HTTPServer) handleSystem(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config, err := h.service.GetConfig(r.Context(), 0)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}
		res := map[string]interface{}{
			"cloud": config.Cloud,
			"oee":   config.OEE,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(res)

	case http.MethodPut:
		var req struct {
			Cloud *domain.CloudConfig `json:"cloud"`
			OEE   *domain.OEEConfig   `json:"oee"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		config, _ := h.service.GetConfig(r.Context(), 0)
		if config == nil {
			config = &domain.LineConfig{
				DeviceID:   "_system",
				LineaID:    0,
				Mode:       "textil",
				ROI:        domain.ROIData{X: 120, Y: 60, Width: 320, Height: 200},
				Thresholds: domain.Thresholds{Edge: 0.4, Color: 0.6, Flow: 0.5, DY: 5.0, Beige: 0.35, High: 0.7, Low: 0.3},
				FSM:        domain.FSMConfig{NFrames: 3, Cooldown: 8, ExitFrames: 5, MaxWaitExitFrames: 750},
				OEE:        domain.ModeDefaultOEE("textil"),
			}
		}
		config.DeviceID = "_system"
		config.LineaID = 0
		if req.Cloud != nil {
			config.Cloud = *req.Cloud
		}
		if req.OEE != nil {
			config.OEE = *req.OEE
		}
		if err := h.service.UpdateConfig(r.Context(), config); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPServer) handleConfigVersion(w http.ResponseWriter, r *http.Request) {
	lineaIDStr := r.URL.Query().Get("linea_id")
	if lineaIDStr == "" {
		http.Error(w, `{"error":"linea_id required"}`, http.StatusBadRequest)
		return
	}
	lineaID, err := strconv.Atoi(lineaIDStr)
	if err != nil {
		http.Error(w, `{"error":"linea_id must be integer"}`, http.StatusBadRequest)
		return
	}

	version, err := h.service.GetConfigVersion(r.Context(), lineaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"version": version})
}

// handleLines — GET: list linea_ids, DELETE: remove a line
func (h *HTTPServer) handleLines(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		ids, err := h.service.GetLineIDs(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if ids == nil {
			ids = []int{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"lines": ids})

	case http.MethodDelete:
		lineaIDStr := r.URL.Query().Get("linea_id")
		if lineaIDStr == "" {
			http.Error(w, "linea_id requerido", http.StatusBadRequest)
			return
		}
		lineaID, err := strconv.Atoi(lineaIDStr)
		if err != nil {
			http.Error(w, "linea_id must be integer", http.StatusBadRequest)
			return
		}
		if err := h.service.DeleteLine(r.Context(), lineaID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "deleted", "linea_id": lineaID})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleDeviceID — GET/PUT the Jetson board identifier
func (h *HTTPServer) handleDeviceID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id, err := h.service.GetDeviceID(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"device_id": id})

	case http.MethodPut:
		var req struct {
			DeviceID string `json:"device_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if req.DeviceID == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}
		if err := h.service.SetDeviceID(r.Context(), req.DeviceID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "updated", "device_id": req.DeviceID})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPServer) handleCalibration(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "calibration_started"})
}

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"service":       "edge-config-service",
		"status":        "ok",
		"config_schema": "textile-v1",
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
