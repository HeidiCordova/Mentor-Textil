package ports

import (
	"cloud-config/internal/domain"
	"context"
)

type PlantaRepository interface {
	FindAll(ctx context.Context, empresaID *int) ([]domain.Planta, error)
	FindByID(ctx context.Context, id int) (*domain.Planta, error)
	Create(ctx context.Context, p *domain.Planta) error
	Update(ctx context.Context, p *domain.Planta) error
	Delete(ctx context.Context, id int) error
}

type LineaRepository interface {
	FindAll(ctx context.Context, plantaID *int, empresaID *int) ([]domain.Linea, error)
	FindByID(ctx context.Context, id int) (*domain.Linea, error)
	Create(ctx context.Context, l *domain.Linea) error
	Update(ctx context.Context, l *domain.Linea) error
	Delete(ctx context.Context, id int) error
}

type DispositivoRepository interface {
	FindAll(ctx context.Context) ([]domain.Dispositivo, error)
	FindByID(ctx context.Context, id int) (*domain.Dispositivo, error)
	FindByDeviceID(ctx context.Context, deviceID string) (*domain.Dispositivo, error)
	Create(ctx context.Context, d *domain.Dispositivo) error
	Update(ctx context.Context, d *domain.Dispositivo) error
	Delete(ctx context.Context, id int) error
	UpdatePing(ctx context.Context, deviceID string) error
}

type VariableRepository interface {
	FindAll(ctx context.Context, dispositivoID *int) ([]domain.Variable, error)
	FindByID(ctx context.Context, id int) (*domain.Variable, error)
	Create(ctx context.Context, v *domain.Variable) error
	Update(ctx context.Context, v *domain.Variable) error
	Delete(ctx context.Context, id int) error
}

type DeviceConfigRepository interface {
	GetPending(ctx context.Context, dispositivoID int) (*domain.DeviceConfig, error)
	GetLatestVersion(ctx context.Context, dispositivoID int) (int, error)
	Create(ctx context.Context, dc *domain.DeviceConfig) error
	MarkApplied(ctx context.Context, dispositivoID int, version int) error
}
