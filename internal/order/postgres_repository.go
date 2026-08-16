package order

import (
	"context"
	"database/sql"

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

func (r *PostgresRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	const query = `
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`

	var order Order

	err := tx.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&order.Currency,
		&order.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Order{}, ErrOrderNotFound
		}

		return Order{}, err
	}

	return order, nil
}

func (r *PostgresRepository) GetItems(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) ([]OrderItem, error) {
	const query = `
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	rows, err := tx.QueryContext(
		ctx,
		query,
		orderID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []OrderItem

	for rows.Next() {
		var item OrderItem

		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *PostgresRepository) UpdateStatus(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	status string,
) error {
	const query = `
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		status,
		orderID,
	)

	return err
}

func (r *PostgresRepository) GetByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	const query = `
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE id = $1
	`

	var order Order

	err := tx.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&order.ID,
		&order.UserID,
		&order.Status,
		&order.TotalAmount,
		&order.Currency,
		&order.CreatedAt,
	)

	if err != nil {
		return Order{}, err
	}

	return order, nil
}
