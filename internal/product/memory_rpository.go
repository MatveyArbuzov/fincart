package product

import (
	"context"
	"errors"
)

var ErrProductNotFound = errors.New("product not found")

type MemoryRepository struct {
	products []Product
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		products: []Product{
			{
				ID:          1,
				Name:        "MacBook Pro",
				Description: "Apple laptop",
				Price:       150000,
				Currency:    "EUR",
				Stock:       10,
			},
			{
				ID:          2,
				Name:        "Mechanical Keyboard",
				Description: "Mechanical keyboard",
				Price:       12000,
				Currency:    "EUR",
				Stock:       50,
			},
		},
	}
}

func (r *MemoryRepository) GetByID(ctx context.Context, id int64) (Product, error) {
	for _, product := range r.products {
		if product.ID == id {
			return product, nil
		}
	}

	return Product{}, ErrProductNotFound
}

func (r *MemoryRepository) List(ctx context.Context) ([]Product, error) {
	return r.products, nil
}
