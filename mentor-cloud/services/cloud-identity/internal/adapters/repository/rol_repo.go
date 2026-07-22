package repository

import (
	"cloud-identity/internal/domain"
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RolRepo struct {
	db *pgxpool.Pool
}

func NewRolRepo(db *pgxpool.Pool) *RolRepo {
	return &RolRepo{db: db}
}

func scanPermisos(raw string) []string {
	if raw == "" || raw == "null" {
		return []string{}
	}
	var p []string
	json.Unmarshal([]byte(raw), &p)
	if p == nil {
		return []string{}
	}
	return p
}

func (r *RolRepo) FindAll(ctx context.Context) ([]domain.Rol, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, nombre, COALESCE(descripcion,''), COALESCE(permisos::text,'[]')
		FROM identity.roles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Rol
	for rows.Next() {
		var rol domain.Rol
		var permisosRaw string
		if err := rows.Scan(&rol.ID, &rol.Nombre, &rol.Descripcion, &permisosRaw); err != nil {
			return nil, err
		}
		rol.Permisos = scanPermisos(permisosRaw)
		result = append(result, rol)
	}
	return result, nil
}

func (r *RolRepo) FindByID(ctx context.Context, id int) (*domain.Rol, error) {
	rol := &domain.Rol{}
	var permisosRaw string
	err := r.db.QueryRow(ctx, `
		SELECT id, nombre, COALESCE(descripcion,''), COALESCE(permisos::text,'[]')
		FROM identity.roles WHERE id=$1`, id).
		Scan(&rol.ID, &rol.Nombre, &rol.Descripcion, &permisosRaw)
	if err != nil {
		return nil, err
	}
	rol.Permisos = scanPermisos(permisosRaw)
	return rol, nil
}

func (r *RolRepo) FindByNombre(ctx context.Context, nombre string) (*domain.Rol, error) {
	rol := &domain.Rol{}
	var permisosRaw string
	err := r.db.QueryRow(ctx, `
		SELECT id, nombre, COALESCE(descripcion,''), COALESCE(permisos::text,'[]')
		FROM identity.roles WHERE nombre=$1`, nombre).
		Scan(&rol.ID, &rol.Nombre, &rol.Descripcion, &permisosRaw)
	if err != nil {
		return nil, err
	}
	rol.Permisos = scanPermisos(permisosRaw)
	return rol, nil
}

func (r *RolRepo) Create(ctx context.Context, rol *domain.Rol) error {
	permisosJSON, err := json.Marshal(rol.Permisos)
	if err != nil {
		return err
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO identity.roles (nombre, descripcion, permisos)
		VALUES ($1, $2, $3::jsonb)
		RETURNING id`,
		rol.Nombre, rol.Descripcion, string(permisosJSON),
	).Scan(&rol.ID)
}

func (r *RolRepo) Update(ctx context.Context, rol *domain.Rol) error {
	permisosJSON, err := json.Marshal(rol.Permisos)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE identity.roles
		SET nombre=$2, descripcion=$3, permisos=$4::jsonb
		WHERE id=$1`,
		rol.ID, rol.Nombre, rol.Descripcion, string(permisosJSON))
	return err
}

func (r *RolRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM identity.roles WHERE id=$1`, id)
	return err
}
