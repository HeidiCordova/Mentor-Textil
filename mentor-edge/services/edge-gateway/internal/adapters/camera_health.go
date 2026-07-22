package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CameraHealthResult struct {
	URL          string `json:"url"`
	URLMasked    string `json:"url_masked"`
	TCPReach     bool   `json:"tcp_reachable"`
	TCPLatencyMs int64  `json:"tcp_latency_ms"`
	RTSPOK       bool   `json:"rtsp_ok"`
	RTSPCodec    string `json:"rtsp_codec,omitempty"`
	RTSPMsg      string `json:"rtsp_message,omitempty"`
	FfprobeAvail bool   `json:"ffprobe_available"`
	Status       string `json:"status"`
	CheckedAt    string `json:"checked_at"`
}

func (s *HTTPServer) handleCameraHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	cameraURL := r.URL.Query().Get("url")
	if cameraURL == "" {
		cameraURL = s.resolveCameraURL(r.Context())
	}

	if cameraURL == "" {
		s.badRequest(w, "no camera URL configured or provided via ?url=")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	result := checkCamera(ctx, cameraURL)
	code := http.StatusOK
	if result.Status != "ok" {
		code = http.StatusServiceUnavailable
	}
	s.jsonResponse(w, code, result)
}

func checkCamera(ctx context.Context, rawURL string) CameraHealthResult {
	res := CameraHealthResult{
		URL:       rawURL,
		URLMasked: maskRTSPCredentials(rawURL),
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// TCP check
	host, port := extractHostPort(rawURL)
	if host != "" {
		t0 := time.Now()
		tcpCtx, tcpCancel := context.WithTimeout(ctx, 3*time.Second)
		defer tcpCancel()
		conn, err := (&net.Dialer{}).DialContext(tcpCtx, "tcp", net.JoinHostPort(host, port))
		latency := time.Since(t0).Milliseconds()
		if err == nil {
			conn.Close()
			res.TCPReach = true
			res.TCPLatencyMs = latency
		} else {
			res.Status = "tcp_unreachable"
			res.RTSPMsg = fmt.Sprintf("TCP %s:%s no alcanzable: %v", host, port, err)
			return res
		}
	}

	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		res.FfprobeAvail = false
		if res.TCPReach {
			res.Status = "tcp_ok_no_ffprobe"
			res.RTSPMsg = "TCP alcanzable pero ffprobe no disponible para verificar RTSP"
		}
		return res
	}
	res.FfprobeAvail = true

	cmd := exec.CommandContext(ctx,
		ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_streams",
		"-rtsp_transport", "udp",
		"-analyzeduration", "2000000",
		"-probesize", "500000",
		rawURL,
	)

	out, err := cmd.Output()
	if err != nil {
		// ffprobe falló — puede ser que la cámara esté al máximo de conexiones concurrentes.
		// Un RTSP OPTIONS no requiere slot de video y confirma si la cámara responde.
		if checkRTSPOptions(ctx, rawURL) {
			res.RTSPOK = true
			res.Status = "ok_busy"
			res.RTSPMsg = "Cámara accesible — en uso por servicios activos (máx. conexiones alcanzado)"
			return res
		}
		res.RTSPOK = false
		res.Status = "rtsp_error"
		exitErr := ""
		if e, ok := err.(*exec.ExitError); ok {
			exitErr = string(e.Stderr)
		}
		res.RTSPMsg = fmt.Sprintf("ffprobe error: %v %s", err, strings.TrimSpace(exitErr))
		return res
	}

	var probe struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &probe); err != nil || len(probe.Streams) == 0 {
		res.RTSPOK = false
		res.Status = "rtsp_no_streams"
		res.RTSPMsg = "RTSP conectado pero sin streams detectados"
		return res
	}

	codecs := []string{}
	for _, st := range probe.Streams {
		if st.CodecType == "video" {
			info := st.CodecName
			if st.Width > 0 {
				info += fmt.Sprintf(" %dx%d", st.Width, st.Height)
			}
			codecs = append(codecs, info)
		}
	}

	res.RTSPOK = true
	res.RTSPCodec = strings.Join(codecs, ", ")
	res.RTSPMsg = fmt.Sprintf("%d stream(s) detectados", len(probe.Streams))
	res.Status = "ok"
	return res
}

// yoloStreamAvailable hace un GET request con timeout corto para saber si
// el stream upstream está disponible (detector /stream o yolo-counter).
func yoloStreamAvailable(yoloURL string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, yoloURL, nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode < 500
}

// proxyMJPEGStream copia el stream MJPEG de src hacia w, pasando headers.
func proxyMJPEGStream(w http.ResponseWriter, r *http.Request, srcURL string) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, srcURL, nil)
	if err != nil {
		http.Error(w, `{"error":"proxy request error"}`, http.StatusBadGateway)
		return
	}
	client := &http.Client{Timeout: 0} // sin timeout — stream continuo
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, `{"error":"upstream unavailable"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copiar headers de contenido
	for _, h := range []string{"Content-Type", "Cache-Control", "Connection"} {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.WriteHeader(resp.StatusCode)

	// El goroutine de flush se detiene con el canal done ANTES de que io.Copy
	// retorne, evitando una carrera con el cierre de chunked-encoding.
	done := make(chan struct{})
	if f, ok := w.(http.Flusher); ok {
		ticker := time.NewTicker(40 * time.Millisecond)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-r.Context().Done():
					return
				case <-ticker.C:
					f.Flush()
				}
			}
		}()
	}
	io.Copy(w, resp.Body)
	close(done)
}

// ffmpegMJPEGStream abre cameraURL con ffmpeg y envía los frames como
// multipart/x-mixed-replace. Es la implementación base del streaming de cámara.
func ffmpegMJPEGStream(w http.ResponseWriter, r *http.Request, cameraURL string) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		http.Error(w, `{"error":"ffmpeg not available on this host"}`, http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-fflags", "nobuffer+fastseek+flush_packets",
		"-flags", "low_delay",
		"-analyzeduration", "0",
		"-probesize", "32768",
		"-rtsp_transport", "udp",
		"-i", cameraURL,
		"-an",
		"-q:v", "5",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-flush_packets", "1",
		"pipe:1",
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, `{"error":"pipe error"}`, http.StatusInternalServerError)
		return
	}
	stderr, _ := cmd.StderrPipe()
	go io.Copy(io.Discard, stderr)

	if err := cmd.Start(); err != nil {
		http.Error(w, `{"error":"failed to start ffmpeg"}`, http.StatusServiceUnavailable)
		return
	}
	defer cmd.Wait()

	rc := http.NewResponseController(w)
	rc.SetWriteDeadline(time.Time{})

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	const maxBuf = 2 * 1024 * 1024
	buf := make([]byte, 0, 512*1024)
	tmp := make([]byte, 128*1024)

	soi := []byte{0xFF, 0xD8}
	eoi := []byte{0xFF, 0xD9}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, readErr := stdout.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)

			var latest []byte
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
				latest = make([]byte, end)
				copy(latest, buf[:end])
				buf = append(buf[:0], buf[end:]...)
			}

			if latest != nil {
				fmt.Fprintf(w, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(latest))
				w.Write(latest)
				fmt.Fprint(w, "\r\n")
				flusher.Flush()
			}

			if len(buf) > maxBuf {
				buf = buf[len(buf)-2:]
			}
		}
		if readErr != nil {
			return
		}
	}
}

// EarlyCameraHealthHandler permite verificar la cámara antes de que haya líneas.
func EarlyCameraHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	cameraURL := r.URL.Query().Get("url")
	if cameraURL == "" {
		http.Error(w, `{"error":"url query param required"}`, http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	result := checkCamera(ctx, cameraURL)
	code := http.StatusOK
	if result.Status != "ok" && result.Status != "ok_busy" {
		code = http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(result)
}

// EarlyCameraStreamHandler es un http.HandlerFunc standalone que puede ser
// usado ANTES de que el gateway tenga líneas configuradas (ej: servidor pre-loop).
// Intenta primero yolo-counter y si no está disponible usa ffmpeg directamente.
func EarlyCameraStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	cameraURL := r.URL.Query().Get("url")
	if cameraURL == "" {
		http.Error(w, `{"error":"url query param required"}`, http.StatusBadRequest)
		return
	}
	yoloStreamURL := os.Getenv("YOLO_STREAM_URL")
	if yoloStreamURL == "" {
		yoloStreamURL = "http://host.docker.internal:8006/lab/stream"
	}
	if yoloStreamAvailable(yoloStreamURL) {
		proxyMJPEGStream(w, r, yoloStreamURL)
		return
	}
	ffmpegMJPEGStream(w, r, cameraURL)
}

func (s *HTTPServer) handleCameraStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	// Primero: intentar stream del detector (corre en host-network, ya tiene RTSP).
	// No necesita cameraURL — el detector tiene acceso directo a la cámara.
	detectorBase := os.Getenv("DETECTOR_URL")
	if detectorBase == "" {
		detectorBase = "http://host.docker.internal:8001"
	}
	detectorStreamURL := strings.TrimRight(detectorBase, "/") + "/stream"

	// Verificar disponibilidad con /frame (endpoint pequeño, no stream infinito).
	if _, err := fetchDetectorFrame(r.Context(), detectorBase); err == nil {
		proxyMJPEGStream(w, r, detectorStreamURL)
		return
	}

	// Segundo: yolo-counter (solo botellas — puerto 8006).
	yoloStreamURL := os.Getenv("YOLO_STREAM_URL")
	if yoloStreamURL == "" {
		yoloStreamURL = "http://host.docker.internal:8006/lab/stream"
	}
	if yoloStreamAvailable(yoloStreamURL) {
		proxyMJPEGStream(w, r, yoloStreamURL)
		return
	}

	// Fallback: ffmpeg directo — requiere cameraURL y acceso RTSP desde host-network.
	cameraURL := r.URL.Query().Get("url")
	if cameraURL == "" {
		cameraURL = s.resolveCameraURL(r.Context())
	}
	if cameraURL == "" {
		s.badRequest(w, "no camera URL configured or provided via ?url=")
		return
	}
	ffmpegMJPEGStream(w, r, cameraURL)
}

func (s *HTTPServer) handleCameraSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.methodNotAllowed(w)
		return
	}

	// Intentar primero obtener el último frame del detector (corre en host-network
	// y ya tiene acceso a la cámara RTSP). Esto evita que el gateway tenga que
	// conectarse directamente a la cámara, lo cual falla en redes bridge de Docker.
	detectorBase := os.Getenv("DETECTOR_URL")
	if detectorBase == "" {
		detectorBase = "http://host.docker.internal:8001"
	}
	if jpeg, err := fetchDetectorFrame(r.Context(), detectorBase); err == nil {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jpeg)))
		w.WriteHeader(http.StatusOK)
		w.Write(jpeg)
		return
	}

	// Fallback: intentar captura directa con ffmpeg (requiere acceso RTSP).
	cameraURL := r.URL.Query().Get("url")
	if cameraURL == "" {
		cameraURL = s.resolveCameraURL(r.Context())
	}
	if cameraURL == "" {
		http.Error(w, `{"error":"no frame available and no camera URL configured"}`, http.StatusServiceUnavailable)
		return
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		http.Error(w, `{"error":"ffmpeg not available on this host"}`, http.StatusServiceUnavailable)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffmpegPath,
		"-fflags", "nobuffer",
		"-flags", "low_delay",
		"-rtsp_transport", "tcp",
		"-analyzeduration", "0",
		"-probesize", "32768",
		"-i", cameraURL,
		"-frames:v", "1",
		"-q:v", "3",
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"pipe:1",
	)

	jpegData, err := cmd.Output()
	if err != nil || len(jpegData) == 0 {
		http.Error(w, `{"error":"failed to capture frame from camera"}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jpegData)))
	w.WriteHeader(http.StatusOK)
	w.Write(jpegData)
}

// fetchDetectorFrame pide el último frame JPEG al servicio detector.
func fetchDetectorFrame(ctx context.Context, detectorBase string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, detectorBase+"/frame", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("detector /frame returned %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("empty frame body")
	}
	return data, nil
}

// checkRTSPOptions envía un RTSP OPTIONS a la cámara sin abrir un slot de video.
// Retorna true si la cámara responde con RTSP/1.0 200, lo que indica que está
// operativa aunque rechace nuevas conexiones de media por estar al límite.
func checkRTSPOptions(ctx context.Context, rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "554"
	}
	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(dialCtx, "tcp", net.JoinHostPort(host, port))
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second)) //nolint:errcheck
	req := fmt.Sprintf("OPTIONS %s RTSP/1.0\r\nCSeq: 1\r\n\r\n", rawURL)
	if _, err := conn.Write([]byte(req)); err != nil {
		return false
	}
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err != nil || n == 0 {
		return false
	}
	return strings.HasPrefix(string(buf[:n]), "RTSP/1.0 200")
}

func maskRTSPCredentials(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.User == nil {
		return rawURL
	}
	masked := u.Scheme + "://****:****@" + u.Host + u.Path
	if u.RawQuery != "" {
		masked += "?" + u.RawQuery
	}
	return masked
}

func extractHostPort(rawURL string) (string, string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "rtsp":
			port = "554"
		case "http":
			port = "80"
		case "https":
			port = "443"
		default:
			port = "554"
		}
	}
	return host, port
}

func (s *HTTPServer) resolveCameraURL(ctx context.Context) string {
	cfg, err := s.defaultLine().Gateway.GetConfig(ctx)
	if err != nil {
		return ""
	}
	if cam, ok := cfg["camera"].(map[string]interface{}); ok {
		if u, ok := cam["url"].(string); ok {
			return u
		}
	}
	if u, ok := cfg["camera_url"].(string); ok {
		return u
	}
	return ""
}
