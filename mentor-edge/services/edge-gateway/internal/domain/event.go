package domain

import "time"

type Event struct {
	ID        int                    `json:"id"`
	EventID   string                 `json:"event_id"`
	DeviceID  string                 `json:"device_id"`
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Synced    bool                   `json:"synced"`
	Dead      bool                   `json:"dead"`
	CreatedAt time.Time              `json:"created_at"`
}

type BufferSummary struct {
	TotalCount    int64      `json:"total_count"`
	PendingCount  int64      `json:"pending_count"`
	SyncedCount   int64      `json:"synced_count"`
	DeadCount     int64      `json:"dead_count"`
	OldestPending *time.Time `json:"oldest_pending,omitempty"`
	NewestPending *time.Time `json:"newest_pending,omitempty"`
	DiskBytes     int64      `json:"disk_bytes"`
}

// VisionCountWindow is resiliencia's bounded, line-scoped count. It is an
// internal dependency response and never represents a lifetime counter.
type VisionCountWindow struct {
	LineaID   int       `json:"linea_id"`
	Count     int64     `json:"count"`
	Since     time.Time `json:"since"`
	Until     time.Time `json:"until"`
	AsOf      time.Time `json:"as_of"`
	EventType string    `json:"event_type"`
}

// VisionCount is the public count contract exposed by edge-gateway. Count and
// CounterEpoch are present only when Status is "active"; callers must stop
// publishing when Status is "no_active_run" or "no_product".
type VisionCount struct {
	Status       string     `json:"status"`
	LineaID      int        `json:"linea_id"`
	DeviceID     string     `json:"device_id"`
	RunID        string     `json:"run_id,omitempty"`
	ProductoID   *int       `json:"producto_id,omitempty"`
	SKU          *string    `json:"sku,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	Count        *int64     `json:"count,omitempty"`
	CounterEpoch string     `json:"counter_epoch,omitempty"`
	AsOf         time.Time  `json:"as_of"`
	EventType    string     `json:"event_type"`
}

type AggregatedHealth struct {
	Service  string            `json:"service"`
	Status   string            `json:"status"`
	DeviceID string            `json:"device_id"`
	Uptime   int64             `json:"uptime"`
	Deps     map[string]string `json:"deps"`
}

type EdgeStatus struct {
	DeviceID       string   `json:"device_id"`
	LineaID        int      `json:"linea_id"`
	BufferPending  int64    `json:"buffer_pending"`
	CloudConnected bool     `json:"cloud_connected"`
	ConfigVersion  int      `json:"config_version"`
	RecentErrors   []string `json:"recent_errors"`
	Uptime         int64    `json:"uptime"`
}

type SSEEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
