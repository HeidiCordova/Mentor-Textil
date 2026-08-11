package ports

import (
	"cloud-analytics/internal/domain"
	"context"
	"time"
)

type DashboardRepository interface {
	GetStats(ctx context.Context, empresaID *int) (*domain.DashboardStats, error)
	GetReports(ctx context.Context, empresaID *int, from, to string) ([]domain.DashboardReport, error)
	GetCharts(ctx context.Context, empresaID *int) (*domain.DashboardCharts, error)
}

type AnalysisRepository interface {
	GetGeneral(ctx context.Context, filter AnalysisFilter) ([]domain.AnalysisPoint, error)
	GetEnergy(ctx context.Context, filter AnalysisFilter) ([]domain.EnergyAnalysis, error)
	GetProduction(ctx context.Context, filter AnalysisFilter) ([]domain.ProductionAnalysis, error)
	GetPareto(ctx context.Context, filter AnalysisFilter) ([]domain.ParetoItem, error)
	GetCombined(ctx context.Context, filter AnalysisFilter) (*domain.FullAnalysis, error)
}

type AlarmaRepository interface {
	FindAll(ctx context.Context, empresaID *int, activa *bool) ([]domain.Alarma, error)
	Create(ctx context.Context, a *domain.Alarma) error
	Acknowledge(ctx context.Context, id int64) error
}

type CompromisoRepository interface {
	FindAll(ctx context.Context, empresaID *int) ([]domain.Compromiso, error)
	Create(ctx context.Context, c *domain.Compromiso) error
	Update(ctx context.Context, c *domain.Compromiso) error
	Delete(ctx context.Context, id int) error
}

type AnalysisFilter struct {
	DeviceID     string
	PlantaID     *int
	EmpresaID    *int
	LineaID      *int
	From         string
	To           string
	Limit        int
	MinIntervalS int // 0 usa el intervalo textil canónico (1800 s)
}

type IngestClient interface {
	GetOEESnapshots(ctx context.Context, filter AnalysisFilter) ([]OEEData, error)
}

type StopRepository interface {
	ListStops(ctx context.Context, filter StopFilter) ([]domain.CloudStop, error)
	StopSummary(ctx context.Context, filter StopFilter) (*domain.StopSummary, error)
	CreateStop(ctx context.Context, stop *domain.CreateCloudStopRequest) (*domain.CloudStop, error)
	UpdateStop(ctx context.Context, stopID string, req *domain.UpdateCloudStopRequest) (*domain.CloudStop, error)
	JustifyStop(ctx context.Context, stopID string, req *domain.JustifyCloudStopRequest) (*domain.CloudStop, error)
	CloseStop(ctx context.Context, stopID string, endedAt time.Time) (*domain.CloudStop, error)
	GetStop(ctx context.Context, stopID string) (*domain.CloudStop, error)
	DeleteStop(ctx context.Context, stopID string) error
	ResolveDeviceID(ctx context.Context, lineaID int) (string, error)
}

// CommandDispatcher sends commands to edge devices via cloud-gateway.
type CommandDispatcher interface {
	Dispatch(ctx context.Context, deviceID string, cmdType string, payload map[string]interface{}, userID int64, empresaID int64) error
}

// BrowserNotifier sends real-time events to connected browser clients via cloud-gateway SSE.
type BrowserNotifier interface {
	Notify(ctx context.Context, eventType string, empresaID int64, lineaID int64) error
}

type StopFilter struct {
	DeviceID  string
	EmpresaID *int
	PlantaID  *int
	LineaID   *int
	Justified *bool
	Open      *bool
	StopType  *string
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

type OEEData struct {
	DeviceID       string  `json:"device_id"`
	Fecha          string  `json:"fecha"`
	Hora           string  `json:"hora"`
	Disponibilidad float64 `json:"disponibilidad"`
	Rendimiento    float64 `json:"rendimiento"`
	Calidad        float64 `json:"calidad"`
	OEE            float64 `json:"oee"`
	Produccion     int     `json:"produccion"`
	EnergiaKwh     float64 `json:"energia_kwh"`
}

type OEEFilter struct {
	DeviceID     string
	EmpresaID    *int
	PlantaID     *int
	LineaID      *int
	From         string
	To           string
	Limit        int
	MinIntervalS int // 0 usa el intervalo textil canónico (1800 s)
}

type OEERepository interface {
	GetSnapshots(ctx context.Context, f OEEFilter) ([]domain.OEESnapshot, error)
	GetSummary(ctx context.Context, f OEEFilter) (*domain.OEESummary, error)
	GetLatest(ctx context.Context, lineaID int) (*domain.OEESnapshot, error)
}

type ProductionRunFilter struct {
	DeviceID  string
	EmpresaID *int
	PlantaID  *int
	LineaID   *int
	Since     *time.Time
	Until     *time.Time
	Limit     int
}

type ProductionRunRepository interface {
	List(ctx context.Context, f ProductionRunFilter) ([]domain.ProductionRun, error)
	Upsert(ctx context.Context, req *domain.UpsertProductionRunRequest) (*domain.ProductionRun, error)
	Delete(ctx context.Context, runID string) error
}

type OEELabRepository interface {
	List(ctx context.Context) ([]domain.OEELabSession, error)
	Get(ctx context.Context, id int64) (*domain.OEELabSession, error)
	Create(ctx context.Context, req *domain.OEELabSessionCreate) (*domain.OEELabSession, error)
	Delete(ctx context.Context, id int64) error
}
