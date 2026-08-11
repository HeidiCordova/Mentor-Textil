package adapters

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"resiliencia/internal/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// pgTextArray wraps []string para encoding como array de texto PostgreSQL en database/sql.
type pgTextArray []string

func (a pgTextArray) Value() (driver.Value, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, v := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		for _, c := range v {
			if c == '\\' || c == '"' {
				b.WriteByte('\\')
			}
			b.WriteRune(c)
		}
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String(), nil
}

type PostgresRepo struct {
	db     *sql.DB
	schema string // ej: "linea_3"
}

func NewPostgresRepo(connStr, lineSchema string) (*PostgresRepo, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresRepo{db: db, schema: lineSchema}, nil
}

func (r *PostgresRepo) DB() *sql.DB { return r.db }

func (r *PostgresRepo) SetSchema(s string) { r.schema = s }

func (r *PostgresRepo) evtbl() string { return `"` + r.schema + `".events_buffer` }

func (r *PostgresRepo) vdtbl() string {
	return `"` + strings.ReplaceAll(r.schema, `"`, `""`) + `"."vision_detections"`
}

func (r *PostgresRepo) vctbl() string {
	return `"` + strings.ReplaceAll(r.schema, `"`, `""`) + `"."vision_counter_state"`
}

func (r *PostgresRepo) vcstbl() string {
	return `"` + strings.ReplaceAll(r.schema, `"`, `""`) + `"."vision_counter_snapshots"`
}

func (r *PostgresRepo) Store(ctx context.Context, event *domain.EventBuffer) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (event_id, device_id, event_type, timestamp, payload, synced, retry_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (event_id) DO NOTHING
	`, r.evtbl())

	_, err := r.db.ExecContext(ctx, query,
		event.EventID,
		event.DeviceID,
		event.EventType,
		event.Timestamp,
		event.Payload,
		event.Synced,
		event.RetryCount,
	)

	return err
}

func (r *PostgresRepo) GetUnsyncedEvents(ctx context.Context, limit int) ([]*domain.EventBuffer, error) {
	query := fmt.Sprintf(`
		SELECT id, event_id, device_id, event_type, timestamp, payload, synced, retry_count, created_at
		FROM %s
		WHERE synced = false AND dead = false
		ORDER BY timestamp ASC
		LIMIT $1
	`, r.evtbl())

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*domain.EventBuffer, 0)
	for rows.Next() {
		event := &domain.EventBuffer{}
		err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.DeviceID,
			&event.EventType,
			&event.Timestamp,
			&event.Payload,
			&event.Synced,
			&event.RetryCount,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresRepo) MarkSynced(ctx context.Context, eventIDs []string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET synced = true, synced_at = $1
		WHERE event_id = ANY($2)
	`, r.evtbl())

	_, err := r.db.ExecContext(ctx, query, time.Now(), pgTextArray(eventIDs))
	return err
}

func (r *PostgresRepo) GetRecentEvents(ctx context.Context, limit int, since *time.Time) ([]*domain.EventBuffer, error) {
	var rows *sql.Rows
	var err error
	if since != nil {
		rows, err = r.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT id, event_id, device_id, event_type, timestamp, payload, synced, retry_count, created_at
			FROM %s
			WHERE timestamp >= $2
			ORDER BY timestamp DESC
			LIMIT $1`, r.evtbl()), limit, *since)
	} else {
		rows, err = r.db.QueryContext(ctx, fmt.Sprintf(`
			SELECT id, event_id, device_id, event_type, timestamp, payload, synced, retry_count, created_at
			FROM %s
			ORDER BY timestamp DESC
			LIMIT $1`, r.evtbl()), limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]*domain.EventBuffer, 0, limit)
	for rows.Next() {
		event := &domain.EventBuffer{}
		if err := rows.Scan(&event.ID, &event.EventID, &event.DeviceID, &event.EventType,
			&event.Timestamp, &event.Payload, &event.Synced, &event.RetryCount, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *PostgresRepo) GetVisionCount(
	ctx context.Context,
	since time.Time,
	until time.Time,
) (*domain.VisionCount, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(DISTINCT detection_id), CURRENT_TIMESTAMP
		FROM %s
		WHERE detected_at >= $1
		  AND detected_at < $2
	`, r.vdtbl())

	result := &domain.VisionCount{
		Since:     since.UTC(),
		Until:     until.UTC(),
		EventType: "CORTE",
	}
	if err := r.db.QueryRowContext(ctx, query, since, until).Scan(&result.Count, &result.AsOf); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepo) GetVisionCounter(
	ctx context.Context,
	until time.Time,
) (*domain.VisionCounter, error) {
	result := &domain.VisionCounter{
		Until:     until.UTC(),
		EventType: "CORTE",
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Locking the single state row serializes first-time boundary finalization
	// with counter increments and with other boundary requests. Each later
	// statement in this READ COMMITTED transaction sees everything committed
	// before the lock was acquired.
	var activeEpoch time.Time
	if err := tx.QueryRowContext(
		ctx,
		fmt.Sprintf(`
			SELECT counter_epoch
			FROM %s
			WHERE counter_name = 'CORTE_TOTAL'
			FOR UPDATE
		`, r.vctbl()),
	).Scan(&activeEpoch); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrVisionCounterBoundary
		}
		return nil, err
	}
	activeEpoch = activeEpoch.UTC()
	if until.Before(activeEpoch) {
		return nil, domain.ErrVisionCounterBoundary
	}

	// A retry always returns the first frozen answer, even when newer
	// boundaries already exist.
	existingQuery := fmt.Sprintf(`
		SELECT counter_epoch, counter_value, created_at, state_updated_at
		FROM %s
		WHERE counter_name = 'CORTE_TOTAL'
		  AND counter_epoch = $1
		  AND counter_until = $2
	`, r.vcstbl())
	err = tx.QueryRowContext(ctx, existingQuery, activeEpoch, until).Scan(
		&result.CounterEpoch,
		&result.Count,
		&result.AsOf,
		&result.StateUpdatedAt,
	)
	if err == nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Never create a boundary older than one already frozen for this epoch.
	// This preserves monotonic snapshots while still allowing exact retries.
	var latestUntil sql.NullTime
	if err := tx.QueryRowContext(
		ctx,
		fmt.Sprintf(`
			SELECT MAX(counter_until)
			FROM %s
			WHERE counter_name = 'CORTE_TOTAL'
			  AND counter_epoch = $1
		`, r.vcstbl()),
		activeEpoch,
	).Scan(&latestUntil); err != nil {
		return nil, err
	}
	if latestUntil.Valid && until.Before(latestUntil.Time.UTC()) {
		return nil, domain.ErrVisionCounterBoundary
	}

	// State and vision_detections are read in one MVCC statement. The state
	// lock prevents a concurrent CORTE insert from committing between this
	// calculation and the frozen snapshot row.
	freezeQuery := fmt.Sprintf(`
		INSERT INTO %s AS stored (
			counter_name,
			counter_epoch,
			counter_until,
			counter_value,
			state_updated_at
		)
		SELECT
			s.counter_name,
			s.counter_epoch,
			$1::timestamptz,
			s.counter_value - (
				SELECT COUNT(*)
				FROM %s d
				WHERE d.detected_at >= $1
				  AND d.detected_at >= s.counter_epoch
			),
			s.updated_at
		FROM %s s
		WHERE s.counter_name = 'CORTE_TOTAL'
		  AND $1 >= s.counter_epoch
		  AND $1 <= CURRENT_TIMESTAMP
		ON CONFLICT (counter_name, counter_epoch, counter_until)
		DO UPDATE SET counter_value = stored.counter_value
		RETURNING counter_epoch, counter_value, created_at, state_updated_at
	`, r.vcstbl(), r.vdtbl(), r.vctbl())
	if err := tx.QueryRowContext(ctx, freezeQuery, until).Scan(
		&result.CounterEpoch,
		&result.Count,
		&result.AsOf,
		&result.StateUpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrVisionCounterBoundary
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepo) GetPendingEvents(ctx context.Context, limit int) ([]*domain.EventBuffer, error) {
	query := fmt.Sprintf(`
		SELECT id, event_id, device_id, event_type, timestamp, payload, synced, retry_count, created_at
		FROM %s
		WHERE synced = false AND dead = false
		ORDER BY timestamp ASC
		LIMIT $1
	`, r.evtbl())
	return r.scanEvents(ctx, query, limit)
}

func (r *PostgresRepo) scanEvents(ctx context.Context, query string, limit int) ([]*domain.EventBuffer, error) {
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*domain.EventBuffer, 0, limit)
	for rows.Next() {
		event := &domain.EventBuffer{}
		if err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.DeviceID,
			&event.EventType,
			&event.Timestamp,
			&event.Payload,
			&event.Synced,
			&event.RetryCount,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *PostgresRepo) EventExists(ctx context.Context, eventID string) (bool, error) {
	query := fmt.Sprintf(`SELECT EXISTS(SELECT 1 FROM %s WHERE event_id = $1)`, r.evtbl())

	var exists bool
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(&exists)
	return exists, err
}

func (r *PostgresRepo) GetPendingCount(ctx context.Context) (int, error) {
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE synced = false AND dead = false`, r.evtbl())

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

func (r *PostgresRepo) PurgeExpired(ctx context.Context) (int, error) {
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE expires_at < NOW()
		  AND (synced = true OR dead = true)
	`, r.evtbl())
	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func (r *PostgresRepo) MarkStaleEventsDead(ctx context.Context, maxAgeHours int) (int, error) {
	query := fmt.Sprintf(`
		UPDATE %s
		SET dead = true
		WHERE synced = false AND dead = false
		  AND created_at < (NOW() - make_interval(hours => $1))
	`, r.evtbl())
	result, err := r.db.ExecContext(ctx, query, maxAgeHours)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}

func (r *PostgresRepo) GetBufferStats(ctx context.Context) (*domain.BufferStats, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE synced = false AND dead = false),
			COUNT(*) FILTER (WHERE synced = true),
			COUNT(*) FILTER (WHERE dead = true),
			MIN(timestamp) FILTER (WHERE synced = false AND dead = false),
			MAX(timestamp) FILTER (WHERE synced = false AND dead = false),
			pg_total_relation_size('%s')
		FROM %s
	`, r.evtbl(), r.evtbl())
	stats := &domain.BufferStats{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalCount,
		&stats.PendingCount,
		&stats.SyncedCount,
		&stats.DeadCount,
		&stats.OldestPending,
		&stats.NewestPending,
		&stats.DiskBytes,
	)
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (r *PostgresRepo) EmergencyPurge(ctx context.Context, keepCount int) (int, error) {
	tbl := r.evtbl()

	// Phase 1: purge oldest synced events
	q1 := fmt.Sprintf(`
		WITH excess AS (
			SELECT id FROM %s
			WHERE synced = true
			ORDER BY timestamp ASC
			LIMIT GREATEST(0, (SELECT COUNT(*) FROM %s WHERE synced = true) - $1)
		)
		DELETE FROM %s WHERE id IN (SELECT id FROM excess)
	`, tbl, tbl, tbl)
	r1, err := r.db.ExecContext(ctx, q1, keepCount)
	if err != nil {
		return 0, err
	}
	phase1, _ := r1.RowsAffected()

	// Phase 2: if still over budget, purge oldest pending events
	q2 := fmt.Sprintf(`
		WITH excess AS (
			SELECT id FROM %s
			WHERE synced = false AND dead = false
			ORDER BY timestamp ASC
			LIMIT GREATEST(0, (SELECT COUNT(*) FROM %s) - $1)
		)
		DELETE FROM %s WHERE id IN (SELECT id FROM excess)
	`, tbl, tbl, tbl)
	r2, err := r.db.ExecContext(ctx, q2, keepCount)
	if err != nil {
		return int(phase1), err
	}
	phase2, _ := r2.RowsAffected()

	return int(phase1 + phase2), nil
}

func (r *PostgresRepo) Close() error {
	return r.db.Close()
}
