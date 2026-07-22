package adapters

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"resiliencia/internal/application"
	"resiliencia/internal/domain"
)

type HTTPServer struct {
	services    map[int]*application.BufferService
	defaultLine int
	server      *http.Server
}

func NewHTTPServer(services map[int]*application.BufferService, defaultLine int, port string) *HTTPServer {
	h := &HTTPServer{services: services, defaultLine: defaultLine}

	mux := http.NewServeMux()
	mux.HandleFunc("/events", h.handleEvents)
	mux.HandleFunc("/events/recent", h.handleRecentEvents)
	mux.HandleFunc("/events/pending", h.handlePendingEvents)
	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/health/buffer", h.handleBufferHealth)
	mux.HandleFunc("/health/stats", h.handleBufferStats)

	h.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return h
}

func (h *HTTPServer) serviceFor(r *http.Request) *application.BufferService {
	if lid := r.URL.Query().Get("linea_id"); lid != "" {
		if id, err := strconv.Atoi(lid); err == nil {
			if svc, ok := h.services[id]; ok {
				return svc
			}
		}
	}
	return h.services[h.defaultLine]
}

func (h *HTTPServer) Start() error {
	return h.server.ListenAndServe()
}

func (h *HTTPServer) Shutdown(ctx context.Context) error {
	return h.server.Shutdown(ctx)
}

func (h *HTTPServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event struct {
		EventID   string          `json:"event_id"`
		DeviceID  string          `json:"device_id"`
		EventType string          `json:"event_type"`
		Timestamp string          `json:"timestamp"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	timestamp, err := time.Parse(time.RFC3339, event.Timestamp)
	if err != nil {
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}

	bufferEvent := &domain.EventBuffer{
		EventID:   event.EventID,
		DeviceID:  event.DeviceID,
		EventType: event.EventType,
		Timestamp: timestamp,
		Payload:   []byte(event.Payload),
	}

	if err := h.serviceFor(r).StoreEvent(r.Context(), bufferEvent); err != nil {
		http.Error(w, "Failed to store event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stored"})
}

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"service":   "resiliencia",
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (h *HTTPServer) handleBufferHealth(w http.ResponseWriter, r *http.Request) {
	count, err := h.serviceFor(r).GetPendingCount(r.Context())
	if err != nil {
		http.Error(w, "Failed to get pending count", http.StatusInternalServerError)
		return
	}

	health := map[string]interface{}{
		"pending_events": count,
		"last_check":     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func (h *HTTPServer) handleBufferStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.serviceFor(r).GetBufferStats(r.Context())
	if err != nil {
		http.Error(w, "Failed to get buffer stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// eventResponseDTO serialises Payload as raw JSON (not base64).
type eventResponseDTO struct {
	ID        int             `json:"id"`
	EventID   string          `json:"event_id"`
	DeviceID  string          `json:"device_id"`
	EventType string          `json:"event_type"`
	Timestamp string          `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
	Synced    bool            `json:"synced"`
	CreatedAt string          `json:"created_at"`
}

func toEventDTOs(events []*domain.EventBuffer) []eventResponseDTO {
	out := make([]eventResponseDTO, 0, len(events))
	for _, e := range events {
		payload := json.RawMessage(e.Payload)
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}
		out = append(out, eventResponseDTO{
			ID:        e.ID,
			EventID:   e.EventID,
			DeviceID:  e.DeviceID,
			EventType: e.EventType,
			Timestamp: e.Timestamp.Format(time.RFC3339),
			Payload:   payload,
			Synced:    e.Synced,
			CreatedAt: e.CreatedAt.Format(time.RFC3339),
		})
	}
	return out
}

func (h *HTTPServer) handleRecentEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := parseLimit(r, 50)
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
	events, err := h.serviceFor(r).GetRecentEvents(r.Context(), limit, since)
	if err != nil {
		log.Printf("[resiliencia] GetRecentEvents error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toEventDTOs(events))
}

func (h *HTTPServer) handlePendingEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	limit := parseLimit(r, 50)
	events, err := h.serviceFor(r).GetPendingEvents(r.Context(), limit)
	if err != nil {
		log.Printf("[resiliencia] GetPendingEvents error: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("[]"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toEventDTOs(events))
}

func parseLimit(r *http.Request, defaultVal int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultVal
	}
	return n
}
