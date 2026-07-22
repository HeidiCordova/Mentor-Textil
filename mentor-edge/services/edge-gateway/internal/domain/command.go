package domain

import (
	"errors"
	"time"
)

type Command struct {
	ID             int                    `json:"id"`
	CommandID      string                 `json:"command_id"`
	DeviceID       string                 `json:"device_id"`
	CommandType    string                 `json:"command_type"`
	Payload        map[string]interface{} `json:"payload"`
	IssuedBy       string                 `json:"issued_by"`
	IssuedAt       time.Time              `json:"issued_at"`
	IdempotencyKey string                 `json:"idempotency_key"`
	Status         string                 `json:"status"`
	Result         map[string]interface{} `json:"result,omitempty"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
	AppliedAt      *time.Time             `json:"applied_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

type CreateCommandRequest struct {
	CommandID      string                 `json:"command_id"`
	DeviceID       string                 `json:"device_id"`
	CommandType    string                 `json:"command_type"`
	Payload        map[string]interface{} `json:"payload"`
	IssuedBy       string                 `json:"issued_by"`
	IdempotencyKey string                 `json:"idempotency_key"`
}

var validCommandTypes = map[string]bool{
	"MODIFICAR_PARADA":              true,
	"JUSTIFICAR_PARADA":             true,
	"CREAR_PARADA":                  true,
	"CERRAR_PARADA":                 true,
	"ELIMINAR_PARADA":               true,
	"ACTUALIZAR_CONFIG":             true,
	"INICIAR_CALIBRACION":           true,
	"REINICIAR_PIPELINE":            true,
	"COMANDO_CUSTOM":                true,
	"SYNC_CATALOG":                  true,
	"SYNC_PRODUCTOS":                true,
	"SYNC_TURNOS":                   true,
	"SYNC_USUARIOS":                 true,
	"SYNC_VARIABLES":                true,
	"SYNC_LINEA_PRODUCTO_VARS":      true,
	"SYNC_PRODUCTO_CARACTERISTICAS": true,
	"SYNC_PLANTAS_LINEAS":           true,
	"SYNC_PARADAS":                  true,
	"SYNC_VELOCIDAD_NOMINAL":        true,
	"SYNC_MOTIVOS_VELOCIDAD":        true,
}

var validCommandStatuses = map[string]bool{
	"RECEIVED": true,
	"APPLIED":  true,
	"FAILED":   true,
}

func (r *CreateCommandRequest) Validate() error {
	if r.DeviceID == "" {
		return ErrMissingDeviceID
	}
	if r.CommandType == "" {
		return errors.New("command_type is required")
	}
	if !validCommandTypes[r.CommandType] {
		return errors.New("invalid command_type")
	}
	if r.IdempotencyKey == "" {
		return errors.New("idempotency_key is required")
	}
	if r.IssuedBy == "" {
		return errors.New("issued_by is required")
	}
	return nil
}
