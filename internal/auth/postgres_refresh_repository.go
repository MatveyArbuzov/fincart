package auth

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type PostgresRefreshTokenRepository struct{}

func NewPostgresRefreshTokenRepository() *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{}
}

func (r *PostgresRefreshTokenRepository) Create(
	ctx context.Context,
	tx database.Tx,
	token RefreshToken,
) error {
	const query = `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	)

	return err
}

func (r *PostgresRefreshTokenRepository) GetByHash(
	ctx context.Context,
	tx database.Tx,
	tokenHash string,
) (RefreshToken, error) {
	const query = `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			created_at,
			revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token RefreshToken

	err := tx.QueryRowContext(
		ctx,
		query,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RefreshToken{}, ErrRefreshTokenNotFound
		}

		return RefreshToken{}, err
	}

	return token, nil
}

func (r *PostgresRefreshTokenRepository) Revoke(
	ctx context.Context,
	tx database.Tx,
	id int64,
) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		id,
	)

	return err
}
