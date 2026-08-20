package user

import (
	"context"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type Repository interface {
	Create(
		ctx context.Context,
		tx database.Tx,
		email string,
		passwordHash string,
		role Role,
	) (User, error)

	GetByEmail(
		ctx context.Context,
		tx database.Tx,
		email string,
	) (User, string, error)

	GetRoleByID(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (string, error)
}
