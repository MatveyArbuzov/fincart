package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrUserNotFound = errors.New("user not found")

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

func (r *PostgresRepository) Create(
	ctx context.Context,
	tx database.Tx,
	email string,
	passwordHash string,
	role Role,
) (User, error) {
	const query = `
		INSERT INTO users (
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3)
		RETURNING id, email, role, created_at
	`

	var user User

	err := tx.QueryRowContext(
		ctx,
		query,
		email,
		passwordHash,
		role,
	).Scan(
		&user.ID,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (r *PostgresRepository) GetByEmail(
	ctx context.Context,
	tx database.Tx,
	email string,
) (User, string, error) {
	const query = `
		SELECT
			id,
			email,
			password_hash,
			role,
			created_at
		FROM users
		WHERE email = $1
	`

	var (
		user         User
		passwordHash string
	)

	err := tx.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&passwordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, "", ErrUserNotFound
		}

		return User{}, "", err
	}

	return user, passwordHash, nil
}
