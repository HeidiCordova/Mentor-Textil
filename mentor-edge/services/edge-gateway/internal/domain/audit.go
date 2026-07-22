package domain

import "time"

type AuditEntry struct {
	ID         int                    `json:"id"`
	DeviceID   string                 `json:"device_id"`
	Actor      string                 `json:"actor"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource"`
	ResourceID *string                `json:"resource_id,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	Result     string                 `json:"result"`
	Timestamp  time.Time              `json:"timestamp"`
}
