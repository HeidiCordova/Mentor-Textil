package ports

import (
	"context"

	"cloud-gateway/internal/domain"
)

type AuditRepository interface {
	Insert(ctx context.Context, e *domain.AuditEntry) error
}

type CommandRepository interface {
	Insert(ctx context.Context, cmd *domain.Command) (*domain.Command, error)
	ListPending(ctx context.Context, deviceID string) ([]domain.Command, error)
	Acknowledge(ctx context.Context, id int64, deviceID string) error
	MarkFailed(ctx context.Context, id int64, reason string) error
}

type DeviceRepository interface {
	FindByAPIKey(ctx context.Context, apiKey string) (*domain.Device, error)
	UpdateLastSeen(ctx context.Context, deviceID string) error
	ListByEmpresaID(ctx context.Context, empresaID int64) ([]domain.Device, error)
	PlantaBelongsToEmpresa(ctx context.Context, plantaID, empresaID int64) bool
	LineaBelongsToEmpresa(ctx context.Context, lineaID, empresaID int64) bool
}
