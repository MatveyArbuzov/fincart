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

	Create(
		ctx context.Context,
		tx database.Tx,
		product Product,
	) (Product, error)

	Update(
		ctx context.Context,
		tx database.Tx,
		product Product,
	) error

	Delete(
		ctx context.Context,
		tx database.Tx,
		id int64,
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

func (r *PostgresTransactionRepository) Create(
	ctx context.Context,
	tx database.Tx,
	product Product,
) (Product, error) {
	const query = `
		INSERT INTO products (
			name,
			description,
			price,
			currency,
			stock
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			id,
			name,
			description,
			price,
			currency,
			stock
	`

	var created Product

	err := tx.QueryRowContext(
		ctx,
		query,
		product.Name,
		product.Description,
		product.Price,
		product.Currency,
		product.Stock,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Description,
		&created.Price,
		&created.Currency,
		&created.Stock,
	)

	if err != nil {
		return Product{}, err
	}

	return created, nil
}

func (r *PostgresTransactionRepository) Update(
	ctx context.Context,
	tx database.Tx,
	product Product,
) error {
	const query = `
		UPDATE products
		SET
			name = $1,
			description = $2,
			price = $3,
			currency = $4,
			stock = $5
		WHERE id = $6
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		product.Name,
		product.Description,
		product.Price,
		product.Currency,
		product.Stock,
		product.ID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}

func (r *PostgresTransactionRepository) Delete(
	ctx context.Context,
	tx database.Tx,
	id int64,
) error {
	const query = `
		DELETE FROM products
		WHERE id = $1
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrProductNotFound
	}

	return nil
}
