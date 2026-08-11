package repository

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"fmt"
	"strings"

	"mentor.local/shared/multitenancy"
)

type productionRunRepo struct {
	pr *multitenancy.PoolResolver
}

func NewProductionRunRepository(pr *multitenancy.PoolResolver) ports.ProductionRunRepository {
	return &productionRunRepo{pr: pr}
}

func (r *productionRunRepo) List(ctx context.Context, f ports.ProductionRunFilter) ([]domain.ProductionRun, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.production_runs")
	isTenant := schema != ""

	var conditions []string
	var args []any
	idx := 1

	if f.DeviceID != "" {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", idx))
		args = append(args, f.DeviceID)
		idx++
	}
	if !isTenant && f.EmpresaID != nil {
		conditions = append(conditions, fmt.Sprintf("empresa_id = $%d", idx))
		args = append(args, *f.EmpresaID)
		idx++
	}
	if !isTenant && f.PlantaID != nil {
		conditions = append(conditions, fmt.Sprintf("planta_id = $%d", idx))
		args = append(args, *f.PlantaID)
		idx++
	}
	if !isTenant && f.LineaID != nil {
		conditions = append(conditions, fmt.Sprintf("linea_id = $%d", idx))
		args = append(args, *f.LineaID)
		idx++
	}
	if f.Since != nil {
		conditions = append(conditions, fmt.Sprintf("(ended_at IS NULL OR ended_at > $%d)", idx))
		args = append(args, *f.Since)
		idx++
	}
	if f.Until != nil {
		conditions = append(conditions, fmt.Sprintf("started_at < $%d", idx))
		args = append(args, *f.Until)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}

	var q string
	if isTenant {
		q = fmt.Sprintf(`
			SELECT id, run_id, device_id,
			       producto_id, sku, nombre,
			       started_at, ended_at, creado_en, actualizado_en
			FROM %s
			%s
			ORDER BY started_at DESC
			LIMIT %d`, tbl, where, limit)
	} else {
		q = fmt.Sprintf(`
			SELECT id, run_id, device_id, linea_id, planta_id, empresa_id,
			       producto_id, sku, nombre,
			       started_at, ended_at, creado_en, actualizado_en
			FROM %s
			%s
			ORDER BY started_at DESC
			LIMIT %d`, tbl, where, limit)
	}

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProductionRun
	for rows.Next() {
		var pr domain.ProductionRun
		if isTenant {
			if err := rows.Scan(
				&pr.ID, &pr.RunID, &pr.DeviceID,
				&pr.ProductoID, &pr.SKU, &pr.Nombre,
				&pr.StartedAt, &pr.EndedAt, &pr.CreatedAt, &pr.UpdatedAt,
			); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(
				&pr.ID, &pr.RunID, &pr.DeviceID,
				&pr.LineaID, &pr.PlantaID, &pr.EmpresaID,
				&pr.ProductoID, &pr.SKU, &pr.Nombre,
				&pr.StartedAt, &pr.EndedAt, &pr.CreatedAt, &pr.UpdatedAt,
			); err != nil {
				return nil, err
			}
		}
		pr.Synced = true
		result = append(result, pr)
	}
	if result == nil {
		result = []domain.ProductionRun{}
	}
	return result, rows.Err()
}

func (r *productionRunRepo) Upsert(ctx context.Context, req *domain.UpsertProductionRunRequest) (*domain.ProductionRun, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.production_runs")

	var pr domain.ProductionRun

	if schema != "" {
		// Si es un run activo nuevo (sin ended_at), cerrar el run activo anterior.
		if req.EndedAt == nil {
			pool.Exec(ctx, fmt.Sprintf(`
				UPDATE %s SET ended_at = $1, actualizado_en = NOW()
				WHERE ended_at IS NULL AND run_id != $2`, tbl),
				req.StartedAt, req.RunID)
		}

		// Tabla tenant: sin linea_id, planta_id, empresa_id
		q := fmt.Sprintf(`
			INSERT INTO %s
			  (run_id, device_id, producto_id, sku, nombre, started_at, ended_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
			ON CONFLICT (device_id, run_id) DO UPDATE SET
			  producto_id    = EXCLUDED.producto_id,
			  sku            = EXCLUDED.sku,
			  nombre         = EXCLUDED.nombre,
			  started_at     = EXCLUDED.started_at,
			  ended_at       = EXCLUDED.ended_at,
			  actualizado_en = NOW()
			RETURNING id, run_id, device_id,
			          producto_id, sku, nombre, started_at, ended_at, creado_en, actualizado_en`, tbl)

		err = pool.QueryRow(ctx, q,
			req.RunID, req.DeviceID,
			req.ProductoID, req.SKU, req.Nombre, req.StartedAt, req.EndedAt,
		).Scan(
			&pr.ID, &pr.RunID, &pr.DeviceID,
			&pr.ProductoID, &pr.SKU, &pr.Nombre,
			&pr.StartedAt, &pr.EndedAt, &pr.CreatedAt, &pr.UpdatedAt,
		)
	} else {
		// Si es un run activo nuevo (sin ended_at), cerrar el run activo anterior.
		if req.EndedAt == nil {
			pool.Exec(ctx, fmt.Sprintf(`
				UPDATE %s SET ended_at = $1, actualizado_en = NOW()
				WHERE ended_at IS NULL AND run_id != $2`, tbl),
				req.StartedAt, req.RunID)
		}

		// Tabla master: con linea_id, planta_id, empresa_id
		q := fmt.Sprintf(`
			INSERT INTO %s
			  (run_id, device_id, linea_id, planta_id, empresa_id, producto_id, sku, nombre, started_at, ended_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			ON CONFLICT (device_id, run_id) DO UPDATE SET
			  linea_id       = EXCLUDED.linea_id,
			  planta_id      = EXCLUDED.planta_id,
			  empresa_id     = EXCLUDED.empresa_id,
			  producto_id    = EXCLUDED.producto_id,
			  sku            = EXCLUDED.sku,
			  nombre         = EXCLUDED.nombre,
			  started_at     = EXCLUDED.started_at,
			  ended_at       = EXCLUDED.ended_at,
			  actualizado_en = NOW()
			RETURNING id, run_id, device_id, linea_id, planta_id, empresa_id,
			          producto_id, sku, nombre, started_at, ended_at, creado_en, actualizado_en`, tbl)

		err = pool.QueryRow(ctx, q,
			req.RunID, req.DeviceID, req.LineaID, req.PlantaID, req.EmpresaID,
			req.ProductoID, req.SKU, req.Nombre, req.StartedAt, req.EndedAt,
		).Scan(
			&pr.ID, &pr.RunID, &pr.DeviceID,
			&pr.LineaID, &pr.PlantaID, &pr.EmpresaID,
			&pr.ProductoID, &pr.SKU, &pr.Nombre,
			&pr.StartedAt, &pr.EndedAt, &pr.CreatedAt, &pr.UpdatedAt,
		)
	}

	if err != nil {
		return nil, err
	}
	pr.Synced = true
	return &pr, nil
}

func (r *productionRunRepo) Delete(ctx context.Context, runID string) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.production_runs")
	_, err = pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE run_id = $1`, tbl), runID)
	return err
}
