package domain

import "time"

type Device struct {
	ID           int64     `json:"id"`
	DeviceID     string    `json:"device_id"`
	EmpresaID    int64     `json:"empresa_id"`
	PlantaID     int64     `json:"planta_id"`
	LineaID      int64     `json:"linea_id"`
	APIKey       string    `json:"-"`
	LastSeenAt   time.Time `json:"last_seen_at"`
	RegisteredAt time.Time `json:"registered_at"`
	Active       bool      `json:"active"`
}
