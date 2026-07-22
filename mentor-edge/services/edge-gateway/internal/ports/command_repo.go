package ports

import (
	"context"
	"edge-gateway/internal/domain"
)

type CommandRepository interface {
	Create(ctx context.Context, req domain.CreateCommandRequest) (*domain.Command, error)
	GetByID(ctx context.Context, commandID string) (*domain.Command, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Command, error)
	MarkApplied(ctx context.Context, commandID string, result map[string]interface{}) error
	MarkFailed(ctx context.Context, commandID string, errMsg string) error
	ListByDevice(ctx context.Context, deviceID string, limit int) ([]domain.Command, error)
}
