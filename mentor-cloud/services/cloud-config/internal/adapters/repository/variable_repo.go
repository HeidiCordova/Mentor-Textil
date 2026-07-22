package repository

import (
	"cloud-config/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type VariableRepo struct {
	db *pgxpool.Pool
}

func NewVariableRepo(db *pgxpool.Pool) *VariableRepo {
	return &VariableRepo{db: db}
}

func (r *VariableRepo) FindAll(ctx context.Context, dispositivoID *int) ([]domain.Variable, error) {
	query := `SELECT id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo FROM config.variables`
	args := []interface{}{}
	if dispositivoID != nil {
		query += ` WHERE dispositivo_id = $1`
		args = append(args, *dispositivoID)
	}
	query += ` ORDER BY id`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Variable
	for rows.Next() {
		var v domain.Variable
		if err := rows.Scan(&v.ID, &v.Nombre, &v.Clave, &v.Valor, &v.Tipo,
			&v.DispositivoID, &v.PlantaID, &v.EmpresaID, &v.Activo); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

func (r *VariableRepo) FindByID(ctx context.Context, id int) (*domain.Variable, error) {
	v := &domain.Variable{}
	err := r.db.QueryRow(ctx,
		`SELECT id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo
		 FROM config.variables WHERE id=$1`, id,
	).Scan(&v.ID, &v.Nombre, &v.Clave, &v.Valor, &v.Tipo,
		&v.DispositivoID, &v.PlantaID, &v.EmpresaID, &v.Activo)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *VariableRepo) Create(ctx context.Context, v *domain.Variable) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO config.variables (nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		v.Nombre, v.Clave, v.Valor, v.Tipo, v.DispositivoID, v.PlantaID, v.EmpresaID, v.Activo,
	).Scan(&v.ID)
}

func (r *VariableRepo) Update(ctx context.Context, v *domain.Variable) error {
	_, err := r.db.Exec(ctx,
		`UPDATE config.variables
		 SET nombre=$2, clave=$3, valor=$4, tipo=$5, dispositivo_id=$6, planta_id=$7, empresa_id=$8, activo=$9, actualizado_en=NOW()
		 WHERE id=$1`,
		v.ID, v.Nombre, v.Clave, v.Valor, v.Tipo, v.DispositivoID, v.PlantaID, v.EmpresaID, v.Activo)
	return err
}

func (r *VariableRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM config.variables WHERE id=$1`, id)
	return err
}

func (r *VariableRepo) FindByClave(ctx context.Context, clave string, dispositivoID int) (*domain.Variable, error) {
	v := &domain.Variable{}
	err := r.db.QueryRow(ctx,
		`SELECT id, nombre, clave, valor, tipo, dispositivo_id, planta_id, empresa_id, activo
		 FROM config.variables WHERE clave=$1 AND dispositivo_id=$2`, clave, dispositivoID,
	).Scan(&v.ID, &v.Nombre, &v.Clave, &v.Valor, &v.Tipo,
		&v.DispositivoID, &v.PlantaID, &v.EmpresaID, &v.Activo)
	if err != nil {
		return nil, err
	}
	return v, nil
}
