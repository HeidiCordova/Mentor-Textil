package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GatewayCommandDispatcher sends commands to the cloud-gateway for SSE delivery to edge devices.
type GatewayCommandDispatcher struct {
	gatewayURL  string
	internalKey string
	client      *http.Client
}

func NewGatewayCommandDispatcher(gatewayURL, internalKey string) *GatewayCommandDispatcher {
	return &GatewayCommandDispatcher{
		gatewayURL:  gatewayURL,
		internalKey: internalKey,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

func (d *GatewayCommandDispatcher) Dispatch(ctx context.Context, deviceID string, cmdType string, payload map[string]interface{}, userID int64, empresaID int64) error {
	body := map[string]interface{}{
		"device_id":  deviceID,
		"type":       cmdType,
		"payload":    payload,
		"user_id":    userID,
		"empresa_id": empresaID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		d.gatewayURL+"/internal/dispatch", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", d.internalKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("dispatch al gateway falló: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("gateway respondió %d", resp.StatusCode)
	}
	return nil
}
