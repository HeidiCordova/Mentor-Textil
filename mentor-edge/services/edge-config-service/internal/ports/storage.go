package ports

import (
	"context"
	"edge-config-service/internal/domain"
)

type ConfigStorage interface {
	// GetConfig retrieves config for a line (linea_id>0) or system defaults (linea_id=0).
	GetConfig(ctx context.Context, lineaID int) (*domain.LineConfig, error)
	UpdateConfig(ctx context.Context, config *domain.LineConfig) error
	GetLineIDs(ctx context.Context) ([]int, error)
	GetConfigVersion(ctx context.Context, lineaID int) (int, error)
	DeleteLine(ctx context.Context, lineaID int) error
	GetDeviceID(ctx context.Context) (string, error)
	SetDeviceID(ctx context.Context, deviceID string) error
}
