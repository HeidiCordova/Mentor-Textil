package repository

import (
	"cloud-analytics/internal/domain"
	"cloud-analytics/internal/ports"
	"context"
	"fmt"
	"time"

	"mentor.local/shared/multitenancy"
)

type compromisoRepo struct {
	pr *multitenancy.PoolResolver
}

func NewCompromisoRepository(pr *multitenancy.PoolResolver) ports.CompromisoRepository {
	return &compromisoRepo{pr: pr}
}

func (r *compromisoRepo) FindAll(ctx context.Context, empresaID *int) ([]domain.Compromiso, error) {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	tbl := multitenancy.Tbl(schema, "analytics.compromisos")

	q := fmt.Sprintf(`SELECT id, empresa_id, planta_id, descripcion, responsable, fecha_limite, estado, creado_en, actualizado_en
      FROM %s`, tbl)
	var args []any
	if empresaID != nil {
		q += " WHERE empresa_id = $1"
		args = append(args, *empresaID)
	}
	q += " ORDER BY creado_en DESC LIMIT 200"

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Compromiso
	for rows.Next() {
		var c domain.Compromiso
		var creadoEn, actualizadoEn time.Time
		var fechaLimite *time.Time
		if err := rows.Scan(
			&c.ID, &c.EmpresaID, &c.PlantaID, &c.Descripcion, &c.Responsable,
			&fechaLimite, &c.Estado, &creadoEn, &actualizadoEn,
		); err != nil {
			return nil, err
		}
		c.CreadoEn = creadoEn.Format(time.RFC3339)
		c.ActualizadoEn = actualizadoEn.Format(time.RFC3339)
		if fechaLimite != nil {
			c.FechaLimite = fechaLimite.Format("2006-01-02")
		}
		result = append(result, c)
	}
	return result, nil
}

func (r *compromisoRepo) Create(ctx context.Context, c *domain.Compromiso) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.compromisos")

	var creadoEn, actualizadoEn time.Time
	err = pool.QueryRow(ctx,
		fmt.Sprintf(`INSERT INTO %s (empresa_id, planta_id, descripcion, responsable, fecha_limite, estado)
 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id, creado_en, actualizado_en`, tbl),
		c.EmpresaID, c.PlantaID, c.Descripcion, c.Responsable, nilIfEmpty(c.FechaLimite), c.Estado,
	).Scan(&c.ID, &creadoEn, &actualizadoEn)
	if err != nil {
		return err
	}
	c.CreadoEn = creadoEn.Format(time.RFC3339)
	c.ActualizadoEn = actualizadoEn.Format(time.RFC3339)
	return nil
}

func (r *compromisoRepo) Update(ctx context.Context, c *domain.Compromiso) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.compromisos")

	tag, err := pool.Exec(ctx,
		fmt.Sprintf(`UPDATE %s
 SET descripcion=$1, responsable=$2, fecha_limite=$3, estado=$4, actualizado_en=NOW()
 WHERE id=$5`, tbl),
		c.Descripcion, c.Responsable, nilIfEmpty(c.FechaLimite), c.Estado, c.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("compromiso %d no encontrado", c.ID)
	}
	return nil
}

func (r *compromisoRepo) Delete(ctx context.Context, id int) error {
	pool, schema, err := r.pr.Resolve(ctx)
	if err != nil {
		return err
	}
	tbl := multitenancy.Tbl(schema, "analytics.compromisos")

	tag, err := pool.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id=$1`, tbl), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("compromiso %d no encontrado", id)
	}
	return nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
