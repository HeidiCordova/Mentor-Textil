package domain

import "time"

type AuditEntry struct {
	Method    string
	Path      string
	Status    int
	LatencyMs int64
	IP        string
	UserID    int64
	EmpresaID int64
	DeviceID  string
	Timestamp time.Time
}
