package product

import (
	"context"
	"testing"
)

type fakeProductRepository struct {
	product Product
	err     error
}

func (f *fakeProductRepository) GetByID(ctx context.Context, id int64) (Product, error) {
	return f.product, f.err
}

func (f *fakeProductRepository) List(ctx context.Context) ([]Product, error) {
	return []Product{f.product}, f.err
}

func TestService_GetProductByID(t *testing.T) {
	repository := &fakeProductRepository{
		product: Product{
			ID:       1,
			Name:     "MacBook Pro",
			Price:    150000,
			Currency: "EUR",
			Stock:    10,
		},
	}

	service := NewService(repository)

	product, err := service.GetProductByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if product.ID != 1 {
		t.Fatalf("expected product ID 1, got %d", product.ID)
	}
}

func TestService_GetProductByID_NotFound(t *testing.T) {
	repository := &fakeProductRepository{
		err: ErrProductNotFound,
	}

	service := NewService(repository)

	_, err := service.GetProductByID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
