package domain

import (
	"encoding/json"
	"time"
)

type CommandStatus string

const (
	StatusPending   CommandStatus = "pending"
	StatusDelivered CommandStatus = "delivered"
	StatusAcked     CommandStatus = "acked"
	StatusFailed    CommandStatus = "failed"
)

type Command struct {
	ID          int64           `json:"id"`
	DeviceID    string          `json:"device_id"`
	EmpresaID   int64           `json:"empresa_id"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      CommandStatus   `json:"status"`
	IssuedBy    int64           `json:"issued_by"`
	CreatedAt   time.Time       `json:"created_at"`
	DeliveredAt *time.Time      `json:"delivered_at,omitempty"`
	AckedAt     *time.Time      `json:"acked_at,omitempty"`
}
