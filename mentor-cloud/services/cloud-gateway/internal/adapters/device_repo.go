package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"cloud-gateway/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"mentor.local/shared/cache"
)

const deviceCacheTTL = 60 * time.Second

type deviceRepo struct {
	pool  *pgxpool.Pool
	cache cache.Store
}

func NewDeviceRepo(pool *pgxpool.Pool, c cache.Store) *deviceRepo {
	return &deviceRepo{pool: pool, cache: c}
}

func (r *deviceRepo) FindByAPIKey(ctx context.Context, apiKey string) (*domain.Device, error) {
	cacheKey := "device:" + apiKey
	if raw, ok := r.cache.Get(cacheKey); ok {
		var d domain.Device
		if json.Unmarshal(raw, &d) == nil {
			return &d, nil
		}
	}

	d := &domain.Device{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, device_id, COALESCE(empresa_id,0), COALESCE(planta_id,0), COALESCE(linea_id,0),
		        api_key, last_seen_at, registered_at, active
		 FROM gateway.device_registry
		 WHERE api_key = $1 AND active = true`,
		apiKey,
	).Scan(&d.ID, &d.DeviceID, &d.EmpresaID, &d.PlantaID, &d.LineaID,
		&d.APIKey, &d.LastSeenAt, &d.RegisteredAt, &d.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err == nil {
		if raw, e := json.Marshal(d); e == nil {
			r.cache.Set(cacheKey, raw, deviceCacheTTL)
		}
	}
	return d, err
}

func (r *deviceRepo) UpdateLastSeen(ctx context.Context, deviceID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateway.device_registry SET last_seen_at = NOW() WHERE device_id = $1`,
		deviceID,
	)
	return err
}

func (r *deviceRepo) ListByEmpresaID(ctx context.Context, empresaID int64) ([]domain.Device, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, COALESCE(empresa_id,0), COALESCE(planta_id,0), COALESCE(linea_id,0),
		        api_key, last_seen_at, registered_at, active
		 FROM gateway.device_registry
		 WHERE empresa_id = $1 AND active = true`,
		empresaID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []domain.Device
	for rows.Next() {
		var d domain.Device
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.EmpresaID, &d.PlantaID, &d.LineaID,
			&d.APIKey, &d.LastSeenAt, &d.RegisteredAt, &d.Active); err != nil {
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

func (r *deviceRepo) PlantaBelongsToEmpresa(ctx context.Context, plantaID, empresaID int64) bool {
	var exists bool
	_ = r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM config.plantas WHERE id=$1 AND empresa_id=$2)`,
		plantaID, empresaID,
	).Scan(&exists)
	return exists
}

func (r *deviceRepo) LineaBelongsToEmpresa(ctx context.Context, lineaID, empresaID int64) bool {
	var exists bool
	_ = r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM config.lineas l
			JOIN config.plantas p ON p.id = l.planta_id
			WHERE l.id=$1 AND p.empresa_id=$2
		)`,
		lineaID, empresaID,
	).Scan(&exists)
	return exists
}
