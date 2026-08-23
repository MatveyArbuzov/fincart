package integration

import (
	"net/http"
	"testing"
)

type orderResponse struct {
	Order struct {
		ID          int64  `json:"id"`
		UserID      int64  `json:"user_id"`
		Status      string `json:"status"`
		TotalAmount int64  `json:"total_amount"`
		Currency    string `json:"currency"`
	} `json:"order"`

	Items []struct {
		ID        int64 `json:"id"`
		OrderID   int64 `json:"order_id"`
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
		UnitPrice int64 `json:"unit_price"`
	} `json:"items"`
}

func createOrderViaCart(
	t *testing.T,
	user testUser,
) int64 {
	t.Helper()

	add := request(
		t,
		http.MethodPost,
		"/api/v1/cart/items",
		map[string]interface{}{
			"product_id": 1,
			"quantity":   1,
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

	var order struct {
		ID int64 `json:"id"`
	}

	order = decodeJSON[struct {
		ID int64 `json:"id"`
	}](t, checkout)

	return order.ID
}

func TestOrder_Get(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	orderID := createOrderViaCart(
		t,
		user,
	)

	resp := request(
		t,
		http.MethodGet,
		path("/api/v1/orders/%d", orderID),
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		resp,
		http.StatusOK,
	)

	data := decodeJSON[orderResponse](
		t,
		resp,
	)

	if data.Order.ID != orderID {
		t.Fatalf(
			"unexpected order ID: got %d, want %d",
			data.Order.ID,
			orderID,
		)
	}

	if data.Order.UserID != user.ID {
		t.Fatalf(
			"unexpected user ID: got %d, want %d",
			data.Order.UserID,
			user.ID,
		)
	}

	if len(data.Items) != 1 {
		t.Fatalf(
			"expected 1 order item, got %d",
			len(data.Items),
		)
	}

	if data.Items[0].ProductID != 1 {
		t.Fatalf(
			"unexpected product ID: %d",
			data.Items[0].ProductID,
		)
	}
}

func TestOrder_Pay(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	orderID := createOrderViaCart(
		t,
		user,
	)

	pay := request(
		t,
		http.MethodPost,
		path("/api/v1/orders/%d/pay", orderID),
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		pay,
		http.StatusNoContent,
	)

	if pay.Body != nil {
		defer pay.Body.Close()
	}

	get := request(
		t,
		http.MethodGet,
		path("/api/v1/orders/%d", orderID),
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		get,
		http.StatusOK,
	)

	data := decodeJSON[orderResponse](
		t,
		get,
	)

	if data.Order.Status != "paid" {
		t.Fatalf(
			"unexpected order status: got %s, want paid",
			data.Order.Status,
		)
	}
}

func TestOrder_Cancel(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	orderID := createOrderViaCart(
		t,
		user,
	)

	cancel := request(
		t,
		http.MethodPost,
		path("/api/v1/orders/%d/cancel", orderID),
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		cancel,
		http.StatusNoContent,
	)

	get := request(
		t,
		http.MethodGet,
		path("/api/v1/orders/%d", orderID),
		nil,
		user.AccessToken,
	)

	assertStatus(
		t,
		get,
		http.StatusOK,
	)

	data := decodeJSON[orderResponse](
		t,
		get,
	)

	if data.Order.Status != "cancelled" {
		t.Fatalf(
			"unexpected order status: got %s, want cancelled",
			data.Order.Status,
		)
	}
}

func TestOrder_NotFound(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/orders/999999",
		nil,
		user.AccessToken,
	)

	assertError(
		t,
		resp,
		http.StatusNotFound,
		"order_not_found",
	)
}

func TestOrder_Forbidden(t *testing.T) {
	resetDatabase(t)

	user1 := createUser(
		t,
		"user1@example.com",
		"password123",
	)

	user2 := createUser(
		t,
		"user2@example.com",
		"password123",
	)

	orderID := createOrderViaCart(
		t,
		user1,
	)

	resp := request(
		t,
		http.MethodGet,
		path("/api/v1/orders/%d", orderID),
		nil,
		user2.AccessToken,
	)

	assertError(
		t,
		resp,
		http.StatusForbidden,
		"order_forbidden",
	)
}
