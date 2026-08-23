package integration

import (
	"io"
	"net/http"
	"testing"
)

type productsResponse struct {
	Items []struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Price       int64  `json:"price"`
		Currency    string `json:"currency"`
		Stock       int    `json:"stock"`
	} `json:"items"`
}

type productResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Currency    string `json:"currency"`
	Stock       int    `json:"stock"`
}

func TestProducts_List(t *testing.T) {
	resetDatabase(t)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/products",
		nil,
		"",
	)

	assertStatus(
		t,
		resp,
		http.StatusOK,
	)

	data := decodeJSON[productsResponse](
		t,
		resp,
	)

	if len(data.Items) != 2 {
		t.Fatalf(
			"unexpected product count: got %d, want 2",
			len(data.Items),
		)
	}

	if data.Items[0].Name != "MacBook Pro" {
		t.Fatalf(
			"unexpected first product: %s",
			data.Items[0].Name,
		)
	}
}

func TestProducts_GetByID(t *testing.T) {
	resetDatabase(t)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/products/1",
		nil,
		"",
	)

	assertStatus(
		t,
		resp,
		http.StatusOK,
	)

	data := decodeJSON[productResponse](
		t,
		resp,
	)

	if data.ID != 1 {
		t.Fatalf(
			"unexpected product ID: %d",
			data.ID,
		)
	}

	if data.Price != 150000 {
		t.Fatalf(
			"unexpected price: %d",
			data.Price,
		)
	}
}

func TestProducts_GetByID_NotFound(t *testing.T) {
	resetDatabase(t)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/products/999999",
		nil,
		"",
	)

	assertStatus(
		t,
		resp,
		http.StatusNotFound,
	)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != "product not found\n" {
		t.Fatalf(
			"unexpected error: got %q, want %q",
			string(body),
			"product not found\n",
		)
	}
}

func TestProducts_GetByID_InvalidID(t *testing.T) {
	resetDatabase(t)

	resp := request(
		t,
		http.MethodGet,
		"/api/v1/products/abc",
		nil,
		"",
	)

	assertStatus(
		t,
		resp,
		http.StatusBadRequest,
	)

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(body) != "invalid product id\n" {
		t.Fatalf(
			"unexpected error: got %q, want %q",
			string(body),
			"invalid product id\n",
		)
	}
}
