package order

import (
	"context"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type PostgresRepository struct {
}

func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

func (r *PostgresRepository) Create(
	ctx context.Context,
	tx database.Tx,
	order Order,
) (Order, error) {
	const query = `
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := tx.QueryRowContext(
		ctx,
		query,
		order.UserID,
		order.Status,
		order.TotalAmount,
		order.Currency,
	).Scan(
		&order.ID,
		&order.CreatedAt,
	)

	if err != nil {
		return Order{}, err
	}

	return order, nil
}

func (r *PostgresRepository) CreateItem(
	ctx context.Context,
	tx database.Tx,
	item OrderItem,
) (OrderItem, error) {
	const query = `
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := tx.QueryRowContext(
		ctx,
		query,
		item.OrderID,
		item.ProductID,
		item.Quantity,
		item.UnitPrice,
	).Scan(&item.ID)

	if err != nil {
		return OrderItem{}, err
	}

	return item, nil
}
