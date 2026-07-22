package adapters

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
)

type PostgresAuditRepo struct {
	db     *sql.DB
	schema string
}

func NewPostgresAuditRepo(db *sql.DB, schema string) ports.AuditRepository {
	return &PostgresAuditRepo{db: db, schema: schema}
}

func (r *PostgresAuditRepo) tbl() string { return r.schema + ".audit_log" }

func (r *PostgresAuditRepo) Log(ctx context.Context, entry domain.AuditEntry) error {
	payloadBytes, _ := json.Marshal(entry.Payload)

	query := fmt.Sprintf(`INSERT INTO %s (device_id, actor, action, resource, resource_id, payload, result)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, r.tbl())

	_, err := r.db.ExecContext(ctx, query,
		entry.DeviceID, entry.Actor, entry.Action, entry.Resource,
		entry.ResourceID, payloadBytes, entry.Result,
	)
	return err
}

func (r *PostgresAuditRepo) ListByDevice(ctx context.Context, deviceID string, limit int) ([]domain.AuditEntry, error) {
	query := fmt.Sprintf(`SELECT id, device_id, actor, action, resource, resource_id, payload, result, timestamp
		FROM %s WHERE device_id = $1
		ORDER BY timestamp DESC LIMIT $2`, r.tbl())

	rows, err := r.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.AuditEntry
	for rows.Next() {
		var e domain.AuditEntry
		var resourceID sql.NullString
		var payloadBytes []byte

		err := rows.Scan(
			&e.ID, &e.DeviceID, &e.Actor, &e.Action, &e.Resource,
			&resourceID, &payloadBytes, &e.Result, &e.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		if resourceID.Valid {
			e.ResourceID = &resourceID.String
		}
		if len(payloadBytes) > 0 {
			json.Unmarshal(payloadBytes, &e.Payload)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PostgresAuditRepo) PurgeOlderThan(ctx context.Context, retentionDays int) (int, error) {
	query := fmt.Sprintf(`DELETE FROM %s WHERE timestamp < (NOW() - make_interval(days => $1))`, r.tbl())
	result, err := r.db.ExecContext(ctx, query, retentionDays)
	if err != nil {
		return 0, err
	}
	affected, _ := result.RowsAffected()
	return int(affected), nil
}
