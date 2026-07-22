package ports

import (
	"context"
	"encoding/json"
	"time"
)

type CloudClient interface {
	SendOEE(ctx context.Context, records []OEERecord) error
	SendStops(ctx context.Context, stops []StopRecord) error
	SendProductionRuns(ctx context.Context, runs []ProductionRunRecord) error
	SendEnergy(ctx context.Context, records []EnergyRecord) error
	// Heartbeat envía latido al cloud y recibe la config canónica del dispositivo.
	Heartbeat(ctx context.Context) (*HeartbeatInfo, error)
	// UpdateCredentials permite cambiar la URL y API key en caliente (sin reiniciar).
	UpdateCredentials(cloudURL, apiKey string)
	// Pending commands: entrega confiable cloud → edge
	GetPendingCommands(ctx context.Context) ([]PendingCommand, error)
	AckPendingCommands(ctx context.Context, ids []int64) error
	// SyncMode sincroniza el modo de operación de la línea al cloud.
	SyncMode(ctx context.Context, lineaID int, mode string) error
}

// HeartbeatInfo contiene la config canónica que el cloud devuelve al edge en cada heartbeat.
type HeartbeatInfo struct {
	DeviceID  string `json:"device_id"`
	EmpresaID int    `json:"empresa_id"`
	PlantaID  int    `json:"planta_id"`
	LineaID   int    `json:"linea_id"`
}

// PendingCommand representa un comando encolado en cloud para aplicar en edge.
type PendingCommand struct {
	ID        int64           `json:"id"`
	DeviceID  string          `json:"device_id"`
	Command   string          `json:"command"`
	Payload   json.RawMessage `json:"payload"`
	Applied   bool            `json:"applied"`
	CreatedAt time.Time       `json:"created_at"`
}

type OEERecord struct {
	Code      string   `json:"code"`
	Time      int64    `json:"time"`
	DeviceID  string   `json:"device_id"`
	IntervalS int      `json:"interval_s"`
	Turno     string   `json:"turno,omitempty"`
	V         int      `json:"v"`
	Head      []string `json:"head"`
	Data      []string `json:"data"`
}

type StopRecord struct {
	StopID      string  `json:"stop_id"`
	DeviceID    string  `json:"device_id"`
	StopType    string  `json:"stop_type"`
	StartedAt   string  `json:"started_at"`
	EndedAt     *string `json:"ended_at,omitempty"`
	DurationMS  *int64  `json:"duration_ms,omitempty"`
	Justified   bool    `json:"justified"`
	Reason      *string `json:"reason,omitempty"`
	Category    *string `json:"category,omitempty"`
	CategoriaID *int64  `json:"categoria_id,omitempty"`
	JustifiedBy *string `json:"justified_by,omitempty"`
	JustifiedAt *string `json:"justified_at,omitempty"`
	Source      string  `json:"source"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type EnergyRecord struct {
	MeterID   string   `json:"meter_id"`
	Timestamp int64    `json:"timestamp"`
	IntervalS int      `json:"interval_s"`
	Head      []string `json:"head"`
	Data      []string `json:"data"`
}
