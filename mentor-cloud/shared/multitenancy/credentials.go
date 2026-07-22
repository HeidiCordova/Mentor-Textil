package multitenancy

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CredentialProvider resuelve la contraseña de Postgres para una BD de planta.
// La implementación concreta varía según el entorno (AES-GCM local o AWS Secrets Manager).
type CredentialProvider interface {
	GetPassword(ctx context.Context, plantaID int) (string, error)
}

type AESCredentialProvider struct {
	key      []byte
	masterDB *pgxpool.Pool
}

func NewAESCredentialProvider(key []byte, masterDB *pgxpool.Pool) (*AESCredentialProvider, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	return &AESCredentialProvider{key: key, masterDB: masterDB}, nil
}

func (p *AESCredentialProvider) GetPassword(ctx context.Context, plantaID int) (string, error) {
	var enc string
	err := p.masterDB.QueryRow(ctx,
		`SELECT pg_password_enc FROM admin.planta_databases WHERE planta_id = $1 AND provisioned = true`,
		plantaID,
	).Scan(&enc)
	if err != nil {
		return "", fmt.Errorf("planta %d credentials: %w", plantaID, err)
	}
	return p.Decrypt(enc)
}

// AWSSecretsCredentials resuelve credenciales desde AWS Secrets Manager.
type AWSSecretsCredentials struct {
	prefix string
}

func NewAWSSecretsCredentials(prefix string) *AWSSecretsCredentials {
	return &AWSSecretsCredentials{prefix: prefix}
}

func (a *AWSSecretsCredentials) GetPassword(_ context.Context, plantaID int) (string, error) {
	return "", fmt.Errorf("AWS Secrets Manager not implemented: %s/planta/%d", a.prefix, plantaID)
}

func (p *AESCredentialProvider) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (p *AESCredentialProvider) Decrypt(ciphertextB64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	block, err := aes.NewCipher(p.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
