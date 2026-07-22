package repository

import (
	"cloud-identity/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*domain.Usuario, error) {
	u := &domain.Usuario{}
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.nombre, COALESCE(u.apellido,''), u.password_hash,
		       u.rol_id, COALESCE(ro.nombre,''), u.empresa_id, u.planta_id, u.activo
		FROM identity.usuarios u
		LEFT JOIN identity.roles ro ON ro.id = u.rol_id
		WHERE u.username = $1 OR u.email = $1`, username,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Nombre, &u.Apellido, &u.PasswordHash,
		&u.RolID, &u.Rol, &u.EmpresaID, &u.PlantaID, &u.Activo)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int) (*domain.Usuario, error) {
	u := &domain.Usuario{}
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.username, u.email, u.nombre, COALESCE(u.apellido,''), u.password_hash,
		       u.rol_id, COALESCE(ro.nombre,''), u.empresa_id, u.planta_id, u.activo
		FROM identity.usuarios u
		LEFT JOIN identity.roles ro ON ro.id = u.rol_id
		WHERE u.id = $1`, id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Nombre, &u.Apellido, &u.PasswordHash,
		&u.RolID, &u.Rol, &u.EmpresaID, &u.PlantaID, &u.Activo)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) FindAll(ctx context.Context, empresaID *int) ([]domain.Usuario, error) {
	query := `
		SELECT u.id, u.username, u.email, u.nombre, COALESCE(u.apellido,''),
		       u.rol_id, COALESCE(ro.nombre,''), u.empresa_id, u.planta_id, u.activo
		FROM identity.usuarios u
		LEFT JOIN identity.roles ro ON ro.id = u.rol_id`
	args := []interface{}{}
	if empresaID != nil {
		query += ` WHERE u.empresa_id = $1`
		args = append(args, *empresaID)
	}
	query += ` ORDER BY u.id`
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Nombre, &u.Apellido, &u.RolID, &u.Rol, &u.EmpresaID, &u.PlantaID, &u.Activo); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.Usuario) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO identity.usuarios (username, email, nombre, apellido, password_hash, rol_id, empresa_id, planta_id, activo)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		u.Username, u.Email, u.Nombre, u.Apellido, u.PasswordHash, u.RolID, u.EmpresaID, u.PlantaID, u.Activo,
	).Scan(&u.ID)
}

func (r *UserRepo) Update(ctx context.Context, u *domain.Usuario) error {
	_, err := r.db.Exec(ctx, `
		UPDATE identity.usuarios
		SET username=$2, email=$3, nombre=$4, apellido=$5, rol_id=$6, empresa_id=$7, planta_id=$8, activo=$9, actualizado_en=NOW()
		WHERE id=$1`,
		u.ID, u.Username, u.Email, u.Nombre, u.Apellido, u.RolID, u.EmpresaID, u.PlantaID, u.Activo)
	return err
}

func (r *UserRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM identity.usuarios WHERE id=$1`, id)
	return err
}

func (r *UserRepo) GetProfile(ctx context.Context, id int) (*domain.UserProfile, error) {
	p := &domain.UserProfile{}
	err := r.db.QueryRow(ctx, `
		SELECT u.id, u.nombre, COALESCE(u.apellido,''), u.email,
		       COALESCE(ro.nombre,''),
		       COALESCE(p.nombre,''),
		       COALESCE(e.nombre,''),
		       u.empresa_id
		FROM identity.usuarios u
		LEFT JOIN identity.roles ro ON ro.id = u.rol_id
		LEFT JOIN identity.empresas e ON e.id = u.empresa_id
		LEFT JOIN config.plantas p ON p.empresa_id = u.empresa_id AND p.activo = true
		WHERE u.id = $1
		LIMIT 1`, id,
	).Scan(&p.ID, &p.Name, &p.LastName, &p.Email, &p.Role, &p.Plant, &p.Company, &p.EmpresaID)
	if err != nil {
		return nil, err
	}
	return p, nil
}
