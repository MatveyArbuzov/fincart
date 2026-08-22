package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/cart"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/product"
	"github.com/MatveyArbuzov/fincart/internal/user"
)

type routerOrderService struct{}

func (routerOrderService) CreateOrder(
	context.Context,
	int64,
	order.CreateOrderRequest,
) (order.Order, error) {
	return order.Order{}, nil
}

func (routerOrderService) CancelOrder(
	context.Context,
	int64,
	int64,
) error {
	return nil
}

func (routerOrderService) GetOrder(
	context.Context,
	int64,
	int64,
) (order.Order, []order.OrderItem, error) {
	return order.Order{}, nil, nil
}

func (routerOrderService) PayOrder(
	context.Context,
	int64,
	int64,
) error {
	return nil
}

func (routerOrderService) ListOrders(
	context.Context,
) ([]order.Order, error) {
	return nil, nil
}

func (routerOrderService) UpdateOrderStatus(
	context.Context,
	int64,
	string,
) error {
	return nil
}

type routerProductService struct{}

func (routerProductService) GetProducts(
	context.Context,
) ([]product.Product, error) {
	return nil, nil
}

func (routerProductService) GetProductByID(
	context.Context,
	int64,
) (product.Product, error) {
	return product.Product{}, nil
}

func (routerProductService) CreateProduct(
	context.Context,
	product.CreateProductRequest,
) (product.Product, error) {
	return product.Product{}, nil
}

func (routerProductService) UpdateProduct(
	context.Context,
	int64,
	product.UpdateProductRequest,
) (product.Product, error) {
	return product.Product{}, nil
}

func (routerProductService) DeleteProduct(
	context.Context,
	int64,
) error {
	return nil
}

type routerCartService struct{}

func (routerCartService) GetCart(
	context.Context,
	int64,
) (cart.Cart, error) {
	return cart.Cart{}, nil
}

func (routerCartService) AddItem(
	context.Context,
	int64,
	cart.AddItemRequest,
) (cart.Cart, error) {
	return cart.Cart{}, nil
}

func (routerCartService) UpdateItem(
	context.Context,
	int64,
	int64,
	int,
) (cart.Cart, error) {
	return cart.Cart{}, nil
}

func (routerCartService) DeleteItem(
	context.Context,
	int64,
	int64,
) (cart.Cart, error) {
	return cart.Cart{}, nil
}

func (routerCartService) Checkout(
	context.Context,
	int64,
) (order.Order, error) {
	return order.Order{}, nil
}

func newTestRouter(t *testing.T) (
	http.Handler,
	*auth.JWTManager,
) {
	t.Helper()

	jwtManager := auth.NewJWTManager(
		"test-secret",
	)

	productHandler := product.NewHandler(
		routerProductService{},
	)

	cartHandler := cart.NewHandler(
		routerCartService{},
	)
	orderHandler := order.NewHandler(
		routerOrderService{},
	)

	router := NewRouter(
		productHandler,
		orderHandler,
		&user.Handler{},
		cartHandler,
		jwtManager,
	)

	return router, jwtManager
}

func TestNewRouter_PublicRoutes(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "get products",
			method:     http.MethodGet,
			path:       "/api/v1/products",
			wantStatus: http.StatusOK,
		},
		{
			name:       "get product by id",
			method:     http.MethodGet,
			path:       "/api/v1/products/1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "register",
			method:     http.MethodPost,
			path:       "/api/v1/auth/register",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "login",
			method:     http.MethodPost,
			path:       "/api/v1/auth/login",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "refresh",
			method:     http.MethodPost,
			path:       "/api/v1/auth/refresh",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "logout",
			method:     http.MethodPost,
			path:       "/api/v1/auth/logout",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(
				tt.method,
				tt.path,
				nil,
			)

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				request,
			)

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.wantStatus,
				)
			}
		})
	}
}

func TestNewRouter_ProtectedRoutes(t *testing.T) {
	t.Parallel()

	router, jwtManager := newTestRouter(t)

	validUserToken, err := jwtManager.GenerateToken(
		42,
		"user",
	)
	if err != nil {
		t.Fatalf(
			"GenerateToken() error = %v",
			err,
		)
	}

	tests := []struct {
		name   string
		method string
		path   string

		token string

		wantStatus int
	}{
		{
			name:       "create order without token",
			method:     http.MethodPost,
			path:       "/api/v1/orders",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "get order without token",
			method:     http.MethodGet,
			path:       "/api/v1/orders/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "cancel order without token",
			method:     http.MethodPost,
			path:       "/api/v1/orders/1/cancel",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "pay order without token",
			method:     http.MethodPost,
			path:       "/api/v1/orders/1/pay",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "get cart without token",
			method:     http.MethodGet,
			path:       "/api/v1/cart",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "add cart item without token",
			method:     http.MethodPost,
			path:       "/api/v1/cart/items",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "update cart item without token",
			method:     http.MethodPatch,
			path:       "/api/v1/cart/items/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "delete cart item without token",
			method:     http.MethodDelete,
			path:       "/api/v1/cart/items/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "checkout without token",
			method:     http.MethodPost,
			path:       "/api/v1/cart/checkout",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "get cart with valid token",
			method:     http.MethodGet,
			path:       "/api/v1/cart",
			token:      validUserToken,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(
				tt.method,
				tt.path,
				nil,
			)

			if tt.token != "" {
				request.Header.Set(
					"Authorization",
					"Bearer "+tt.token,
				)
			}

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				request,
			)

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.wantStatus,
				)
			}
		})
	}
}

func TestNewRouter_AdminRoutes(t *testing.T) {
	t.Parallel()

	router, jwtManager := newTestRouter(t)

	userToken, err := jwtManager.GenerateToken(
		42,
		"user",
	)
	if err != nil {
		t.Fatalf(
			"GenerateToken(user) error = %v",
			err,
		)
	}

	adminToken, err := jwtManager.GenerateToken(
		1,
		"admin",
	)
	if err != nil {
		t.Fatalf(
			"GenerateToken(admin) error = %v",
			err,
		)
	}

	tests := []struct {
		name   string
		method string
		path   string
		token  string

		wantStatus int
	}{
		{
			name:       "create product without token",
			method:     http.MethodPost,
			path:       "/api/v1/admin/products",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "create product with user token",
			method:     http.MethodPost,
			path:       "/api/v1/admin/products",
			token:      userToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "create product with admin token",
			method:     http.MethodPost,
			path:       "/api/v1/admin/products",
			token:      adminToken,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "update product without token",
			method:     http.MethodPatch,
			path:       "/api/v1/admin/products/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "update product with user token",
			method:     http.MethodPatch,
			path:       "/api/v1/admin/products/1",
			token:      userToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "update product with admin token",
			method:     http.MethodPatch,
			path:       "/api/v1/admin/products/1",
			token:      adminToken,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "delete product without token",
			method:     http.MethodDelete,
			path:       "/api/v1/admin/products/1",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "delete product with user token",
			method:     http.MethodDelete,
			path:       "/api/v1/admin/products/1",
			token:      userToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "delete product with admin token",
			method:     http.MethodDelete,
			path:       "/api/v1/admin/products/1",
			token:      adminToken,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "list orders without token",
			method:     http.MethodGet,
			path:       "/api/v1/admin/orders",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "list orders with user token",
			method:     http.MethodGet,
			path:       "/api/v1/admin/orders",
			token:      userToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "list orders with admin token",
			method:     http.MethodGet,
			path:       "/api/v1/admin/orders",
			token:      adminToken,
			wantStatus: http.StatusOK,
		},
		{
			name:       "update order status without token",
			method:     http.MethodPatch,
			path:       "/api/v1/admin/orders/1/status",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "update order status with user token",
			method:     http.MethodPatch,
			path:       "/api/v1/admin/orders/1/status",
			token:      userToken,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "update order status with admin token",
			method:     http.MethodPatch,
			path:       "/api/v1/admin/orders/1/status",
			token:      adminToken,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(
				tt.method,
				tt.path,
				nil,
			)

			if tt.token != "" {
				request.Header.Set(
					"Authorization",
					"Bearer "+tt.token,
				)
			}

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				request,
			)

			if recorder.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					tt.wantStatus,
				)
			}
		})
	}
}

func TestNewRouter_InvalidJWT(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	tests := []struct {
		name          string
		authorization string
	}{
		{
			name:          "missing authorization",
			authorization: "",
		},
		{
			name:          "invalid token",
			authorization: "Bearer invalid-token",
		},
		{
			name:          "missing bearer prefix",
			authorization: "invalid-token",
		},
		{
			name:          "wrong auth scheme",
			authorization: "Basic invalid-token",
		},
		{
			name:          "bearer without token",
			authorization: "Bearer",
		},
		{
			name:          "too many authorization parts",
			authorization: "Bearer token extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/cart",
				nil,
			)

			if tt.authorization != "" {
				request.Header.Set(
					"Authorization",
					tt.authorization,
				)
			}

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				request,
			)

			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					http.StatusUnauthorized,
				)
			}
		})
	}
}

func TestNewRouter_WrongMethods(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "POST products",
			method: http.MethodPost,
			path:   "/api/v1/products",
		},
		{
			name:   "POST product by id",
			method: http.MethodPost,
			path:   "/api/v1/products/1",
		},
		{
			name:   "GET checkout",
			method: http.MethodGet,
			path:   "/api/v1/cart/checkout",
		},
		{
			name:   "GET admin products",
			method: http.MethodGet,
			path:   "/api/v1/admin/products",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(
				tt.method,
				tt.path,
				nil,
			)

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				request,
			)

			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					http.StatusMethodNotAllowed,
				)
			}
		})
	}
}

func TestNewRouter_NotFound(t *testing.T) {
	t.Parallel()

	router, _ := newTestRouter(t)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "unknown endpoint",
			method: http.MethodGet,
			path:   "/api/v1/unknown",
		},
		{
			name:   "unknown product endpoint",
			method: http.MethodGet,
			path:   "/api/v1/products/1/unknown",
		},
		{
			name:   "unknown admin endpoint",
			method: http.MethodGet,
			path:   "/api/v1/admin/unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(
				tt.method,
				tt.path,
				nil,
			)

			recorder := httptest.NewRecorder()

			router.ServeHTTP(
				recorder,
				request,
			)

			if recorder.Code != http.StatusNotFound {
				t.Fatalf(
					"status = %d, want %d",
					recorder.Code,
					http.StatusNotFound,
				)
			}
		})
	}
}
