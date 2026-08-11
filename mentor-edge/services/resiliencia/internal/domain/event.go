package domain

import (
	"errors"
	"time"
)

var ErrVisionCounterBoundary = errors.New("vision counter boundary is outside the active epoch")

type EventBuffer struct {
	ID         int        `json:"id"`
	EventID    string     `json:"event_id"`
	DeviceID   string     `json:"device_id"`
	EventType  string     `json:"event_type"`
	Timestamp  time.Time  `json:"timestamp"`
	Payload    []byte     `json:"payload"`
	Synced     bool       `json:"synced"`
	Dead       bool       `json:"dead"`
	RetryCount int        `json:"retry_count"`
	CreatedAt  time.Time  `json:"created_at"`
	SyncedAt   *time.Time `json:"synced_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
}

type BufferStats struct {
	TotalCount    int64      `json:"total_count"`
	PendingCount  int64      `json:"pending_count"`
	SyncedCount   int64      `json:"synced_count"`
	DeadCount     int64      `json:"dead_count"`
	OldestPending *time.Time `json:"oldest_pending,omitempty"`
	NewestPending *time.Time `json:"newest_pending,omitempty"`
	DiskBytes     int64      `json:"disk_bytes"`
}

// VisionCount is a bounded count of unique CORTE detections stored in one
// line's vision_detections table. The caller must provide both boundaries;
// resiliencia deliberately does not expose an unbounded lifetime counter.
type VisionCount struct {
	LineaID   int       `json:"linea_id"`
	Count     int64     `json:"count"`
	Since     time.Time `json:"since"`
	Until     time.Time `json:"until"`
	AsOf      time.Time `json:"as_of"`
	EventType string    `json:"event_type"`
}

// VisionCounter is the durable raw-machine counter consumed by the legacy
// Node-RED/Modbus adapter. It is not reset by calibration or product changes.
// CounterEpoch changes only after an explicit operational reset.
type VisionCounter struct {
	LineaID        int       `json:"linea_id"`
	Count          int64     `json:"count"`
	CounterEpoch   time.Time `json:"counter_epoch"`
	Until          time.Time `json:"until"`
	AsOf           time.Time `json:"as_of"`
	StateUpdatedAt time.Time `json:"state_updated_at"`
	EventType      string    `json:"event_type"`
}

type DeduplicationPolicy interface {
	IsDuplicate(eventID string) bool
	MarkProcessed(eventID string)
}

type QueuePolicy interface {
	ShouldAccept(event *EventBuffer) bool
	GetPriority(event *EventBuffer) int
}
