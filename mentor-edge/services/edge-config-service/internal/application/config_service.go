package application

import (
	"context"

	"edge-config-service/internal/domain"
	"edge-config-service/internal/ports"
)

type ConfigService struct {
	storage ports.ConfigStorage
}

func NewConfigService(storage ports.ConfigStorage) *ConfigService {
	return &ConfigService{
		storage: storage,
	}
}

func (s *ConfigService) GetConfig(ctx context.Context, lineaID int) (*domain.LineConfig, error) {
	return s.storage.GetConfig(ctx, lineaID)
}

func (s *ConfigService) UpdateConfig(ctx context.Context, config *domain.LineConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	return s.storage.UpdateConfig(ctx, config)
}

func (s *ConfigService) GetConfigVersion(ctx context.Context, lineaID int) (int, error) {
	return s.storage.GetConfigVersion(ctx, lineaID)
}

func (s *ConfigService) GetLineIDs(ctx context.Context) ([]int, error) {
	return s.storage.GetLineIDs(ctx)
}

func (s *ConfigService) DeleteLine(ctx context.Context, lineaID int) error {
	return s.storage.DeleteLine(ctx, lineaID)
}

func (s *ConfigService) GetDeviceID(ctx context.Context) (string, error) {
	return s.storage.GetDeviceID(ctx)
}

func (s *ConfigService) SetDeviceID(ctx context.Context, deviceID string) error {
	return s.storage.SetDeviceID(ctx, deviceID)
}
