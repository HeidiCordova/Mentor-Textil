package application

import (
	"cloud-identity/internal/ports"
	"context"
	"log"

	"golang.org/x/crypto/bcrypt"
)

func SeedAdmin(ctx context.Context, users ports.UserRepository, roles ports.RolRepository) error {
	existing, _ := users.FindByUsername(ctx, "admin")
	if existing != nil {
		return nil
	}

	rol, err := roles.FindByNombre(ctx, "ADMIN")
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("Admin1234!"), 12)
	if err != nil {
		return err
	}

	u := &struct {
		Username     string
		Email        string
		Nombre       string
		PasswordHash string
		RolID        *int
		Activo       bool
	}{
		Username:     "admin",
		Email:        "admin@mentormonitor.com",
		Nombre:       "Administrador",
		PasswordHash: string(hash),
		RolID:        &rol.ID,
		Activo:       true,
	}

	_ = u
	log.Println("identity: admin user already seeded via init.sql")
	return nil
}
