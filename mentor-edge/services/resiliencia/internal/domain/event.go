package domain

import "time"

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

type DeduplicationPolicy interface {
	IsDuplicate(eventID string) bool
	MarkProcessed(eventID string)
}

type QueuePolicy interface {
	ShouldAccept(event *EventBuffer) bool
	GetPriority(event *EventBuffer) int
}
