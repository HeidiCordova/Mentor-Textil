package domain

import "time"

type ProductionRun struct {
	ID         int        `json:"id"`
	RunID      string     `json:"run_id"`
	DeviceID   string     `json:"device_id"`
	LineaID    *int       `json:"linea_id,omitempty"`
	ProductoID *int       `json:"producto_id,omitempty"`
	SKU        *string    `json:"sku,omitempty"`
	Nombre     *string    `json:"nombre,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	Synced     bool       `json:"synced"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type UpsertProductionRunRequest struct {
	RunID      string     `json:"run_id,omitempty"`
	DeviceID   string     `json:"device_id,omitempty"`
	LineaID    *int       `json:"linea_id,omitempty"`
	ProductoID *int       `json:"producto_id,omitempty"`
	SKU        *string    `json:"sku,omitempty"`
	Nombre     *string    `json:"nombre,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
}

type ProductionRunFilter struct {
	DeviceID string
	LineaID  *int
	Since    *time.Time
	Until    *time.Time
	Limit    int
}
