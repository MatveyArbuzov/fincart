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
}
