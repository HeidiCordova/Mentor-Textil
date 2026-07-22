package repository

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"fmt"

	"mentor.local/shared/multitenancy"
)

type OEERepository struct {
	pr *multitenancy.PoolResolver
}

func NewOEERepository(pr *multitenancy.PoolResolver) *OEERepository {
	return &OEERepository{pr: pr}
}

func minIntervalS(v int) int {
	if v > 0 {
		return v
	}
	return 300
}

func (r *OEERepository) GetSnapshots(ctx context.Context, f ports.OEEFilter) ([]domain.OEESnapshot, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tblSnap := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	tblRuns := multitenancy.Tbl(schema, "analytics.production_runs")

	q := fmt.Sprintf(`SELECT s.id, s.device_id, s.linea_id, s.planta_id, s.empresa_id,
	             COALESCE(s.turno,''), TO_CHAR(s.fecha,'YYYY-MM-DD'), s.hora,
	             COALESCE(s.disponibilidad,0), COALESCE(s.rendimiento,0),
	             COALESCE(s.calidad,0), COALESCE(s.oee,0),
	             COALESCE(s.produccion,0), COALESCE(s.energia_kwh,0),
	             COALESCE(s.interval_s,0), COALESCE(s.head,'{}'), COALESCE(s.data,'{}'),
	             COALESCE(pr.nombre,''), COALESCE(pr.sku,'')
	      FROM %s s
	      LEFT JOIN LATERAL (
	        SELECT nombre, sku FROM %s
	        WHERE device_id = s.device_id
	          AND s.hora >= started_at
	          AND (ended_at IS NULL OR s.hora <= ended_at)
	        ORDER BY started_at DESC LIMIT 1
	      ) pr ON true
		      WHERE s.interval_s >= %d`, tblSnap, tblRuns, minIntervalS(f.MinIntervalS))
	args := []interface{}{}
	idx := 1

	if f.EmpresaID != nil {
		q += fmt.Sprintf(` AND s.empresa_id = $%d`, idx)
		args = append(args, *f.EmpresaID)
		idx++
	}
	if f.PlantaID != nil {
		q += fmt.Sprintf(` AND s.planta_id = $%d`, idx)
		args = append(args, *f.PlantaID)
		idx++
	}
	if f.LineaID != nil {
		q += fmt.Sprintf(` AND s.linea_id = $%d`, idx)
		args = append(args, *f.LineaID)
		idx++
	}
	if f.DeviceID != "" {
		q += fmt.Sprintf(` AND s.device_id = $%d`, idx)
		args = append(args, f.DeviceID)
		idx++
	}
	if f.From != "" {
		q += fmt.Sprintf(` AND s.hora >= $%d::timestamptz`, idx)
		args = append(args, f.From)
		idx++
	}
	if f.To != "" {
		q += fmt.Sprintf(` AND s.hora <= $%d::timestamptz`, idx)
		args = append(args, f.To)
		idx++
	}

	q += ` ORDER BY s.hora DESC`
	limit := f.Limit
	if limit <= 0 {
		limit = 500
	}
	q += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.OEESnapshot
	for rows.Next() {
		var s domain.OEESnapshot
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
			&s.Turno, &s.Fecha, &s.Hora,
			&s.Disponibilidad, &s.Rendimiento, &s.Calidad, &s.OEE,
			&s.Produccion, &s.EnergiaKwh, &s.IntervalS, &s.Head, &s.Data,
			&s.RunNombre, &s.RunSKU,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *OEERepository) GetSummary(ctx context.Context, f ports.OEEFilter) (*domain.OEESummary, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "ingest.oee_snapshots")

	q := fmt.Sprintf(`SELECT COALESCE(AVG(disponibilidad),0), COALESCE(AVG(rendimiento),0),
	             COALESCE(AVG(calidad),0), COALESCE(AVG(oee),0),
	             COALESCE(SUM(produccion),0), COUNT(*)
	      FROM %s WHERE interval_s >= %d`, tbl, minIntervalS(f.MinIntervalS))
	args := []interface{}{}
	idx := 1

	if f.EmpresaID != nil {
		q += fmt.Sprintf(` AND empresa_id = $%d`, idx)
		args = append(args, *f.EmpresaID)
		idx++
	}
	if f.PlantaID != nil {
		q += fmt.Sprintf(` AND planta_id = $%d`, idx)
		args = append(args, *f.PlantaID)
		idx++
	}
	if f.LineaID != nil {
		q += fmt.Sprintf(` AND linea_id = $%d`, idx)
		args = append(args, *f.LineaID)
		idx++
	}
	if f.From != "" {
		q += fmt.Sprintf(` AND hora >= $%d::timestamptz`, idx)
		args = append(args, f.From)
		idx++
	}
	if f.To != "" {
		q += fmt.Sprintf(` AND hora <= $%d::timestamptz`, idx)
		args = append(args, f.To)
		idx++
	}

	s := &domain.OEESummary{}
	err = pool.QueryRow(ctx, q, args...).Scan(
		&s.Disponibilidad, &s.Rendimiento, &s.Calidad, &s.OEE,
		&s.ProduccionTotal, &s.Snapshots,
	)
	return s, err
}

func (r *OEERepository) GetLatest(ctx context.Context, lineaID int) (*domain.OEESnapshot, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tblSnap := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	tblRuns := multitenancy.Tbl(schema, "analytics.production_runs")

	var s domain.OEESnapshot
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT s.id, s.device_id, s.linea_id, s.planta_id, s.empresa_id,
		       COALESCE(s.turno,''), TO_CHAR(s.fecha,'YYYY-MM-DD'), s.hora,
		       COALESCE(s.disponibilidad,0), COALESCE(s.rendimiento,0),
		       COALESCE(s.calidad,0), COALESCE(s.oee,0),
		       COALESCE(s.produccion,0), COALESCE(s.energia_kwh,0),
		       COALESCE(s.interval_s,0), COALESCE(s.head,'{}'), COALESCE(s.data,'{}'),
		       COALESCE(pr.nombre,''), COALESCE(pr.sku,'')
		FROM %s s
		LEFT JOIN LATERAL (
		  SELECT nombre, sku FROM %s
		  WHERE device_id = s.device_id
		    AND s.hora >= started_at
		    AND (ended_at IS NULL OR s.hora <= ended_at)
		  ORDER BY started_at DESC LIMIT 1
		) pr ON true
		WHERE s.linea_id = $1 AND s.interval_s >= 300
		ORDER BY s.hora DESC LIMIT 1`, tblSnap, tblRuns), lineaID,
	).Scan(
		&s.ID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
		&s.Turno, &s.Fecha, &s.Hora,
		&s.Disponibilidad, &s.Rendimiento, &s.Calidad, &s.OEE,
		&s.Produccion, &s.EnergiaKwh, &s.IntervalS, &s.Head, &s.Data,
		&s.RunNombre, &s.RunSKU,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
