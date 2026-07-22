package ports

import (
	"cloud-integration/internal/domain"
	"context"
)

type APIKeyRepository interface {
	Create(ctx context.Context, req domain.CreateKeyRequest, keyHash, keyPrefix string) (*domain.APIKey, error)
	List(ctx context.Context, empresaID int) ([]domain.APIKey, error)
	Revoke(ctx context.Context, id int, empresaID int) error
	Validate(ctx context.Context, raw string) (*domain.APIKey, error)
}

type QueryRepository interface {
	LatestSnapshot(ctx context.Context, lineaID int, empresaID int) (*domain.OEESnapshot, error)
	ListSnapshots(ctx context.Context, lineaID int, empresaID int, limit int) ([]domain.OEESnapshot, error)
	ListParadas(ctx context.Context, lineaID int, empresaID int, desde, hasta string, limit int) ([]domain.Parada, error)
	// ListEnergy devuelve snapshots de energía paginados.
	// deviceID y meterID son filtros opcionales (string vacío = sin filtro).
	// desde/hasta son timestamps ISO 8601 opcionales.
	ListEnergy(ctx context.Context, empresaID int, deviceID, meterID, desde, hasta string, limit, offset int) ([]domain.EnergySnapshot, int, error)
}
