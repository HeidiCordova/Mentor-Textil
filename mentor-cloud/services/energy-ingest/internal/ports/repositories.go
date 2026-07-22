package ports

import (
	"context"
	"energy-ingest/internal/domain"
)

type EnergyRepository interface {
	UpsertSnapshot(ctx context.Context, s *domain.EnergySnapshot) error
	UpsertMeter(ctx context.Context, m *domain.Meter) error
	UpdateMeter(ctx context.Context, id int, m *domain.MeterUpdate) error
	DeleteMeter(ctx context.Context, id int) error
	LogSync(ctx context.Context, l *domain.DeviceSyncLog) error
	GetSnapshots(ctx context.Context, filter SnapshotFilter) ([]domain.EnergySnapshot, int, error)
	GetMeters(ctx context.Context, empresaID *int) ([]domain.Meter, error)
	GetMeterByID(ctx context.Context, id int) (*domain.Meter, error)
}

type ScopeResolver interface {
	ResolveByDevice(ctx context.Context, deviceID string) (*domain.DeviceScope, error)
	ResolveByAPIKey(ctx context.Context, apiKey string) (deviceID string, scope *domain.DeviceScope, err error)
	UpdateLastSeen(ctx context.Context, deviceID string)
}

type SnapshotFilter struct {
	DeviceID  string
	MeterID   string
	EmpresaID *int
	PlantaID  *int
	From      string
	To        string
	Limit     int
	Offset    int
}
