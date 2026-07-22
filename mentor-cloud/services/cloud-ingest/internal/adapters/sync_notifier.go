package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SyncNotifier forwards catalog updates to the cloud-gateway so they are
// pushed down to every connected edge device.
type SyncNotifier struct {
	gatewayURL  string
	internalKey string
	client      *http.Client
}

func NewSyncNotifier(gatewayURL, internalKey string) *SyncNotifier {
	return &SyncNotifier{
		gatewayURL:  gatewayURL,
		internalKey: internalKey,
		client:      &http.Client{Timeout: 10 * time.Second},
	}
}

// BroadcastVariables sends a SYNC_VARIABLES command to the gateway so all
// edge devices update their sync_variables table.
func (n *SyncNotifier) BroadcastVariables(ctx context.Context, empresaID int64, vars interface{}) {
	if n.gatewayURL == "" || n.internalKey == "" {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"resource":   "variables",
		"empresa_id": empresaID,
		"records":    vars,
	})
	if err != nil {
		log.Printf("[sync-notifier] marshal variables error: %v", err)
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"empresa_id": empresaID,
		"type":       "SYNC_VARIABLES",
		"payload":    json.RawMessage(payload),
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/internal/broadcast", n.gatewayURL), bytes.NewReader(body))
	if err != nil {
		log.Printf("[sync-notifier] request error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", n.internalKey)

	resp, err := n.client.Do(req)
	if err != nil {
		log.Printf("[sync-notifier] broadcast variables empresa=%d: %v", empresaID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[sync-notifier] broadcast variables status=%d empresa=%d", resp.StatusCode, empresaID)
		return
	}
	log.Printf("[sync-notifier] broadcast variables ok empresa=%d", empresaID)
}
