package repository

import (
	"cloud-analytics/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mentor.local/shared/cache"
	"mentor.local/shared/multitenancy"

	"golang.org/x/sync/errgroup"
)

type DashboardRepo struct {
	pr        *multitenancy.PoolResolver
	ingestURL string
	cache     cache.Store
}

func NewDashboardRepo(pr *multitenancy.PoolResolver, ingestURL string, c cache.Store) *DashboardRepo {
	return &DashboardRepo{pr: pr, ingestURL: ingestURL, cache: c}
}

func (r *DashboardRepo) GetStats(ctx context.Context, empresaID *int) (*domain.DashboardStats, error) {
	cacheKey := "dashboard:stats"
	if empresaID != nil {
		cacheKey = fmt.Sprintf("dashboard:stats:%d", *empresaID)
	}

	if raw, ok := r.cache.Get(cacheKey); ok {
		var cached domain.DashboardStats
		if err := json.Unmarshal(raw, &cached); err == nil {
			return &cached, nil
		}
	}

	master := r.pr.Master()
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	alarmasTbl := multitenancy.Tbl(schema, "analytics.alarmas")
	oeeTbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")

	stats := &domain.DashboardStats{}
	today := time.Now().Format("2006-01-02")

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return master.QueryRow(gctx, `SELECT COUNT(*) FROM config.plantas WHERE activo=true`).Scan(&stats.TotalPlantas)
	})
	g.Go(func() error {
		return master.QueryRow(gctx, `SELECT COUNT(*) FROM config.lineas WHERE activo=true`).Scan(&stats.TotalLineas)
	})
	g.Go(func() error {
		return master.QueryRow(gctx, `SELECT COUNT(*) FROM config.dispositivos WHERE activo=true`).Scan(&stats.TotalDispositivos)
	})
	g.Go(func() error {
		return pool.QueryRow(gctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE activa=true`, alarmasTbl)).Scan(&stats.AlarmasActivas)
	})
	g.Go(func() error {
		return pool.QueryRow(gctx,
			fmt.Sprintf(`SELECT COALESCE(AVG(oee), 0) FROM %s WHERE fecha=$1 AND interval_s >= 1800`, oeeTbl),
			today,
		).Scan(&stats.EficienciaPromedio)
	})
	g.Go(func() error {
		return pool.QueryRow(gctx,
			fmt.Sprintf(`SELECT COALESCE(SUM(produccion), 0) FROM %s WHERE fecha=$1 AND interval_s >= 1800`, oeeTbl),
			today,
		).Scan(&stats.ProduccionHoy)
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	if raw, err := json.Marshal(stats); err == nil {
		r.cache.Set(cacheKey, raw, 30*time.Second)
	}

	return stats, nil
}

func (r *DashboardRepo) GetCharts(ctx context.Context, empresaID *int) (*domain.DashboardCharts, error) {
	cacheKey := "dashboard:charts"
	if empresaID != nil {
		cacheKey = fmt.Sprintf("dashboard:charts:%d", *empresaID)
	}

	if raw, ok := r.cache.Get(cacheKey); ok {
		var cached domain.DashboardCharts
		if err := json.Unmarshal(raw, &cached); err == nil {
			return &cached, nil
		}
	}

	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	oeeTbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")

	charts := &domain.DashboardCharts{}
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var q string
		if schema != "" {
			q = fmt.Sprintf(`SELECT TO_CHAR(mes, 'TMMonth'), COALESCE(total, 0)
				FROM %s.mv_produccion_mensual ORDER BY mes LIMIT 12`, schema)
		} else {
			q = fmt.Sprintf(`SELECT TO_CHAR(fecha, 'TMMonth'), COALESCE(SUM(produccion), 0)
				FROM %s
				WHERE interval_s >= 1800 AND fecha >= NOW() - INTERVAL '12 months'
				GROUP BY DATE_TRUNC('month', fecha), TO_CHAR(fecha, 'TMMonth')
				ORDER BY DATE_TRUNC('month', fecha)`, oeeTbl)
		}
		rows, err := pool.Query(gctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var cp domain.ChartPoint
			if err := rows.Scan(&cp.Mes, &cp.Valor); err != nil {
				return err
			}
			charts.Produccion = append(charts.Produccion, cp)
		}
		return rows.Err()
	})

	g.Go(func() error {
		rows, err := pool.Query(gctx, fmt.Sprintf(`
			SELECT TO_CHAR(hora, 'HH24:MI'), COALESCE(AVG(energia_kwh), 0)
			FROM %s
			WHERE interval_s >= 1800 AND fecha = CURRENT_DATE
			GROUP BY DATE_TRUNC('hour', hora), TO_CHAR(hora, 'HH24:MI')
			ORDER BY DATE_TRUNC('hour', hora)`, oeeTbl))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var ep domain.EnergyPoint
			if err := rows.Scan(&ep.Hora, &ep.Consumo); err != nil {
				return err
			}
			charts.Energia = append(charts.Energia, ep)
		}
		return rows.Err()
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("get charts: %w", err)
	}

	if raw, err := json.Marshal(charts); err == nil {
		r.cache.Set(cacheKey, raw, 60*time.Minute)
	}

	return charts, nil
}

func (r *DashboardRepo) GetReports(ctx context.Context, empresaID *int, from, to string) ([]domain.DashboardReport, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	oeeTbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")

	query := fmt.Sprintf(`
		SELECT fecha::text, COALESCE(SUM(produccion), 0), 'Programada'
		FROM %s WHERE interval_s >= 1800`, oeeTbl)
	args := []interface{}{}
	idx := 1

	if from != "" {
		query += fmt.Sprintf(` AND fecha >= $%d`, idx)
		args = append(args, from)
		idx++
	}
	if to != "" {
		query += fmt.Sprintf(` AND fecha <= $%d`, idx)
		args = append(args, to)
		idx++
	}
	query += ` GROUP BY fecha ORDER BY fecha DESC LIMIT 30`

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.DashboardReport
	for rows.Next() {
		var dr domain.DashboardReport
		if err := rows.Scan(&dr.Fecha, &dr.Valor, &dr.Tipo); err != nil {
			return nil, err
		}
		result = append(result, dr)
	}
	return result, rows.Err()
}
