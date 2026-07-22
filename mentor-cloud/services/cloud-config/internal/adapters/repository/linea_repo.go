package repository

import (
	"cloud-config/internal/domain"
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LineaRepo struct {
	db *pgxpool.Pool
}

func NewLineaRepo(db *pgxpool.Pool) *LineaRepo {
	return &LineaRepo{db: db}
}

func (r *LineaRepo) FindAll(ctx context.Context, plantaID *int, empresaID *int) ([]domain.Linea, error) {
	query := `
		SELECT l.id, l.nombre, l.planta_id, l.tipo, l.subtipo, l.mode, l.activo
		FROM config.lineas l
		JOIN config.plantas p ON p.id = l.planta_id`
	args := []interface{}{}
	conditions := []string{}

	if plantaID != nil {
		conditions = append(conditions, `l.planta_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *plantaID)
	}
	if empresaID != nil {
		conditions = append(conditions, `p.empresa_id = $`+strconv.Itoa(len(args)+1))
		args = append(args, *empresaID)
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + conditions[0]
		for _, cond := range conditions[1:] {
			query += ` AND ` + cond
		}
	}
	query += ` ORDER BY l.id`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Linea
	for rows.Next() {
		var l domain.Linea
		if err := rows.Scan(&l.ID, &l.Nombre, &l.PlantaID, &l.Tipo, &l.Subtipo, &l.Mode, &l.Activo); err != nil {
			return nil, err
		}
		result = append(result, l)
	}
	return result, nil
}

func (r *LineaRepo) FindByID(ctx context.Context, id int) (*domain.Linea, error) {
	l := &domain.Linea{}
	err := r.db.QueryRow(ctx,
		`SELECT id, nombre, planta_id, tipo, subtipo, mode, activo FROM config.lineas WHERE id=$1`, id,
	).Scan(&l.ID, &l.Nombre, &l.PlantaID, &l.Tipo, &l.Subtipo, &l.Mode, &l.Activo)
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (r *LineaRepo) Create(ctx context.Context, l *domain.Linea) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO config.lineas (nombre, planta_id, tipo, subtipo, mode, activo) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		l.Nombre, l.PlantaID, l.Tipo, l.Subtipo, l.Mode, l.Activo,
	).Scan(&l.ID)
}

func (r *LineaRepo) Update(ctx context.Context, l *domain.Linea) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config.lineas SET nombre=$2, planta_id=$3, tipo=$4, subtipo=$5, mode=$6, activo=$7 WHERE id=$1`,
		l.ID, l.Nombre, l.PlantaID, l.Tipo, l.Subtipo, l.Mode, l.Activo)
	return err
}

func (r *LineaRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM config.lineas WHERE id=$1`, id)
	return err
}
