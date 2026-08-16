package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

type fakeTransactionManager struct{}

func (f *fakeTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	return fn(nil)
}

type fakeProductRepository struct {
	product    product.Product
	getByIDErr error
}

func (f *fakeProductRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (product.Product, error) {
	if f.getByIDErr != nil {
		return product.Product{}, f.getByIDErr
	}

	if id != f.product.ID {
		return product.Product{}, errors.New("product not found")
	}

	return f.product, nil
}

func (f *fakeProductRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	return nil
}

type fakeOrderRepository struct {
	order     Order
	orderItem OrderItem
}

func (f *fakeOrderRepository) Create(
	ctx context.Context,
	tx database.Tx,
	order Order,
) (Order, error) {
	order.ID = 100
	order.CreatedAt = time.Now()

	f.order = order

	return order, nil
}

func (f *fakeOrderRepository) CreateItem(
	ctx context.Context,
	tx database.Tx,
	item OrderItem,
) (OrderItem, error) {
	item.ID = 1

	f.orderItem = item

	return item, nil
}

func TestCreateOrderHandler_Success(t *testing.T) {
	productRepository := &fakeProductRepository{
		product: product.Product{
			ID:       1,
			Name:     "MacBook",
			Price:    150000,
			Currency: "EUR",
			Stock:    10,
		},
	}

	orderRepository := &fakeOrderRepository{}

	transactionManager := &fakeTransactionManager{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	handler := NewHandler(service)

	body := `
		{
			"items": [
				{
					"product_id": 1,
					"quantity": 2
				}
			]
		}
	`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", "10")

	recorder := httptest.NewRecorder()

	handler.CreateOrder(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf(
			"expected Content-Type application/json, got %s",
			contentType,
		)
	}

	var response Order

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if response.ID != 100 {
		t.Fatalf(
			"expected order ID 100, got %d",
			response.ID,
		)
	}

	if response.UserID != 10 {
		t.Fatalf(
			"expected user ID 10, got %d",
			response.UserID,
		)
	}

	if response.Status != "pending" {
		t.Fatalf(
			"expected status pending, got %s",
			response.Status,
		)
	}

	if response.TotalAmount != 300000 {
		t.Fatalf(
			"expected total amount 300000, got %d",
			response.TotalAmount,
		)
	}

	if response.Currency != "EUR" {
		t.Fatalf(
			"expected currency EUR, got %s",
			response.Currency,
		)
	}
}

func TestCreateOrderHandler_InvalidJSON(t *testing.T) {
	productRepository := &fakeProductRepository{
		product: product.Product{
			ID:       1,
			Name:     "MacBook",
			Price:    150000,
			Currency: "EUR",
			Stock:    10,
		},
	}

	orderRepository := &fakeOrderRepository{}
	transactionManager := &fakeTransactionManager{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	handler := NewHandler(service)

	body := `{
		"items": [
			{
				"product_id": 1,
				"quantity":
			}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", "10")

	recorder := httptest.NewRecorder()

	handler.CreateOrder(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateOrderHandler_InvalidUserID(t *testing.T) {
	productRepository := &fakeProductRepository{
		product: product.Product{
			ID:       1,
			Name:     "MacBook",
			Price:    150000,
			Currency: "EUR",
			Stock:    10,
		},
	}

	orderRepository := &fakeOrderRepository{}
	transactionManager := &fakeTransactionManager{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	handler := NewHandler(service)

	body := `{
		"items": [
			{
				"product_id": 1,
				"quantity": 2
			}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")

	// X-User-ID специально не устанавливаем.

	recorder := httptest.NewRecorder()

	handler.CreateOrder(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateOrderHandler_InsufficientStock(t *testing.T) {
	productRepository := &fakeProductRepository{
		product: product.Product{
			ID:       1,
			Name:     "MacBook",
			Price:    150000,
			Currency: "EUR",
			Stock:    2,
		},
	}

	orderRepository := &fakeOrderRepository{}
	transactionManager := &fakeTransactionManager{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	handler := NewHandler(service)

	body := `{
		"items": [
			{
				"product_id": 1,
				"quantity": 5
			}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", "10")

	recorder := httptest.NewRecorder()

	handler.CreateOrder(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusConflict,
			recorder.Code,
		)
	}
}

func TestCreateOrderHandler_InternalServerError(t *testing.T) {
	productRepository := &fakeProductRepository{
		getByIDErr: errors.New("database connection failed"),
	}

	orderRepository := &fakeOrderRepository{}
	transactionManager := &fakeTransactionManager{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	handler := NewHandler(service)

	body := `{
		"items": [
			{
				"product_id": 1,
				"quantity": 2
			}
		]
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-User-ID", "10")

	recorder := httptest.NewRecorder()

	handler.CreateOrder(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}
