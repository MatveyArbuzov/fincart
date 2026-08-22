package order

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

const testJWTSecret = "test-secret"

func authenticatedRequest(
	t *testing.T,
	method string,
	target string,
	body []byte,
	userID int64,
	role string,
) *http.Request {
	t.Helper()

	req := httptest.NewRequest(
		method,
		target,
		bytes.NewReader(body),
	)

	jwtManager := auth.NewJWTManager(testJWTSecret)

	token, err := jwtManager.GenerateToken(userID, role)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	return req
}

func serveAuthenticated(
	t *testing.T,
	handler http.Handler,
	req *http.Request,
) *httptest.ResponseRecorder {
	t.Helper()

	jwtManager := auth.NewJWTManager(testJWTSecret)

	recorder := httptest.NewRecorder()

	jwtManager.Middleware(handler).ServeHTTP(
		recorder,
		req,
	)

	return recorder
}

func setPathValue(
	req *http.Request,
	key string,
	value string,
) {
	req.SetPathValue(key, value)
}

func decodeErrorResponse(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) errorResponse {
	t.Helper()

	var response errorResponse

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf(
			"failed to decode error response: %v; body=%q",
			err,
			recorder.Body.String(),
		)
	}

	return response
}

func newTestOrderHandler(
	transactions TransactionManager,
	products ProductRepository,
	orders Repository,
	paymentService payment.Service,
) *Handler {
	service := NewService(
		transactions,
		products,
		orders,
		paymentService,
	)

	return NewHandler(service)
}

func TestHandler_CreateOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		body           []byte
		userID         int64
		role           string
		products       *fakeOrderProductRepository
		orders         *fakeOrderRepository
		transactions   *fakeOrderTransactionManager
		paymentService *fakePaymentService
		authenticated  bool
		wantStatus     int
		wantError      string
		wantOrder      *Order
	}{
		{
			name:           "unauthorized",
			body:           []byte(`{"items":[]}`),
			authenticated:  false,
			products:       &fakeOrderProductRepository{},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusUnauthorized,
			wantError:      "unauthorized",
		},
		{
			name:           "invalid body",
			body:           []byte(`{"items":`),
			userID:         42,
			role:           "user",
			authenticated:  true,
			products:       &fakeOrderProductRepository{},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_request_body",
		},
		{
			name:           "invalid order",
			body:           []byte(`{"items":[]}`),
			userID:         42,
			role:           "user",
			authenticated:  true,
			products:       &fakeOrderProductRepository{},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_order",
		},
		{
			name: "insufficient stock",
			body: []byte(`{
				"items": [
					{
						"product_id": 1,
						"quantity": 5
					}
				]
			}`),
			userID:        42,
			role:          "user",
			authenticated: true,
			products: &fakeOrderProductRepository{
				products: map[int64]product.Product{
					1: {
						ID:       1,
						Price:    100,
						Currency: "USD",
						Stock:    2,
					},
				},
			},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusConflict,
			wantError:      "insufficient_stock",
		},
		{
			name: "product not found",
			body: []byte(`{
				"items": [
					{
						"product_id": 999,
						"quantity": 1
					}
				]
			}`),
			userID:        42,
			role:          "user",
			authenticated: true,
			products: &fakeOrderProductRepository{
				products: map[int64]product.Product{},
			},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusNotFound,
			wantError:      "product_not_found",
		},
		{
			name: "different currencies",
			body: []byte(`{
				"items": [
					{
						"product_id": 1,
						"quantity": 1
					},
					{
						"product_id": 2,
						"quantity": 1
					}
				]
			}`),
			userID:        42,
			role:          "user",
			authenticated: true,
			products: &fakeOrderProductRepository{
				products: map[int64]product.Product{
					1: {
						ID:       1,
						Price:    100,
						Currency: "USD",
						Stock:    10,
					},
					2: {
						ID:       2,
						Price:    200,
						Currency: "EUR",
						Stock:    10,
					},
				},
			},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusBadRequest,
			wantError:      "different_currencies",
		},
		{
			name: "repository error",
			body: []byte(`{
				"items": [
					{
						"product_id": 1,
						"quantity": 1
					}
				]
			}`),
			userID:        42,
			role:          "user",
			authenticated: true,
			products: &fakeOrderProductRepository{
				products: map[int64]product.Product{
					1: {
						ID:       1,
						Price:    100,
						Currency: "USD",
						Stock:    10,
					},
				},
			},
			orders: &fakeOrderRepository{
				createErr: errors.New("database unavailable"),
			},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusInternalServerError,
			wantError:      "internal_server_error",
		},
		{
			name: "transaction error",
			body: []byte(`{
				"items": [
					{
						"product_id": 1,
						"quantity": 1
					}
				]
			}`),
			userID:        42,
			role:          "user",
			authenticated: true,
			products: &fakeOrderProductRepository{
				products: map[int64]product.Product{
					1: {
						ID:       1,
						Price:    100,
						Currency: "USD",
						Stock:    10,
					},
				},
			},
			orders: &fakeOrderRepository{},
			transactions: &fakeOrderTransactionManager{
				err: errors.New("transaction failed"),
			},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusInternalServerError,
			wantError:      "internal_server_error",
		},
		{
			name: "success",
			body: []byte(`{
				"items": [
					{
						"product_id": 1,
						"quantity": 2
					}
				]
			}`),
			userID:        42,
			role:          "user",
			authenticated: true,
			products: &fakeOrderProductRepository{
				products: map[int64]product.Product{
					1: {
						ID:       1,
						Name:     "Phone",
						Price:    100,
						Currency: "USD",
						Stock:    10,
					},
				},
			},
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			wantStatus:     http.StatusCreated,
			wantOrder: &Order{
				ID:          100,
				UserID:      42,
				TotalAmount: 200,
				Currency:    "USD",
				Status:      string(OrderStatusPending),
			},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				tt.transactions,
				tt.products,
				tt.orders,
				tt.paymentService,
			)

			var req *http.Request

			if tt.authenticated {
				req = authenticatedRequest(
					t,
					http.MethodPost,
					"/api/v1/orders",
					tt.body,
					tt.userID,
					tt.role,
				)
			} else {
				req = httptest.NewRequest(
					http.MethodPost,
					"/api/v1/orders",
					bytes.NewReader(tt.body),
				)
			}

			recorder := httptest.NewRecorder()

			if tt.authenticated {
				recorder = serveAuthenticated(
					t,
					http.HandlerFunc(handler.CreateOrder),
					req,
				)
			} else {
				handler.CreateOrder(recorder, req)
			}

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%q",
					recorder.Code,
					tt.wantStatus,
					recorder.Body.String(),
				)
			}

			if tt.wantError != "" {
				response := decodeErrorResponse(t, recorder)

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}

				return
			}

			if tt.wantOrder == nil {
				return
			}

			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf(
					"Content-Type = %q, want application/json",
					recorder.Header().Get("Content-Type"),
				)
			}

			var got Order

			if err := json.Unmarshal(
				recorder.Body.Bytes(),
				&got,
			); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if got != *tt.wantOrder {
				t.Fatalf(
					"order = %+v, want %+v",
					got,
					*tt.wantOrder,
				)
			}
		})
	}
}

func TestHandler_CancelOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		userID        int64
		orders        *fakeOrderRepository
		transactions  *fakeOrderTransactionManager
		authenticated bool
		usePathValue  bool
		wantStatus    int
		wantError     string
	}{
		{
			name:          "unauthorized",
			id:            "10",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: false,
			usePathValue:  true,
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "empty id",
			id:            "",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_order_id",
		},
		{
			name:          "non numeric id",
			id:            "abc",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_order_id",
		},
		{
			name:          "zero id",
			id:            "0",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_order_id",
		},
		{
			name:          "negative id",
			id:            "-1",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_order_id",
		},
		{
			name: "not found",
			id:   "10",
			orders: &fakeOrderRepository{
				getByIDForUpdateErr: ErrOrderNotFound,
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			userID:        42,
			wantStatus:    http.StatusNotFound,
			wantError:     "order_not_found",
		},
		{
			name: "forbidden",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 999,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			userID:        42,
			wantStatus:    http.StatusForbidden,
			wantError:     "order_forbidden",
		},
		{
			name: "invalid state",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
					Status: string(OrderStatusPaid),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			userID:        42,
			wantStatus:    http.StatusConflict,
			wantError:     "invalid_order_state",
		},
		{
			name: "repository error",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
					Status: string(OrderStatusPending),
				},
				getItemsErr: errors.New("database error"),
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			userID:        42,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name: "transaction error",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
					Status: string(OrderStatusPending),
				},
			},
			transactions: &fakeOrderTransactionManager{
				err: errors.New("transaction failed"),
			},
			authenticated: true,
			usePathValue:  true,
			userID:        42,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name: "success",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  true,
			userID:        42,
			wantStatus:    http.StatusNoContent,
		},
		{
			name: "fallback path parsing",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			usePathValue:  false,
			userID:        42,
			wantStatus:    http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				tt.transactions,
				&fakeOrderProductRepository{},
				tt.orders,
				&fakePaymentService{},
			)

			req := authenticatedRequest(
				t,
				http.MethodPost,
				"/api/v1/orders/"+tt.id+"/cancel",
				nil,
				tt.userID,
				"user",
			)

			if tt.usePathValue {
				setPathValue(req, "id", tt.id)
			}

			var recorder *httptest.ResponseRecorder

			if tt.authenticated {
				recorder = serveAuthenticated(
					t,
					http.HandlerFunc(handler.CancelOrder),
					req,
				)
			} else {
				recorder = httptest.NewRecorder()
				handler.CancelOrder(recorder, httptest.NewRequest(
					http.MethodPost,
					"/api/v1/orders/"+tt.id+"/cancel",
					nil,
				))
			}

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%q",
					recorder.Code,
					tt.wantStatus,
					recorder.Body.String(),
				)
			}

			if tt.wantError != "" {
				response := decodeErrorResponse(t, recorder)

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}

				return
			}

			if tt.wantStatus == http.StatusNoContent &&
				recorder.Body.Len() != 0 {
				t.Fatalf(
					"body = %q, want empty",
					recorder.Body.String(),
				)
			}
		})
	}
}

func TestHandler_GetOrder(t *testing.T) {
	t.Parallel()

	wantOrder := Order{
		ID:          10,
		UserID:      42,
		Status:      string(OrderStatusPending),
		TotalAmount: 300,
		Currency:    "USD",
	}

	wantItems := []OrderItem{
		{
			ID:        1,
			OrderID:   10,
			ProductID: 5,
			Quantity:  2,
			UnitPrice: 150,
		},
	}

	tests := []struct {
		name          string
		id            string
		userID        int64
		orders        *fakeOrderRepository
		transactions  *fakeOrderTransactionManager
		authenticated bool
		wantStatus    int
		wantError     string
		wantOrder     *Order
		wantItems     []OrderItem
	}{
		{
			name:          "unauthorized",
			id:            "10",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: false,
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "invalid id",
			id:            "abc",
			userID:        42,
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_order_id",
		},
		{
			name: "not found",
			id:   "10",
			orders: &fakeOrderRepository{
				getByIDErr: ErrOrderNotFound,
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusNotFound,
			wantError:     "order_not_found",
		},
		{
			name: "forbidden",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 99,
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusForbidden,
			wantError:     "order_forbidden",
		},
		{
			name: "repository error",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
				},
				getItemsErr: errors.New("database error"),
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name: "transaction error",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
				},
			},
			transactions: &fakeOrderTransactionManager{
				err: errors.New("transaction failed"),
			},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name:          "success",
			id:            "10",
			userID:        42,
			orders:        &fakeOrderRepository{order: wantOrder, items: wantItems},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusOK,
			wantOrder:     &wantOrder,
			wantItems:     wantItems,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				tt.transactions,
				&fakeOrderProductRepository{},
				tt.orders,
				&fakePaymentService{},
			)

			var req *http.Request

			if tt.authenticated {
				req = authenticatedRequest(
					t,
					http.MethodGet,
					"/api/v1/orders/"+tt.id,
					nil,
					tt.userID,
					"user",
				)
			} else {
				req = httptest.NewRequest(
					http.MethodGet,
					"/api/v1/orders/"+tt.id,
					nil,
				)
			}

			setPathValue(req, "id", tt.id)

			var recorder *httptest.ResponseRecorder

			if tt.authenticated {
				recorder = serveAuthenticated(
					t,
					http.HandlerFunc(handler.GetOrder),
					req,
				)
			} else {
				recorder = httptest.NewRecorder()
				handler.GetOrder(recorder, req)
			}

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%q",
					recorder.Code,
					tt.wantStatus,
					recorder.Body.String(),
				)
			}

			if tt.wantError != "" {
				response := decodeErrorResponse(t, recorder)

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}

				return
			}

			if tt.wantOrder == nil {
				return
			}

			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf(
					"Content-Type = %q, want application/json",
					recorder.Header().Get("Content-Type"),
				)
			}

			var response struct {
				Order Order       `json:"order"`
				Items []OrderItem `json:"items"`
			}

			if err := json.Unmarshal(
				recorder.Body.Bytes(),
				&response,
			); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Order != *tt.wantOrder {
				t.Fatalf(
					"order = %+v, want %+v",
					response.Order,
					*tt.wantOrder,
				)
			}

			if len(response.Items) != len(tt.wantItems) {
				t.Fatalf(
					"items = %d, want %d",
					len(response.Items),
					len(tt.wantItems),
				)
			}

			for i := range tt.wantItems {
				if response.Items[i] != tt.wantItems[i] {
					t.Fatalf(
						"items[%d] = %+v, want %+v",
						i,
						response.Items[i],
						tt.wantItems[i],
					)
				}
			}
		})
	}
}

func TestHandler_PayOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		id             string
		userID         int64
		orders         *fakeOrderRepository
		transactions   *fakeOrderTransactionManager
		paymentService *fakePaymentService
		authenticated  bool
		wantStatus     int
		wantError      string
		wantPaid       bool
	}{
		{
			name:           "unauthorized",
			id:             "10",
			userID:         42,
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			authenticated:  false,
			wantStatus:     http.StatusUnauthorized,
		},
		{
			name:           "invalid id",
			id:             "abc",
			userID:         42,
			orders:         &fakeOrderRepository{},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			authenticated:  true,
			wantStatus:     http.StatusBadRequest,
			wantError:      "invalid_order_id",
		},
		{
			name: "not found",
			id:   "10",
			orders: &fakeOrderRepository{
				getByIDForUpdateErr: ErrOrderNotFound,
			},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			authenticated:  true,
			userID:         42,
			wantStatus:     http.StatusNotFound,
			wantError:      "order_not_found",
		},
		{
			name: "forbidden",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 99,
					Status: string(OrderStatusPending),
				},
			},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			authenticated:  true,
			userID:         42,
			wantStatus:     http.StatusForbidden,
			wantError:      "order_forbidden",
		},
		{
			name: "invalid state",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					UserID: 42,
					Status: string(OrderStatusPaid),
				},
			},
			transactions:   &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{},
			authenticated:  true,
			userID:         42,
			wantStatus:     http.StatusConflict,
			wantError:      "invalid_order_state",
		},
		{
			name: "payment failed",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:       10,
					UserID:   42,
					Status:   string(OrderStatusPending),
					Currency: "USD",
				},
			},
			transactions: &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{
				result: payment.ResultFailed,
			},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusPaymentRequired,
			wantError:     "payment_failed",
		},
		{
			name: "payment timeout",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:       10,
					UserID:   42,
					Status:   string(OrderStatusPending),
					Currency: "USD",
				},
			},
			transactions: &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{
				result: payment.ResultTimeout,
			},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusGatewayTimeout,
			wantError:     "payment_timeout",
		},
		{
			name: "payment service error",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:       10,
					UserID:   42,
					Status:   string(OrderStatusPending),
					Currency: "USD",
				},
			},
			transactions: &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{
				err: errors.New("payment unavailable"),
			},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name: "transaction error",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:          10,
					UserID:      42,
					Status:      string(OrderStatusPending),
					TotalAmount: 1000,
					Currency:    "USD",
				},
			},
			transactions: &fakeOrderTransactionManager{
				err: errors.New("transaction failed"),
			},
			paymentService: &fakePaymentService{
				result: payment.ResultSuccess,
			},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name: "success",
			id:   "10",
			orders: &fakeOrderRepository{
				order: Order{
					ID:          10,
					UserID:      42,
					Status:      string(OrderStatusPending),
					TotalAmount: 1000,
					Currency:    "USD",
				},
			},
			transactions: &fakeOrderTransactionManager{},
			paymentService: &fakePaymentService{
				result: payment.ResultSuccess,
			},
			authenticated: true,
			userID:        42,
			wantStatus:    http.StatusNoContent,
			wantPaid:      true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				tt.transactions,
				&fakeOrderProductRepository{},
				tt.orders,
				tt.paymentService,
			)

			var req *http.Request

			if tt.authenticated {
				req = authenticatedRequest(
					t,
					http.MethodPost,
					"/api/v1/orders/"+tt.id+"/pay",
					nil,
					tt.userID,
					"user",
				)
				setPathValue(req, "id", tt.id)
			} else {
				req = httptest.NewRequest(
					http.MethodPost,
					"/api/v1/orders/"+tt.id+"/pay",
					nil,
				)
				setPathValue(req, "id", tt.id)
			}

			var recorder *httptest.ResponseRecorder

			if tt.authenticated {
				recorder = serveAuthenticated(
					t,
					http.HandlerFunc(handler.PayOrder),
					req,
				)
			} else {
				recorder = httptest.NewRecorder()
				handler.PayOrder(recorder, req)
			}

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%q",
					recorder.Code,
					tt.wantStatus,
					recorder.Body.String(),
				)
			}

			if tt.wantError != "" {
				response := decodeErrorResponse(t, recorder)

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}
			}

			if tt.wantPaid {
				if tt.orders.updatedStatus != string(OrderStatusPaid) {
					t.Fatalf(
						"status = %q, want paid",
						tt.orders.updatedStatus,
					)
				}

				if tt.paymentService.orderID != 10 {
					t.Fatalf(
						"payment order ID = %d, want 10",
						tt.paymentService.orderID,
					)
				}

				if tt.paymentService.amount != 1000 {
					t.Fatalf(
						"payment amount = %d, want 1000",
						tt.paymentService.amount,
					)
				}

				if tt.paymentService.currency != "USD" {
					t.Fatalf(
						"payment currency = %q, want USD",
						tt.paymentService.currency,
					)
				}
			}
		})
	}
}

func TestHandler_ListOrders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		orders     *fakeOrderRepository
		wantStatus int
		wantOrders []Order
		wantError  string
	}{
		{
			name: "success",
			orders: &fakeOrderRepository{
				orders: []Order{
					{
						ID:     3,
						UserID: 3,
					},
					{
						ID:     2,
						UserID: 2,
					},
				},
			},
			wantStatus: http.StatusOK,
			wantOrders: []Order{
				{
					ID:     3,
					UserID: 3,
				},
				{
					ID:     2,
					UserID: 2,
				},
			},
		},
		{
			name: "empty list",
			orders: &fakeOrderRepository{
				orders: []Order{},
			},
			wantStatus: http.StatusOK,
			wantOrders: []Order{},
		},
		{
			name: "internal error",
			orders: &fakeOrderRepository{
				listErr: errors.New("database unavailable"),
			},
			wantStatus: http.StatusInternalServerError,
			wantError:  "internal_server_error",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				&fakeOrderTransactionManager{},
				&fakeOrderProductRepository{},
				tt.orders,
				&fakePaymentService{},
			)

			req := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/orders",
				nil,
			)

			recorder := httptest.NewRecorder()

			handler.ListOrders(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.wantStatus,
				)
			}

			if tt.wantError != "" {
				response := decodeErrorResponse(t, recorder)

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}

				return
			}

			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf(
					"Content-Type = %q, want application/json",
					recorder.Header().Get("Content-Type"),
				)
			}

			var response struct {
				Orders []Order `json:"orders"`
			}

			if err := json.Unmarshal(
				recorder.Body.Bytes(),
				&response,
			); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Orders == nil {
				t.Fatal("orders is nil, want empty/non-nil slice")
			}

			if len(response.Orders) != len(tt.wantOrders) {
				t.Fatalf(
					"orders = %d, want %d",
					len(response.Orders),
					len(tt.wantOrders),
				)
			}

			for i := range tt.wantOrders {
				if response.Orders[i] != tt.wantOrders[i] {
					t.Fatalf(
						"orders[%d] = %+v, want %+v",
						i,
						response.Orders[i],
						tt.wantOrders[i],
					)
				}
			}
		})
	}
}

func TestHandler_UpdateOrderStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		id            string
		requestStatus string
		userID        int64
		role          string
		orders        *fakeOrderRepository
		transactions  *fakeOrderTransactionManager
		authenticated bool
		wantStatus    int
		wantError     string
		wantUpdated   string
	}{
		{
			name:          "invalid id",
			id:            "abc",
			requestStatus: string(OrderStatusPaid),
			userID:        42,
			role:          "admin",
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_order_id",
		},
		{
			name:          "invalid body",
			id:            "10",
			requestStatus: "",
			userID:        42,
			role:          "admin",
			orders:        &fakeOrderRepository{},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusBadRequest,
			wantError:     "invalid_request_body",
		},
		{
			name:          "invalid state",
			id:            "10",
			requestStatus: "unknown",
			userID:        42,
			role:          "admin",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusConflict,
			wantError:     "invalid_order_state",
		},
		{
			name:          "not found",
			id:            "10",
			requestStatus: string(OrderStatusPaid),
			userID:        42,
			role:          "admin",
			orders: &fakeOrderRepository{
				getByIDForUpdateErr: ErrOrderNotFound,
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusNotFound,
			wantError:     "order_not_found",
		},
		{
			name:          "invalid transition",
			id:            "10",
			requestStatus: string(OrderStatusProcessing),
			userID:        42,
			role:          "admin",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusConflict,
			wantError:     "invalid_order_state",
		},
		{
			name:          "repository error",
			id:            "10",
			requestStatus: string(OrderStatusPaid),
			userID:        42,
			role:          "admin",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					Status: string(OrderStatusPending),
				},
				updateStatusErr: errors.New("database error"),
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name:          "transaction error",
			id:            "10",
			requestStatus: string(OrderStatusPaid),
			userID:        42,
			role:          "admin",
			orders:        &fakeOrderRepository{},
			transactions: &fakeOrderTransactionManager{
				err: errors.New("transaction failed"),
			},
			authenticated: true,
			wantStatus:    http.StatusInternalServerError,
			wantError:     "internal_server_error",
		},
		{
			name:          "success paid",
			id:            "10",
			requestStatus: string(OrderStatusPaid),
			userID:        42,
			role:          "admin",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusNoContent,
			wantUpdated:   string(OrderStatusPaid),
		},
		{
			name:          "success cancelled",
			id:            "10",
			requestStatus: string(OrderStatusCancelled),
			userID:        42,
			role:          "admin",
			orders: &fakeOrderRepository{
				order: Order{
					ID:     10,
					Status: string(OrderStatusPending),
				},
			},
			transactions:  &fakeOrderTransactionManager{},
			authenticated: true,
			wantStatus:    http.StatusNoContent,
			wantUpdated:   string(OrderStatusCancelled),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				tt.transactions,
				&fakeOrderProductRepository{},
				tt.orders,
				&fakePaymentService{},
			)

			var body []byte

			if tt.name == "invalid body" {
				body = []byte(`{"status":`)
			} else {
				body, _ = json.Marshal(
					UpdateOrderStatusRequest{
						Status: tt.requestStatus,
					},
				)
			}

			req := authenticatedRequest(
				t,
				http.MethodPatch,
				"/api/v1/orders/"+tt.id+"/status",
				body,
				tt.userID,
				tt.role,
			)

			setPathValue(req, "id", tt.id)

			var recorder *httptest.ResponseRecorder

			if tt.authenticated {
				recorder = serveAuthenticated(
					t,
					http.HandlerFunc(handler.UpdateOrderStatus),
					req,
				)
			} else {
				recorder = httptest.NewRecorder()
				handler.UpdateOrderStatus(recorder, req)
			}

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d; body=%q",
					recorder.Code,
					tt.wantStatus,
					recorder.Body.String(),
				)
			}

			if tt.wantError != "" {
				response := decodeErrorResponse(t, recorder)

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}

				return
			}

			if recorder.Body.Len() != 0 {
				t.Fatalf(
					"body = %q, want empty",
					recorder.Body.String(),
				)
			}

			if tt.wantUpdated != "" {
				if tt.orders.updatedOrderID != 10 {
					t.Fatalf(
						"order ID = %d, want 10",
						tt.orders.updatedOrderID,
					)
				}

				if tt.orders.updatedStatus != tt.wantUpdated {
					t.Fatalf(
						"status = %q, want %q",
						tt.orders.updatedStatus,
						tt.wantUpdated,
					)
				}
			}
		})
	}
}

func TestHandler_AuthMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		authorization string
		userID        int64
		role          string
		wantStatus    int
		wantUserID    int64
		wantRole      string
	}{
		{
			name:          "invalid token",
			authorization: "Bearer invalid-token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:       "valid token preserves claims",
			userID:     123,
			role:       "admin",
			wantStatus: http.StatusNoContent,
			wantUserID: 123,
			wantRole:   "admin",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var gotUserID int64
			var gotRole string

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				claims, ok := auth.ClaimsFromContext(
					r.Context(),
				)

				if !ok {
					t.Fatal("claims are missing from context")
				}

				gotUserID = claims.UserID
				gotRole = claims.Role

				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)

			if tt.authorization != "" {
				req.Header.Set(
					"Authorization",
					tt.authorization,
				)
			} else {
				token, err := auth.NewJWTManager(
					testJWTSecret,
				).GenerateToken(
					tt.userID,
					tt.role,
				)
				if err != nil {
					t.Fatalf(
						"failed to generate token: %v",
						err,
					)
				}

				req.Header.Set(
					"Authorization",
					"Bearer "+token,
				)
			}

			recorder := httptest.NewRecorder()

			auth.NewJWTManager(testJWTSecret).
				Middleware(next).
				ServeHTTP(recorder, req)

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.wantStatus,
				)
			}

			if tt.wantStatus == http.StatusNoContent {
				if gotUserID != tt.wantUserID {
					t.Fatalf(
						"user ID = %d, want %d",
						gotUserID,
						tt.wantUserID,
					)
				}

				if gotRole != tt.wantRole {
					t.Fatalf(
						"role = %q, want %q",
						gotRole,
						tt.wantRole,
					)
				}
			}
		})
	}
}

func TestHandler_ContextCancellation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cancel       bool
		wantStatuses []int
	}{
		{
			name:         "cancelled context",
			cancel:       true,
			wantStatuses: []int{http.StatusCreated, http.StatusInternalServerError},
		},
		{
			name:         "active context",
			cancel:       false,
			wantStatuses: []int{http.StatusCreated},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := newTestOrderHandler(
				&fakeOrderTransactionManager{},
				&fakeOrderProductRepository{
					products: map[int64]product.Product{
						1: {
							ID:       1,
							Price:    100,
							Currency: "USD",
							Stock:    10,
						},
					},
				},
				&fakeOrderRepository{},
				&fakePaymentService{},
			)

			ctx, cancel := context.WithCancel(
				context.Background(),
			)

			if tt.cancel {
				cancel()
			} else {
				defer cancel()
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/orders",
				bytes.NewBufferString(
					`{"items":[{"product_id":1,"quantity":1}]}`,
				),
			).WithContext(ctx)

			token, err := auth.NewJWTManager(
				testJWTSecret,
			).GenerateToken(42, "user")
			if err != nil {
				t.Fatalf(
					"failed to generate token: %v",
					err,
				)
			}

			req.Header.Set(
				"Authorization",
				"Bearer "+token,
			)

			recorder := httptest.NewRecorder()

			auth.NewJWTManager(testJWTSecret).
				Middleware(
					http.HandlerFunc(handler.CreateOrder),
				).
				ServeHTTP(recorder, req)

			validStatus := false

			for _, status := range tt.wantStatuses {
				if recorder.Code == status {
					validStatus = true
					break
				}
			}

			if !validStatus {
				t.Fatalf(
					"unexpected status = %d, want one of %v",
					recorder.Code,
					tt.wantStatuses,
				)
			}
		})
	}
}
