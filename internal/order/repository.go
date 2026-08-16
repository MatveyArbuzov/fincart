package order

import (
	"context"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type Repository interface {
	Create(
		ctx context.Context,
		tx database.Tx,
		order Order,
	) (Order, error)

	CreateItem(
		ctx context.Context,
		tx database.Tx,
		item OrderItem,
	) (OrderItem, error)

	GetByIDForUpdate(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (Order, error)

	GetItems(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
	) ([]OrderItem, error)

	UpdateStatus(
		ctx context.Context,
		tx database.Tx,
		orderID int64,
		status string,
	) error

	GetByID(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (Order, error)
}
