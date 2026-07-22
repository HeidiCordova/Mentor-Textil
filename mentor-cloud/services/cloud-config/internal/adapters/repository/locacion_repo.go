package repository

import (
	"cloud-config/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LocacionRepo struct {
	pool *pgxpool.Pool
}

func NewLocacionRepo(pool *pgxpool.Pool) *LocacionRepo {
	return &LocacionRepo{pool: pool}
}

func (r *LocacionRepo) FindAll(ctx context.Context, lineaID *int, empresaID *int) ([]domain.Locacion, error) {
	query := `
		SELECT l.id, l.nombre, COALESCE(l.descripcion,''), l.linea_id,
		       COALESCE(li.nombre,''), l.empresa_id, l.activo
		FROM config.locaciones l
		LEFT JOIN config.lineas li ON li.id = l.linea_id
		WHERE ($1::int IS NULL OR l.linea_id = $1)
		  AND ($2::int IS NULL OR l.empresa_id = $2)
		ORDER BY l.id`
	rows, err := r.pool.Query(ctx, query, lineaID, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []domain.Locacion
	for rows.Next() {
		var loc domain.Locacion
		if err := rows.Scan(&loc.ID, &loc.Nombre, &loc.Descripcion, &loc.LineaID, &loc.LineaNombre, &loc.EmpresaID, &loc.Activo); err != nil {
			return nil, err
		}
		result = append(result, loc)
	}
	if result == nil {
		result = []domain.Locacion{}
	}
	return result, nil
}

func (r *LocacionRepo) FindByID(ctx context.Context, id int) (*domain.Locacion, error) {
	query := `
		SELECT l.id, l.nombre, COALESCE(l.descripcion,''), l.linea_id,
		       COALESCE(li.nombre,''), l.empresa_id, l.activo
		FROM config.locaciones l
		LEFT JOIN config.lineas li ON li.id = l.linea_id
		WHERE l.id = $1`
	var loc domain.Locacion
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&loc.ID, &loc.Nombre, &loc.Descripcion, &loc.LineaID,
		&loc.LineaNombre, &loc.EmpresaID, &loc.Activo,
	)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func (r *LocacionRepo) Create(ctx context.Context, loc *domain.Locacion) error {
	query := `
		INSERT INTO config.locaciones (nombre, descripcion, linea_id, empresa_id, activo)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	return r.pool.QueryRow(ctx, query,
		loc.Nombre, loc.Descripcion, loc.LineaID, loc.EmpresaID, loc.Activo,
	).Scan(&loc.ID)
}

func (r *LocacionRepo) Update(ctx context.Context, loc *domain.Locacion) error {
	query := `
		UPDATE config.locaciones
		SET nombre=$2, descripcion=$3, linea_id=$4, empresa_id=$5, activo=$6,
		    actualizado_en=NOW()
		WHERE id=$1`
	_, err := r.pool.Exec(ctx, query,
		loc.ID, loc.Nombre, loc.Descripcion, loc.LineaID, loc.EmpresaID, loc.Activo,
	)
	return err
}

func (r *LocacionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM config.locaciones WHERE id=$1`, id)
	return err
}
