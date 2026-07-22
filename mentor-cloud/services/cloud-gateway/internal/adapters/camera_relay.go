package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// CameraHub relays live MJPEG video from edge devices to browser clients without
// storing any frames to disk. All data flows through in-memory channels.
//
// Push path  (edge → cloud):   POST /api/v1/camera/push?device_id=X
//
//	Edge authenticates with X-API-Key and streams a multipart/x-mixed-replace body.
//
// Stream path (cloud → browser): GET /api/v1/camera/stream?device_id=X&token=JWT
//
//	Browser receives standard MJPEG multipart stream.
type CameraHub struct {
	mu      sync.RWMutex
	devices map[string]*deviceStream
}

type deviceStream struct {
	mu      sync.Mutex
	latest  []byte
	clients []chan []byte
	roiJSON string // last ROI config received from edge (X-Camera-ROI header)
}

func NewCameraHub() *CameraHub {
	return &CameraHub{devices: make(map[string]*deviceStream)}
}

func (h *CameraHub) getOrCreate(deviceID string) *deviceStream {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ds, ok := h.devices[deviceID]; ok {
		return ds
	}
	ds := &deviceStream{}
	h.devices[deviceID] = ds
	return ds
}

// HandlePush receives a streaming MJPEG upload from an edge device.
// Auth is enforced by the router (APIKey) before this handler is called.
func (h *CameraHub) HandlePush(w http.ResponseWriter, r *http.Request) {
	deviceID := r.Header.Get("X-Device-ID")
	if deviceID == "" {
		deviceID = r.URL.Query().Get("device_id")
	}
	if deviceID == "" {
		http.Error(w, `{"error":"device_id required"}`, http.StatusBadRequest)
		return
	}

	// Remove read deadline so the long-lived streaming body is not killed
	// by the server's ReadTimeout (30 s by default).
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Time{})

	// Store ROI config if edge sent it
	if roiJSON := r.Header.Get("X-Camera-ROI"); roiJSON != "" {
		ds0 := h.getOrCreate(deviceID)
		ds0.mu.Lock()
		ds0.roiJSON = roiJSON
		ds0.mu.Unlock()
	}

	log.Printf("[camera-hub] edge %s connected (push)", deviceID)
	ds := h.getOrCreate(deviceID)

	buf := make([]byte, 0, 512*1024)
	tmp := make([]byte, 128*1024)
	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)

			// Extract complete JPEG frames by scanning for SOI/EOI markers.
			for {
				soiIdx := bytes.Index(buf, soi)
				if soiIdx < 0 {
					if len(buf) > 1 {
						buf = buf[len(buf)-1:]
					}
					break
				}
				buf = buf[soiIdx:]
				eoiIdx := bytes.Index(buf[2:], eoi)
				if eoiIdx < 0 {
					break
				}
				end := 2 + eoiIdx + 2
				frame := make([]byte, end)
				copy(frame, buf[:end])
				buf = append(buf[:0], buf[end:]...)

				// Fan-out to all watching browser clients.
				ds.mu.Lock()
				ds.latest = frame
				for _, ch := range ds.clients {
					select {
					case ch <- frame:
					default: // client too slow — drop frame
					}
				}
				ds.mu.Unlock()
			}

			if len(buf) > 4*1024*1024 {
				buf = buf[len(buf)-2:]
			}
		}
		if err != nil {
			break
		}
	}

	log.Printf("[camera-hub] edge %s disconnected (push)", deviceID)
	w.WriteHeader(http.StatusOK)
}

// HandleStream serves a live MJPEG stream to a browser client.
// Auth is enforced by the router (JWTStream) before this handler is called.
func (h *CameraHub) HandleStream(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		http.Error(w, `{"error":"device_id required"}`, http.StatusBadRequest)
		return
	}

	ds := h.getOrCreate(deviceID)

	// Buffer de 2 frames: máximo ~200ms de lag acumulado a 10fps.
	// Con buffer=8 a 3fps se podían acumular >2s de lag antes de descartar.
	ch := make(chan []byte, 2)
	ds.mu.Lock()
	ds.clients = append(ds.clients, ch)
	if len(ds.latest) > 0 {
		// Send the most-recent frame immediately so the browser shows something
		// as soon as the <img> tag connects, without waiting for the next push.
		select {
		case ch <- ds.latest:
		default:
		}
	}
	ds.mu.Unlock()

	defer func() {
		ds.mu.Lock()
		for i, c := range ds.clients {
			if c == ch {
				ds.clients = append(ds.clients[:i], ds.clients[i+1:]...)
				close(ch)
				break
			}
		}
		ds.mu.Unlock()
	}()

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// Tell upstream nginx/caddy not to buffer this response.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case frame, ok := <-ch:
			if !ok {
				return
			}
			keepalive.Reset(20 * time.Second)
			fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
			if _, err := w.Write(frame); err != nil {
				return
			}
			fmt.Fprint(w, "\r\n")
			flusher.Flush()
		case <-keepalive.C:
			// Empty keep-alive part so nginx/proxies don't drop an idle connection.
			fmt.Fprint(w, "--frame\r\nContent-Length: 0\r\n\r\n\r\n")
			flusher.Flush()
		}
	}
}

// HandleROI returns the ROI configuration last received from the edge device.
// Supports two formats from the edge:
//   - New:  {"lines": {"14": {roi, roi_presencia, source_w, source_h}}}
//   - Legacy: {"roi": {...}, "roi_presencia": {...}}
//
// Query params: device_id (required), linea_id (optional, cloud linea id).
// Auth is enforced by the router (JWTStream) before this handler is called.
func (h *CameraHub) HandleROI(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")
	lineaID := r.URL.Query().Get("linea_id")
	if deviceID == "" {
		http.Error(w, `{"error":"device_id required"}`, http.StatusBadRequest)
		return
	}

	h.mu.RLock()
	ds, ok := h.devices[deviceID]
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	empty := `{"roi":null,"roi_presencia":null}`

	if !ok {
		fmt.Fprint(w, empty)
		return
	}

	ds.mu.Lock()
	roiJSON := ds.roiJSON
	ds.mu.Unlock()

	if roiJSON == "" || !json.Valid([]byte(roiJSON)) {
		fmt.Fprint(w, empty)
		return
	}

	// Detect new multi-line format: {"lines": {"14": {...}}}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(roiJSON), &raw); err != nil {
		fmt.Fprint(w, empty)
		return
	}

	if linesRaw, ok := raw["lines"]; ok {
		var linesMap map[string]json.RawMessage
		if err := json.Unmarshal(linesRaw, &linesMap); err == nil {
			var payload json.RawMessage
			if lineaID != "" {
				payload = linesMap[lineaID]
				// Si se especificó una línea concreta y no está en el mapa,
				// devolver vacío en lugar de hacer fallback a otra línea.
			} else {
				// Sin linea_id: primer línea disponible (compatibilidad legacy)
				for _, v := range linesMap {
					payload = v
					break
				}
			}
			if payload != nil {
				w.WriteHeader(http.StatusOK)
				w.Write(payload) //nolint:errcheck
				return
			}
		}
	}

	// Formato legacy: {"roi":{}, "roi_presencia":{}}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, roiJSON)
}
