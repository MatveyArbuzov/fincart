package integration

import (
	"context"
	"database/sql"
	"net/http"
	"testing"
)

type checkoutResponse struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Status      string `json:"status"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`
}

func TestCart_Checkout(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	add := request(
		t,
		http.MethodPost,
		"/api/v1/cart/items",
		map[string]interface{}{
			"product_id": 1,
			"quantity":   2,
		},
		user.AccessToken,
	)

	assertStatus(
		t,
		add,
		http.StatusOK,
	)

	if add.Body != nil {
		defer add.Body.Close()
	}

	checkout := request(
		t,
		http.MethodPost,
		"/api/v1/cart/checkout",
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		checkout,
		http.StatusCreated,
	)

	order := decodeJSON[checkoutResponse](
		t,
		checkout,
	)

	if order.UserID != user.ID {
		t.Fatalf(
			"unexpected user ID: got %d, want %d",
			order.UserID,
			user.ID,
		)
	}

	if order.Status != "pending" {
		t.Fatalf(
			"unexpected order status: %s",
			order.Status,
		)
	}

	if order.TotalAmount != 300000 {
		t.Fatalf(
			"unexpected total: got %d, want 300000",
			order.TotalAmount,
		)
	}

	if order.Currency != "EUR" {
		t.Fatalf(
			"unexpected currency: %s",
			order.Currency,
		)
	}

	// Verify stock was decreased in PostgreSQL.

	var stock int

	err := testServer.db.QueryRowContext(
		context.Background(),
		`
		SELECT stock
		FROM products
		WHERE id = 1
		`,
	).Scan(&stock)

	if err != nil {
		t.Fatalf(
			"failed to read product stock: %v",
			err,
		)
	}

	if stock != 8 {
		t.Fatalf(
			"unexpected stock: got %d, want 8",
			stock,
		)
	}

	// Verify draft cart no longer exists.

	var draftCount int

	err = testServer.db.QueryRowContext(
		context.Background(),
		`
		SELECT COUNT(*)
		FROM orders
		WHERE user_id = $1
		  AND status = 'draft'
		`,
		user.ID,
	).Scan(&draftCount)

	if err != nil {
		t.Fatalf(
			"failed to check draft cart: %v",
			err,
		)
	}

	if draftCount != 0 {
		t.Fatalf(
			"expected no draft cart, got %d",
			draftCount,
		)
	}

	_ = sql.ErrNoRows
}
