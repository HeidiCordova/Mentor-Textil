package adapters

import (
	"context"

	"cloud-gateway/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type auditRepo struct{ pool *pgxpool.Pool }

func NewAuditRepo(pool *pgxpool.Pool) *auditRepo {
	return &auditRepo{pool: pool}
}

func (r *auditRepo) Insert(ctx context.Context, e *domain.AuditEntry) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gateway.audit_log
		 (method, path, status, latency_ms, ip, user_id, empresa_id, device_id, ts)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		e.Method, e.Path, e.Status, e.LatencyMs,
		nullStr(e.IP), nullInt(e.UserID), nullInt(e.EmpresaID), nullStr(e.DeviceID),
		e.Timestamp,
	)
	return err
}

func nullInt(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullStr(v string) any {
	if v == "" {
		return nil
	}
	return v
}
