package service

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"company-registry/internal/domain/entity"
	"company-registry/internal/domain/errs"
)

type UserSource interface {
	GetUser(ctx context.Context, username, passwordHash string) (*entity.User, error)
}

type tokenClaims struct {
	jwt.RegisteredClaims
	UserID int `json:"user_id"`
}

type AuthService struct {
	userSource UserSource
	salt       string
	signingKey []byte
	tokenTTL   time.Duration
}

func NewAuthService(
	userSource UserSource,
	salt string,
	signingKey string,
	tokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		userSource: userSource,
		salt:       salt,
		signingKey: []byte(signingKey),
		tokenTTL:   tokenTTL,
	}
}

func (s *AuthService) GenerateToken(ctx context.Context, username, password string) (string, error) {
	user, err := s.userSource.GetUser(ctx, username, s.generatePasswordHash(password))
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return "", errs.ErrInvalidCredentials
		}

		return "", fmt.Errorf("userSource.GetUser: %w", err)
	}

	now := time.Now()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserID: user.ID,
	})

	signed, err := token.SignedString(s.signingKey)
	if err != nil {
		return "", fmt.Errorf("token.SignedString: %w", err)
	}

	return signed, nil
}

func (s *AuthService) ParseToken(accessToken string) (int, error) {
	var claims tokenClaims

	_, err := jwt.ParseWithClaims(
		accessToken,
		&claims,
		func(*jwt.Token) (any, error) {
			return s.signingKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return 0, fmt.Errorf("jwt.ParseWithClaims: %w", err)
	}

	return claims.UserID, nil
}

func (s *AuthService) generatePasswordHash(password string) string {
	hash := sha1.New() //nolint:gosec
	hash.Write([]byte(password))

	return fmt.Sprintf("%x", hash.Sum([]byte(s.salt)))
}
