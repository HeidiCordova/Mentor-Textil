package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"enviador/internal/ports"
)

type HTTPCloudClient struct {
	mu         sync.RWMutex
	cloudURL   string
	httpClient *http.Client
	deviceID   string
	apiKey     string
	lineaID    string
}

func NewHTTPCloudClient(cloudURL, deviceID, apiKey, lineaID string, timeout time.Duration) ports.CloudClient {
	return &HTTPCloudClient{
		cloudURL: cloudURL,
		deviceID: deviceID,
		apiKey:   apiKey,
		lineaID:  lineaID,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// UpdateCredentials actualiza la URL y API key en caliente sin reiniciar el servicio.
func (c *HTTPCloudClient) UpdateCredentials(cloudURL, apiKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if cloudURL != "" {
		c.cloudURL = cloudURL
	}
	if apiKey != "" {
		c.apiKey = apiKey
	}
}

func (c *HTTPCloudClient) SendOEE(ctx context.Context, records []ports.OEERecord) error {
	return c.postJSON(ctx, "/api/v1/edge/oee", records)
}

func (c *HTTPCloudClient) SendStops(ctx context.Context, stops []ports.StopRecord) error {
	return c.postJSON(ctx, "/api/v1/edge/stops-sync", stops)
}

func (c *HTTPCloudClient) SendProductionRuns(ctx context.Context, runs []ports.ProductionRunRecord) error {
	return c.postJSON(ctx, "/api/v1/edge/production-runs-sync", runs)
}

func (c *HTTPCloudClient) SendEnergy(ctx context.Context, records []ports.EnergyRecord) error {
	return c.postJSON(ctx, "/api/v1/energy/snapshots", records)
}

func (c *HTTPCloudClient) Heartbeat(ctx context.Context) (*ports.HeartbeatInfo, error) {
	c.mu.RLock()
	cloudURL := c.cloudURL
	apiKey := c.apiKey
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "POST", cloudURL+"/api/v1/edge/heartbeat", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Device-ID", c.deviceID)
	req.Header.Set("X-Linea-ID", c.lineaID)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("heartbeat: cloud returned status %d", resp.StatusCode)
	}
	var info ports.HeartbeatInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		// Si el cloud no devuelve JSON válido (ej. versión vieja), no es error
		return nil, nil //nolint
	}
	return &info, nil
}

func (c *HTTPCloudClient) postJSON(ctx context.Context, path string, payload interface{}) error {
	c.mu.RLock()
	cloudURL := c.cloudURL
	apiKey := c.apiKey
	c.mu.RUnlock()

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cloudURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", c.deviceID)
	req.Header.Set("X-Linea-ID", c.lineaID)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("cloud returned status %d for %s", resp.StatusCode, path)
	}

	return nil
}

// GetPendingCommands consulta los comandos pendientes (no aplicados) para este dispositivo.
func (c *HTTPCloudClient) GetPendingCommands(ctx context.Context) ([]ports.PendingCommand, error) {
	c.mu.RLock()
	cloudURL := c.cloudURL
	apiKey := c.apiKey
	c.mu.RUnlock()

	req, err := http.NewRequestWithContext(ctx, "GET", cloudURL+"/api/v1/edge/pending-commands", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Device-ID", c.deviceID)
	req.Header.Set("X-Linea-ID", c.lineaID)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pending-commands: cloud returned status %d", resp.StatusCode)
	}

	var cmds []ports.PendingCommand
	if err := json.NewDecoder(resp.Body).Decode(&cmds); err != nil {
		return nil, fmt.Errorf("pending-commands decode: %w", err)
	}
	return cmds, nil
}

// AckPendingCommands notifica al cloud que los comandos con los IDs dados fueron aplicados.
func (c *HTTPCloudClient) AckPendingCommands(ctx context.Context, ids []int64) error {
	return c.postJSON(ctx, "/api/v1/edge/pending-commands/ack", map[string]interface{}{"ids": ids})
}

// SyncMode sincroniza el modo de operación de la línea (textil/botellas) al cloud.
func (c *HTTPCloudClient) SyncMode(ctx context.Context, lineaID int, mode string) error {
	return c.putJSON(ctx, "/api/v1/edge/linea-config", map[string]interface{}{
		"linea_id": lineaID,
		"mode":     mode,
	})
}

func (c *HTTPCloudClient) putJSON(ctx context.Context, path string, payload interface{}) error {
	c.mu.RLock()
	cloudURL := c.cloudURL
	apiKey := c.apiKey
	c.mu.RUnlock()

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", cloudURL+path, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device-ID", c.deviceID)
	req.Header.Set("X-Linea-ID", c.lineaID)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("cloud returned status %d for %s", resp.StatusCode, path)
	}
	return nil
}
