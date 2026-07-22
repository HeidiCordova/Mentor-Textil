package domain

import (
	"errors"
	"time"
)

// Architecture: Edge Gateway (Single Entry Point)
// Cloud never calls internal services directly.
// Tablet never calls internal services directly (in hybrid mode).
// Commands are idempotent and audited.
// SSE is used for real-time UI updates (incremental rendering on tablet).

type Stop struct {
	ID          int        `json:"id"`
	StopID      string     `json:"stop_id"`
	DeviceID    string     `json:"device_id"`
	StopType    string     `json:"stop_type"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	DurationMs  *int       `json:"duration_ms,omitempty"`
	Justified   bool       `json:"justified"`
	Reason      *string    `json:"reason,omitempty"`
	Category    *string    `json:"category,omitempty"`
	CategoriaID *int       `json:"categoria_id,omitempty"`
	JustifiedBy *string    `json:"justified_by,omitempty"`
	JustifiedAt *time.Time `json:"justified_at,omitempty"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Synced      bool       `json:"synced"`
	SyncedAt    *time.Time `json:"synced_at,omitempty"`
}

type StopSummary struct {
	TotalStops       int64            `json:"total_stops"`
	OpenStops        int64            `json:"open_stops"`
	JustifiedStops   int64            `json:"justified_stops"`
	UnjustifiedStops int64            `json:"unjustified_stops"`
	TotalDowntimeMs  int64            `json:"total_downtime_ms"`
	ByType           map[string]int64 `json:"by_type"`
}

type CreateStopRequest struct {
	StopID      string     `json:"stop_id,omitempty"` // optional: if provided, use this UUID instead of DB default
	DeviceID    string     `json:"device_id"`
	StopType    string     `json:"stop_type"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Reason      *string    `json:"reason,omitempty"`
	Category    *string    `json:"category,omitempty"`
	CategoriaID *int       `json:"categoria_id,omitempty"`
	Source      string     `json:"source"`
}

type JustifyStopRequest struct {
	StopID      string `json:"stop_id"`
	StopType    string `json:"stop_type"`
	Reason      string `json:"reason"`
	Category    string `json:"category"`
	CategoriaID *int   `json:"categoria_id,omitempty"`
	JustifiedBy string `json:"justified_by"`
}

type StopFilter struct {
	DeviceID  string
	Justified *bool
	Open      *bool
	StopType  *string
	Since     *time.Time
	Until     *time.Time
	Limit     int
	Offset    int
}

var validStopTypes = map[string]bool{
	"MICROPARADA":        true,
	"PARADA_NO_ASIGNADA": true,
	"PROGRAMADA":         true,
	"NO_PROGRAMADA":      true,
	"MECANICA":           true,
	"ELECTRICA":          true,
	"CAMBIO_FORMATO":     true,
	"FALTA_MATERIAL":     true,
	"CALIDAD":            true,
	"REFRIGERIO":         true,
	"CAPACITACION":       true,
	"MANTENIMIENTO":      true,
	"OTRA":               true,
}

var validSources = map[string]bool{
	"detector": true,
	"operator": true,
	"cloud":    true,
	"system":   true,
}

func (r *CreateStopRequest) Validate() error {
	if r.DeviceID == "" {
		return ErrMissingDeviceID
	}
	if !validStopTypes[r.StopType] {
		return ErrInvalidStopType
	}
	if r.StartedAt.IsZero() {
		return ErrMissingTimestamp
	}
	if r.Source == "" {
		r.Source = "operator"
	}
	if !validSources[r.Source] {
		return errors.New("invalid source: must be detector, operator, cloud or system")
	}
	if r.EndedAt != nil && r.EndedAt.Before(r.StartedAt) {
		return errors.New("ended_at must be after started_at")
	}
	return nil
}

func (r *JustifyStopRequest) Validate() error {
	if r.StopID == "" {
		return errors.New("stop_id is required")
	}
	if r.Reason == "" {
		return errors.New("reason is required")
	}
	if r.JustifiedBy == "" {
		return errors.New("justified_by is required")
	}
	if r.StopType != "" && !validStopTypes[r.StopType] {
		return ErrInvalidStopType
	}
	return nil
}
