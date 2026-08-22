package cart

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/order"
)

type handlerServiceMock struct {
	getCartFn    func(context.Context, int64) (Cart, error)
	addItemFn    func(context.Context, int64, AddItemRequest) (Cart, error)
	updateItemFn func(context.Context, int64, int64, int) (Cart, error)
	deleteItemFn func(context.Context, int64, int64) (Cart, error)
	checkoutFn   func(context.Context, int64) (order.Order, error)
}

func (m *handlerServiceMock) GetCart(
	ctx context.Context,
	userID int64,
) (Cart, error) {
	if m.getCartFn == nil {
		return Cart{}, nil
	}

	return m.getCartFn(ctx, userID)
}

func (m *handlerServiceMock) AddItem(
	ctx context.Context,
	userID int64,
	request AddItemRequest,
) (Cart, error) {
	if m.addItemFn == nil {
		return Cart{}, nil
	}

	return m.addItemFn(ctx, userID, request)
}

func (m *handlerServiceMock) UpdateItem(
	ctx context.Context,
	userID int64,
	productID int64,
	quantity int,
) (Cart, error) {
	if m.updateItemFn == nil {
		return Cart{}, nil
	}

	return m.updateItemFn(
		ctx,
		userID,
		productID,
		quantity,
	)
}

func (m *handlerServiceMock) DeleteItem(
	ctx context.Context,
	userID int64,
	productID int64,
) (Cart, error) {
	if m.deleteItemFn == nil {
		return Cart{}, nil
	}

	return m.deleteItemFn(
		ctx,
		userID,
		productID,
	)
}

func (m *handlerServiceMock) Checkout(
	ctx context.Context,
	userID int64,
) (order.Order, error) {
	if m.checkoutFn == nil {
		return order.Order{}, nil
	}

	return m.checkoutFn(ctx, userID)
}

func newAuthenticatedRequest(
	t *testing.T,
	method string,
	target string,
	body []byte,
	userID int64,
) *http.Request {
	t.Helper()

	req := httptest.NewRequest(
		method,
		target,
		bytes.NewReader(body),
	)

	manager := auth.NewJWTManager("test-secret")

	token, err := manager.GenerateToken(
		userID,
		"user",
	)
	if err != nil {
		t.Fatalf(
			"GenerateToken() error = %v",
			err,
		)
	}

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	return req
}

func TestHandler_GetCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		auth       bool
		serviceErr error
		status     int
		wantError  string
	}{
		{
			name:   "unauthorized",
			auth:   false,
			status: http.StatusUnauthorized,
		},
		{
			name:   "success",
			auth:   true,
			status: http.StatusOK,
		},
		{
			name:       "invalid cart",
			auth:       true,
			serviceErr: ErrInvalidCart,
			status:     http.StatusBadRequest,
			wantError:  "invalid_cart",
		},
		{
			name:       "product not found",
			auth:       true,
			serviceErr: ErrProductNotFound,
			status:     http.StatusNotFound,
			wantError:  "product_not_found",
		},
		{
			name:       "internal error",
			auth:       true,
			serviceErr: errors.New("database error"),
			status:     http.StatusInternalServerError,
			wantError:  "internal_server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &handlerServiceMock{
				getCartFn: func(
					context.Context,
					int64,
				) (Cart, error) {
					return Cart{
						ID:     1,
						UserID: 10,
						Status: "draft",
						Items:  []Item{},
					}, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			var req *http.Request

			if tt.auth {
				req = newAuthenticatedRequest(
					t,
					http.MethodGet,
					"/cart",
					nil,
					10,
				)
			} else {
				req = httptest.NewRequest(
					http.MethodGet,
					"/cart",
					nil,
				)
			}

			recorder := httptest.NewRecorder()

			auth.NewJWTManager("test-secret").
				Middleware(http.HandlerFunc(handler.GetCart)).
				ServeHTTP(recorder, req)

			if recorder.Code != tt.status {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					recorder.Code,
					tt.status,
					recorder.Body.String(),
				)
			}

			if tt.wantError != "" {
				var response struct {
					Error string `json:"error"`
				}

				if err := json.Unmarshal(
					recorder.Body.Bytes(),
					&response,
				); err != nil {
					t.Fatalf(
						"decode response: %v",
						err,
					)
				}

				if response.Error != tt.wantError {
					t.Fatalf(
						"error = %q, want %q",
						response.Error,
						tt.wantError,
					)
				}
			}
		})
	}
}

func TestHandler_AddItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		auth       bool
		body       string
		serviceErr error
		status     int
		wantError  string
	}{
		{
			name:   "unauthorized",
			auth:   false,
			body:   `{"product_id":1,"quantity":2}`,
			status: http.StatusUnauthorized,
		},
		{
			name:      "invalid json",
			auth:      true,
			body:      `{`,
			status:    http.StatusBadRequest,
			wantError: "invalid_request_body",
		},
		{
			name:       "invalid cart",
			auth:       true,
			body:       `{"product_id":1,"quantity":2}`,
			serviceErr: ErrInvalidCart,
			status:     http.StatusBadRequest,
			wantError:  "invalid_cart",
		},
		{
			name:       "product not found",
			auth:       true,
			body:       `{"product_id":1,"quantity":2}`,
			serviceErr: ErrProductNotFound,
			status:     http.StatusNotFound,
			wantError:  "product_not_found",
		},
		{
			name:       "insufficient stock",
			auth:       true,
			body:       `{"product_id":1,"quantity":2}`,
			serviceErr: ErrInsufficientStock,
			status:     http.StatusConflict,
			wantError:  "insufficient_stock",
		},
		{
			name:   "success",
			auth:   true,
			body:   `{"product_id":1,"quantity":2}`,
			status: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &handlerServiceMock{
				addItemFn: func(
					_ context.Context,
					_ int64,
					request AddItemRequest,
				) (Cart, error) {
					return Cart{
						ID: 1,
					}, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			var req *http.Request

			if tt.auth {
				req = newAuthenticatedRequest(
					t,
					http.MethodPost,
					"/cart/items",
					[]byte(tt.body),
					10,
				)
			} else {
				req = httptest.NewRequest(
					http.MethodPost,
					"/cart/items",
					bytes.NewBufferString(tt.body),
				)
			}

			recorder := httptest.NewRecorder()

			auth.NewJWTManager("test-secret").
				Middleware(http.HandlerFunc(handler.AddItem)).
				ServeHTTP(recorder, req)

			if recorder.Code != tt.status {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					recorder.Code,
					tt.status,
					recorder.Body.String(),
				)
			}

			assertHandlerError(
				t,
				recorder,
				tt.wantError,
			)
		})
	}
}

func TestHandler_UpdateItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		productID  string
		body       string
		serviceErr error
		status     int
		wantError  string
	}{
		{
			name:      "invalid product id",
			productID: "abc",
			body:      `{"quantity":2}`,
			status:    http.StatusBadRequest,
			wantError: "invalid_product_id",
		},
		{
			name:      "zero product id",
			productID: "0",
			body:      `{"quantity":2}`,
			status:    http.StatusBadRequest,
			wantError: "invalid_product_id",
		},
		{
			name:      "invalid json",
			productID: "1",
			body:      `{`,
			status:    http.StatusBadRequest,
			wantError: "invalid_request_body",
		},
		{
			name:       "cart item not found",
			productID:  "1",
			body:       `{"quantity":2}`,
			serviceErr: ErrCartItemNotFound,
			status:     http.StatusNotFound,
			wantError:  "cart_item_not_found",
		},
		{
			name:       "insufficient stock",
			productID:  "1",
			body:       `{"quantity":2}`,
			serviceErr: ErrInsufficientStock,
			status:     http.StatusConflict,
			wantError:  "insufficient_stock",
		},
		{
			name:      "success",
			productID: "1",
			body:      `{"quantity":2}`,
			status:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &handlerServiceMock{
				updateItemFn: func(
					_ context.Context,
					_ int64,
					productID int64,
					quantity int,
				) (Cart, error) {
					return Cart{
						ID:          int64(productID),
						TotalAmount: int64(quantity),
					}, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := newAuthenticatedRequest(
				t,
				http.MethodPatch,
				"/cart/items/"+tt.productID,
				[]byte(tt.body),
				10,
			)

			req.SetPathValue(
				"product_id",
				tt.productID,
			)

			recorder := httptest.NewRecorder()

			auth.NewJWTManager("test-secret").
				Middleware(http.HandlerFunc(handler.UpdateItem)).
				ServeHTTP(recorder, req)

			if recorder.Code != tt.status {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					recorder.Code,
					tt.status,
					recorder.Body.String(),
				)
			}

			assertHandlerError(
				t,
				recorder,
				tt.wantError,
			)
		})
	}
}

func TestHandler_DeleteItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		productID  string
		serviceErr error
		status     int
		wantError  string
	}{
		{
			name:      "invalid product id",
			productID: "abc",
			status:    http.StatusBadRequest,
			wantError: "invalid_product_id",
		},
		{
			name:       "cart not found",
			productID:  "1",
			serviceErr: ErrCartNotFound,
			status:     http.StatusNotFound,
			wantError:  "cart_not_found",
		},
		{
			name:       "cart item not found",
			productID:  "1",
			serviceErr: ErrCartItemNotFound,
			status:     http.StatusNotFound,
			wantError:  "cart_item_not_found",
		},
		{
			name:      "success",
			productID: "1",
			status:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &handlerServiceMock{
				deleteItemFn: func(
					_ context.Context,
					_ int64,
					_ int64,
				) (Cart, error) {
					return Cart{ID: 1}, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := newAuthenticatedRequest(
				t,
				http.MethodDelete,
				"/cart/items/"+tt.productID,
				nil,
				10,
			)

			req.SetPathValue(
				"product_id",
				tt.productID,
			)

			recorder := httptest.NewRecorder()

			auth.NewJWTManager("test-secret").
				Middleware(http.HandlerFunc(handler.DeleteItem)).
				ServeHTTP(recorder, req)

			if recorder.Code != tt.status {
				t.Fatalf(
					"status = %d, want %d; body=%s",
					recorder.Code,
					tt.status,
					recorder.Body.String(),
				)
			}

			assertHandlerError(
				t,
				recorder,
				tt.wantError,
			)
		})
	}
}

func TestHandler_Checkout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		auth       bool
		serviceErr error
		status     int
		wantError  string
	}{
		{
			name:   "unauthorized",
			auth:   false,
			status: http.StatusUnauthorized,
		},
		{
			name:       "cart not found",
			auth:       true,
			serviceErr: ErrCartNotFound,
			status:     http.StatusNotFound,
			wantError:  "cart_not_found",
		},
		{
			name:       "empty cart",
			auth:       true,
			serviceErr: ErrEmptyCart,
			status:     http.StatusConflict,
			wantError:  "empty_cart",
		},
		{
			name:   "success",
			auth:   true,
			status: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &handlerServiceMock{
				checkoutFn: func(
					context.Context,
					int64,
				) (order.Order, error) {
					return order.Order{
						ID:     1,
						UserID: 10,
						Status: string(order.OrderStatusPending),
					}, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			var req *http.Request

			if tt.auth {
				req = newAuthenticatedRequest(
					t,
					http.MethodPost,
					"/cart/checkout",
					nil,
					10,
				)
			} else {
				req = httptest.NewRequest(
					http.MethodPost,
					"/cart/checkout",
					nil,
				)
			}

			recorder := httptest.NewRecorder()

			auth.NewJWTManager("test-secret").
				Middleware(http.HandlerFunc(handler.Checkout)).
				ServeHTTP(recorder, req)

			if recorder.Code != tt.status {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.status,
				)
			}

			assertHandlerError(
				t,
				recorder,
				tt.wantError,
			)
		})
	}
}

func TestWriteServiceError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		status    int
		wantError string
	}{
		{
			name:      "invalid cart",
			err:       ErrInvalidCart,
			status:    http.StatusBadRequest,
			wantError: "invalid_cart",
		},
		{
			name:      "invalid quantity",
			err:       ErrInvalidQuantity,
			status:    http.StatusBadRequest,
			wantError: "invalid_cart",
		},
		{
			name:      "product not found",
			err:       ErrProductNotFound,
			status:    http.StatusNotFound,
			wantError: "product_not_found",
		},
		{
			name:      "cart not found",
			err:       ErrCartNotFound,
			status:    http.StatusNotFound,
			wantError: "cart_not_found",
		},
		{
			name:      "cart item not found",
			err:       ErrCartItemNotFound,
			status:    http.StatusNotFound,
			wantError: "cart_item_not_found",
		},
		{
			name:      "insufficient stock",
			err:       ErrInsufficientStock,
			status:    http.StatusConflict,
			wantError: "insufficient_stock",
		},
		{
			name:      "different currency",
			err:       ErrDifferentCurrency,
			status:    http.StatusBadRequest,
			wantError: "different_currencies",
		},
		{
			name:      "empty cart",
			err:       ErrEmptyCart,
			status:    http.StatusConflict,
			wantError: "empty_cart",
		},
		{
			name:      "unknown error",
			err:       errors.New("unknown"),
			status:    http.StatusInternalServerError,
			wantError: "internal_server_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			writeServiceError(
				recorder,
				tt.err,
			)

			if recorder.Code != tt.status {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.status,
				)
			}

			assertHandlerError(
				t,
				recorder,
				tt.wantError,
			)
		})
	}
}

func assertHandlerError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	want string,
) {
	t.Helper()

	if want == "" {
		return
	}

	var response struct {
		Error string `json:"error"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatalf(
			"decode error response: %v; body=%s",
			err,
			recorder.Body.String(),
		)
	}

	if response.Error != want {
		t.Fatalf(
			"error = %q, want %q",
			response.Error,
			want,
		)
	}
}
