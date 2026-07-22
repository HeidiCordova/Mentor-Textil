package adapters

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"enviador/internal/ports"

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

type PostgresReader struct {
	db     *sql.DB
	schema string // ej: "linea_3"
}

func NewPostgresReader(connStr, lineSchema string) (*PostgresReader, error) {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresReader{db: db, schema: lineSchema}, nil
}

func (r *PostgresReader) DB() *sql.DB { return r.db }

func (r *PostgresReader) SetSchema(s string) { r.schema = s }

func (r *PostgresReader) evtbl() string   { return `"` + r.schema + `".events_buffer` }
func (r *PostgresReader) stoptbl() string { return `"` + r.schema + `".stops` }
func (r *PostgresReader) runtbl() string  { return `"` + r.schema + `".production_runs` }

func (r *PostgresReader) GetUnsyncedEvents(ctx context.Context, limit int) ([]ports.Event, error) {
	query := fmt.Sprintf(`
		SELECT id, event_id, device_id, event_type, timestamp, payload, retry_count
		FROM %s
		WHERE synced = false AND dead = false AND event_type = 'OEE_SNAPSHOT'
		ORDER BY timestamp ASC
		LIMIT $1
	`, r.evtbl())

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]ports.Event, 0)
	for rows.Next() {
		var event ports.Event
		err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.DeviceID,
			&event.EventType,
			&event.Timestamp,
			&event.Payload,
			&event.RetryCount,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (r *PostgresReader) MarkSynced(ctx context.Context, eventIDs []string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET synced = true, synced_at = NOW()
		WHERE event_id = ANY($1)
	`, r.evtbl())

	_, err := r.db.ExecContext(ctx, query, pgTextArray(eventIDs))
	return err
}

func (r *PostgresReader) IncrementRetry(ctx context.Context, eventID string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET retry_count = retry_count + 1
		WHERE event_id = $1
	`, r.evtbl())

	_, err := r.db.ExecContext(ctx, query, eventID)
	return err
}

func (r *PostgresReader) MarkDead(ctx context.Context, eventIDs []string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET dead = true
		WHERE event_id = ANY($1)
	`, r.evtbl())

	_, err := r.db.ExecContext(ctx, query, pgTextArray(eventIDs))
	return err
}

func (r *PostgresReader) GetCloudSyncInterval(ctx context.Context, lineaID int) (int, error) {
	query := `SELECT COALESCE((cloud->>'sync_interval_s')::int, 300) FROM config.line_config WHERE linea_id = $1`
	var interval int
	err := r.db.QueryRowContext(ctx, query, lineaID).Scan(&interval)
	if err != nil {
		return 300, nil
	}
	return interval, nil
}

func (r *PostgresReader) GetCloudCredentials(ctx context.Context, lineaID int) (cloudURL, apiKey string, err error) {
	query := `
		SELECT
			COALESCE(cloud->>'cloud_url', ''),
			COALESCE(cloud->>'cloud_api_key', '')
		FROM config.line_config
		WHERE linea_id = $1
	`
	err = r.db.QueryRowContext(ctx, query, lineaID).Scan(&cloudURL, &apiKey)
	if err != nil {
		return "", "", nil
	}
	return cloudURL, apiKey, nil
}

func (r *PostgresReader) GetUnsyncedStops(ctx context.Context, limit int) ([]ports.StopRecord, error) {
	query := fmt.Sprintf(`
		SELECT stop_id, device_id, stop_type,
		       started_at, ended_at, duration_ms,
		       justified, reason, category, categoria_id,
		       justified_by, justified_at, source,
		       created_at, updated_at
		FROM %s
		WHERE synced = false
		ORDER BY created_at ASC
		LIMIT $1
	`, r.stoptbl())

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ports.StopRecord
	for rows.Next() {
		var s ports.StopRecord
		var startedAt, createdAt, updatedAt sql.NullTime
		var endedAt, justifiedAt sql.NullTime
		var durationMS sql.NullInt64
		var reason, category, justifiedBy sql.NullString
		var categoriaID sql.NullInt64

		err := rows.Scan(
			&s.StopID, &s.DeviceID, &s.StopType,
			&startedAt, &endedAt, &durationMS,
			&s.Justified, &reason, &category, &categoriaID,
			&justifiedBy, &justifiedAt, &s.Source,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, err
		}

		if startedAt.Valid {
			s.StartedAt = startedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if endedAt.Valid {
			v := endedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			s.EndedAt = &v
		}
		if durationMS.Valid {
			s.DurationMS = &durationMS.Int64
		}
		if reason.Valid {
			s.Reason = &reason.String
		}
		if category.Valid {
			s.Category = &category.String
		}
		if categoriaID.Valid {
			s.CategoriaID = &categoriaID.Int64
		}
		if justifiedBy.Valid {
			s.JustifiedBy = &justifiedBy.String
		}
		if justifiedAt.Valid {
			v := justifiedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			s.JustifiedAt = &v
		}
		if createdAt.Valid {
			s.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if updatedAt.Valid {
			s.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}

		result = append(result, s)
	}

	return result, nil
}

func (r *PostgresReader) MarkStopsSynced(ctx context.Context, stopIDs []string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET synced = true, synced_at = NOW()
		WHERE stop_id::text = ANY($1)
	`, r.stoptbl())

	_, err := r.db.ExecContext(ctx, query, pgTextArray(stopIDs))
	return err
}

func (r *PostgresReader) GetUnsyncedProductionRuns(ctx context.Context, limit int) ([]ports.ProductionRunRecord, error) {
	query := fmt.Sprintf(`
		SELECT run_id, device_id, producto_id, sku, nombre,
		       started_at, ended_at, created_at, updated_at
		FROM %s
		WHERE synced = false
		ORDER BY started_at ASC
		LIMIT $1
	`, r.runtbl())

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ports.ProductionRunRecord
	for rows.Next() {
		var pr ports.ProductionRunRecord
		var productoID sql.NullInt64
		var sku, nombre sql.NullString
		var startedAt, createdAt, updatedAt sql.NullTime
		var endedAt sql.NullTime

		if err := rows.Scan(
			&pr.RunID, &pr.DeviceID, &productoID, &sku, &nombre,
			&startedAt, &endedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}

		if productoID.Valid {
			v := int(productoID.Int64)
			pr.ProductoID = &v
		}
		if sku.Valid {
			pr.SKU = &sku.String
		}
		if nombre.Valid {
			pr.Nombre = &nombre.String
		}
		if startedAt.Valid {
			pr.StartedAt = startedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if endedAt.Valid {
			v := endedAt.Time.Format("2006-01-02T15:04:05Z07:00")
			pr.EndedAt = &v
		}
		if createdAt.Valid {
			pr.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}
		if updatedAt.Valid {
			pr.UpdatedAt = updatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		}

		result = append(result, pr)
	}

	return result, nil
}

func (r *PostgresReader) MarkProductionRunsSynced(ctx context.Context, runIDs []string) error {
	query := fmt.Sprintf(`
		UPDATE %s
		SET synced = true, updated_at = NOW()
		WHERE run_id::text = ANY($1)
	`, r.runtbl())

	_, err := r.db.ExecContext(ctx, query, pgTextArray(runIDs))
	return err
}

func (r *PostgresReader) GetUnsyncedEnergyEvents(ctx context.Context, limit int) ([]ports.Event, error) {
	query := fmt.Sprintf(`
		SELECT id, event_id, device_id, event_type, timestamp, payload, retry_count
		FROM %s
		WHERE synced = false AND dead = false AND event_type = 'ENERGIA_SNAPSHOT'
		ORDER BY timestamp ASC
		LIMIT $1
	`, r.evtbl())

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]ports.Event, 0)
	for rows.Next() {
		var event ports.Event
		if err := rows.Scan(
			&event.ID, &event.EventID, &event.DeviceID,
			&event.EventType, &event.Timestamp, &event.Payload, &event.RetryCount,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (r *PostgresReader) Close() error {
	return r.db.Close()
}

// GetCurrentMode devuelve el modo de operación actual de la línea desde config.line_config.
func (r *PostgresReader) GetCurrentMode(ctx context.Context, lineaID int) (string, error) {
	var mode string
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(mode, 'botellas') FROM config.line_config WHERE linea_id = $1`, lineaID,
	).Scan(&mode)
	if err != nil {
		return "botellas", nil
	}
	return mode, nil
}

// ApplyPendingCommand aplica un comando recibido desde cloud al edge DB.
// Si el stop ya fue justificado localmente en el edge, NO sobreescribe (edge priority).
func (r *PostgresReader) ApplyPendingCommand(ctx context.Context, cmd ports.PendingCommand) error {
	switch cmd.Command {
	case "justificar_parada":
		return r.applyJustifyStop(ctx, cmd)
	case "upsert_production_run":
		return r.applyUpsertProductionRun(ctx, cmd)
	case "SYNC_USUARIOS":
		return r.applySyncUsuarios(ctx, cmd)
	default:
		return nil
	}
}

func (r *PostgresReader) applySyncUsuarios(ctx context.Context, cmd ports.PendingCommand) error {
	var payload struct {
		Records []struct {
			ID           int    `json:"id"`
			Username     string `json:"username"`
			Email        string `json:"email"`
			Nombre       string `json:"nombre"`
			Apellido     string `json:"apellido"`
			PasswordHash string `json:"password_hash"`
			RolID        int    `json:"rol_id"`
			Rol          string `json:"rol"`
			EmpresaID    int    `json:"empresa_id"`
			Activo       bool   `json:"activo"`
		} `json:"records"`
	}
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return fmt.Errorf("parse SYNC_USUARIOS payload: %w", err)
	}
	if len(payload.Records) == 0 {
		return nil
	}

	tbl := `"` + r.schema + `".usuarios`
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM `+tbl); err != nil {
		return err
	}
	for _, rec := range payload.Records {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO `+tbl+` (id, username, email, nombre, apellido, password_hash, rol_id, rol, empresa_id, activo, synced_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())`,
			rec.ID, rec.Username, rec.Email, rec.Nombre, rec.Apellido,
			rec.PasswordHash, rec.RolID, rec.Rol, rec.EmpresaID, rec.Activo,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PostgresReader) applyJustifyStop(ctx context.Context, cmd ports.PendingCommand) error {
	var payload struct {
		StopID      string  `json:"stop_id"`
		Reason      *string `json:"reason"`
		Category    *string `json:"category"`
		CategoriaID *int64  `json:"categoria_id"`
		StopType    *string `json:"stop_type"`
		JustifiedBy *string `json:"justified_by"`
	}
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return fmt.Errorf("parse pending command payload: %w", err)
	}
	if payload.StopID == "" {
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET justified    = true,
		    justified_by = COALESCE($1, justified_by),
		    justified_at = COALESCE(justified_at, NOW()),
		    reason       = COALESCE($2, reason),
		    category     = COALESCE($3, category),
		    categoria_id = COALESCE($4, categoria_id)
		WHERE stop_id::text = $5
		  AND (justified = false OR justified IS NULL)
	`, r.stoptbl())

	_, err := r.db.ExecContext(ctx, query,
		payload.JustifiedBy,
		payload.Reason,
		payload.Category,
		payload.CategoriaID,
		payload.StopID,
	)
	return err
}

func (r *PostgresReader) applyUpsertProductionRun(ctx context.Context, cmd ports.PendingCommand) error {
	var payload struct {
		RunID     string  `json:"run_id"`
		SKU       *string `json:"sku"`
		Nombre    *string `json:"nombre"`
		StartedAt string  `json:"started_at"`
		EndedAt   *string `json:"ended_at"`
	}
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return fmt.Errorf("parse upsert_production_run payload: %w", err)
	}
	if payload.RunID == "" || payload.StartedAt == "" {
		return nil
	}

	// Solo cerrar el run activo anterior cuando el nuevo run es el activo (sin ended_at).
	// Si es una asignación histórica (con ended_at), no tocamos el run activo actual.
	if payload.EndedAt == nil {
		_, _ = r.db.ExecContext(ctx, fmt.Sprintf(`
			UPDATE %s SET ended_at = $1, updated_at = NOW(), synced = false
			WHERE ended_at IS NULL AND run_id != $2
		`, r.runtbl()), payload.StartedAt, payload.RunID)
	}

	// Intentar UPDATE primero; si no actualiza nada, INSERT
	// synced = true para evitar bounce de vuelta al cloud (ya existe allá)
	res, err := r.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s SET sku = $1, nombre = $2, started_at = $3, ended_at = $4, updated_at = NOW(), synced = true
		WHERE run_id = $5
	`, r.runtbl()), payload.SKU, payload.Nombre, payload.StartedAt, payload.EndedAt, payload.RunID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}

	_, err = r.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (run_id, device_id, sku, nombre, started_at, ended_at, synced)
		VALUES ($1, $2, $3, $4, $5, $6, true)
	`, r.runtbl()), payload.RunID, cmd.DeviceID, payload.SKU, payload.Nombre, payload.StartedAt, payload.EndedAt)
	return err
}
