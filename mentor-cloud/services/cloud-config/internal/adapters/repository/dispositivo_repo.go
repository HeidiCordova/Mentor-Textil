package repository

import (
	"cloud-config/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DispositivoRepo struct {
	db *pgxpool.Pool
}

func NewDispositivoRepo(db *pgxpool.Pool) *DispositivoRepo {
	return &DispositivoRepo{db: db}
}

func (r *DispositivoRepo) FindAll(ctx context.Context) ([]domain.Dispositivo, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, device_id, nombre, linea_id, planta_id, empresa_id, estado, ultimo_ping, activo
		 FROM config.dispositivos ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Dispositivo
	for rows.Next() {
		var d domain.Dispositivo
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.Nombre, &d.LineaID, &d.PlantaID,
			&d.EmpresaID, &d.Estado, &d.UltimoPing, &d.Activo); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

func (r *DispositivoRepo) FindByID(ctx context.Context, id int) (*domain.Dispositivo, error) {
	d := &domain.Dispositivo{}
	err := r.db.QueryRow(ctx,
		`SELECT id, device_id, nombre, linea_id, planta_id, empresa_id, estado, ultimo_ping, activo
		 FROM config.dispositivos WHERE id=$1`, id,
	).Scan(&d.ID, &d.DeviceID, &d.Nombre, &d.LineaID, &d.PlantaID,
		&d.EmpresaID, &d.Estado, &d.UltimoPing, &d.Activo)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DispositivoRepo) FindByDeviceID(ctx context.Context, deviceID string) (*domain.Dispositivo, error) {
	d := &domain.Dispositivo{}
	err := r.db.QueryRow(ctx,
		`SELECT id, device_id, nombre, linea_id, planta_id, empresa_id, estado, ultimo_ping, activo
		 FROM config.dispositivos WHERE device_id=$1`, deviceID,
	).Scan(&d.ID, &d.DeviceID, &d.Nombre, &d.LineaID, &d.PlantaID,
		&d.EmpresaID, &d.Estado, &d.UltimoPing, &d.Activo)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *DispositivoRepo) Create(ctx context.Context, d *domain.Dispositivo) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO config.dispositivos (device_id, nombre, linea_id, planta_id, empresa_id, estado, activo)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		d.DeviceID, d.Nombre, d.LineaID, d.PlantaID, d.EmpresaID, d.Estado, d.Activo,
	).Scan(&d.ID)
}

func (r *DispositivoRepo) Update(ctx context.Context, d *domain.Dispositivo) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config.dispositivos
		 SET nombre=$2, linea_id=$3, planta_id=$4, empresa_id=$5, estado=$6, activo=$7, actualizado_en=NOW()
		 WHERE id=$1`,
		d.ID, d.Nombre, d.LineaID, d.PlantaID, d.EmpresaID, d.Estado, d.Activo)
	return err
}

func (r *DispositivoRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM config.dispositivos WHERE id=$1`, id)
	return err
}

func (r *DispositivoRepo) UpdatePing(ctx context.Context, deviceID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config.dispositivos SET ultimo_ping=NOW(), estado='online' WHERE device_id=$1`, deviceID)
	return err
}

func (r *DispositivoRepo) FindByLinea(ctx context.Context, lineaID int) ([]domain.Dispositivo, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, device_id, nombre, linea_id, planta_id, empresa_id, estado, ultimo_ping, activo
		 FROM config.dispositivos WHERE linea_id=$1 AND activo=true ORDER BY id`, lineaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Dispositivo
	for rows.Next() {
		var d domain.Dispositivo
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.Nombre, &d.LineaID, &d.PlantaID,
			&d.EmpresaID, &d.Estado, &d.UltimoPing, &d.Activo); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}
