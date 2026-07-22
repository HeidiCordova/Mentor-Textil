package ports

import (
	"context"
	"time"

	"edge-gateway/internal/domain"
)

type StopRepository interface {
	Create(ctx context.Context, req domain.CreateStopRequest) (*domain.Stop, error)
	GetByID(ctx context.Context, stopID string) (*domain.Stop, error)
	Update(ctx context.Context, stop *domain.Stop) error
	Justify(ctx context.Context, req domain.JustifyStopRequest) (*domain.Stop, error)
	List(ctx context.Context, filter domain.StopFilter) ([]domain.Stop, error)
	CloseStop(ctx context.Context, stopID string, endedAt interface{}) (*domain.Stop, error)
	Delete(ctx context.Context, stopID string) error
	GetSummary(ctx context.Context, deviceID string, hours int) (*domain.StopSummary, error)
	GetDurationByType(ctx context.Context, deviceID string, intervalS int) (map[string]int64, error)
	CloseStaleStops(ctx context.Context, staleAfter time.Duration) (int, error)
}
