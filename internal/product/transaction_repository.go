package product

import (
	"context"
	"database/sql"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type TransactionRepository interface {
	GetByIDForUpdate(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (Product, error)

	DecreaseStock(
		ctx context.Context,
		tx database.Tx,
		id int64,
		quantity int,
	) error

	IncreaseStock(
		ctx context.Context,
		tx database.Tx,
		id int64,
		quantity int,
	) error
}

type PostgresTransactionRepository struct{}

func NewPostgresTransactionRepository() *PostgresTransactionRepository {
	return &PostgresTransactionRepository{}
}

func (r *PostgresTransactionRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Product, error) {
	const query = `
		SELECT
			id,
			name,
			description,
			price,
			currency,
			stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`

	var product Product

	err := tx.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Currency,
		&product.Stock,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return Product{}, ErrProductNotFound
		}

		return Product{}, err
	}

	return product, nil
}

func (r *PostgresTransactionRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	const query = `
		UPDATE products
		SET stock = stock - $1
		WHERE id = $2
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		quantity,
		id,
	)

	return err
}

func (r *PostgresTransactionRepository) IncreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	const query = `
		UPDATE products
		SET stock = stock + $1
		WHERE id = $2
	`

	_, err := tx.ExecContext(
		ctx,
		query,
		quantity,
		id,
	)

	return err
}
