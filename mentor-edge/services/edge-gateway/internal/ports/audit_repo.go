package ports

import (
	"context"
	"edge-gateway/internal/domain"
)

type AuditRepository interface {
	Log(ctx context.Context, entry domain.AuditEntry) error
	ListByDevice(ctx context.Context, deviceID string, limit int) ([]domain.AuditEntry, error)
	PurgeOlderThan(ctx context.Context, retentionDays int) (int, error)
}
