package application

import (
	"context"
	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
	"time"
)

type StatusService struct {
	buffer    ports.BufferClient
	config    ports.ConfigClient
	enviador  ports.EnviadorClient
	detector  ports.DetectorClient
	audit     ports.AuditRepository
	deviceID  string
	startedAt time.Time
}

func NewStatusService(
	buffer ports.BufferClient,
	config ports.ConfigClient,
	enviador ports.EnviadorClient,
	detector ports.DetectorClient,
	audit ports.AuditRepository,
	deviceID string,
) *StatusService {
	return &StatusService{
		buffer:    buffer,
		config:    config,
		enviador:  enviador,
		detector:  detector,
		audit:     audit,
		deviceID:  deviceID,
		startedAt: time.Now(),
	}
}

func (s *StatusService) GetStatus(ctx context.Context) (*domain.EdgeStatus, error) {
	status := &domain.EdgeStatus{
		DeviceID: s.deviceID,
		Uptime:   int64(time.Since(s.startedAt).Seconds()),
	}

	summary, err := s.buffer.GetSummary(ctx)
	if err == nil && summary != nil {
		status.BufferPending = summary.PendingCount
	}

	cloudStatus, err := s.enviador.Health(ctx)
	if err == nil {
		status.CloudConnected = (cloudStatus == "ok")
	}

	version, err := s.config.GetConfigVersion(ctx)
	if err == nil {
		status.ConfigVersion = version
	}

	recentAudit, err := s.audit.ListByDevice(ctx, s.deviceID, 10)
	if err == nil {
		for _, entry := range recentAudit {
			if entry.Result == "ERROR" {
				status.RecentErrors = append(status.RecentErrors, entry.Action+": "+entry.Result)
			}
		}
	}
	if status.RecentErrors == nil {
		status.RecentErrors = []string{}
	}

	return status, nil
}
