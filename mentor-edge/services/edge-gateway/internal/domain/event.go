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
