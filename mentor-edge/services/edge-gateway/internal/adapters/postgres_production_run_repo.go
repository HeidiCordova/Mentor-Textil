package adapters

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
)

const runCols = `id, run_id, device_id, linea_id, producto_id, sku, nombre,
	started_at, ended_at, synced, created_at, updated_at`

type PostgresProductionRunRepo struct {
	db     *sql.DB
	schema string // ej: "linea_3"
}

func NewPostgresProductionRunRepo(db *sql.DB, schema string) ports.ProductionRunRepository {
	return &PostgresProductionRunRepo{db: db, schema: schema}
}

func (r *PostgresProductionRunRepo) tbl() string { return r.schema + ".production_runs" }

func scanRun(s interface {
	Scan(...any) error
}) (*domain.ProductionRun, error) {
	r := &domain.ProductionRun{}
	return r, s.Scan(
		&r.ID, &r.RunID, &r.DeviceID, &r.LineaID, &r.ProductoID, &r.SKU, &r.Nombre,
		&r.StartedAt, &r.EndedAt, &r.Synced, &r.CreatedAt, &r.UpdatedAt,
	)
}

func (r *PostgresProductionRunRepo) Upsert(ctx context.Context, req domain.UpsertProductionRunRequest) ([]domain.ProductionRun, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	T1 := req.StartedAt
	T2 := req.EndedAt

	var (
		rows    *sql.Rows
		scanErr error
	)

	tbl := r.tbl()
	if T2 == nil {
		rows, scanErr = tx.QueryContext(ctx, fmt.Sprintf(`
			SELECT %s
			FROM %s
			WHERE device_id = $1
			  AND (ended_at IS NULL OR ended_at > $2)
			ORDER BY started_at
			FOR UPDATE
		`, runCols, tbl), req.DeviceID, T1)
	} else {
		rows, scanErr = tx.QueryContext(ctx, fmt.Sprintf(`
			SELECT %s
			FROM %s
			WHERE device_id = $1
			  AND started_at < $2
			  AND (ended_at IS NULL OR ended_at > $3)
			ORDER BY started_at
			FOR UPDATE
		`, runCols, tbl), req.DeviceID, *T2, T1)
	}
	if scanErr != nil {
		return nil, scanErr
	}

	var overlap []domain.ProductionRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		overlap = append(overlap, *run)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, run := range overlap {
		if run.StartedAt.Before(T1) {
			// Clip this run's end to T1.
			_, err = tx.ExecContext(ctx, fmt.Sprintf(`
				UPDATE %s
				SET ended_at = $1, synced = false, updated_at = NOW()
				WHERE run_id = $2
			`, tbl), T1, run.RunID)
			if err != nil {
				return nil, fmt.Errorf("clip run: %w", err)
			}

			// If the run extended beyond T2, create a tail.
			if T2 != nil && (run.EndedAt == nil || run.EndedAt.After(*T2)) {
				_, err = tx.ExecContext(ctx, fmt.Sprintf(`
					INSERT INTO %s
					    (device_id, linea_id, producto_id, sku, nombre, started_at, ended_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
				`, tbl), run.DeviceID, run.LineaID, run.ProductoID, run.SKU, run.Nombre, *T2, run.EndedAt)
				if err != nil {
					return nil, fmt.Errorf("insert tail: %w", err)
				}
			}
		} else {
			// Run starts at or after T1.
			if T2 != nil && (run.EndedAt == nil || run.EndedAt.After(*T2)) {
				// Move its start to T2.
				_, err = tx.ExecContext(ctx, fmt.Sprintf(`
					UPDATE %s
					SET started_at = $1, synced = false, updated_at = NOW()
					WHERE run_id = $2
				`, tbl), *T2, run.RunID)
				if err != nil {
					return nil, fmt.Errorf("shift run: %w", err)
				}
			} else {
				// Fully contained: delete.
				_, err = tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE run_id = $1`, tbl), run.RunID)
				if err != nil {
					return nil, fmt.Errorf("delete contained: %w", err)
				}
			}
		}
	}

	// Remove zero-duration runs that clipping may have created.
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE device_id = $1 AND started_at = ended_at
	`, tbl), req.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("cleanup zero-duration: %w", err)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (device_id, linea_id, producto_id, sku, nombre, started_at, ended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tbl), req.DeviceID, req.LineaID, req.ProductoID, req.SKU, req.Nombre, T1, T2)
	if err != nil {
		return nil, fmt.Errorf("insert new run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return r.List(ctx, domain.ProductionRunFilter{DeviceID: req.DeviceID, Limit: 500})
}

func (r *PostgresProductionRunRepo) List(ctx context.Context, f domain.ProductionRunFilter) ([]domain.ProductionRun, error) {
	where := []string{"device_id = $1"}
	args := []any{f.DeviceID}
	i := 2

	if f.LineaID != nil {
		where = append(where, fmt.Sprintf("linea_id = $%d", i))
		args = append(args, *f.LineaID)
		i++
	}
	if f.Since != nil {
		where = append(where, fmt.Sprintf("started_at >= $%d", i))
		args = append(args, *f.Since)
		i++
	}
	if f.Until != nil {
		where = append(where, fmt.Sprintf("started_at < $%d", i))
		args = append(args, *f.Until)
		i++
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	args = append(args, limit)

	query := fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE %s
		ORDER BY started_at ASC
		LIMIT $%d
	`, runCols, r.tbl(), strings.Join(where, " AND "), i)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.ProductionRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	return runs, rows.Err()
}

func (r *PostgresProductionRunRepo) Delete(ctx context.Context, runID string) ([]domain.ProductionRun, error) {
	var deviceID string
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		DELETE FROM %s WHERE run_id = $1 RETURNING device_id
	`, r.tbl()), runID).Scan(&deviceID)
	if err != nil {
		return nil, err
	}
	return r.List(ctx, domain.ProductionRunFilter{DeviceID: deviceID, Limit: 500})
}

func (r *PostgresProductionRunRepo) ListUnsynced(ctx context.Context, limit int) ([]domain.ProductionRun, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s
		FROM %s
		WHERE synced = false
		ORDER BY started_at ASC
		LIMIT $1
	`, runCols, r.tbl()), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []domain.ProductionRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	return runs, rows.Err()
}

// SinProgramacionSeconds returns the total seconds within [from, to) covered
// by production_runs with sku IS NULL ("sin programación") for the given device.
func (r *PostgresProductionRunRepo) SinProgramacionSeconds(ctx context.Context, deviceID string, from, to time.Time) (int64, error) {
	var secs int64
	err := r.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT COALESCE(SUM(
			EXTRACT(EPOCH FROM (
				LEAST(COALESCE(ended_at, $3), $3) - GREATEST(started_at, $2)
			))
		), 0)::BIGINT
		FROM %s
		WHERE device_id = $1
		  AND sku IS NULL
		  AND started_at < $3
		  AND (ended_at IS NULL OR ended_at > $2)
	`, r.tbl()), deviceID, from, to).Scan(&secs)
	return secs, err
}

func (r *PostgresProductionRunRepo) MarkSynced(ctx context.Context, runIDs []string) error {
	if len(runIDs) == 0 {
		return nil
	}
	placeholders := make([]string, len(runIDs))
	args := make([]any, len(runIDs))
	for i, id := range runIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET synced = true, updated_at = NOW()
		WHERE run_id IN (%s)
	`, r.tbl(), strings.Join(placeholders, ",")), args...)
	return err
}

// ScanTime is exported for testing
var _ = time.Now
