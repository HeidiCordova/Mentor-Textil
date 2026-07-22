package application

import (
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrUserNotFound = errors.New("usuario no encontrado")

type UserService struct {
	users ports.UserRepository
	roles ports.RolRepository
}

func NewUserService(users ports.UserRepository, roles ports.RolRepository) *UserService {
	return &UserService{users: users, roles: roles}
}

func (s *UserService) List(ctx context.Context, empresaID *int) ([]domain.Usuario, error) {
	return s.users.FindAll(ctx, empresaID)
}

func (s *UserService) Get(ctx context.Context, id int) (*domain.Usuario, error) {
	return s.users.FindByID(ctx, id)
}

func (s *UserService) Create(ctx context.Context, u *domain.Usuario, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	u.Activo = true
	return s.users.Create(ctx, u)
}

func (s *UserService) Update(ctx context.Context, u *domain.Usuario) error {
	return s.users.Update(ctx, u)
}

func (s *UserService) Delete(ctx context.Context, id int) error {
	return s.users.Delete(ctx, id)
}

func (s *UserService) GetProfile(ctx context.Context, id int) (*domain.UserProfile, error) {
	return s.users.GetProfile(ctx, id)
}
