package ports

import (
	"context"
	"time"

	"edge-gateway/internal/domain"
)

type ConfigClient interface {
	GetConfig(ctx context.Context) (map[string]interface{}, error)
	UpdateConfig(ctx context.Context, patch map[string]interface{}) (map[string]interface{}, error)
	GetConfigVersion(ctx context.Context) (int, error)
	StartCalibration(ctx context.Context) error
}

type BufferClient interface {
	GetSummary(ctx context.Context) (*domain.BufferSummary, error)
	GetRecentEvents(ctx context.Context, limit int, since *time.Time) ([]domain.Event, error)
	GetPendingEvents(ctx context.Context, limit int) ([]domain.Event, error)
	Health(ctx context.Context) (string, error)
}

type DetectorClient interface {
	Health(ctx context.Context) (string, error)
	CalibrationStatus(ctx context.Context) (map[string]interface{}, error)
}

type EnviadorClient interface {
	Health(ctx context.Context) (string, error)
}
