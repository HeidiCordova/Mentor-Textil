package adapters

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// CloudCommand es el comando recibido desde el cloud via SSE.
type CloudCommand struct {
	ID        int64                  `json:"id"`
	DeviceID  string                 `json:"device_id"`
	EmpresaID int64                  `json:"empresa_id"`
	Type      string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Status    string                 `json:"status"`
	IssuedBy  int64                  `json:"issued_by"`
}

// CloudSSEClient mantiene una conexión SSE persistente con el cloud-gateway.
// En cuanto el cloud despacha un comando, llega aquí en tiempo real sin polling.
type CloudSSEClient struct {
	cloudURL  string
	apiKey    string
	deviceID  string
	client    *http.Client
	onCommand func(cmd CloudCommand) error
}

func NewCloudSSEClient(cloudURL, apiKey, deviceID string, onCommand func(CloudCommand) error) *CloudSSEClient {
	return &CloudSSEClient{
		cloudURL:  cloudURL,
		apiKey:    apiKey,
		deviceID:  deviceID,
		client:    &http.Client{}, // sin timeout — conexión larga por diseño
		onCommand: onCommand,
	}
}

// Start ejecuta el loop SSE para siempre, reconectando en cualquier error.
// Debe llamarse en una goroutine.
func (c *CloudSSEClient) Start(ctx context.Context) {
	log.Printf("[cloud-sse] iniciando → %s/api/v1/edge/stream (device: %s)", c.cloudURL, c.deviceID)
	for {
		if err := c.connect(ctx); err != nil {
			if ctx.Err() != nil {
				log.Printf("[cloud-sse] contexto cancelado, deteniendo")
				return
			}
			log.Printf("[cloud-sse] conexión perdida: %v — reintentando en 5s", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

func (c *CloudSSEClient) connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.cloudURL+"/api/v1/edge/stream", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Device-ID", c.deviceID)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloud devolvió %d", resp.StatusCode)
	}

	log.Printf("[cloud-sse] conectado ✓ (device: %s)", c.deviceID)

	scanner := bufio.NewScanner(resp.Body)
	var eventType, data string

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		case line == "": // línea vacía = fin del evento
			if data != "" && eventType == "command" {
				var cmd CloudCommand
				if jsonErr := json.Unmarshal([]byte(data), &cmd); jsonErr == nil {
					if execErr := c.onCommand(cmd); execErr != nil {
						log.Printf("[cloud-sse] error ejecutando cmd %d: %v", cmd.ID, execErr)
					}
				} else {
					log.Printf("[cloud-sse] json parse error: %v", jsonErr)
				}
			}
			// líneas ": heartbeat" o ": connected" son comentarios SSE — se ignoran
			eventType, data = "", ""
		}
	}
	return scanner.Err()
}

// AckCloud confirma la recepción del comando al cloud-gateway.
func (c *CloudSSEClient) AckCloud(ctx context.Context, cmdID int64) error {
	url := fmt.Sprintf("%s/api/v1/edge/commands/%d/ack", c.cloudURL, cmdID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("X-Device-ID", c.deviceID)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ack devolvió %d", resp.StatusCode)
	}
	log.Printf("[cloud-sse] cmd %d ACK enviado al cloud ✓", cmdID)
	return nil
}
