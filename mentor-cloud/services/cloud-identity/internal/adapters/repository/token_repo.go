package repository

import (
	"cloud-identity/internal/domain"
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TokenRepo struct {
	db *pgxpool.Pool
}

func NewTokenRepo(db *pgxpool.Pool) *TokenRepo {
	return &TokenRepo{db: db}
}

func (r *TokenRepo) Store(ctx context.Context, t *domain.RefreshToken) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO identity.refresh_tokens (usuario_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id`,
		t.UsuarioID, t.TokenHash, t.ExpiresAt,
	).Scan(&t.ID)
}

func (r *TokenRepo) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	t := &domain.RefreshToken{}
	err := r.db.QueryRow(ctx, `
		SELECT id, usuario_id, token_hash, expires_at, revocado
		FROM identity.refresh_tokens
		WHERE token_hash=$1`, hash,
	).Scan(&t.ID, &t.UsuarioID, &t.TokenHash, &t.ExpiresAt, &t.Revocado)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *TokenRepo) RevokeAllForUser(ctx context.Context, userID int) error {
	_, err := r.db.Exec(ctx, `
		UPDATE identity.refresh_tokens SET revocado=true WHERE usuario_id=$1`, userID)
	return err
}

func (r *TokenRepo) Revoke(ctx context.Context, tokenID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE identity.refresh_tokens SET revocado=true WHERE id=$1`, tokenID)
	return err
}
