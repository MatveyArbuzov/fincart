package product

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryRepository_GetByID(t *testing.T) {
	repository := NewMemoryRepository()

	product, err := repository.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if product.ID != 1 {
		t.Fatalf("expected product ID 1, got %d", product.ID)
	}

	if product.Name != "MacBook Pro" {
		t.Fatalf("expected product name MacBook Pro, got %s", product.Name)
	}
}

func TestMemoryRepository_GetByID_NotFound(t *testing.T) {
	repository := NewMemoryRepository()

	_, err := repository.GetByID(context.Background(), 999)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf("expected ErrProductNotFound, got %v", err)
	}
}
