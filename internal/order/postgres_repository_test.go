package order

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	mock.ExpectBegin()

	createdAt := time.Now()

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
				createdAt,
			),
		)
	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	result, err := repository.Create(
		context.Background(),
		tx,
		Order{
			UserID:      10,
			Status:      "pending",
			TotalAmount: 300000,
			Currency:    "EUR",
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 100 {
		t.Fatalf("expected ID 100, got %d", result.ID)
	}

	if result.UserID != 10 {
		t.Fatalf("expected user ID 10, got %d", result.UserID)
	}

	if result.Status != "pending" {
		t.Fatalf("expected status pending, got %s", result.Status)
	}

	if result.TotalAmount != 300000 {
		t.Fatalf(
			"expected total amount 300000, got %d",
			result.TotalAmount,
		)
	}

	if result.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", result.Currency)
	}

	if !result.CreatedAt.Equal(createdAt) {
		t.Fatalf(
			"expected created_at %v, got %v",
			createdAt,
			result.CreatedAt,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	expectedErr := errors.New("insert order failed")

	mock.ExpectBegin()

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
		WillReturnError(expectedErr)

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	_, err = repository.Create(
		context.Background(),
		tx,
		Order{
			UserID:      10,
			Status:      "pending",
			TotalAmount: 300000,
			Currency:    "EUR",
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected %v, got %v",
			expectedErr,
			err,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_CreateItem(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	mock.ExpectBegin()

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
			sqlmock.NewRows([]string{"id"}).
				AddRow(int64(500)),
		)

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	result, err := repository.CreateItem(
		context.Background(),
		tx,
		OrderItem{
			OrderID:   100,
			ProductID: 1,
			Quantity:  2,
			UnitPrice: 150000,
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 500 {
		t.Fatalf("expected item ID 500, got %d", result.ID)
	}

	if result.OrderID != 100 {
		t.Fatalf("expected order ID 100, got %d", result.OrderID)
	}

	if result.ProductID != 1 {
		t.Fatalf("expected product ID 1, got %d", result.ProductID)
	}

	if result.Quantity != 2 {
		t.Fatalf("expected quantity 2, got %d", result.Quantity)
	}

	if result.UnitPrice != 150000 {
		t.Fatalf(
			"expected unit price 150000, got %d",
			result.UnitPrice,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_CreateItem_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	expectedErr := errors.New("insert order item failed")

	mock.ExpectBegin()

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
		WillReturnError(expectedErr)

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	_, err = repository.CreateItem(
		context.Background(),
		tx,
		OrderItem{
			OrderID:   100,
			ProductID: 1,
			Quantity:  2,
			UnitPrice: 150000,
		},
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected %v, got %v",
			expectedErr,
			err,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_GetByIDForUpdate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	createdAt := time.Now()

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`)).
		WithArgs(int64(100)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				int64(100),
				int64(10),
				"pending",
				int64(300000),
				"EUR",
				createdAt,
			),
		)

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	result, err := repository.GetByIDForUpdate(
		context.Background(),
		tx,
		100,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 100 {
		t.Fatalf("expected ID 100, got %d", result.ID)
	}

	if result.UserID != 10 {
		t.Fatalf("expected user ID 10, got %d", result.UserID)
	}

	if result.Status != "pending" {
		t.Fatalf("expected status pending, got %s", result.Status)
	}

	if result.TotalAmount != 300000 {
		t.Fatalf(
			"expected total amount 300000, got %d",
			result.TotalAmount,
		)
	}

	if result.Currency != "EUR" {
		t.Fatalf("expected EUR, got %s", result.Currency)
	}

	if !result.CreatedAt.Equal(createdAt) {
		t.Fatalf(
			"expected created_at %v, got %v",
			createdAt,
			result.CreatedAt,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_GetByIDForUpdate_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`)).
		WithArgs(int64(999)).
		WillReturnError(errors.New("sql: no rows in result set"))

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	_, err = repository.GetByIDForUpdate(
		context.Background(),
		tx,
		999,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_GetItems(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`)).
		WithArgs(int64(100)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"order_id",
				"product_id",
				"quantity",
				"unit_price",
			}).
				AddRow(
					int64(1),
					int64(100),
					int64(1),
					2,
					int64(150000),
				).
				AddRow(
					int64(2),
					int64(100),
					int64(2),
					3,
					int64(12000),
				),
		)

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	items, err := repository.GetItems(
		context.Background(),
		tx,
		100,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf(
			"expected 2 items, got %d",
			len(items),
		)
	}

	if items[0].ID != 1 {
		t.Fatalf("expected first item ID 1, got %d", items[0].ID)
	}

	if items[0].ProductID != 1 {
		t.Fatalf(
			"expected first product ID 1, got %d",
			items[0].ProductID,
		)
	}

	if items[0].Quantity != 2 {
		t.Fatalf(
			"expected first quantity 2, got %d",
			items[0].Quantity,
		)
	}

	if items[1].ID != 2 {
		t.Fatalf("expected second item ID 2, got %d", items[1].ID)
	}

	if items[1].ProductID != 2 {
		t.Fatalf(
			"expected second product ID 2, got %d",
			items[1].ProductID,
		)
	}

	if items[1].Quantity != 3 {
		t.Fatalf(
			"expected second quantity 3, got %d",
			items[1].Quantity,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_GetItems_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	expectedErr := errors.New("query order items failed")

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`)).
		WithArgs(int64(100)).
		WillReturnError(expectedErr)

	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	_, err = repository.GetItems(
		context.Background(),
		tx,
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected %v, got %v",
			expectedErr,
			err,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`)).
		WithArgs(
			"cancelled",
			int64(100),
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)
	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	err = repository.UpdateStatus(
		context.Background(),
		tx,
		100,
		"cancelled",
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func TestPostgresRepository_UpdateStatus_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repository := NewPostgresRepository()

	expectedErr := errors.New("update status failed")

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`)).
		WithArgs(
			"cancelled",
			int64(100),
		).
		WillReturnError(expectedErr)
	mock.ExpectRollback()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	err = repository.UpdateStatus(
		context.Background(),
		tx,
		100,
		"cancelled",
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected %v, got %v",
			expectedErr,
			err,
		)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("failed to rollback test transaction: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
