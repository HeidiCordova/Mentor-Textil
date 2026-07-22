package application

import (
	"cloud-identity/internal/domain"
	"cloud-identity/internal/ports"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("credenciales invalidas")
	ErrTokenExpired       = errors.New("token expirado")
	ErrTokenRevoked       = errors.New("token revocado")
)

type AuthService struct {
	users     ports.UserRepository
	tokens    ports.TokenRepository
	jwtSecret []byte
}

func NewAuthService(users ports.UserRepository, tokens ports.TokenRepository, secret string) *AuthService {
	return &AuthService{
		users:     users,
		tokens:    tokens,
		jwtSecret: []byte(secret),
	}
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResponse, error) {
	user, err := s.users.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.Activo {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	profile, err := s.users.GetProfile(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         *profile,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*domain.LoginResponse, error) {
	hash := hashToken(rawToken)
	stored, err := s.tokens.FindByHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if stored.Revocado {
		return nil, ErrTokenRevoked
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	_ = s.tokens.Revoke(ctx, stored.ID)

	user, err := s.users.FindByID(ctx, stored.UsuarioID)
	if err != nil {
		return nil, err
	}

	profile, err := s.users.GetProfile(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, err
	}

	newRefresh, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &domain.LoginResponse{
		Token:        accessToken,
		RefreshToken: newRefresh,
		User:         *profile,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, userID int) error {
	return s.tokens.RevokeAllForUser(ctx, userID)
}

func (s *AuthService) Me(ctx context.Context, userID int) (*domain.UserProfile, error) {
	return s.users.GetProfile(ctx, userID)
}

func (s *AuthService) generateAccessToken(user *domain.Usuario) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"role": user.Rol,
		"exp":  time.Now().Add(8 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	if user.EmpresaID != nil {
		claims["empresa_id"] = *user.EmpresaID
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *AuthService) generateRefreshToken(ctx context.Context, userID int) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	rawToken := hex.EncodeToString(b)
	hash := hashToken(rawToken)

	rt := &domain.RefreshToken{
		UsuarioID: userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := s.tokens.Store(ctx, rt); err != nil {
		return "", err
	}

	return rawToken, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
