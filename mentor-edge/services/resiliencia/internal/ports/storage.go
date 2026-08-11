package ports

import (
	"context"
	"time"

	"resiliencia/internal/domain"
)

type EventStorage interface {
	Store(ctx context.Context, event *domain.EventBuffer) error
	GetUnsyncedEvents(ctx context.Context, limit int) ([]*domain.EventBuffer, error)
	GetRecentEvents(ctx context.Context, limit int, since *time.Time) ([]*domain.EventBuffer, error)
	GetVisionCount(ctx context.Context, since, until time.Time) (*domain.VisionCount, error)
	GetVisionCounter(ctx context.Context, until time.Time) (*domain.VisionCounter, error)
	GetPendingEvents(ctx context.Context, limit int) ([]*domain.EventBuffer, error)
	MarkSynced(ctx context.Context, eventIDs []string) error
	EventExists(ctx context.Context, eventID string) (bool, error)
	GetPendingCount(ctx context.Context) (int, error)
	PurgeExpired(ctx context.Context) (int, error)
	MarkStaleEventsDead(ctx context.Context, maxAgeHours int) (int, error)
	GetBufferStats(ctx context.Context) (*domain.BufferStats, error)
	EmergencyPurge(ctx context.Context, keepCount int) (int, error)
}
