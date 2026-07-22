package repository

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"mentor.local/shared/multitenancy"
)

type analysisRepo struct {
	pr *multitenancy.PoolResolver
}

func NewAnalysisRepository(pr *multitenancy.PoolResolver) ports.AnalysisRepository {
	return &analysisRepo{pr: pr}
}

func (r *analysisRepo) GetGeneral(ctx context.Context, f ports.AnalysisFilter) ([]domain.AnalysisPoint, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	clauses, args := r.buildWhere(f)
	q := fmt.Sprintf(`
		SELECT TO_CHAR(fecha, 'YYYY-MM-DD') || 'T' || TO_CHAR(hora, 'HH24:MI:SS') AS ts,
		       COALESCE(oee, 0),
		       CASE WHEN oee >= 85 THEN 'optimal'
		            WHEN oee >= 60 THEN 'warning'
		            ELSE 'critical' END AS status
		FROM %s
		%s
		ORDER BY fecha DESC, hora DESC
		LIMIT %d`, tbl, clauses, r.limit(f))

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AnalysisPoint
	for rows.Next() {
		var p domain.AnalysisPoint
		if err := rows.Scan(&p.Timestamp, &p.Value, &p.Status); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *analysisRepo) GetEnergy(ctx context.Context, f ports.AnalysisFilter) ([]domain.EnergyAnalysis, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	clauses, args := r.buildWhere(f)
	q := fmt.Sprintf(`
		SELECT TO_CHAR(fecha, 'YYYY-MM-DD') || 'T' || TO_CHAR(hora, 'HH24:MI:SS') AS ts,
		       COALESCE(energia_kwh, 0),
		       COALESCE(energia_kwh, 0) / NULLIF(EXTRACT(EPOCH FROM interval '1 hour'), 0) AS power
		FROM %s
		%s
		ORDER BY fecha DESC, hora DESC
		LIMIT %d`, tbl, clauses, r.limit(f))

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.EnergyAnalysis
	for rows.Next() {
		var e domain.EnergyAnalysis
		if err := rows.Scan(&e.Timestamp, &e.Consumption, &e.Power); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *analysisRepo) GetProduction(ctx context.Context, f ports.AnalysisFilter) ([]domain.ProductionAnalysis, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	clauses, args := r.buildWhere(f)
	q := fmt.Sprintf(`
		SELECT TO_CHAR(fecha, 'YYYY-MM-DD') || 'T' || TO_CHAR(hora, 'HH24:MI:SS') AS ts,
		       COALESCE(produccion, 0),
		       COALESCE(produccion, 0) AS target,
		       COALESCE(oee, 0) AS efficiency
		FROM %s
		%s
		ORDER BY fecha DESC, hora DESC
		LIMIT %d`, tbl, clauses, r.limit(f))

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProductionAnalysis
	for rows.Next() {
		var p domain.ProductionAnalysis
		if err := rows.Scan(&p.Timestamp, &p.Produced, &p.Target, &p.Efficiency); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

func (r *analysisRepo) GetPareto(ctx context.Context, f ports.AnalysisFilter) ([]domain.ParetoItem, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tblP := multitenancy.Tbl(schema, "analytics.paradas")

	var args []any
	idx := 1

	var conditions []string
	if f.EmpresaID != nil {
		conditions = append(conditions, fmt.Sprintf("p.empresa_id = $%d", idx))
		args = append(args, *f.EmpresaID)
		idx++
	}
	if f.PlantaID != nil {
		conditions = append(conditions, fmt.Sprintf("p.planta_id = $%d", idx))
		args = append(args, *f.PlantaID)
		idx++
	}
	if f.LineaID != nil {
		conditions = append(conditions, fmt.Sprintf("p.linea_id = $%d", idx))
		args = append(args, *f.LineaID)
		idx++
	}
	if f.From != "" {
		conditions = append(conditions, fmt.Sprintf("p.inicio >= $%d", idx))
		args = append(args, f.From)
		idx++
	}
	if f.To != "" {
		conditions = append(conditions, fmt.Sprintf("p.inicio <= $%d", idx))
		args = append(args, f.To)
		idx++
	}
	clauses := ""
	if len(conditions) > 0 {
		clauses = "WHERE " + strings.Join(conditions, " AND ")
	}

	// When schema is set (planta-scoped) JOIN against the new per-linea
	// category tables (cat_programada + cat_no_programada).
	// In fallback mode (no scope) there are no category lookup tables, so
	// we rely on the categoria_nombre column stored in the parada row.
	var q string
	if schema != "" {
		tblCP := schema + ".cat_programada"
		tblCN := schema + ".cat_no_programada"
		q = fmt.Sprintf(`
		WITH cats AS (
			SELECT id, nombre FROM %s
			UNION ALL
			SELECT id, nombre FROM %s
		)
		SELECT COALESCE(c.nombre, p.categoria_nombre, 'Sin categoria') AS categoria,
		       COUNT(*)::int AS frecuencia,
		       ROUND(COUNT(*) * 100.0 / NULLIF(SUM(COUNT(*)) OVER(), 0), 2) AS porcentaje
		FROM %s p
		LEFT JOIN cats c ON c.id = p.categoria_id
		%s
		GROUP BY COALESCE(c.nombre, p.categoria_nombre, 'Sin categoria')
		ORDER BY frecuencia DESC
		LIMIT 20`, tblCP, tblCN, tblP, clauses)
	} else {
		q = fmt.Sprintf(`
		SELECT COALESCE(p.categoria_nombre, 'Sin categoria') AS categoria,
		       COUNT(*)::int AS frecuencia,
		       ROUND(COUNT(*) * 100.0 / NULLIF(SUM(COUNT(*)) OVER(), 0), 2) AS porcentaje
		FROM %s p
		%s
		GROUP BY COALESCE(p.categoria_nombre, 'Sin categoria')
		ORDER BY frecuencia DESC
		LIMIT 20`, tblP, clauses)
	}

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ParetoItem
	for rows.Next() {
		var item domain.ParetoItem
		if err := rows.Scan(&item.Categoria, &item.Frecuencia, &item.Porcentaje); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *analysisRepo) GetCombined(ctx context.Context, f ports.AnalysisFilter) (*domain.FullAnalysis, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	result := &domain.FullAnalysis{}
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")
		clauses, args := r.buildWhere(f)
		q := fmt.Sprintf(`
			WITH base AS (
				SELECT fecha, hora,
				       COALESCE(oee, 0) AS oee,
				       COALESCE(energia_kwh, 0) AS energia,
				       COALESCE(produccion, 0) AS produccion
				FROM %s
				%s
				ORDER BY fecha DESC, hora DESC
				LIMIT %d
			)
			SELECT TO_CHAR(fecha, 'YYYY-MM-DD') || 'T' || TO_CHAR(hora, 'HH24:MI:SS'),
			       oee,
			       CASE WHEN oee >= 85 THEN 'optimal'
			            WHEN oee >= 60 THEN 'warning'
			            ELSE 'critical' END,
			       energia,
			       energia / NULLIF(EXTRACT(EPOCH FROM interval '1 hour'), 0),
			       produccion::int,
			       oee
			FROM base`, tbl, clauses, r.limit(f))

		rows, err := pool.Query(gctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var c domain.CombinedAnalysis
			if err := rows.Scan(&c.Timestamp, &c.OEE, &c.OEEStatus, &c.Consumption, &c.Power, &c.Produced, &c.Efficiency); err != nil {
				return err
			}
			result.Snapshots = append(result.Snapshots, c)
		}
		return rows.Err()
	})

	g.Go(func() error {
		pareto, err := r.GetPareto(gctx, f)
		if err != nil {
			return err
		}
		result.Pareto = pareto
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *analysisRepo) buildWhere(f ports.AnalysisFilter) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if f.DeviceID != "" {
		conditions = append(conditions, fmt.Sprintf("device_id = $%d", idx))
		args = append(args, f.DeviceID)
		idx++
	}
	if f.From != "" {
		conditions = append(conditions, fmt.Sprintf("fecha >= $%d", idx))
		args = append(args, f.From)
		idx++
	}
	if f.To != "" {
		conditions = append(conditions, fmt.Sprintf("fecha <= $%d", idx))
		args = append(args, f.To)
		idx++
	}
	minIS := f.MinIntervalS
	if minIS <= 0 {
		minIS = 300
	}
	conditions = append(conditions, fmt.Sprintf("interval_s >= %d", minIS))
	if len(conditions) == 1 {
		today := time.Now().Format("2006-01-02")
		conditions = append(conditions, fmt.Sprintf("fecha = $%d", idx))
		args = append(args, today)
	}
	return "WHERE " + strings.Join(conditions, " AND "), args
}

func (r *analysisRepo) limit(f ports.AnalysisFilter) int {
	if f.Limit > 0 && f.Limit <= 1000 {
		return f.Limit
	}
	return 200
}
