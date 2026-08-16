package order

import (
	"context"
	"os"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func TestCancelOrder_Integration(t *testing.T) {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fincart:fincart@localhost:5432/fincart?sslmode=disable"
	}

	db, err := database.NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	transactionManager := database.NewManager(db)

	productRepository :=
		product.NewPostgresTransactionRepository()

	orderRepository :=
		NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
		payment.NewFakeService(payment.ResultSuccess),
	)

	// Создаём тестовый товар.
	var productID int64

	err = db.QueryRowContext(
		ctx,
		`
		INSERT INTO products (
			name,
			description,
			price,
			currency,
			stock
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
		`,
		"Integration Test Product",
		"Test product",
		int64(1000),
		"EUR",
		10,
	).Scan(&productID)

	if err != nil {
		t.Fatalf("failed to create product: %v", err)
	}

	t.Cleanup(func() {
		_, err := db.ExecContext(
			ctx,
			"DELETE FROM order_items WHERE product_id = $1",
			productID,
		)
		if err != nil {
			t.Errorf("failed to cleanup order items: %v", err)
		}

		_, err = db.ExecContext(
			ctx,
			"DELETE FROM orders WHERE user_id = $1",
			int64(999999),
		)
		if err != nil {
			t.Errorf("failed to cleanup orders: %v", err)
		}

		_, err = db.ExecContext(
			ctx,
			"DELETE FROM products WHERE id = $1",
			productID,
		)
		if err != nil {
			t.Errorf("failed to cleanup product: %v", err)
		}
	})

	// Создаём заказ.
	createdOrder, err := service.CreateOrder(
		ctx,
		999999,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: productID,
					Quantity:  3,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	if createdOrder.Status != "pending" {
		t.Fatalf(
			"expected pending status, got %s",
			createdOrder.Status,
		)
	}

	// После покупки stock должен быть 7.
	var stock int

	err = db.QueryRowContext(
		ctx,
		`
		SELECT stock
		FROM products
		WHERE id = $1
		`,
		productID,
	).Scan(&stock)

	if err != nil {
		t.Fatalf("failed to read stock: %v", err)
	}

	if stock != 7 {
		t.Fatalf(
			"expected stock 7 after order creation, got %d",
			stock,
		)
	}

	// Отменяем заказ.
	err = service.CancelOrder(
		ctx,
		createdOrder.ID,
	)

	if err != nil {
		t.Fatalf("failed to cancel order: %v", err)
	}

	// Проверяем статус.
	var status string

	err = db.QueryRowContext(
		ctx,
		`
		SELECT status
		FROM orders
		WHERE id = $1
		`,
		createdOrder.ID,
	).Scan(&status)

	if err != nil {
		t.Fatalf("failed to read order status: %v", err)
	}

	if status != "cancelled" {
		t.Fatalf(
			"expected cancelled status, got %s",
			status,
		)
	}

	// После отмены stock должен вернуться к 10.
	err = db.QueryRowContext(
		ctx,
		`
		SELECT stock
		FROM products
		WHERE id = $1
		`,
		productID,
	).Scan(&stock)

	if err != nil {
		t.Fatalf("failed to read stock after cancellation: %v", err)
	}

	if stock != 10 {
		t.Fatalf(
			"expected stock 10 after cancellation, got %d",
			stock,
		)
	}
}
