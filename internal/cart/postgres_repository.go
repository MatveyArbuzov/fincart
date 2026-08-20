package cart

import (
	"context"
	"database/sql"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository {
	return &PostgresRepository{}
}

func (r *PostgresRepository) GetDraft(
	ctx context.Context,
	tx database.Tx,
	userID int64,
) (Cart, error) {
	const query = `
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE user_id = $1
		  AND status = 'draft'
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`

	var cart Cart

	err := tx.QueryRowContext(
		ctx,
		query,
		userID,
	).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.Status,
		&cart.TotalAmount,
		&cart.Currency,
		&cart.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Cart{}, nil
		}

		return Cart{}, err
	}

	return cart, nil
}

func (r *PostgresRepository) CreateDraft(
	ctx context.Context,
	tx database.Tx,
	userID int64,
) (Cart, error) {
	const query = `
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, 'draft', 0, '')
		RETURNING
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
	`

	var cart Cart

	err := tx.QueryRowContext(
		ctx,
		query,
		userID,
	).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.Status,
		&cart.TotalAmount,
		&cart.Currency,
		&cart.CreatedAt,
	)

	if err != nil {
		return Cart{}, err
	}

	return cart, nil
}

func (r *PostgresRepository) GetItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
) (Item, error) {
	const query = `
		SELECT
			oi.id,
			oi.product_id,
			oi.quantity,
			oi.unit_price,
			p.name,
			p.description,
			p.currency,
			p.stock
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
		  AND oi.product_id = $2
	`

	var item Item

	err := tx.QueryRowContext(
		ctx,
		query,
		orderID,
		productID,
	).Scan(
		&item.ID,
		&item.ProductID,
		&item.Quantity,
		&item.UnitPrice,
		&item.Name,
		&item.Description,
		&item.Currency,
		&item.Stock,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Item{}, sql.ErrNoRows
		}

		return Item{}, err
	}

	return item, nil
}

func (r *PostgresRepository) AddItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
	quantity int,
	unitPrice int64,
) (Item, error) {
	const query = `
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, product_id, quantity, unit_price
	`

	var item Item

	err := tx.QueryRowContext(
		ctx,
		query,
		orderID,
		productID,
		quantity,
		unitPrice,
	).Scan(
		&item.ID,
		&item.ProductID,
		&item.Quantity,
		&item.UnitPrice,
	)

	if err != nil {
		return Item{}, err
	}

	return item, nil
}

func (r *PostgresRepository) UpdateItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
	quantity int,
) error {
	const query = `
		UPDATE order_items
		SET quantity = $1
		WHERE order_id = $2
		  AND product_id = $3
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		quantity,
		orderID,
		productID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresRepository) DeleteItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
) error {
	const query = `
		DELETE FROM order_items
		WHERE order_id = $1
		  AND product_id = $2
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		orderID,
		productID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresRepository) GetItems(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) ([]Item, error) {
	const query = `
		SELECT
			oi.id,
			oi.product_id,
			oi.quantity,
			oi.unit_price,
			p.name,
			p.description,
			p.currency,
			p.stock
		FROM order_items oi
		JOIN products p ON p.id = oi.product_id
		WHERE oi.order_id = $1
		ORDER BY oi.id
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

	items := make([]Item, 0)

	for rows.Next() {
		var item Item

		if err := rows.Scan(
			&item.ID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Name,
			&item.Description,
			&item.Currency,
			&item.Stock,
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

func (r *PostgresRepository) UpdateTotal(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	totalAmount int64,
	currency string,
) error {
	const query = `
		UPDATE orders
		SET
			total_amount = $1,
			currency = $2
		WHERE id = $3
		  AND status = 'draft'
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		totalAmount,
		currency,
		orderID,
	)

	return err
}

func (r *PostgresRepository) Checkout(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) error {
	const query = `
		UPDATE orders
		SET status = 'pending'
		WHERE id = $1
		  AND status = 'draft'
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		orderID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *PostgresRepository) GetDraftByID(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) (Cart, error) {
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
		  AND status = 'draft'
		FOR UPDATE
	`

	var cart Cart

	err := tx.QueryRowContext(
		ctx,
		query,
		orderID,
	).Scan(
		&cart.ID,
		&cart.UserID,
		&cart.Status,
		&cart.TotalAmount,
		&cart.Currency,
		&cart.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Cart{}, ErrCartNotFound
		}

		return Cart{}, err
	}

	return cart, nil
}

func (r *PostgresRepository) UpdateItemPrice(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
	unitPrice int64,
) error {
	const query = `
		UPDATE order_items
		SET unit_price = $1
		WHERE order_id = $2
		  AND product_id = $3
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		unitPrice,
		orderID,
		productID,
	)

	return err
}
