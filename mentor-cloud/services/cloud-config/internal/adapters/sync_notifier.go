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

	cmdType := "SYNC_CATALOG"
	switch resource {
	case "productos":
		cmdType = "SYNC_PRODUCTOS"
	case "turnos":
		cmdType = "SYNC_TURNOS"
	case "usuarios":
		cmdType = "SYNC_USUARIOS"
	case "variables":
		cmdType = "SYNC_VARIABLES"
	case "velocidad_nominal":
		cmdType = "SYNC_VELOCIDAD_NOMINAL"
	case "motivos_velocidad":
		cmdType = "SYNC_MOTIVOS_VELOCIDAD"
	case "plantas_lineas":
		cmdType = "SYNC_PLANTAS_LINEAS"
	case "linea_producto_vars":
		cmdType = "SYNC_LINEA_PRODUCTO_VARS"
	case "producto_caracteristicas":
		cmdType = "SYNC_PRODUCTO_CARACTERISTICAS"
	}

	body, _ := json.Marshal(map[string]interface{}{
		"empresa_id": empresaID,
		"type":       cmdType,
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

// NotifyBrowsers envía un evento catalog.changed al BrowserSSEHub del gateway
// para que las tablets en modo CLOUD y el browser cloud se actualicen en tiempo real.
func (n *SyncNotifier) NotifyBrowsers(ctx context.Context, eventType string, empresaID int64, lineaID int64) {
	if n.gatewayURL == "" || n.internalKey == "" {
		return
	}
	body, err := json.Marshal(map[string]interface{}{
		"type":       eventType,
		"empresa_id": empresaID,
		"linea_id":   lineaID,
	})
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		n.gatewayURL+"/internal/notify", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Key", n.internalKey)
	resp, err := n.client.Do(req)
	if err != nil {
		log.Printf("[sync-notifier] notify-browsers error type=%s empresa=%d: %v", eventType, empresaID, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[sync-notifier] notify-browsers ok type=%s empresa=%d linea=%d", eventType, empresaID, lineaID)
}
