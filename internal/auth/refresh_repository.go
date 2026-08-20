package auth

import (
	"context"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type RefreshTokenRepository interface {
	Create(
		ctx context.Context,
		tx database.Tx,
		token RefreshToken,
	) error

	GetByHash(
		ctx context.Context,
		tx database.Tx,
		tokenHash string,
	) (RefreshToken, error)

	Revoke(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) error
}
