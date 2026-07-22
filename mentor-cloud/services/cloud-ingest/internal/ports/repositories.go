package ports

import (
	"cloud-ingest/internal/domain"
	"context"
)

type EventRepository interface {
	InsertRawEvent(ctx context.Context, e *domain.RawEvent) error
	InsertOEESnapshot(ctx context.Context, s *domain.OEESnapshot) error
	UpdateVariablesFromHeadData(ctx context.Context, deviceID string, head []string, data []string) error
	GetVariablesForEmpresa(ctx context.Context, empresaID int) ([]domain.Variable, error)
	UpsertParada(ctx context.Context, stop *domain.StopRecord, scope *domain.DeviceScope) error
	UpsertProductionRun(ctx context.Context, run *domain.ProductionRunRecord, scope *domain.DeviceScope) error
	LogSync(ctx context.Context, log *domain.DeviceSyncLog) error
	GetRawEvents(ctx context.Context, filter RawEventsFilter) ([]domain.RawEvent, int, error)
	// Pending commands: entrega confiable de comandos cloud → edge
	GetPendingCommands(ctx context.Context, deviceID string) ([]domain.PendingCommand, error)
	AckPendingCommands(ctx context.Context, deviceID string, ids []int64) error
	InsertPendingCommand(ctx context.Context, deviceID string, command string, payload []byte) error
}

type OEEQueryRepository interface {
	GetSnapshots(ctx context.Context, filter OEEFilter) ([]domain.OEESnapshot, error)
	GetLatestByDevice(ctx context.Context, deviceID string) (*domain.OEESnapshot, error)
}

type RawEventsFilter struct {
	DeviceID  string
	EmpresaID *int
	PlantaID  *int
	LineaID   *int
	EventType string
	From      string // ISO string
	To        string // ISO string
	Limit     int
	Offset    int
}

type OEEFilter struct {
	DeviceID  string
	PlantaID  *int
	EmpresaID *int
	FechaFrom string
	FechaTo   string
	Limit     int
	Offset    int
}

// ScopeResolver resuelve el scope organizacional completo.
// Prioriza linea_id (resuelve planta/empresa via config.lineas -> config.plantas).
// Fallback: device_id (resuelve via gateway.device_registry).
type ScopeResolver interface {
	ResolveByLinea(ctx context.Context, lineaID int) (*domain.DeviceScope, error)
	ResolveByDevice(ctx context.Context, deviceID string) (*domain.DeviceScope, error)
	UpdateLastSeen(ctx context.Context, deviceID string)
}
