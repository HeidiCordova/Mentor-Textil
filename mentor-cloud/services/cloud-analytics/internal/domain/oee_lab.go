package domain

import "time"

type OEELabSession struct {
	ID        int64                  `json:"id"`
	Nombre    string                 `json:"nombre"`
	LineaID   *int                   `json:"linea_id,omitempty"`
	Inputs    map[string]interface{} `json:"inputs"`
	Results   map[string]interface{} `json:"results"`
	Notas     string                 `json:"notas,omitempty"`
	CreatedBy string                 `json:"created_by,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type OEELabSessionCreate struct {
	Nombre    string                 `json:"nombre"`
	LineaID   *int                   `json:"linea_id,omitempty"`
	Inputs    map[string]interface{} `json:"inputs"`
	Results   map[string]interface{} `json:"results"`
	Notas     string                 `json:"notas,omitempty"`
	CreatedBy string                 `json:"created_by,omitempty"`
}
