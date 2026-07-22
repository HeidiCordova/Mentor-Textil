package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"cloud-integration/internal/domain"
	"cloud-integration/internal/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type apiKeyRepo struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepo(db *pgxpool.Pool) ports.APIKeyRepository {
	return &apiKeyRepo{db: db}
}

func generateKey() (plaintext, prefix, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw := "mk_" + hex.EncodeToString(b)
	plaintext = raw
	prefix = raw[:12]
	sum := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(sum[:])
	return
}

func hashKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (r *apiKeyRepo) Create(ctx context.Context, req domain.CreateKeyRequest, keyHash, keyPrefix string) (*domain.APIKey, error) {
	row := r.db.QueryRow(ctx, `
		INSERT INTO integration.api_keys (empresa_id, nombre, key_prefix, key_hash, scopes)
		VALUES ($1, $2, $3, $4, ARRAY['oee:read','snapshots:read','paradas:read','energy:read'])
		RETURNING id, empresa_id, nombre, key_prefix, scopes, activo, creado_en, ultimo_uso`,
		req.EmpresaID, req.Nombre, keyPrefix, keyHash,
	)
	return scanKey(row)
}

func (r *apiKeyRepo) List(ctx context.Context, empresaID int) ([]domain.APIKey, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, empresa_id, nombre, key_prefix, scopes, activo, creado_en, ultimo_uso
		FROM integration.api_keys
		WHERE empresa_id = $1
		ORDER BY creado_en DESC`, empresaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.APIKey
	for rows.Next() {
		k := domain.APIKey{}
		if err := rows.Scan(&k.ID, &k.EmpresaID, &k.Nombre, &k.KeyPrefix, &k.Scopes, &k.Activo, &k.CreadoEn, &k.UltimoUso); err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, nil
}

func (r *apiKeyRepo) Revoke(ctx context.Context, id int, empresaID int) error {
	cmd, err := r.db.Exec(ctx, `
		UPDATE integration.api_keys SET activo = false
		WHERE id = $1 AND empresa_id = $2`, id, empresaID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("clave no encontrada o sin permiso")
	}
	return nil
}

func (r *apiKeyRepo) Validate(ctx context.Context, raw string) (*domain.APIKey, error) {
	if len(raw) < 12 {
		return nil, errors.New("clave invalida")
	}
	prefix := raw[:12]
	stored := hashKey(raw)

	var k domain.APIKey
	row := r.db.QueryRow(ctx, `
		SELECT id, empresa_id, nombre, key_prefix, scopes, activo, creado_en, ultimo_uso
		FROM integration.api_keys
		WHERE key_prefix = $1 AND key_hash = $2`, prefix, stored)
	scanned, err := scanKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("clave invalida")
		}
		return nil, err
	}
	k = *scanned

	if k.Activo {
		_, _ = r.db.Exec(ctx, `UPDATE integration.api_keys SET ultimo_uso = now() WHERE id = $1`, k.ID)
	}
	return &k, nil
}

// GenerateKey expone la generación para uso desde el handler.
func GenerateKey() (plaintext, prefix, hash string, err error) {
	return generateKey()
}

func scanKey(row pgx.Row) (*domain.APIKey, error) {
	k := &domain.APIKey{}
	err := row.Scan(&k.ID, &k.EmpresaID, &k.Nombre, &k.KeyPrefix, &k.Scopes, &k.Activo, &k.CreadoEn, &k.UltimoUso)
	if err != nil {
		return nil, err
	}
	return k, nil
}
