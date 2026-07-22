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

func (n *SyncNotifier) Broadcast(ctx context.Context, resource string, empresaID int64, records interface{}) {
	if n.gatewayURL == "" || n.internalKey == "" {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"resource":   resource,
		"empresa_id": empresaID,
		"records":    records,
	})
	if err != nil {
		log.Printf("[sync-notifier] marshal error: %v", err)
		return
	}

	body, _ := json.Marshal(map[string]interface{}{
		"empresa_id": empresaID,
		"type":       "SYNC_USUARIOS",
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
		log.Printf("[sync-notifier] broadcast error resource=%s empresa=%d: %v", resource, empresaID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("[sync-notifier] broadcast status=%d resource=%s empresa=%d", resp.StatusCode, resource, empresaID)
		return
	}
	log.Printf("[sync-notifier] broadcast ok resource=%s empresa=%d", resource, empresaID)
}
