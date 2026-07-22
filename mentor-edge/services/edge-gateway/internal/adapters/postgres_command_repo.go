package adapters

import (
	"context"
	"database/sql"
	"edge-gateway/internal/domain"
	"edge-gateway/internal/ports"
	"encoding/json"
	"fmt"
	"time"
)

type PostgresCommandRepo struct {
	db     *sql.DB
	schema string
}

func NewPostgresCommandRepo(db *sql.DB, schema string) ports.CommandRepository {
	return &PostgresCommandRepo{db: db, schema: schema}
}

func (r *PostgresCommandRepo) tbl() string { return r.schema + ".commands_buffer" }

func (r *PostgresCommandRepo) Create(ctx context.Context, req domain.CreateCommandRequest) (*domain.Command, error) {
	existing, _ := r.GetByIdempotencyKey(ctx, req.IdempotencyKey)
	if existing != nil {
		return existing, domain.ErrDuplicateCommand
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	query := fmt.Sprintf(`INSERT INTO %s
		(command_id, device_id, command_type, payload, issued_by, idempotency_key)
		VALUES (COALESCE(NULLIF($1, '')::UUID, uuid_generate_v4()), $2, $3, $4, $5, $6)
		RETURNING id, command_id, device_id, command_type, payload, issued_by,
			issued_at, idempotency_key, status, result, error_message, applied_at, created_at`, r.tbl())

	cmdID := req.CommandID
	return r.scanCommand(r.db.QueryRowContext(ctx, query,
		cmdID, req.DeviceID, req.CommandType, payloadBytes,
		req.IssuedBy, req.IdempotencyKey,
	))
}

func (r *PostgresCommandRepo) GetByID(ctx context.Context, commandID string) (*domain.Command, error) {
	query := fmt.Sprintf(`SELECT id, command_id, device_id, command_type, payload, issued_by,
		issued_at, idempotency_key, status, result, error_message, applied_at, created_at
		FROM %s WHERE command_id = $1`, r.tbl())

	cmd, err := r.scanCommand(r.db.QueryRowContext(ctx, query, commandID))
	if err == sql.ErrNoRows {
		return nil, domain.ErrCommandNotFound
	}
	return cmd, err
}

func (r *PostgresCommandRepo) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Command, error) {
	query := fmt.Sprintf(`SELECT id, command_id, device_id, command_type, payload, issued_by,
		issued_at, idempotency_key, status, result, error_message, applied_at, created_at
		FROM %s WHERE idempotency_key = $1`, r.tbl())

	cmd, err := r.scanCommand(r.db.QueryRowContext(ctx, query, key))
	if err == sql.ErrNoRows {
		return nil, domain.ErrCommandNotFound
	}
	return cmd, err
}

func (r *PostgresCommandRepo) MarkApplied(ctx context.Context, commandID string, result map[string]interface{}) error {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}

	query := fmt.Sprintf(`UPDATE %s SET status = 'APPLIED', result = $1, applied_at = $2
		WHERE command_id = $3`, r.tbl())

	res, err := r.db.ExecContext(ctx, query, resultBytes, time.Now().UTC(), commandID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrCommandNotFound
	}
	return nil
}

func (r *PostgresCommandRepo) MarkFailed(ctx context.Context, commandID string, errMsg string) error {
	query := fmt.Sprintf(`UPDATE %s SET status = 'FAILED', error_message = $1, applied_at = $2
		WHERE command_id = $3`, r.tbl())

	res, err := r.db.ExecContext(ctx, query, errMsg, time.Now().UTC(), commandID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrCommandNotFound
	}
	return nil
}

func (r *PostgresCommandRepo) ListByDevice(ctx context.Context, deviceID string, limit int) ([]domain.Command, error) {
	query := fmt.Sprintf(`SELECT id, command_id, device_id, command_type, payload, issued_by,
		issued_at, idempotency_key, status, result, error_message, applied_at, created_at
		FROM %s WHERE device_id = $1
		ORDER BY issued_at DESC LIMIT $2`, r.tbl())

	rows, err := r.db.QueryContext(ctx, query, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []domain.Command
	for rows.Next() {
		cmd, err := r.scanCommand(rows)
		if err != nil {
			return nil, err
		}
		commands = append(commands, *cmd)
	}
	return commands, rows.Err()
}

func (r *PostgresCommandRepo) scanCommand(row scanner) (*domain.Command, error) {
	var cmd domain.Command
	var payloadBytes, resultBytes []byte
	var errorMsg sql.NullString
	var appliedAt sql.NullTime

	err := row.Scan(
		&cmd.ID, &cmd.CommandID, &cmd.DeviceID, &cmd.CommandType,
		&payloadBytes, &cmd.IssuedBy, &cmd.IssuedAt, &cmd.IdempotencyKey,
		&cmd.Status, &resultBytes, &errorMsg, &appliedAt, &cmd.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if len(payloadBytes) > 0 {
		json.Unmarshal(payloadBytes, &cmd.Payload)
	}
	if len(resultBytes) > 0 {
		json.Unmarshal(resultBytes, &cmd.Result)
	}
	if errorMsg.Valid {
		cmd.ErrorMessage = &errorMsg.String
	}
	if appliedAt.Valid {
		cmd.AppliedAt = &appliedAt.Time
	}
	return &cmd, nil
}
