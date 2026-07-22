package domain

import "time"

type DashboardStats struct {
	TotalPlantas       int     `json:"totalPlantas"`
	TotalLineas        int     `json:"totalLineas"`
	TotalDispositivos  int     `json:"totalDispositivos"`
	AlarmasActivas     int     `json:"alarmasActivas"`
	EficienciaPromedio float64 `json:"eficienciaPromedio"`
	ProduccionHoy      int     `json:"produccionHoy"`
}

type DashboardReport struct {
	Fecha string  `json:"fecha"`
	Valor float64 `json:"valor"`
	Tipo  string  `json:"tipo"`
}

type DashboardCharts struct {
	Produccion []ChartPoint  `json:"produccion"`
	Energia    []EnergyPoint `json:"energia"`
}

type ChartPoint struct {
	Mes   string  `json:"mes"`
	Valor float64 `json:"valor"`
}

type EnergyPoint struct {
	Hora    string  `json:"hora"`
	Consumo float64 `json:"consumo"`
}

type AnalysisPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Status    string  `json:"status,omitempty"`
}

type EnergyAnalysis struct {
	Timestamp   string  `json:"timestamp"`
	Consumption float64 `json:"consumption"`
	Power       float64 `json:"power"`
}

type ProductionAnalysis struct {
	Timestamp  string  `json:"timestamp"`
	Produced   int     `json:"produced"`
	Target     int     `json:"target"`
	Efficiency float64 `json:"efficiency"`
}

type ParetoItem struct {
	Categoria  string  `json:"categoria"`
	Frecuencia int     `json:"frecuencia"`
	Porcentaje float64 `json:"porcentaje"`
}

type CombinedAnalysis struct {
	Timestamp   string  `json:"timestamp"`
	OEE         float64 `json:"oee"`
	OEEStatus   string  `json:"oee_status"`
	Consumption float64 `json:"consumption"`
	Power       float64 `json:"power"`
	Produced    int     `json:"produced"`
	Efficiency  float64 `json:"efficiency"`
}

type FullAnalysis struct {
	Snapshots []CombinedAnalysis `json:"snapshots"`
	Pareto    []ParetoItem       `json:"pareto"`
}

// CloudStop represents a stop record as exposed by the cloud API (read from analytics.paradas).
type CloudStop struct {
	ID                  int64      `json:"id"`
	StopID              *string    `json:"stop_id,omitempty"`
	DeviceID            string     `json:"device_id"`
	LineaID             *int       `json:"linea_id,omitempty"`
	PlantaID            *int       `json:"planta_id,omitempty"`
	EmpresaID           *int       `json:"empresa_id,omitempty"`
	CategoriaID         *int64     `json:"categoria_id,omitempty"`
	StopType            *string    `json:"stop_type,omitempty"`
	Inicio              time.Time  `json:"started_at"`
	Fin                 *time.Time `json:"ended_at,omitempty"`
	DuracionMin         *float64   `json:"duration_min,omitempty"`
	Justified           bool       `json:"justified"`
	Reason              *string    `json:"reason,omitempty"`
	Category            *string    `json:"category,omitempty"`
	JustifiedBy         *string    `json:"justified_by,omitempty"`
	JustifiedAt         *time.Time `json:"justified_at,omitempty"`
	Source              *string    `json:"source,omitempty"`
	CategoriaNombre     *string    `json:"categoria_nombre,omitempty"`
	SubcategoriaNombre  *string    `json:"subcategoria_nombre,omitempty"`
	Subcategoria2Nombre *string    `json:"subcategoria_2_nombre,omitempty"`
	CreadoEn            time.Time  `json:"created_at"`
}

// StopSummary mirrors the edge StopSummary for API compatibility.
type StopSummary struct {
	TotalStops       int            `json:"total_stops"`
	OpenStops        int            `json:"open_stops"`
	JustifiedStops   int            `json:"justified_stops"`
	UnjustifiedStops int            `json:"unjustified_stops"`
	TotalDowntimeMS  int64          `json:"total_downtime_ms"`
	ByType           map[string]int `json:"by_type"`
}

// CreateCloudStopRequest is the request body for creating a stop from cloud.
type CreateCloudStopRequest struct {
	DeviceID    string  `json:"device_id,omitempty"`
	StopType    string  `json:"stop_type" binding:"required"`
	Category    *string `json:"category,omitempty"`
	CategoriaID *int64  `json:"categoria_id,omitempty"`
	Reason      *string `json:"reason,omitempty"`
	Source      string  `json:"source,omitempty"`
	EmpresaID   int     `json:"empresa_id,omitempty"`
	PlantaID    *int    `json:"planta_id,omitempty"`
	LineaID     *int    `json:"linea_id,omitempty"`
}

// JustifyCloudStopRequest is the request body for justifying a stop from cloud.
type JustifyCloudStopRequest struct {
	Reason      string `json:"reason" binding:"required"`
	Category    string `json:"category,omitempty"`
	StopType    string `json:"stop_type,omitempty"`
	CategoriaID *int64 `json:"categoria_id,omitempty"`
	JustifiedBy string `json:"justified_by" binding:"required"`
}

// UpdateCloudStopRequest is the request body for modifying arbitrary fields of a stop from cloud.
// All fields are optional; only non-nil values are applied.
type UpdateCloudStopRequest struct {
	StopType    *string    `json:"stop_type,omitempty"`
	Category    *string    `json:"category,omitempty"`
	CategoriaID *int64     `json:"categoria_id,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
}

type Alarma struct {
	ID              int64      `json:"id"`
	DeviceID        string     `json:"device_id"`
	LineaID         *int       `json:"linea_id,omitempty"`
	PlantaID        *int       `json:"planta_id,omitempty"`
	EmpresaID       *int       `json:"empresa_id,omitempty"`
	Tipo            string     `json:"tipo"`
	Mensaje         string     `json:"mensaje"`
	Severidad       string     `json:"severidad"`
	Activa          bool       `json:"activa"`
	TimestampEvento time.Time  `json:"timestamp_evento"`
	ReconocidoEn    *time.Time `json:"reconocido_en,omitempty"`
	CreadoEn        time.Time  `json:"creado_en"`
}

type Compromiso struct {
	ID            int    `json:"id"`
	EmpresaID     *int   `json:"empresa_id,omitempty"`
	PlantaID      *int   `json:"planta_id,omitempty"`
	Descripcion   string `json:"descripcion"`
	Responsable   string `json:"responsable,omitempty"`
	FechaLimite   string `json:"fecha_limite,omitempty"`
	Estado        string `json:"estado"`
	CreadoEn      string `json:"creado_en"`
	ActualizadoEn string `json:"actualizado_en"`
}

type ProductionRun struct {
	ID         int64      `json:"id"`
	RunID      string     `json:"run_id"`
	DeviceID   string     `json:"device_id"`
	LineaID    *int       `json:"linea_id"`
	PlantaID   *int       `json:"planta_id,omitempty"`
	EmpresaID  *int       `json:"empresa_id,omitempty"`
	ProductoID *int       `json:"producto_id"`
	SKU        *string    `json:"sku"`
	Nombre     *string    `json:"nombre"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	Synced     bool       `json:"synced"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type UpsertProductionRunRequest struct {
	RunID      string     `json:"run_id"`
	DeviceID   string     `json:"device_id"`
	LineaID    *int       `json:"linea_id,omitempty"`
	PlantaID   *int       `json:"planta_id,omitempty"`
	EmpresaID  *int       `json:"empresa_id,omitempty"`
	ProductoID *int       `json:"producto_id,omitempty"`
	SKU        *string    `json:"sku,omitempty"`
	Nombre     *string    `json:"nombre,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}
