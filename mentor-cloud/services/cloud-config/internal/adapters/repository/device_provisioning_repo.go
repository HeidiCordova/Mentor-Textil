package repository

import (
	"cloud-config/internal/domain"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DeviceProvisioningRepo struct {
	db *pgxpool.Pool
}

func NewDeviceProvisioningRepo(db *pgxpool.Pool) *DeviceProvisioningRepo {
	return &DeviceProvisioningRepo{db: db}
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *DeviceProvisioningRepo) Provision(ctx context.Context, d *domain.Dispositivo) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO config.dispositivos (device_id, nombre, linea_id, planta_id, empresa_id, estado, activo)
		 VALUES ($1,$2,$3,$4,$5,'offline',true) RETURNING id`,
		d.DeviceID, d.Nombre, d.LineaID, d.PlantaID, d.EmpresaID,
	).Scan(&d.ID)
	if err != nil {
		return "", fmt.Errorf("insert dispositivo: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO gateway.device_registry (device_id, empresa_id, planta_id, linea_id, api_key, active)
		 VALUES ($1,$2,$3,$4,$5,true)`,
		d.DeviceID, d.EmpresaID, d.PlantaID, d.LineaID, apiKey,
	)
	if err != nil {
		return "", fmt.Errorf("insert device_registry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	d.Activo = true
	d.Estado = "offline"
	return apiKey, nil
}

func (r *DeviceProvisioningRepo) UpdateWithRegistry(ctx context.Context, d *domain.Dispositivo) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var deviceID string
	err = tx.QueryRow(ctx,
		`UPDATE config.dispositivos
		 SET nombre=$2, linea_id=$3, planta_id=$4, empresa_id=$5, locacion_id=$6, activo=$7, actualizado_en=NOW()
		 WHERE id=$1 RETURNING device_id`,
		d.ID, d.Nombre, d.LineaID, d.PlantaID, d.EmpresaID, d.LocacionID, d.Activo,
	).Scan(&deviceID)
	if err != nil {
		return fmt.Errorf("update dispositivo: %w", err)
	}

	var empresaID int64
	if d.EmpresaID != nil {
		empresaID = int64(*d.EmpresaID)
	}

	_, err = tx.Exec(ctx,
		`UPDATE gateway.device_registry SET empresa_id=$1, planta_id=$2, linea_id=$3, active=$4 WHERE device_id=$5`,
		empresaID, d.PlantaID, d.LineaID, d.Activo, deviceID,
	)
	if err != nil {
		return fmt.Errorf("update registry: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *DeviceProvisioningRepo) Revoke(ctx context.Context, id int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var deviceID string
	err = tx.QueryRow(ctx,
		`UPDATE config.dispositivos SET activo=false, estado='revoked', actualizado_en=NOW()
		 WHERE id=$1 RETURNING device_id`, id,
	).Scan(&deviceID)
	if err != nil {
		return fmt.Errorf("revoke dispositivo: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE gateway.device_registry SET active=false WHERE device_id=$1`, deviceID,
	)
	if err != nil {
		return fmt.Errorf("revoke registry: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *DeviceProvisioningRepo) Activate(ctx context.Context, id int) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var deviceID string
	err = tx.QueryRow(ctx,
		`UPDATE config.dispositivos SET activo=true, estado='offline', actualizado_en=NOW()
		 WHERE id=$1 RETURNING device_id`, id,
	).Scan(&deviceID)
	if err != nil {
		return "", fmt.Errorf("activate dispositivo: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE gateway.device_registry SET active=true, api_key=$1 WHERE device_id=$2`,
		apiKey, deviceID,
	)
	if err != nil {
		return "", fmt.Errorf("activate registry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	return apiKey, nil
}

func (r *DeviceProvisioningRepo) RegenerateKey(ctx context.Context, id int) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var deviceID string
	err = tx.QueryRow(ctx,
		`SELECT device_id FROM config.dispositivos WHERE id=$1 AND activo=true`, id,
	).Scan(&deviceID)
	if err != nil {
		return "", fmt.Errorf("find dispositivo: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE gateway.device_registry SET api_key=$1 WHERE device_id=$2`,
		apiKey, deviceID,
	)
	if err != nil {
		return "", fmt.Errorf("update key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}

	return apiKey, nil
}

func (r *DeviceProvisioningRepo) Delete(ctx context.Context, id int) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var deviceID string
	err = tx.QueryRow(ctx,
		`DELETE FROM config.dispositivos WHERE id=$1 RETURNING device_id`, id,
	).Scan(&deviceID)
	if err != nil {
		return fmt.Errorf("delete dispositivo: %w", err)
	}

	_, err = tx.Exec(ctx,
		`DELETE FROM gateway.device_registry WHERE device_id=$1`, deviceID,
	)
	if err != nil {
		return fmt.Errorf("delete registry: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *DeviceProvisioningRepo) FindAllWithStatus(ctx context.Context, empresaID *int) ([]domain.DispositivoConEstado, error) {
	query := `SELECT d.id, d.device_id, d.nombre, d.linea_id, d.locacion_id, d.planta_id, d.empresa_id,
	                 d.activo,
	                 gr.last_seen_at, COALESCE(gr.active, false),
	                 CASE WHEN gr.last_seen_at > NOW() - INTERVAL '5 minutes' THEN 'online' ELSE 'offline' END
	          FROM config.dispositivos d
	          LEFT JOIN gateway.device_registry gr ON gr.device_id = d.device_id`
	args := []interface{}{}
	if empresaID != nil {
		query += ` WHERE d.empresa_id = $1`
		args = append(args, *empresaID)
	}
	query += ` ORDER BY d.id`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.DispositivoConEstado
	for rows.Next() {
		var d domain.DispositivoConEstado
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.Nombre, &d.LineaID, &d.LocacionID, &d.PlantaID,
			&d.EmpresaID, &d.Activo,
			&d.LastSeenAt, &d.GatewayActive, &d.Estado); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, nil
}

func (r *DeviceProvisioningRepo) FindByIDWithStatus(ctx context.Context, id int) (*domain.DispositivoConEstado, error) {
	d := &domain.DispositivoConEstado{}
	err := r.db.QueryRow(ctx,
		`SELECT d.id, d.device_id, d.nombre, d.linea_id, d.locacion_id, d.planta_id, d.empresa_id,
		        d.activo,
		        gr.last_seen_at, COALESCE(gr.active, false),
		        CASE WHEN gr.last_seen_at > NOW() - INTERVAL '5 minutes' THEN 'online' ELSE 'offline' END
		 FROM config.dispositivos d
		 LEFT JOIN gateway.device_registry gr ON gr.device_id = d.device_id
		 WHERE d.id=$1`, id,
	).Scan(&d.ID, &d.DeviceID, &d.Nombre, &d.LineaID, &d.LocacionID, &d.PlantaID,
		&d.EmpresaID, &d.Activo,
		&d.LastSeenAt, &d.GatewayActive, &d.Estado)
	if err != nil {
		return nil, err
	}
	return d, nil
}
