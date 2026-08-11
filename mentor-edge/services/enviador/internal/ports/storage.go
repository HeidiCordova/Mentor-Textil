package ports

import (
	"context"
	"time"
)

type EventStorage interface {
	GetUnsyncedEvents(ctx context.Context, limit int) ([]Event, error)
	MarkSynced(ctx context.Context, eventIDs []string) error
	IncrementRetry(ctx context.Context, eventID string) error
	MarkDead(ctx context.Context, eventIDs []string) error
	GetCloudSyncInterval(ctx context.Context, lineaID int) (int, error)
	GetCloudCredentials(ctx context.Context, lineaID int) (cloudURL, apiKey string, err error)
	GetUnsyncedStops(ctx context.Context, limit int) ([]StopRecord, error)
	MarkStopsSynced(ctx context.Context, stopIDs []string) error
	GetUnsyncedProductionRuns(ctx context.Context, limit int) ([]ProductionRunRecord, error)
	MarkProductionRunsSynced(ctx context.Context, runIDs []string) error
	GetUnsyncedEnergyEvents(ctx context.Context, limit int) ([]Event, error)
	// ApplyPendingCommand aplica un comando recibido desde cloud al edge DB.
	ApplyPendingCommand(ctx context.Context, cmd PendingCommand) error
	// GetCurrentMode devuelve el modo de operación textil de la línea.
	GetCurrentMode(ctx context.Context, lineaID int) (string, error)
}

type ProductionRunRecord struct {
	RunID      string  `json:"run_id"`
	DeviceID   string  `json:"device_id"`
	ProductoID *int    `json:"producto_id,omitempty"`
	SKU        *string `json:"sku,omitempty"`
	Nombre     *string `json:"nombre,omitempty"`
	StartedAt  string  `json:"started_at"`
	EndedAt    *string `json:"ended_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type Event struct {
	ID         int       `json:"id"`
	EventID    string    `json:"event_id"`
	DeviceID   string    `json:"device_id"`
	EventType  string    `json:"event_type"`
	Timestamp  time.Time `json:"timestamp"`
	Payload    []byte    `json:"payload"`
	RetryCount int       `json:"retry_count"`
}
