package product

import (
	"context"
	"database/sql"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{
		db: db,
	}
}

func (r *PostgresRepository) GetByID(ctx context.Context, id int64) (Product, error) {
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
	`

	var product Product

	err := r.db.QueryRowContext(ctx, query, id).Scan(
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

func (r *PostgresRepository) List(ctx context.Context) ([]Product, error) {
	const query = `
		SELECT
			id,
			name,
			description,
			price,
			currency,
			stock
		FROM products
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product

	for rows.Next() {
		var product Product

		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Currency,
			&product.Stock,
		); err != nil {
			return nil, err
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
