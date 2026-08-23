package integration

import (
	"net/http"
	"testing"
)

type cartResponse struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Status      string `json:"status"`
	TotalAmount int64  `json:"total_amount"`
	Currency    string `json:"currency"`

	Items []struct {
		ID        int64 `json:"id"`
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
		UnitPrice int64 `json:"unit_price"`
		Stock     int   `json:"stock"`
	} `json:"items"`
}

func TestCart_AddUpdateDelete(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	// Initially cart does not exist.

	getCart := request(
		t,
		http.MethodGet,
		"/api/v1/cart",
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		getCart,
		http.StatusOK,
	)

	initialCart := decodeJSON[cartResponse](
		t,
		getCart,
	)

	if initialCart.UserID != user.ID {
		t.Fatalf(
			"unexpected user ID: got %d, want %d",
			initialCart.UserID,
			user.ID,
		)
	}

	if len(initialCart.Items) != 0 {
		t.Fatalf(
			"expected empty cart, got %d items",
			len(initialCart.Items),
		)
	}

	// Add product.

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

	cart := decodeJSON[cartResponse](
		t,
		add,
	)

	if len(cart.Items) != 1 {
		t.Fatalf(
			"expected 1 cart item, got %d",
			len(cart.Items),
		)
	}

	if cart.Items[0].ProductID != 1 {
		t.Fatalf(
			"unexpected product ID: %d",
			cart.Items[0].ProductID,
		)
	}

	if cart.Items[0].Quantity != 2 {
		t.Fatalf(
			"unexpected quantity: %d",
			cart.Items[0].Quantity,
		)
	}

	if cart.TotalAmount != 300000 {
		t.Fatalf(
			"unexpected cart total: got %d, want 300000",
			cart.TotalAmount,
		)
	}

	// Add same product again.

	addAgain := request(
		t,
		http.MethodPost,
		"/api/v1/cart/items",
		map[string]interface{}{
			"product_id": 1,
			"quantity":   3,
		},
		user.AccessToken,
	)

	assertStatus(
		t,
		addAgain,
		http.StatusOK,
	)

	cart = decodeJSON[cartResponse](
		t,
		addAgain,
	)

	if cart.Items[0].Quantity != 5 {
		t.Fatalf(
			"expected quantity 5, got %d",
			cart.Items[0].Quantity,
		)
	}

	if cart.TotalAmount != 750000 {
		t.Fatalf(
			"unexpected total: got %d, want 750000",
			cart.TotalAmount,
		)
	}

	// Update.

	update := request(
		t,
		http.MethodPatch,
		"/api/v1/cart/items/1",
		map[string]interface{}{
			"quantity": 3,
		},
		user.AccessToken,
	)

	assertStatus(
		t,
		update,
		http.StatusOK,
	)

	cart = decodeJSON[cartResponse](
		t,
		update,
	)

	if cart.Items[0].Quantity != 3 {
		t.Fatalf(
			"expected quantity 3, got %d",
			cart.Items[0].Quantity,
		)
	}

	// Delete.

	deleteResp := request(
		t,
		http.MethodDelete,
		"/api/v1/cart/items/1",
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		deleteResp,
		http.StatusOK,
	)

	cart = decodeJSON[cartResponse](
		t,
		deleteResp,
	)

	if len(cart.Items) != 0 {
		t.Fatalf(
			"expected empty cart after delete, got %d items",
			len(cart.Items),
		)
	}

	if cart.TotalAmount != 0 {
		t.Fatalf(
			"expected total 0, got %d",
			cart.TotalAmount,
		)
	}
}

func TestCart_InsufficientStock(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodPost,
		"/api/v1/cart/items",
		map[string]interface{}{
			"product_id": 1,
			"quantity":   999,
		},
		user.AccessToken,
	)

	assertError(
		t,
		resp,
		http.StatusConflict,
		"insufficient_stock",
	)
}

func TestCart_ProductNotFound(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodPost,
		"/api/v1/cart/items",
		map[string]interface{}{
			"product_id": 999999,
			"quantity":   1,
		},
		user.AccessToken,
	)

	assertError(
		t,
		resp,
		http.StatusNotFound,
		"product_not_found",
	)
}

func TestCart_InvalidQuantity(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodPost,
		"/api/v1/cart/items",
		map[string]interface{}{
			"product_id": 1,
			"quantity":   0,
		},
		user.AccessToken,
	)

	assertError(
		t,
		resp,
		http.StatusBadRequest,
		"invalid_cart",
	)
}
