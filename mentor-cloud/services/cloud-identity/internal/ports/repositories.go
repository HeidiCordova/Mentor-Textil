package ports

import (
	"cloud-identity/internal/domain"
	"context"
)

type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*domain.Usuario, error)
	FindByID(ctx context.Context, id int) (*domain.Usuario, error)
	FindAll(ctx context.Context, empresaID *int) ([]domain.Usuario, error)
	Create(ctx context.Context, u *domain.Usuario) error
	Update(ctx context.Context, u *domain.Usuario) error
	Delete(ctx context.Context, id int) error
	GetProfile(ctx context.Context, id int) (*domain.UserProfile, error)
}

type EmpresaRepository interface {
	FindAll(ctx context.Context) ([]domain.Empresa, error)
	FindByID(ctx context.Context, id int) (*domain.Empresa, error)
	Create(ctx context.Context, e *domain.Empresa) error
	Update(ctx context.Context, e *domain.Empresa) error
	Delete(ctx context.Context, id int) error
}

type RolRepository interface {
	FindAll(ctx context.Context) ([]domain.Rol, error)
	FindByID(ctx context.Context, id int) (*domain.Rol, error)
	FindByNombre(ctx context.Context, nombre string) (*domain.Rol, error)
	Create(ctx context.Context, r *domain.Rol) error
	Update(ctx context.Context, r *domain.Rol) error
	Delete(ctx context.Context, id int) error
}

type TokenRepository interface {
	Store(ctx context.Context, t *domain.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error)
	RevokeAllForUser(ctx context.Context, userID int) error
	Revoke(ctx context.Context, tokenID string) error
}
