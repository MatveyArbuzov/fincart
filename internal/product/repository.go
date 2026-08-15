package product

import "context"

type Repository interface {
	GetByID(ctx context.Context, id int64) (Product, error)
	List(ctx context.Context) ([]Product, error)
}
