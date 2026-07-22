package domain

import "time"

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
	EnergiaKwh     float64   `json:"energia_kwh"`
	IntervalS      int       `json:"interval_s"`
	Head           []string  `json:"head,omitempty"`
	Data           []string  `json:"data,omitempty"`
}

type OEESummary struct {
	Disponibilidad  float64 `json:"disponibilidad"`
	Rendimiento     float64 `json:"rendimiento"`
	Calidad         float64 `json:"calidad"`
	OEE             float64 `json:"oee"`
	ProduccionTotal int     `json:"produccion_total"`
	Snapshots       int     `json:"snapshots"`
}
