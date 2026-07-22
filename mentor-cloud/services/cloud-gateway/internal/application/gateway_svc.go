package application

import (
	"context"
	"encoding/json"

	"cloud-gateway/internal/domain"
	"cloud-gateway/internal/ports"
)

type GatewayService struct {
	commands ports.CommandRepository
	devices  ports.DeviceRepository
}

func NewGatewayService(commands ports.CommandRepository, devices ports.DeviceRepository) *GatewayService {
	return &GatewayService{commands: commands, devices: devices}
}

func (s *GatewayService) DispatchCommand(ctx context.Context, cmd *domain.Command) (*domain.Command, error) {
	cmd.Status = domain.StatusPending
	return s.commands.Insert(ctx, cmd)
}

func (s *GatewayService) PendingCommands(ctx context.Context, deviceID string) ([]domain.Command, error) {
	return s.commands.ListPending(ctx, deviceID)
}

func (s *GatewayService) AcknowledgeCommand(ctx context.Context, id int64, deviceID string) error {
	return s.commands.Acknowledge(ctx, id, deviceID)
}

func (s *GatewayService) BroadcastToEmpresa(ctx context.Context, empresaID int64, cmdType string, payload json.RawMessage) ([]*domain.Command, error) {
	devices, err := s.devices.ListByEmpresaID(ctx, empresaID)
	if err != nil {
		return nil, err
	}
	var result []*domain.Command
	for _, d := range devices {
		cmd := &domain.Command{
			DeviceID:  d.DeviceID,
			EmpresaID: empresaID,
			Type:      cmdType,
			Payload:   payload,
			Status:    domain.StatusPending,
			IssuedBy:  0,
		}
		created, err := s.commands.Insert(ctx, cmd)
		if err != nil {
			return result, err
		}
		result = append(result, created)
	}
	return result, nil
}

// ListDevicesByEmpresa returns all active devices for an empresa.
func (s *GatewayService) ListDevicesByEmpresa(ctx context.Context, empresaID int64) ([]domain.Device, error) {
	return s.devices.ListByEmpresaID(ctx, empresaID)
}

// CreateCommand inserts a single command for a specific device.
func (s *GatewayService) CreateCommand(ctx context.Context, cmd *domain.Command) (*domain.Command, error) {
	cmd.Status = domain.StatusPending
	return s.commands.Insert(ctx, cmd)
}
