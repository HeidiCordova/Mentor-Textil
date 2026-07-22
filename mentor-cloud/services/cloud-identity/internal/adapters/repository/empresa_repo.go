package repository

import (
	"cloud-identity/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type EmpresaRepo struct {
	db *pgxpool.Pool
}

func NewEmpresaRepo(db *pgxpool.Pool) *EmpresaRepo {
	return &EmpresaRepo{db: db}
}

func (r *EmpresaRepo) FindAll(ctx context.Context) ([]domain.Empresa, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, nombre, ruc, COALESCE(direccion,''), COALESCE(telefono,''),
		       COALESCE(email,''), COALESCE(responsable,''), estado, creado_en
		FROM identity.empresas
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Empresa
	for rows.Next() {
		var e domain.Empresa
		if err := rows.Scan(&e.ID, &e.Nombre, &e.RUC, &e.Direccion, &e.Telefono,
			&e.Email, &e.Responsable, &e.Estado, &e.CreadoEn); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

func (r *EmpresaRepo) FindByID(ctx context.Context, id int) (*domain.Empresa, error) {
	e := &domain.Empresa{}
	err := r.db.QueryRow(ctx, `
		SELECT id, nombre, ruc, COALESCE(direccion,''), COALESCE(telefono,''),
		       COALESCE(email,''), COALESCE(responsable,''), estado, creado_en
		FROM identity.empresas WHERE id=$1`, id,
	).Scan(&e.ID, &e.Nombre, &e.RUC, &e.Direccion, &e.Telefono,
		&e.Email, &e.Responsable, &e.Estado, &e.CreadoEn)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *EmpresaRepo) Create(ctx context.Context, e *domain.Empresa) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO identity.empresas (nombre, ruc, direccion, telefono, email, responsable, estado)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creado_en`,
		e.Nombre, e.RUC, e.Direccion, e.Telefono, e.Email, e.Responsable, e.Estado,
	).Scan(&e.ID, &e.CreadoEn)
}

func (r *EmpresaRepo) Update(ctx context.Context, e *domain.Empresa) error {
	_, err := r.db.Exec(ctx, `
		UPDATE identity.empresas
		SET nombre=$2, ruc=$3, direccion=$4, telefono=$5, email=$6,
		    responsable=$7, estado=$8, actualizado_en=NOW()
		WHERE id=$1`,
		e.ID, e.Nombre, e.RUC, e.Direccion, e.Telefono, e.Email, e.Responsable, e.Estado)
	return err
}

func (r *EmpresaRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM identity.empresas WHERE id=$1`, id)
	return err
}
