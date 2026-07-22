package domain

import "time"

type APIKey struct {
	ID        int        `json:"id"`
	EmpresaID int        `json:"empresa_id"`
	Nombre    string     `json:"nombre"`
	KeyPrefix string     `json:"key_prefix"`
	Scopes    []string   `json:"scopes"`
	Activo    bool       `json:"activo"`
	CreadoEn  time.Time  `json:"creado_en"`
	UltimoUso *time.Time `json:"ultimo_uso,omitempty"`
}

type CreateKeyRequest struct {
	EmpresaID int    `json:"empresa_id" binding:"required"`
	Nombre    string `json:"nombre" binding:"required,max=100"`
}

type CreateKeyResponse struct {
	APIKey
	Plaintext string `json:"key"`
}

type OEESnapshot struct {
	ID             int64     `json:"id"`
	DeviceID       string    `json:"device_id"`
	LineaID        *int      `json:"linea_id,omitempty"`
	PlantaID       *int      `json:"planta_id,omitempty"`
	EmpresaID      *int      `json:"empresa_id,omitempty"`
	Turno          string    `json:"turno,omitempty"`
	RunNombre      string    `json:"run_nombre,omitempty"`
	RunSKU         string    `json:"run_sku,omitempty"`
	Fecha          string    `json:"fecha"`
	Hora           time.Time `json:"hora"`
	Disponibilidad float64   `json:"disponibilidad"`
	Rendimiento    float64   `json:"rendimiento"`
	Calidad        float64   `json:"calidad"`
	OEE            float64   `json:"oee"`
	Produccion     int       `json:"produccion"`
	IntervalS      int       `json:"interval_s"`
}

type Parada struct {
	ID              int64      `json:"id"`
	DeviceID        string     `json:"device_id"`
	LineaID         *int       `json:"linea_id,omitempty"`
	PlantaID        *int       `json:"planta_id,omitempty"`
	EmpresaID       *int       `json:"empresa_id,omitempty"`
	StopType        string     `json:"stop_type"`
	Inicio          time.Time  `json:"inicio"`
	Fin             *time.Time `json:"fin,omitempty"`
	DuracionMin     *float64   `json:"duracion_min,omitempty"`
	Justified       bool       `json:"justified"`
	Reason          *string    `json:"reason,omitempty"`
	CategoriaNombre *string    `json:"categoria_nombre,omitempty"`
	Source          string     `json:"source"`
}

// EnergySnapshot representa una lectura del medidor MC60.
// Campos nil significan que el medidor no retornó ese valor.
type EnergySnapshot struct {
	ID               int64    `json:"id"`
	DeviceID         string   `json:"device_id"`
	MeterID          string   `json:"meter_id"`
	Hora             time.Time `json:"hora"`
	IntervalS        int      `json:"interval_s"`
	EmpresaID        *int     `json:"empresa_id,omitempty"`
	PlantaID         *int     `json:"planta_id,omitempty"`
	// Corrientes [A]
	CorrienteA       *float64 `json:"corriente_a,omitempty"`
	CorrienteB       *float64 `json:"corriente_b,omitempty"`
	CorrienteC       *float64 `json:"corriente_c,omitempty"`
	CorrienteAvg     *float64 `json:"corriente_avg,omitempty"`
	// Voltajes fase-neutro [V]
	VoltajeA         *float64 `json:"voltaje_a,omitempty"`
	VoltajeB         *float64 `json:"voltaje_b,omitempty"`
	VoltajeC         *float64 `json:"voltaje_c,omitempty"`
	VoltajeAvg       *float64 `json:"voltaje_avg,omitempty"`
	// Voltajes línea-línea [V]
	VoltajeAB        *float64 `json:"voltaje_ab,omitempty"`
	VoltajeBC        *float64 `json:"voltaje_bc,omitempty"`
	VoltajeAC        *float64 `json:"voltaje_ac,omitempty"`
	// Potencia instantánea
	PotenciaActiva   *float64 `json:"potencia_activa,omitempty"`   // [kW]
	PotenciaReactiva *float64 `json:"potencia_reactiva,omitempty"` // [kVAR]
	PotenciaAparente *float64 `json:"potencia_aparente,omitempty"` // [kVA]
	FactorPotencia   *float64 `json:"factor_potencia,omitempty"`   // 0–1
	FrecuenciaHz     *float64 `json:"frecuencia_hz,omitempty"`     // [Hz]
	// Energía acumulada
	EnergiaActiva    *float64 `json:"energia_activa,omitempty"`    // [kWh]
	EnergiaReactiva  *float64 `json:"energia_reactiva,omitempty"`  // [kVARh]
	EnergiaAparente  *float64 `json:"energia_aparente,omitempty"`  // [kVAh]
	// Distorsión armónica total (THD)
	THDIA            *float64 `json:"thd_ia,omitempty"` // [%]
	THDIB            *float64 `json:"thd_ib,omitempty"`
	THDIC            *float64 `json:"thd_ic,omitempty"`
	THDUA            *float64 `json:"thd_ua,omitempty"`
	THDUB            *float64 `json:"thd_ub,omitempty"`
	THDUC            *float64 `json:"thd_uc,omitempty"`
}

// EnergyPage envuelve la lista paginada para Power BI y sistemas BI similares.
type EnergyPage struct {
	Data   []EnergySnapshot `json:"data"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}
