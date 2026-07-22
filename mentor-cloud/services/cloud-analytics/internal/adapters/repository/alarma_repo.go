package repository

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"fmt"
	"strings"
	"time"

	"mentor.local/shared/multitenancy"
)

type alarmaRepo struct {
	pr *multitenancy.PoolResolver
}

func NewAlarmaRepository(pr *multitenancy.PoolResolver) ports.AlarmaRepository {
	return &alarmaRepo{pr: pr}
}

func (r *alarmaRepo) FindAll(ctx context.Context, empresaID *int, activa *bool) ([]domain.Alarma, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.alarmas")

	var conditions []string
	var args []any
	idx := 1

	if empresaID != nil {
		conditions = append(conditions, fmt.Sprintf("empresa_id = $%d", idx))
		args = append(args, *empresaID)
		idx++
	}
	if activa != nil {
		conditions = append(conditions, fmt.Sprintf("activa = $%d", idx))
		args = append(args, *activa)
		idx++
	}

	q := fmt.Sprintf("SELECT id, device_id, linea_id, planta_id, empresa_id, tipo, mensaje, severidad, activa, timestamp_evento, reconocido_en, creado_en FROM %s", tbl)
	if len(conditions) > 0 {
		q += " WHERE " + strings.Join(conditions, " AND ")
	}
	q += " ORDER BY creado_en DESC LIMIT 200"

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Alarma
	for rows.Next() {
		var a domain.Alarma
		if err := rows.Scan(
			&a.ID, &a.DeviceID, &a.LineaID, &a.PlantaID, &a.EmpresaID,
			&a.Tipo, &a.Mensaje, &a.Severidad, &a.Activa,
			&a.TimestampEvento, &a.ReconocidoEn, &a.CreadoEn,
		); err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, nil
}

func (r *alarmaRepo) Create(ctx context.Context, a *domain.Alarma) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.alarmas")
	return pool.QueryRow(ctx, fmt.Sprintf(
		`INSERT INTO %s (device_id, linea_id, planta_id, empresa_id, tipo, mensaje, severidad, activa, timestamp_evento)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, creado_en`, tbl),
		a.DeviceID, a.LineaID, a.PlantaID, a.EmpresaID,
		a.Tipo, a.Mensaje, a.Severidad, a.Activa, a.TimestampEvento,
	).Scan(&a.ID, &a.CreadoEn)
}

func (r *alarmaRepo) Acknowledge(ctx context.Context, id int64) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.alarmas")
	tag, err := pool.Exec(ctx, fmt.Sprintf(
		`UPDATE %s SET activa = false, reconocido_en = $1 WHERE id = $2 AND activa = true`, tbl),
		time.Now(), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("alarma %d no encontrada o ya reconocida", id)
	}
	return nil
}
