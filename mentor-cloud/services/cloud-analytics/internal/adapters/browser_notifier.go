package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GatewayBrowserNotifier notifies connected browser clients via cloud-gateway /internal/notify.
type GatewayBrowserNotifier struct {
	gatewayURL  string
	internalKey string
	client      *http.Client
}

func NewGatewayBrowserNotifier(gatewayURL, internalKey string) *GatewayBrowserNotifier {
	return &GatewayBrowserNotifier{
		gatewayURL:  gatewayURL,
		internalKey: internalKey,
		client:      &http.Client{Timeout: 5 * time.Second},
	}
}

func (n *GatewayBrowserNotifier) Notify(ctx context.Context, eventType string, empresaID int64, lineaID int64) error {
	body := map[string]interface{}{
		"type":       eventType,
		"empresa_id": empresaID,
		"linea_id":   lineaID,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.gatewayURL+"/internal/notify", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", n.internalKey)

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("notify browser falló: %w", err)
	}
	defer resp.Body.Close()
	return nil
}
