package adapters

import (
	"context"
	"time"

	"cloud-gateway/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type commandRepo struct{ pool *pgxpool.Pool }

func NewCommandRepo(pool *pgxpool.Pool) *commandRepo {
	return &commandRepo{pool: pool}
}

func (r *commandRepo) Insert(ctx context.Context, cmd *domain.Command) (*domain.Command, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO gateway.commands (device_id, empresa_id, type, payload, issued_by)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, created_at`,
		cmd.DeviceID, nullInt(cmd.EmpresaID), cmd.Type, cmd.Payload, nullInt(cmd.IssuedBy),
	).Scan(&cmd.ID, &cmd.CreatedAt)
	return cmd, err
}

func (r *commandRepo) ListPending(ctx context.Context, deviceID string) ([]domain.Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, device_id, empresa_id, type, payload, status, created_at
		 FROM gateway.commands
		 WHERE device_id = $1 AND status = 'pending'
		 ORDER BY created_at ASC LIMIT 50`,
		deviceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Command
	for rows.Next() {
		var c domain.Command
		if err := rows.Scan(&c.ID, &c.DeviceID, &c.EmpresaID, &c.Type,
			&c.Payload, &c.Status, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *commandRepo) Acknowledge(ctx context.Context, id int64, deviceID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateway.commands
		 SET status = 'acked', acked_at = $1
		 WHERE id = $2 AND device_id = $3`,
		time.Now(), id, deviceID,
	)
	return err
}

func (r *commandRepo) MarkFailed(ctx context.Context, id int64, reason string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gateway.commands SET status = 'failed', fail_reason = $1 WHERE id = $2`,
		reason, id,
	)
	return err
}
