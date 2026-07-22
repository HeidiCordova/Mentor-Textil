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
