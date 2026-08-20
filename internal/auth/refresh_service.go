package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrRefreshTokenExpired = errors.New("refresh token expired")
	ErrRefreshTokenRevoked = errors.New("refresh token revoked")
)

const refreshTokenLifetime = 30 * 24 * time.Hour

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error
}

type UserRoleProvider interface {
	GetRoleByID(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (string, error)
}

type RefreshService struct {
	transactions TransactionManager
	repository   RefreshTokenRepository
	users        UserRoleProvider
	jwtManager   *JWTManager
}

func NewRefreshService(
	transactions TransactionManager,
	repository RefreshTokenRepository,
	users UserRoleProvider,
	jwtManager *JWTManager,
) *RefreshService {
	return &RefreshService{
		transactions: transactions,
		repository:   repository,
		users:        users,
		jwtManager:   jwtManager,
	}
}

func generateRefreshToken() (string, error) {
	data := make([]byte, 32)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(data), nil
}

func hashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))

	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func (s *RefreshService) Create(
	ctx context.Context,
	userID int64,
	role string,
) (string, string, error) {
	if userID <= 0 || role == "" {
		return "", "", ErrInvalidRefreshToken
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	tokenHash := hashRefreshToken(refreshToken)

	var accessToken string

	err = s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			if err := s.repository.Create(
				ctx,
				tx,
				RefreshToken{
					UserID:    userID,
					TokenHash: tokenHash,
					ExpiresAt: time.Now().Add(
						refreshTokenLifetime,
					),
				},
			); err != nil {
				return err
			}

			var err error

			accessToken, err = s.jwtManager.GenerateToken(
				userID,
				role,
			)
			if err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *RefreshService) Refresh(
	ctx context.Context,
	refreshToken string,
) (string, string, error) {
	if refreshToken == "" {
		return "", "", ErrInvalidRefreshToken
	}

	tokenHash := hashRefreshToken(refreshToken)

	var (
		accessToken     string
		newRefreshToken string
	)

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			storedToken, err := s.repository.GetByHash(
				ctx,
				tx,
				tokenHash,
			)
			if err != nil {
				if errors.Is(
					err,
					ErrRefreshTokenNotFound,
				) {
					return ErrInvalidRefreshToken
				}

				return err
			}

			if storedToken.RevokedAt != nil {
				return ErrRefreshTokenRevoked
			}

			if !time.Now().Before(
				storedToken.ExpiresAt,
			) {
				return ErrRefreshTokenExpired
			}

			role, err := s.users.GetRoleByID(
				ctx,
				tx,
				storedToken.UserID,
			)
			if err != nil {
				return err
			}

			if err := s.repository.Revoke(
				ctx,
				tx,
				storedToken.ID,
			); err != nil {
				return err
			}

			newRefreshToken, err = generateRefreshToken()
			if err != nil {
				return err
			}

			if err := s.repository.Create(
				ctx,
				tx,
				RefreshToken{
					UserID: storedToken.UserID,
					TokenHash: hashRefreshToken(
						newRefreshToken,
					),
					ExpiresAt: time.Now().Add(
						refreshTokenLifetime,
					),
				},
			); err != nil {
				return err
			}

			accessToken, err = s.jwtManager.GenerateToken(
				storedToken.UserID,
				role,
			)
			if err != nil {
				return err
			}

			return nil
		},
	)

	if err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

func (s *RefreshService) Revoke(
	ctx context.Context,
	refreshToken string,
) error {
	if refreshToken == "" {
		return ErrInvalidRefreshToken
	}

	tokenHash := hashRefreshToken(refreshToken)

	return s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			token, err := s.repository.GetByHash(
				ctx,
				tx,
				tokenHash,
			)
			if err != nil {
				if errors.Is(
					err,
					ErrRefreshTokenNotFound,
				) {
					return ErrInvalidRefreshToken
				}

				return err
			}

			if token.RevokedAt != nil {
				return nil
			}

			return s.repository.Revoke(
				ctx,
				tx,
				token.ID,
			)
		},
	)
}
