package repository

import (
	"context"
	"fmt"

	"cloud-integration/internal/domain"
	"cloud-integration/internal/ports"

	"mentor.local/shared/multitenancy"
)

type queryRepo struct {
	pr *multitenancy.PoolResolver
}

func NewQueryRepo(pr *multitenancy.PoolResolver) ports.QueryRepository {
	return &queryRepo{pr: pr}
}

func (r *queryRepo) LatestSnapshot(ctx context.Context, lineaID int, empresaID int) (*domain.OEESnapshot, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	oee := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	runs := multitenancy.Tbl(schema, "analytics.production_runs")

	row := pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT s.id, s.device_id, s.linea_id, s.planta_id, s.empresa_id,
		       COALESCE(s.turno,''), TO_CHAR(s.fecha,'YYYY-MM-DD'),
		       s.hora, COALESCE(s.disponibilidad,0), COALESCE(s.rendimiento,0),
		       COALESCE(s.calidad,0), COALESCE(s.oee,0),
		       COALESCE(s.produccion,0), COALESCE(s.interval_s,0),
		       COALESCE(pr.nombre,''), COALESCE(pr.sku,'')
		FROM %s s
		LEFT JOIN LATERAL (
		  SELECT nombre, sku FROM %s
		  WHERE device_id = s.device_id
		    AND s.hora >= started_at
		    AND (ended_at IS NULL OR s.hora <= ended_at)
		  ORDER BY started_at DESC LIMIT 1
		) pr ON true
		WHERE s.linea_id = $1 AND s.empresa_id = $2 AND s.interval_s >= 300
		ORDER BY s.hora DESC LIMIT 1`, oee, runs), lineaID, empresaID)

	s := &domain.OEESnapshot{}
	err = row.Scan(
		&s.ID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
		&s.Turno, &s.Fecha, &s.Hora,
		&s.Disponibilidad, &s.Rendimiento, &s.Calidad, &s.OEE,
		&s.Produccion, &s.IntervalS, &s.RunNombre, &s.RunSKU,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *queryRepo) ListSnapshots(ctx context.Context, lineaID int, empresaID int, limit int) ([]domain.OEESnapshot, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	oee := multitenancy.Tbl(schema, "ingest.oee_snapshots")
	runs := multitenancy.Tbl(schema, "analytics.production_runs")

	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT s.id, s.device_id, s.linea_id, s.planta_id, s.empresa_id,
		       COALESCE(s.turno,''), TO_CHAR(s.fecha,'YYYY-MM-DD'),
		       s.hora, COALESCE(s.disponibilidad,0), COALESCE(s.rendimiento,0),
		       COALESCE(s.calidad,0), COALESCE(s.oee,0),
		       COALESCE(s.produccion,0), COALESCE(s.interval_s,0),
		       COALESCE(pr.nombre,''), COALESCE(pr.sku,'')
		FROM %s s
		LEFT JOIN LATERAL (
		  SELECT nombre, sku FROM %s
		  WHERE device_id = s.device_id
		    AND s.hora >= started_at
		    AND (ended_at IS NULL OR s.hora <= ended_at)
		  ORDER BY started_at DESC LIMIT 1
		) pr ON true
		WHERE s.linea_id = $1 AND s.empresa_id = $2 AND s.interval_s >= 300
		ORDER BY s.hora DESC LIMIT %d`, oee, runs, limit), lineaID, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.OEESnapshot
	for rows.Next() {
		s := domain.OEESnapshot{}
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
			&s.Turno, &s.Fecha, &s.Hora,
			&s.Disponibilidad, &s.Rendimiento, &s.Calidad, &s.OEE,
			&s.Produccion, &s.IntervalS, &s.RunNombre, &s.RunSKU,
		); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func (r *queryRepo) ListParadas(ctx context.Context, lineaID int, empresaID int, desde, hasta string, limit int) ([]domain.Parada, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	args := []any{lineaID, empresaID}
	idx := 3
	extra := ""
	if desde != "" {
		extra += fmt.Sprintf(" AND inicio >= $%d::timestamptz", idx)
		args = append(args, desde)
		idx++
	}
	if hasta != "" {
		extra += fmt.Sprintf(" AND inicio <= $%d::timestamptz", idx)
		args = append(args, hasta)
	}

	q := fmt.Sprintf(`
		SELECT id, device_id, linea_id, planta_id, empresa_id,
		       stop_type, inicio, fin, duracion_min,
		       COALESCE(justified, false), reason, categoria_nombre, source
		FROM %s
		WHERE linea_id = $1 AND empresa_id = $2%s
		ORDER BY inicio DESC LIMIT %d`, tbl, extra, limit)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Parada
	for rows.Next() {
		p := domain.Parada{}
		if err := rows.Scan(
			&p.ID, &p.DeviceID, &p.LineaID, &p.PlantaID, &p.EmpresaID,
			&p.StopType, &p.Inicio, &p.Fin, &p.DuracionMin,
			&p.Justified, &p.Reason, &p.CategoriaNombre, &p.Source,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}

// ListEnergy devuelve snapshots de energía paginados desde energy.snapshots.
// Soporta filtrado por device_id, meter_id y rango temporal.
// limit máx 2000 para compatibilidad con Power BI (carga por lotes).
func (r *queryRepo) ListEnergy(ctx context.Context, empresaID int, deviceID, meterID, desde, hasta string, limit, offset int) ([]domain.EnergySnapshot, int, error) {
	pool, _, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 2000 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	args := []any{empresaID}
	idx := 2
	where := ""

	if deviceID != "" {
		where += fmt.Sprintf(" AND device_id = $%d", idx)
		args = append(args, deviceID)
		idx++
	}
	if meterID != "" {
		where += fmt.Sprintf(" AND meter_id = $%d", idx)
		args = append(args, meterID)
		idx++
	}
	if desde != "" {
		where += fmt.Sprintf(" AND hora >= $%d::timestamptz", idx)
		args = append(args, desde)
		idx++
	}
	if hasta != "" {
		where += fmt.Sprintf(" AND hora <= $%d::timestamptz", idx)
		args = append(args, hasta)
		idx++
	}

	// Contar total para paginación
	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM energy.snapshots WHERE empresa_id = $1%s`, where)
	if err := pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Construir args para query con LIMIT/OFFSET
	argsPage := append(args, limit, offset)
	dataQ := fmt.Sprintf(`
		SELECT id, device_id, meter_id, hora, interval_s, empresa_id, planta_id,
		       corriente_a, corriente_b, corriente_c, corriente_avg,
		       voltaje_a, voltaje_b, voltaje_c, voltaje_avg,
		       voltaje_ab, voltaje_bc, voltaje_ac,
		       potencia_activa, potencia_reactiva, potencia_aparente,
		       factor_potencia, frecuencia_hz,
		       energia_activa, energia_reactiva, energia_aparente,
		       thd_ia, thd_ib, thd_ic, thd_ua, thd_ub, thd_uc
		FROM energy.snapshots
		WHERE empresa_id = $1%s
		ORDER BY hora DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1)

	rows, err := pool.Query(ctx, dataQ, argsPage...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []domain.EnergySnapshot
	for rows.Next() {
		s := domain.EnergySnapshot{}
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.MeterID, &s.Hora, &s.IntervalS,
			&s.EmpresaID, &s.PlantaID,
			&s.CorrienteA, &s.CorrienteB, &s.CorrienteC, &s.CorrienteAvg,
			&s.VoltajeA, &s.VoltajeB, &s.VoltajeC, &s.VoltajeAvg,
			&s.VoltajeAB, &s.VoltajeBC, &s.VoltajeAC,
			&s.PotenciaActiva, &s.PotenciaReactiva, &s.PotenciaAparente,
			&s.FactorPotencia, &s.FrecuenciaHz,
			&s.EnergiaActiva, &s.EnergiaReactiva, &s.EnergiaAparente,
			&s.THDIA, &s.THDIB, &s.THDIC, &s.THDUA, &s.THDUB, &s.THDUC,
		); err != nil {
			return nil, 0, err
		}
		result = append(result, s)
	}
	return result, total, nil
}
