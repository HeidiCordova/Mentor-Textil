package ports

import (
	"context"
	"time"

	"edge-gateway/internal/domain"
)

type ProductionRunRepository interface {
	Upsert(ctx context.Context, req domain.UpsertProductionRunRequest) ([]domain.ProductionRun, error)
	List(ctx context.Context, filter domain.ProductionRunFilter) ([]domain.ProductionRun, error)
	FindActive(ctx context.Context, deviceID string, lineaID int, at time.Time) (*domain.ProductionRun, error)
	Delete(ctx context.Context, runID string) ([]domain.ProductionRun, error)
	ListUnsynced(ctx context.Context, limit int) ([]domain.ProductionRun, error)
	MarkSynced(ctx context.Context, runIDs []string) error
	// SinProgramacionSeconds returns the number of seconds in [from, to) that are
	// covered by production_runs with sku IS NULL ("sin programación") for the device.
	SinProgramacionSeconds(ctx context.Context, deviceID string, from, to time.Time) (int64, error)
}
