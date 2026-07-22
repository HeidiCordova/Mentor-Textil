package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type HealthServer struct {
	server            *http.Server
	lastAlive         atomic.Value
	cloudLastOK       atomic.Value
	consecutiveErrors atomic.Int64
	totalSynced       atomic.Int64
}

func NewHealthServer(port string) *HealthServer {
	hs := &HealthServer{}
	hs.lastAlive.Store(time.Now())
	hs.cloudLastOK.Store(time.Time{})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", hs.handleHealth)

	hs.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return hs
}

func (h *HealthServer) MarkAlive() {
	h.lastAlive.Store(time.Now())
}

func (h *HealthServer) MarkCloudOK() {
	h.cloudLastOK.Store(time.Now())
	h.consecutiveErrors.Store(0)
}

// MarkError incrementa el contador de errores consecutivos.
func (h *HealthServer) MarkError() {
	h.consecutiveErrors.Add(1)
}

// IncrementSynced suma n registros al total acumulado de sincronizaciones exitosas.
func (h *HealthServer) IncrementSynced(n int) {
	h.totalSynced.Add(int64(n))
}

func (h *HealthServer) Start() error {
	return h.server.ListenAndServe()
}

func (h *HealthServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *HealthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	last := h.lastAlive.Load().(time.Time)
	status := "ok"
	if time.Since(last) > 360*time.Second {
		status = "degraded"
	}

	cloudLastOK := h.cloudLastOK.Load().(time.Time)
	cloudStatus := "unknown"
	cloudLastOKStr := ""
	if !cloudLastOK.IsZero() {
		cloudLastOKStr = cloudLastOK.Format(time.RFC3339)
		if time.Since(cloudLastOK) < 5*time.Minute {
			cloudStatus = "ok"
		} else {
			cloudStatus = "degraded"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":            "enviador",
		"status":             status,
		"cloud_status":       cloudStatus,
		"cloud_last_ok":      cloudLastOKStr,
		"consecutive_errors": h.consecutiveErrors.Load(),
		"total_synced":       h.totalSynced.Load(),
		"timestamp":          time.Now().Format(time.RFC3339),
	})
}
