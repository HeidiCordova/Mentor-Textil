package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
)

// PostgresStopRepo opera sobre el schema de una línea concreta (ej: "linea_3").
type PostgresStopRepo struct {
	db     *sql.DB
	schema string // ej: "linea_3"
}

// NewPostgresStopRepo crea el repo apuntando al schema de la línea.
// schema debe ser "linea_{id}", ej: "linea_3".
func NewPostgresStopRepo(db *sql.DB, schema string) ports.StopRepository {
	return &PostgresStopRepo{db: db, schema: schema}
}

// tbl devuelve el nombre calificado de la tabla stops para este schema.
func (r *PostgresStopRepo) tbl() string { return r.schema + ".stops" }

func (r *PostgresStopRepo) Create(ctx context.Context, req domain.CreateStopRequest) (*domain.Stop, error) {
	var query string
	var args []interface{}

	if req.StopID != "" {
		// Cloud-originated stop: use the provided UUID for consistency
		query = fmt.Sprintf(`INSERT INTO %s (stop_id, device_id, stop_type, started_at, ended_at, reason, category, categoria_id, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (stop_id) DO NOTHING
			RETURNING id, stop_id, device_id, stop_type, started_at, ended_at,
				duration_ms, justified, reason, category, categoria_id, justified_by, justified_at,
				source, created_at, updated_at, synced, synced_at`, r.tbl())
		args = []interface{}{req.StopID, req.DeviceID, req.StopType, req.StartedAt, req.EndedAt,
			req.Reason, req.Category, req.CategoriaID, req.Source}
	} else {
		query = fmt.Sprintf(`INSERT INTO %s (device_id, stop_type, started_at, ended_at, reason, category, categoria_id, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, stop_id, device_id, stop_type, started_at, ended_at,
				duration_ms, justified, reason, category, categoria_id, justified_by, justified_at,
				source, created_at, updated_at, synced, synced_at`, r.tbl())
		args = []interface{}{req.DeviceID, req.StopType, req.StartedAt, req.EndedAt,
			req.Reason, req.Category, req.CategoriaID, req.Source}
	}

	stop, err := r.scanStop(r.db.QueryRowContext(ctx, query, args...))
	if err != nil && req.StopID != "" {
		// ON CONFLICT DO NOTHING returns no rows — stop already exists
		existing, err2 := r.GetByID(ctx, req.StopID)
		if err2 == nil {
			return existing, nil
		}
		return nil, err
	}
	return stop, err
}

func (r *PostgresStopRepo) GetByID(ctx context.Context, stopID string) (*domain.Stop, error) {
	query := fmt.Sprintf(`SELECT id, stop_id, device_id, stop_type, started_at, ended_at,
		duration_ms, justified, reason, category, categoria_id, justified_by, justified_at,
		source, created_at, updated_at, synced, synced_at
		FROM %s WHERE stop_id = $1`, r.tbl())

	s, err := r.scanStop(r.db.QueryRowContext(ctx, query, stopID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrStopNotFound
	}
	return s, err
}

func (r *PostgresStopRepo) Update(ctx context.Context, stop *domain.Stop) error {
	query := fmt.Sprintf(`UPDATE %s SET stop_type = $1, reason = $2, category = $3,
		ended_at = $4, justified = $5, justified_by = $6, justified_at = $7, categoria_id = $8
		WHERE stop_id = $9`, r.tbl())

	res, err := r.db.ExecContext(ctx, query,
		stop.StopType, stop.Reason, stop.Category,
		stop.EndedAt, stop.Justified, stop.JustifiedBy, stop.JustifiedAt,
		stop.CategoriaID, stop.StopID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrStopNotFound
	}
	return nil
}

func (r *PostgresStopRepo) Justify(ctx context.Context, req domain.JustifyStopRequest) (*domain.Stop, error) {
	now := time.Now().UTC()
	query := fmt.Sprintf(`UPDATE %s SET justified = true, reason = $1, category = $2,
		justified_by = $3, justified_at = $4, categoria_id = $5,
		synced = false, updated_at = NOW()`, r.tbl())

	args := []interface{}{req.Reason, req.Category, req.JustifiedBy, now, req.CategoriaID}
	paramIdx := 6

	if req.StopType != "" {
		query += `, stop_type = $` + itoa(paramIdx)
		args = append(args, req.StopType)
		paramIdx++
	}

	query += ` WHERE stop_id = $` + itoa(paramIdx)
	args = append(args, req.StopID)
	query += ` RETURNING id, stop_id, device_id, stop_type, started_at, ended_at,
		duration_ms, justified, reason, category, categoria_id, justified_by, justified_at,
		source, created_at, updated_at, synced, synced_at`

	s, err := r.scanStop(r.db.QueryRowContext(ctx, query, args...))
	if err == sql.ErrNoRows {
		return nil, domain.ErrStopNotFound
	}
	return s, err
}

func (r *PostgresStopRepo) List(ctx context.Context, filter domain.StopFilter) ([]domain.Stop, error) {
	query := fmt.Sprintf(`SELECT id, stop_id, device_id, stop_type, started_at, ended_at,
		duration_ms, justified, reason, category, categoria_id, justified_by, justified_at,
		source, created_at, updated_at, synced, synced_at
		FROM %s WHERE 1=1`, r.tbl())
	args := []interface{}{}
	paramIdx := 1

	if filter.DeviceID != "" {
		query += ` AND device_id = $` + itoa(paramIdx)
		args = append(args, filter.DeviceID)
		paramIdx++
	}

	if filter.Justified != nil {
		query += ` AND justified = $` + itoa(paramIdx)
		args = append(args, *filter.Justified)
		paramIdx++
	}
	if filter.Open != nil {
		if *filter.Open {
			query += ` AND ended_at IS NULL`
		} else {
			query += ` AND ended_at IS NOT NULL`
		}
	}
	if filter.StopType != nil {
		query += ` AND stop_type = $` + itoa(paramIdx)
		args = append(args, *filter.StopType)
		paramIdx++
	}
	if filter.Since != nil {
		query += ` AND started_at >= $` + itoa(paramIdx)
		args = append(args, *filter.Since)
		paramIdx++
	}
	if filter.Until != nil {
		query += ` AND started_at <= $` + itoa(paramIdx)
		args = append(args, *filter.Until)
		paramIdx++
	}

	query += ` ORDER BY started_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT $` + itoa(paramIdx)
		args = append(args, filter.Limit)
		paramIdx++
	}
	if filter.Offset > 0 {
		query += ` OFFSET $` + itoa(paramIdx)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stops []domain.Stop
	for rows.Next() {
		s, err := r.scanStopFromRows(rows)
		if err != nil {
			return nil, err
		}
		stops = append(stops, *s)
	}
	return stops, rows.Err()
}

func (r *PostgresStopRepo) CloseStop(ctx context.Context, stopID string, endedAt interface{}) (*domain.Stop, error) {
	var t time.Time
	switch v := endedAt.(type) {
	case time.Time:
		t = v
	default:
		t = time.Now().UTC()
	}

	query := fmt.Sprintf(`UPDATE %s SET ended_at = $1
		WHERE stop_id = $2 AND ended_at IS NULL
		RETURNING id, stop_id, device_id, stop_type, started_at, ended_at,
			duration_ms, justified, reason, category, categoria_id, justified_by, justified_at,
			source, created_at, updated_at, synced, synced_at`, r.tbl())

	s, err := r.scanStop(r.db.QueryRowContext(ctx, query, t, stopID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrStopNotFound
	}
	return s, err
}

func (r *PostgresStopRepo) Delete(ctx context.Context, stopID string) error {
	res, err := r.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE stop_id = $1`, r.tbl()), stopID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrStopNotFound
	}
	return nil
}

func (r *PostgresStopRepo) GetSummary(ctx context.Context, deviceID string, hours int) (*domain.StopSummary, error) {
	query := fmt.Sprintf(`SELECT
		COUNT(*)::BIGINT,
		COUNT(*) FILTER (WHERE ended_at IS NULL)::BIGINT,
		COUNT(*) FILTER (WHERE justified = TRUE)::BIGINT,
		COUNT(*) FILTER (WHERE justified = FALSE AND ended_at IS NOT NULL)::BIGINT,
		COALESCE(SUM(duration_ms) FILTER (WHERE duration_ms IS NOT NULL), 0)::BIGINT
		FROM %s
		WHERE started_at >= (NOW() - make_interval(hours => $1))`, r.tbl())
	args := []interface{}{hours}
	if deviceID != "" {
		query += ` AND device_id = $2`
		args = append(args, deviceID)
	}

	var summary domain.StopSummary
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&summary.TotalStops, &summary.OpenStops, &summary.JustifiedStops,
		&summary.UnjustifiedStops, &summary.TotalDowntimeMs,
	)
	if err != nil {
		return nil, err
	}

	byTypeQuery := fmt.Sprintf(`SELECT stop_type, COUNT(*) FROM %s
		WHERE started_at >= (NOW() - make_interval(hours => $1))
		GROUP BY stop_type`, r.tbl())
	byTypeArgs := []interface{}{hours}
	if deviceID != "" {
		byTypeQuery = fmt.Sprintf(`SELECT stop_type, COUNT(*) FROM %s
		WHERE started_at >= (NOW() - make_interval(hours => $1)) AND device_id = $2
		GROUP BY stop_type`, r.tbl())
		byTypeArgs = append(byTypeArgs, deviceID)
	}
	rows, err := r.db.QueryContext(ctx, byTypeQuery, byTypeArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary.ByType = make(map[string]int64)
	for rows.Next() {
		var st string
		var c int64
		if err := rows.Scan(&st, &c); err != nil {
			return nil, err
		}
		summary.ByType[st] = c
	}
	return &summary, rows.Err()
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func (r *PostgresStopRepo) scanStop(row scanner) (*domain.Stop, error) {
	var s domain.Stop
	var endedAt, justifiedAt, syncedAt sql.NullTime
	var durationMs sql.NullInt32
	var reason, category, justifiedBy sql.NullString
	var categoriaID sql.NullInt32

	err := row.Scan(
		&s.ID, &s.StopID, &s.DeviceID, &s.StopType, &s.StartedAt,
		&endedAt, &durationMs, &s.Justified, &reason, &category,
		&categoriaID, &justifiedBy, &justifiedAt, &s.Source, &s.CreatedAt, &s.UpdatedAt,
		&s.Synced, &syncedAt,
	)
	if err != nil {
		return nil, err
	}

	if endedAt.Valid {
		s.EndedAt = &endedAt.Time
	}
	if durationMs.Valid {
		v := int(durationMs.Int32)
		s.DurationMs = &v
	}
	if reason.Valid {
		s.Reason = &reason.String
	}
	if category.Valid {
		s.Category = &category.String
	}
	if categoriaID.Valid {
		v := int(categoriaID.Int32)
		s.CategoriaID = &v
	}
	if justifiedBy.Valid {
		s.JustifiedBy = &justifiedBy.String
	}
	if justifiedAt.Valid {
		s.JustifiedAt = &justifiedAt.Time
	}
	if syncedAt.Valid {
		s.SyncedAt = &syncedAt.Time
	}
	return &s, nil
}

func (r *PostgresStopRepo) scanStopFromRows(rows *sql.Rows) (*domain.Stop, error) {
	return r.scanStop(rows)
}

// Encode payload for JSONB columns
func encodeJSON(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return fmt.Sprintf("%d", i)
}

func (r *PostgresStopRepo) CloseStaleStops(ctx context.Context, staleAfter time.Duration) (int, error) {
	query := fmt.Sprintf(`
		UPDATE %s
		SET ended_at = NOW(), updated_at = NOW()
		WHERE ended_at IS NULL
		  AND started_at < (NOW() - $1::interval)
	`, r.tbl())
	result, err := r.db.ExecContext(ctx, query, fmt.Sprintf("%.0f seconds", staleAfter.Seconds()))
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

// GetDurationByType returns the sum of stop duration (in ms) per stop_type
// that overlaps with the last intervalS seconds.
// Uses intersection math — LEAST(ended_at, NOW()) - GREATEST(started_at, window_start) —
// so a stop spanning two intervals is split proportionally across both.
// This powers the T_PARADA_PROGRAMADA, T_PARADA_NO_PROGRAMADA, T_REFRIGERIO,
// T_CAPACITACION_OBLIGATORIA, and T_MANTENIMIENTO_PLANIFICADO OEE variables.
func (r *PostgresStopRepo) GetDurationByType(ctx context.Context, deviceID string, intervalS int) (map[string]int64, error) {
	if intervalS <= 0 {
		intervalS = 60
	}
	// Calculate the intersection between each stop's [started_at, ended_at]
	// and the query window [NOW()-interval, NOW()], in milliseconds.
	query := fmt.Sprintf(`
		SELECT stop_type,
		       COALESCE(SUM(
		           EXTRACT(EPOCH FROM (
		               LEAST(ended_at, NOW()) - GREATEST(started_at, NOW() - make_interval(secs => $1))
		           )) * 1000
		       ), 0)::BIGINT
		FROM %s
		WHERE justified = TRUE
		  AND ended_at IS NOT NULL
		  AND started_at < NOW()
		  AND ended_at   > (NOW() - make_interval(secs => $1))
	`, r.tbl())
	args := []interface{}{float64(intervalS)}
	if deviceID != "" {
		query += ` AND device_id = $2`
		args = append(args, deviceID)
	}
	query += ` GROUP BY stop_type`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var st string
		var ms int64
		if err := rows.Scan(&st, &ms); err != nil {
			return nil, err
		}
		result[st] = ms
	}
	return result, rows.Err()
}
