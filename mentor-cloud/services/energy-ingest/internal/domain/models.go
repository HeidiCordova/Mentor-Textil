package domain

import "time"

type EnergyRecord struct {
	DeviceID  string   `json:"device_id,omitempty"`
	MeterID   string   `json:"meter_id"`
	Timestamp int64    `json:"timestamp"`
	IntervalS int      `json:"interval_s"`
	Head      []string `json:"head"`
	Data      []string `json:"data"`
	Ubicacion string   `json:"ubicacion,omitempty"`
}

type EnergySnapshot struct {
	ID               int       `json:"id"`
	DeviceID         string    `json:"device_id"`
	MeterID          string    `json:"meter_id"`
	Hora             time.Time `json:"hora"`
	IntervalS        int       `json:"interval_s"`
	Head             []string  `json:"head,omitempty"`
	Data             []string  `json:"data,omitempty"`
	EmpresaID        *int      `json:"empresa_id,omitempty"`
	PlantaID         *int      `json:"planta_id,omitempty"`
	CorrienteA       float64   `json:"corriente_a"`
	CorrienteB       float64   `json:"corriente_b"`
	CorrienteC       float64   `json:"corriente_c"`
	CorrienteAVG     float64   `json:"corriente_avg"`
	VoltajeA         float64   `json:"voltaje_a"`
	VoltajeB         float64   `json:"voltaje_b"`
	VoltajeC         float64   `json:"voltaje_c"`
	VoltajeAVG       float64   `json:"voltaje_avg"`
	VoltajeAB        float64   `json:"voltaje_ab"`
	VoltajeBC        float64   `json:"voltaje_bc"`
	VoltajeAC        float64   `json:"voltaje_ac"`
	PotenciaActiva   float64   `json:"potencia_activa"`
	PotenciaReactiva float64   `json:"potencia_reactiva"`
	PotenciaAparente float64   `json:"potencia_aparente"`
	FactorPotencia   float64   `json:"factor_potencia"`
	FrecuenciaHz     float64   `json:"frecuencia_hz"`
	EnergiaActiva    float64   `json:"energia_activa"`
	EnergiaReactiva  float64   `json:"energia_reactiva"`
	EnergiaAparente  float64   `json:"energia_aparente"`
	THDIA            float64   `json:"thd_ia"`
	THDIB            float64   `json:"thd_ib"`
	THDIC            float64   `json:"thd_ic"`
	THDUA            float64   `json:"thd_ua"`
	THDUB            float64   `json:"thd_ub"`
	THDUC            float64   `json:"thd_uc"`
	CreatedAt        time.Time `json:"created_at"`
	// Consumo por intervalo: delta respecto al snapshot anterior del mismo medidor.
	// NULL cuando no existe fila anterior o cuando el contador fue reiniciado.
	ConsumoActiva   *float64 `json:"consumo_activa"`
	ConsumoReactiva *float64 `json:"consumo_reactiva"`
	ConsumoAparente *float64 `json:"consumo_aparente"`
}

type Meter struct {
	ID        int        `json:"id"`
	DeviceID  string     `json:"device_id"`
	MeterID   string     `json:"meter_id"`
	EmpresaID *int       `json:"empresa_id,omitempty"`
	PlantaID  *int       `json:"planta_id,omitempty"`
	Nombre    *string    `json:"nombre,omitempty"`
	Ubicacion *string    `json:"ubicacion,omitempty"`
	Activo    bool       `json:"activo"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type MeterUpdate struct {
	Nombre    *string `json:"nombre"`
	Ubicacion *string `json:"ubicacion"`
	Activo    *bool   `json:"activo"`
}

type MeterCreate struct {
	DeviceID  string `json:"device_id"`
	MeterID   string `json:"meter_id"`
	Nombre    string `json:"nombre"`
	Ubicacion string `json:"ubicacion"`
	EmpresaID *int   `json:"empresa_id"`
	PlantaID  *int   `json:"planta_id"`
}

type DeviceScope struct {
	EmpresaID *int
	PlantaID  *int
	LineaID   *int
}

type DeviceSyncLog struct {
	DeviceID  string `json:"device_id"`
	BatchSize int    `json:"batch_size"`
	Status    string `json:"status"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}
