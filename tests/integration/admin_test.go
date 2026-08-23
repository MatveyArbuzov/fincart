package integration

import (
	"io"
	"net/http"
	"testing"
)

func TestAdmin_CreateProduct(t *testing.T) {
	resetDatabase(t)

	admin := createAdmin(
		t,
		"admin@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodPost,
		"/api/v1/admin/products",
		map[string]interface{}{
			"name":        "Gaming Mouse",
			"description": "Wireless mouse",
			"price":       5000,
			"currency":    "eur",
			"stock":       25,
		},
		admin.AccessToken,
	)

	assertStatus(
		t,
		resp,
		http.StatusCreated,
	)

	product := decodeJSON[productResponse](
		t,
		resp,
	)

	if product.Name != "Gaming Mouse" {
		t.Fatalf(
			"unexpected product name: %s",
			product.Name,
		)
	}

	if product.Currency != "EUR" {
		t.Fatalf(
			"expected EUR, got %s",
			product.Currency,
		)
	}

	if product.Price != 5000 {
		t.Fatalf(
			"unexpected price: %d",
			product.Price,
		)
	}
}

func TestAdmin_UpdateProduct(t *testing.T) {
	resetDatabase(t)

	admin := createAdmin(
		t,
		"admin@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodPatch,
		"/api/v1/admin/products/1",
		map[string]interface{}{
			"name":        "MacBook Pro Updated",
			"description": "Updated laptop",
			"price":       160000,
			"currency":    "eur",
			"stock":       5,
		},
		admin.AccessToken,
	)

	assertStatus(
		t,
		resp,
		http.StatusOK,
	)

	product := decodeJSON[productResponse](
		t,
		resp,
	)

	if product.ID != 1 {
		t.Fatalf(
			"unexpected product ID: %d",
			product.ID,
		)
	}

	if product.Name != "MacBook Pro Updated" {
		t.Fatalf(
			"unexpected product name: %s",
			product.Name,
		)
	}

	if product.Price != 160000 {
		t.Fatalf(
			"unexpected price: %d",
			product.Price,
		)
	}
}

func TestAdmin_DeleteProduct(t *testing.T) {
	resetDatabase(t)

	admin := createAdmin(
		t,
		"admin@example.com",
		"password123",
	)

	resp := request(
		t,
		http.MethodDelete,
		"/api/v1/admin/products/1",
		nil,
		admin.AccessToken,
	)

	assertStatus(
		t,
		resp,
		http.StatusNoContent,
	)

	get := request(
		t,
		http.MethodGet,
		"/api/v1/products/1",
		nil,
		"",
	)

	assertStatus(
		t,
		get,
		http.StatusNotFound,
	)

	defer get.Body.Close()

	body, err := io.ReadAll(get.Body)
	if err != nil {
		t.Fatalf(
			"failed to read response body: %v",
			err,
		)
	}

	if string(body) != "product not found\n" {
		t.Fatalf(
			"unexpected error: got %q, want %q",
			string(body),
			"product not found\n",
		)
	}
}

func TestAdmin_UnauthorizedCannotCreateProduct(t *testing.T) {
	resetDatabase(t)

	resp := request(
		t,
		http.MethodPost,
		"/api/v1/admin/products",
		map[string]interface{}{
			"name":        "Hacked Product",
			"description": "Should not be created",
			"price":       1,
			"currency":    "EUR",
			"stock":       1,
		},
		"",
	)

	assertStatus(
		t,
		resp,
		http.StatusUnauthorized,
	)
}

func TestAdmin_ListOrders(t *testing.T) {
	resetDatabase(t)

	user := createUser(
		t,
		"user@example.com",
		"password123",
	)

	admin := createAdmin(
		t,
		"admin@example.com",
		"password123",
	)

	createOrderViaCart(
		t,
		user,
	)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/admin/orders",
		nil,
		admin.AccessToken,
	)

	assertStatus(
		t,
		resp,
		http.StatusOK,
	)

	data := decodeJSON[struct {
		Orders []struct {
			ID     int64  `json:"id"`
			UserID int64  `json:"user_id"`
			Status string `json:"status"`
		} `json:"orders"`
	}](
		t,
		resp,
	)

	if len(data.Orders) != 1 {
		t.Fatalf(
			"expected 1 order, got %d",
			len(data.Orders),
		)
	}

	if data.Orders[0].UserID != user.ID {
		t.Fatalf(
			"unexpected user ID: %d",
			data.Orders[0].UserID,
		)
	}
}
