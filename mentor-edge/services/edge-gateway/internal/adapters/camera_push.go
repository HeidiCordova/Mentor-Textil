package adapters

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"
)

// CameraPushClient streams live MJPEG video from the local RTSP camera to
// the cloud-gateway via a long-lived HTTP POST. The cloud-gateway receives
// the stream and fans it out to browser clients without storing anything.
//
// Endpoint: POST <cloudURL>/api/v1/camera/push?device_id=X
// Auth:     X-API-Key + X-Device-ID headers
//
// ffmpeg encodes at 3 fps / 640 px wide to keep upstream bandwidth low (~60 KB/s).
type CameraPushClient struct {
	cloudURL     string
	apiKey       string
	deviceID     string
	getCameraURL func() string // called on every reconnect attempt
	getROI       func() string // returns JSON string with roi+roi_presencia or ""
}

func NewCameraPushClient(cloudURL, apiKey, deviceID string, getCameraURL func() string, getROI func() string) *CameraPushClient {
	return &CameraPushClient{
		cloudURL:     cloudURL,
		apiKey:       apiKey,
		deviceID:     deviceID,
		getCameraURL: getCameraURL,
		getROI:       getROI,
	}
}

// Start runs the push loop until ctx is cancelled. Call in a goroutine.
func (c *CameraPushClient) Start(ctx context.Context) {
	log.Printf("[camera-push] starting → %s", c.cloudURL)
	for {
		if ctx.Err() != nil {
			return
		}
		cameraURL := c.getCameraURL()
		if cameraURL == "" {
			log.Printf("[camera-push] no camera URL configured, retrying in 30s")
			select {
			case <-ctx.Done():
				return
			case <-time.After(30 * time.Second):
			}
			continue
		}

		if err := c.push(ctx, cameraURL); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[camera-push] disconnected: %v — retrying in 10s", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}

func (c *CameraPushClient) push(ctx context.Context, cameraURL string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("[camera-push] ffmpeg not found, disabling camera push (retrying in 60s)")
		time.Sleep(60 * time.Second)
		return nil
	}

	// Per-push context so we can kill ffmpeg when the HTTP connection ends.
	pushCtx, pushCancel := context.WithCancel(ctx)

	cmd := exec.CommandContext(pushCtx, ffmpegPath,
		"-fflags", "nobuffer+flush_packets",
		"-flags", "low_delay",
		"-analyzeduration", "0",
		"-probesize", "32768",
		"-rtsp_transport", "udp",
		"-i", cameraURL,
		"-an",      // no audio
		"-r", "10", // 10 fps — fluido sin aumentar costo neto de ancho de banda
		"-q:v", "18", // JPEG quality: frames más pequeños que q10 pero suficientes
		"-vf", "scale=480:-2", // 480 px wide — menor carga por frame
		"-f", "image2pipe",
		"-vcodec", "mjpeg",
		"-flush_packets", "1",
		"pipe:1",
	)

	stdout, pipeErr := cmd.StdoutPipe()
	if pipeErr != nil {
		return pipeErr
	}
	stderrPipe, _ := cmd.StderrPipe()
	go io.Copy(io.Discard, stderrPipe)

	if startErr := cmd.Start(); startErr != nil {
		pushCancel()
		return startErr
	}
	// LIFO: pushCancel (cancela ffmpeg) corre ANTES que cmd.Wait (lo espera).
	// Registrar en orden inverso al deseado de ejecución.
	defer cmd.Wait()   // corre 2do: ffmpeg ya terminó, retorna rápido
	defer pushCancel() // corre 1ero: mata ffmpeg vía contexto

	// Pipe reader/writer bridge: goroutine converts raw JPEG bytes from ffmpeg
	// into a multipart/x-mixed-replace stream and writes it to pr.
	pr, pw := io.Pipe()
	defer pr.CloseWithError(fmt.Errorf("push ended")) // desbloquea goroutine escritor al salir
	go func() {
		defer pw.Close()

		buf := make([]byte, 0, 512*1024)
		tmp := make([]byte, 128*1024)
		soi := []byte{0xFF, 0xD8}
		eoi := []byte{0xFF, 0xD9}

		for {
			n, readErr := stdout.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
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
					frame := buf[:end]

					fmt.Fprintf(pw, "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %d\r\n\r\n", len(frame))
					if _, err := pw.Write(frame); err != nil {
						return
					}
					fmt.Fprint(pw, "\r\n")

					buf = append(buf[:0], buf[end:]...)
				}
				if len(buf) > 4*1024*1024 {
					buf = buf[len(buf)-2:]
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	targetURL := fmt.Sprintf("%s/api/v1/camera/push?device_id=%s", c.cloudURL, c.deviceID)
	req, reqErr := http.NewRequestWithContext(pushCtx, http.MethodPost, targetURL, pr)
	if reqErr != nil {
		pr.Close()
		return reqErr
	}
	req.Header.Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Device-ID", c.deviceID)
	if c.getROI != nil {
		if roiJSON := c.getROI(); roiJSON != "" {
			req.Header.Set("X-Camera-ROI", roiJSON)
		}
	}

	httpClient := &http.Client{} // no global timeout — this is a long-lived connection
	resp, doErr := httpClient.Do(req)
	if doErr != nil {
		return doErr
	}
	defer resp.Body.Close()
	return fmt.Errorf("push connection closed with status %d", resp.StatusCode)
}
