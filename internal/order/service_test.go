package order

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func TestCreateOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				10,
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`)).
		WithArgs(
			int64(10),
			"pending",
			int64(300000),
			"EUR",
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				int64(100),
				time.Now(),
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`)).
		WithArgs(
			int64(100),
			int64(1),
			2,
			int64(150000),
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
			}).AddRow(
				int64(1),
			),
		)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE products
		SET stock = stock - $1
		WHERE id = $2
	`)).
		WithArgs(
			2,
			int64(1),
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	mock.ExpectCommit()

	result, err := service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 100 {
		t.Fatalf(
			"expected order ID 100, got %d",
			result.ID,
		)
	}

	if result.UserID != 10 {
		t.Fatalf(
			"expected user ID 10, got %d",
			result.UserID,
		)
	}

	if result.Status != "pending" {
		t.Fatalf(
			"expected status pending, got %s",
			result.Status,
		)
	}

	if result.TotalAmount != 300000 {
		t.Fatalf(
			"expected total amount 300000, got %d",
			result.TotalAmount,
		)
	}

	if result.Currency != "EUR" {
		t.Fatalf(
			"expected currency EUR, got %s",
			result.Currency,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				1,
			),
		)

	mock.ExpectRollback()

	_, err = service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		},
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf(
			"expected ErrInsufficientStock, got %v",
			err,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}

func TestCreateOrder_OrderItemCreationFailed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				10,
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`)).
		WithArgs(
			int64(10),
			"pending",
			int64(300000),
			"EUR",
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				int64(100),
				time.Now(),
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`)).
		WithArgs(
			int64(100),
			int64(1),
			2,
			int64(150000),
		).
		WillReturnError(
			errors.New("failed to create order item"),
		)

	mock.ExpectRollback()

	// Act
	_, err = service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		},
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create order item") {
		t.Fatalf(
			"expected order item error, got %v",
			err,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}

func TestCreateOrder_DuplicateProducts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
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
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				10,
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`)).
		WithArgs(
			int64(10),
			"pending",
			int64(750000),
			"EUR",
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				int64(100),
				time.Now(),
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`)).
		WithArgs(
			int64(100),
			int64(1),
			5,
			int64(150000),
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
			}).AddRow(
				int64(1),
			),
		)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE products
		SET stock = stock - $1
		WHERE id = $2
	`)).
		WithArgs(
			5,
			int64(1),
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	mock.ExpectCommit()

	result, err := service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
				{
					ProductID: 1,
					Quantity:  3,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.TotalAmount != 750000 {
		t.Fatalf(
			"expected total amount 750000, got %d",
			result.TotalAmount,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}
