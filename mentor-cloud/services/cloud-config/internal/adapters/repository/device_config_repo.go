package repository

import (
	"cloud-config/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceConfigRepo struct {
	db *pgxpool.Pool
}

func NewDeviceConfigRepo(db *pgxpool.Pool) *DeviceConfigRepo {
	return &DeviceConfigRepo{db: db}
}

func (r *DeviceConfigRepo) GetPending(ctx context.Context, dispositivoID int) (*domain.DeviceConfig, error) {
	dc := &domain.DeviceConfig{}
	err := r.db.QueryRow(ctx,
		`SELECT id, dispositivo_id, version, payload, aplicado, creado_en
		 FROM config.device_configs
		 WHERE dispositivo_id=$1 AND aplicado=false
		 ORDER BY version DESC LIMIT 1`, dispositivoID,
	).Scan(&dc.ID, &dc.DispositivoID, &dc.Version, &dc.Payload, &dc.Aplicado, &dc.CreadoEn)
	if err != nil {
		return nil, err
	}
	return dc, nil
}

func (r *DeviceConfigRepo) GetLatestVersion(ctx context.Context, dispositivoID int) (int, error) {
	var v int
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM config.device_configs WHERE dispositivo_id=$1`, dispositivoID,
	).Scan(&v)
	return v, err
}

func (r *DeviceConfigRepo) Create(ctx context.Context, dc *domain.DeviceConfig) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO config.device_configs (dispositivo_id, version, payload) VALUES ($1,$2,$3) RETURNING id`,
		dc.DispositivoID, dc.Version, dc.Payload,
	).Scan(&dc.ID)
}

func (r *DeviceConfigRepo) MarkApplied(ctx context.Context, dispositivoID int, version int) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config.device_configs SET aplicado=true WHERE dispositivo_id=$1 AND version=$2`,
		dispositivoID, version)
	return err
}
