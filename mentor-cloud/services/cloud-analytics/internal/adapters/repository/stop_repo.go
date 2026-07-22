package repository

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"mentor.local/shared/multitenancy"
)

type stopRepo struct {
	pr *multitenancy.PoolResolver
}

func NewStopRepository(pr *multitenancy.PoolResolver) ports.StopRepository {
	return &stopRepo{pr: pr}
}

func (r *stopRepo) ListStops(ctx context.Context, f ports.StopFilter) ([]domain.CloudStop, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	var conditions []string
	var args []any
	idx := 1

	if f.DeviceID != "" {
		conditions = append(conditions, fmt.Sprintf("p.device_id = $%d", idx))
		args = append(args, f.DeviceID)
		idx++
	}
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
	if f.Justified != nil {
		conditions = append(conditions, fmt.Sprintf("p.justified = $%d", idx))
		args = append(args, *f.Justified)
		idx++
	}
	if f.Open != nil {
		if *f.Open {
			conditions = append(conditions, "p.fin IS NULL")
		} else {
			conditions = append(conditions, "p.fin IS NOT NULL")
		}
	}
	if f.StopType != nil {
		conditions = append(conditions, fmt.Sprintf("p.stop_type = $%d", idx))
		args = append(args, *f.StopType)
		idx++
	}
	if f.Since != nil {
		conditions = append(conditions, fmt.Sprintf("p.inicio >= $%d", idx))
		args = append(args, *f.Since)
		idx++
	}
	if f.Until != nil {
		conditions = append(conditions, fmt.Sprintf("p.inicio <= $%d", idx))
		args = append(args, *f.Until)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	limit := 200
	if f.Limit > 0 && f.Limit <= 1000 {
		limit = f.Limit
	}

	q := fmt.Sprintf(`
		SELECT p.id, p.stop_id, p.device_id, p.linea_id, p.planta_id, p.empresa_id,
		       p.categoria_id, p.stop_type, p.inicio, p.fin, p.duracion_min,
		       COALESCE(p.justified, false), p.reason, p.categoria_nombre,
		       p.justified_by, p.justified_at, p.source,
		       p.subcategoria_nombre, p.subcategoria_2_nombre, p.creado_en
		FROM %s p
		%s
		ORDER BY p.inicio DESC
		LIMIT %d`, tbl, where, limit)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.CloudStop
	for rows.Next() {
		var s domain.CloudStop
		if err := rows.Scan(
			&s.ID, &s.StopID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
			&s.CategoriaID, &s.StopType, &s.Inicio, &s.Fin, &s.DuracionMin,
			&s.Justified, &s.Reason, &s.CategoriaNombre,
			&s.JustifiedBy, &s.JustifiedAt, &s.Source,
			&s.SubcategoriaNombre, &s.Subcategoria2Nombre, &s.CreadoEn,
		); err != nil {
			return nil, err
		}
		s.Category = s.CategoriaNombre
		result = append(result, s)
	}
	if result == nil {
		result = []domain.CloudStop{}
	}
	return result, nil
}

func (r *stopRepo) StopSummary(ctx context.Context, f ports.StopFilter) (*domain.StopSummary, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	var conditions []string
	var args []any
	idx := 1

	if f.EmpresaID != nil {
		conditions = append(conditions, fmt.Sprintf("empresa_id = $%d", idx))
		args = append(args, *f.EmpresaID)
		idx++
	}
	if f.PlantaID != nil {
		conditions = append(conditions, fmt.Sprintf("planta_id = $%d", idx))
		args = append(args, *f.PlantaID)
		idx++
	}
	if f.LineaID != nil {
		conditions = append(conditions, fmt.Sprintf("linea_id = $%d", idx))
		args = append(args, *f.LineaID)
		idx++
	}
	if f.Since != nil {
		conditions = append(conditions, fmt.Sprintf("inicio >= $%d", idx))
		args = append(args, *f.Since)
		idx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	summary := &domain.StopSummary{ByType: make(map[string]int)}

	q := fmt.Sprintf(`
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE fin IS NULL) AS open_stops,
			COUNT(*) FILTER (WHERE justified = true) AS justified,
			COUNT(*) FILTER (WHERE justified = false OR justified IS NULL) AS unjustified,
			COALESCE(SUM(duracion_min * 60000)::bigint, 0) AS downtime_ms
		FROM %s %s`, tbl, where)

	if err := pool.QueryRow(ctx, q, args...).Scan(
		&summary.TotalStops, &summary.OpenStops,
		&summary.JustifiedStops, &summary.UnjustifiedStops,
		&summary.TotalDowntimeMS,
	); err != nil {
		return nil, err
	}

	q2 := fmt.Sprintf(`
		SELECT COALESCE(stop_type, 'UNKNOWN'), COUNT(*)::int
		FROM %s %s
		GROUP BY stop_type`, tbl, where)

	rows, err := pool.Query(ctx, q2, args...)
	if err != nil {
		return summary, nil
	}
	defer rows.Close()

	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err == nil {
			summary.ByType[typ] = count
		}
	}

	return summary, nil
}

func (r *stopRepo) CreateStop(ctx context.Context, req *domain.CreateCloudStopRequest) (*domain.CloudStop, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	stopID := uuid.New().String()
	now := time.Now()
	source := req.Source
	if source == "" {
		source = "cloud"
	}

	var id int64
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		INSERT INTO %s
			(device_id, linea_id, planta_id, empresa_id, categoria_id,
			 descripcion, inicio, creado_en,
			 categoria_nombre, stop_id, stop_type, justified, source, reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::uuid,$11,false,$12,$13)
		RETURNING id`, tbl),
		req.DeviceID, req.LineaID, req.PlantaID, req.EmpresaID,
		req.CategoriaID, req.Reason, now, now,
		req.Category, stopID, req.StopType, source, req.Reason,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return &domain.CloudStop{
		ID:              id,
		StopID:          &stopID,
		DeviceID:        req.DeviceID,
		LineaID:         req.LineaID,
		PlantaID:        req.PlantaID,
		EmpresaID:       &req.EmpresaID,
		CategoriaID:     req.CategoriaID,
		StopType:        &req.StopType,
		Inicio:          now,
		Justified:       false,
		Reason:          req.Reason,
		Category:        req.Category,
		CategoriaNombre: req.Category,
		Source:          &source,
		CreadoEn:        now,
	}, nil
}

func (r *stopRepo) UpdateStop(ctx context.Context, stopID string, req *domain.UpdateCloudStopRequest) (*domain.CloudStop, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	var setClauses []string
	var args []any
	idx := 1

	if req.StopType != nil {
		setClauses = append(setClauses, fmt.Sprintf("stop_type = $%d", idx))
		args = append(args, *req.StopType)
		idx++
	}
	if req.Category != nil {
		setClauses = append(setClauses, fmt.Sprintf("categoria_nombre = $%d", idx))
		args = append(args, *req.Category)
		idx++
	}
	if req.CategoriaID != nil {
		setClauses = append(setClauses, fmt.Sprintf("categoria_id = $%d", idx))
		args = append(args, *req.CategoriaID)
		idx++
	}
	if req.Reason != nil {
		setClauses = append(setClauses, fmt.Sprintf("reason = $%d", idx))
		args = append(args, *req.Reason)
		idx++
	}
	if req.StartedAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("inicio = $%d", idx))
		args = append(args, *req.StartedAt)
		idx++
	}
	if req.EndedAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("fin = $%d", idx))
		args = append(args, *req.EndedAt)
		idx++
		setClauses = append(setClauses, fmt.Sprintf(
			"duracion_min = EXTRACT(EPOCH FROM ($%d - COALESCE(inicio, NOW()))) / 60.0", idx-1,
		))
	}

	if len(setClauses) == 0 {
		return r.GetStop(ctx, stopID)
	}

	args = append(args, stopID)
	q := fmt.Sprintf(
		`UPDATE %s SET %s WHERE stop_id = $%d::uuid`,
		tbl, strings.Join(setClauses, ", "), idx,
	)
	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("parada no encontrada: %s", stopID)
	}

	return r.GetStop(ctx, stopID)
}

func (r *stopRepo) JustifyStop(ctx context.Context, stopID string, req *domain.JustifyCloudStopRequest) (*domain.CloudStop, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")
	now := time.Now()

	tag, err := pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET justified = true,
		    justified_by = $1,
		    justified_at = $2,
		    reason = $3,
		    categoria_nombre = CASE WHEN $4 = '' THEN categoria_nombre ELSE $4 END,
		    stop_type = CASE WHEN $5 = '' THEN stop_type ELSE $5 END,
		    categoria_id = COALESCE($6, categoria_id)
		WHERE stop_id = $7::uuid`, tbl),
		req.JustifiedBy, now, req.Reason, req.Category, req.StopType, req.CategoriaID, stopID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("parada no encontrada: %s", stopID)
	}

	var s domain.CloudStop
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT p.id, p.stop_id, p.device_id, p.linea_id, p.planta_id, p.empresa_id,
		       p.categoria_id, p.stop_type, p.inicio, p.fin, p.duracion_min,
		       COALESCE(p.justified, false), p.reason, p.categoria_nombre,
		       p.justified_by, p.justified_at, p.source,
		       p.subcategoria_nombre, p.subcategoria_2_nombre, p.creado_en
		FROM %s p
		WHERE p.stop_id = $1::uuid`, tbl), stopID,
	).Scan(
		&s.ID, &s.StopID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
		&s.CategoriaID, &s.StopType, &s.Inicio, &s.Fin, &s.DuracionMin,
		&s.Justified, &s.Reason, &s.CategoriaNombre,
		&s.JustifiedBy, &s.JustifiedAt, &s.Source,
		&s.SubcategoriaNombre, &s.Subcategoria2Nombre, &s.CreadoEn,
	)
	if err != nil {
		return nil, err
	}
	s.Category = s.CategoriaNombre

	// Encolar pending_command para entrega confiable al edge en el próximo ciclo
	// del Enviador, por si la conexión SSE estaba caída durante la justificación.
	if s.DeviceID != "" {
		pcTbl := multitenancy.Tbl(schema, "analytics.pending_commands")
		pcPayload, _ := json.Marshal(map[string]interface{}{
			"stop_id":      stopID,
			"reason":       req.Reason,
			"category":     req.Category,
			"categoria_id": req.CategoriaID,
			"stop_type":    req.StopType,
			"justified_by": req.JustifiedBy,
		})
		if _, err2 := pool.Exec(ctx, fmt.Sprintf(`
			INSERT INTO %s (device_id, command, payload)
			VALUES ($1, 'justificar_parada', $2::jsonb)
		`, pcTbl), s.DeviceID, string(pcPayload)); err2 != nil {
			log.Printf("[stop_repo] warn: no se pudo encolar pending_command para %s: %v", stopID, err2)
		}
	}

	return &s, nil
}

func (r *stopRepo) GetStop(ctx context.Context, stopID string) (*domain.CloudStop, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	var s domain.CloudStop
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT p.id, p.stop_id, p.device_id, p.linea_id, p.planta_id, p.empresa_id,
		       p.categoria_id, p.stop_type, p.inicio, p.fin, p.duracion_min,
		       COALESCE(p.justified, false), p.reason, p.categoria_nombre,
		       p.justified_by, p.justified_at, p.source,
		       p.subcategoria_nombre, p.subcategoria_2_nombre, p.creado_en
		FROM %s p WHERE p.stop_id = $1::uuid`, tbl), stopID,
	).Scan(
		&s.ID, &s.StopID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
		&s.CategoriaID, &s.StopType, &s.Inicio, &s.Fin, &s.DuracionMin,
		&s.Justified, &s.Reason, &s.CategoriaNombre,
		&s.JustifiedBy, &s.JustifiedAt, &s.Source,
		&s.SubcategoriaNombre, &s.Subcategoria2Nombre, &s.CreadoEn,
	)
	if err != nil {
		return nil, err
	}
	s.Category = s.CategoriaNombre
	return &s, nil
}

func (r *stopRepo) CloseStop(ctx context.Context, stopID string, endedAt time.Time) (*domain.CloudStop, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")

	_, err = pool.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET fin = $1, duracion_min = EXTRACT(EPOCH FROM ($1 - inicio)) / 60.0
		WHERE stop_id = $2::uuid`, tbl), endedAt, stopID)
	if err != nil {
		return nil, err
	}

	var s domain.CloudStop
	err = pool.QueryRow(ctx, fmt.Sprintf(`
		SELECT p.id, p.stop_id, p.device_id, p.linea_id, p.planta_id, p.empresa_id,
		       p.categoria_id, p.stop_type, p.inicio, p.fin, p.duracion_min,
		       COALESCE(p.justified, false), p.reason, p.categoria_nombre,
		       p.justified_by, p.justified_at, p.source,
		       p.subcategoria_nombre, p.subcategoria_2_nombre, p.creado_en
		FROM %s p WHERE p.stop_id = $1::uuid`, tbl), stopID,
	).Scan(
		&s.ID, &s.StopID, &s.DeviceID, &s.LineaID, &s.PlantaID, &s.EmpresaID,
		&s.CategoriaID, &s.StopType, &s.Inicio, &s.Fin, &s.DuracionMin,
		&s.Justified, &s.Reason, &s.CategoriaNombre,
		&s.JustifiedBy, &s.JustifiedAt, &s.Source,
		&s.SubcategoriaNombre, &s.Subcategoria2Nombre, &s.CreadoEn,
	)
	return &s, err
}

func (r *stopRepo) DeleteStop(ctx context.Context, stopID string) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.paradas")
	tag, err := pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE stop_id = $1::uuid`, tbl), stopID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("parada no encontrada: %s", stopID)
	}
	return nil
}

func (r *stopRepo) ResolveDeviceID(ctx context.Context, lineaID int) (string, error) {
	pool := r.pr.Master()
	var deviceID string
	err := pool.QueryRow(ctx,
		`SELECT device_id FROM config.dispositivos WHERE linea_id = $1 AND activo = true LIMIT 1`,
		lineaID,
	).Scan(&deviceID)
	return deviceID, err
}
