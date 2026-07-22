package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mentor.local/shared/multitenancy"

	"github.com/jackc/pgx/v5"
)

type VelocidadNominalRepo struct {
	pr *multitenancy.PoolResolver
}

func NewVelocidadNominalRepo(pr *multitenancy.PoolResolver) *VelocidadNominalRepo {
	return &VelocidadNominalRepo{pr: pr}
}

func (r *VelocidadNominalRepo) GetActiveVelocidad(ctx context.Context, lineaID int, ts time.Time) (float64, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return 0, err
	}
	tblRuns := multitenancy.Tbl(schema, "analytics.production_runs")
	tblProd := multitenancy.Tbl(schema, "config.productos")
	tblVel := multitenancy.Tbl(schema, "config.velocidad_nominal")

	var vel float64
	var queryErr error

	if schema != "" {
		// Modo tenant: production_runs está en el schema de la línea,
		// por lo que ya está filtrado implícitamente — no hace falta linea_id.
		queryErr = pool.QueryRow(ctx, fmt.Sprintf(`
			SELECT vn.velocidad_us
			FROM %s pr
			LEFT JOIN %s p ON p.codigo = pr.sku
			JOIN %s vn
				ON vn.producto_id = COALESCE(pr.producto_id, p.id)
			WHERE pr.started_at <= $1
			  AND (pr.ended_at IS NULL OR pr.ended_at >= $1)
			  AND COALESCE(pr.producto_id, p.id) IS NOT NULL
			ORDER BY pr.started_at DESC
			LIMIT 1`, tblRuns, tblProd, tblVel), ts,
		).Scan(&vel)
	} else {
		// Modo legacy: tabla compartida, filtrar por linea_id.
		queryErr = pool.QueryRow(ctx, fmt.Sprintf(`
			SELECT vn.velocidad_us
			FROM %s pr
			LEFT JOIN %s p ON p.codigo = pr.sku
			JOIN %s vn
				ON vn.producto_id = COALESCE(pr.producto_id, p.id)
			WHERE pr.linea_id = $1
			  AND pr.started_at <= $2
			  AND (pr.ended_at IS NULL OR pr.ended_at >= $2)
			  AND COALESCE(pr.producto_id, p.id) IS NOT NULL
			ORDER BY pr.started_at DESC
			LIMIT 1`, tblRuns, tblProd, tblVel), lineaID, ts,
		).Scan(&vel)
	}

	if errors.Is(queryErr, pgx.ErrNoRows) {
		return 0, nil
	}
	return vel, queryErr
}
