package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockProductService struct {
	getProductsFunc    func(ctx context.Context) ([]Product, error)
	getProductByIDFunc func(ctx context.Context, id int64) (Product, error)
	createProductFunc  func(ctx context.Context, request CreateProductRequest) (Product, error)
	updateProductFunc  func(ctx context.Context, id int64, request UpdateProductRequest) (Product, error)
	deleteProductFunc  func(ctx context.Context, id int64) error
}

func (m *mockProductService) GetProducts(
	ctx context.Context,
) ([]Product, error) {
	if m.getProductsFunc == nil {
		panic("getProductsFunc is nil")
	}

	return m.getProductsFunc(ctx)
}

func (m *mockProductService) GetProductByID(
	ctx context.Context,
	id int64,
) (Product, error) {
	if m.getProductByIDFunc == nil {
		panic("getProductByIDFunc is nil")
	}

	return m.getProductByIDFunc(ctx, id)
}

func (m *mockProductService) CreateProduct(
	ctx context.Context,
	request CreateProductRequest,
) (Product, error) {
	if m.createProductFunc == nil {
		panic("createProductFunc is nil")
	}

	return m.createProductFunc(ctx, request)
}

func (m *mockProductService) UpdateProduct(
	ctx context.Context,
	id int64,
	request UpdateProductRequest,
) (Product, error) {
	if m.updateProductFunc == nil {
		panic("updateProductFunc is nil")
	}

	return m.updateProductFunc(ctx, id, request)
}

func (m *mockProductService) DeleteProduct(
	ctx context.Context,
	id int64,
) error {
	if m.deleteProductFunc == nil {
		panic("deleteProductFunc is nil")
	}

	return m.deleteProductFunc(ctx, id)
}

func TestHandler_GetProducts(t *testing.T) {
	t.Parallel()

	internalErr := errors.New("database error")

	tests := []struct {
		name string

		products   []Product
		serviceErr error

		wantStatus      int
		wantContentType string
		wantBody        string
	}{
		{
			name: "success",

			products: []Product{
				{
					ID:          1,
					Name:        "MacBook Pro",
					Description: "Apple laptop",
					Price:       150000,
					Currency:    "EUR",
					Stock:       10,
				},
				{
					ID:          2,
					Name:        "Keyboard",
					Description: "Mechanical keyboard",
					Price:       12000,
					Currency:    "EUR",
					Stock:       50,
				},
			},

			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantBody: `{
				"items": [
					{
						"id": 1,
						"name": "MacBook Pro",
						"description": "Apple laptop",
						"price": 150000,
						"currency": "EUR",
						"stock": 10
					},
					{
						"id": 2,
						"name": "Keyboard",
						"description": "Mechanical keyboard",
						"price": 12000,
						"currency": "EUR",
						"stock": 50
					}
				]
			}`,
		},

		{
			name: "empty list",

			products: []Product{},

			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantBody: `{
				"items": []
			}`,
		},

		{
			name: "service error",

			serviceErr: internalErr,

			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := &mockProductService{
				getProductsFunc: func(ctx context.Context) ([]Product, error) {
					return tt.products, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodGet,
				"/products",
				nil,
			)

			rec := httptest.NewRecorder()

			handler.GetProducts(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"GetProducts() status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}

			if tt.wantContentType != "" {
				contentType := rec.Header().Get("Content-Type")

				if !strings.HasPrefix(
					contentType,
					tt.wantContentType,
				) {
					t.Fatalf(
						"GetProducts() Content-Type = %q, want prefix %q",
						contentType,
						tt.wantContentType,
					)
				}
			}

			if tt.wantBody != "" {
				var got any
				var want any

				if err := json.Unmarshal(
					rec.Body.Bytes(),
					&got,
				); err != nil {
					if rec.Body.String() != tt.wantBody {
						t.Fatalf(
							"GetProducts() body = %q, want %q",
							rec.Body.String(),
							tt.wantBody,
						)
					}

					return
				}

				if err := json.Unmarshal(
					[]byte(tt.wantBody),
					&want,
				); err != nil {
					t.Fatalf("invalid expected JSON: %v", err)
				}

				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(want)

				if string(gotJSON) != string(wantJSON) {
					t.Fatalf(
						"GetProducts() body = %s, want %s",
						gotJSON,
						wantJSON,
					)
				}
			}
		})
	}
}

func TestHandler_GetProductByID(t *testing.T) {
	t.Parallel()

	internalErr := errors.New("database error")

	product := Product{
		ID:          1,
		Name:        "MacBook Pro",
		Description: "Apple laptop",
		Price:       150000,
		Currency:    "EUR",
		Stock:       10,
	}

	tests := []struct {
		name string

		path string

		product    Product
		serviceErr error

		wantStatus int
		wantBody   string
	}{
		{
			name: "success",

			path: "/products/1",

			product: product,

			wantStatus: http.StatusOK,
			wantBody: `{
				"id": 1,
				"name": "MacBook Pro",
				"description": "Apple laptop",
				"price": 150000,
				"currency": "EUR",
				"stock": 10
			}`,
		},

		{
			name: "invalid id",

			path: "/products/abc",

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "zero id",

			path: "/products/0",

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "negative id",

			path: "/products/-1",

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "not found",

			path: "/products/999",

			serviceErr: ErrProductNotFound,

			wantStatus: http.StatusNotFound,
			wantBody:   "product not found\n",
		},

		{
			name: "invalid product",

			path: "/products/1",

			serviceErr: ErrInvalidProduct,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "service error",

			path: "/products/1",

			serviceErr: internalErr,

			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			service := &mockProductService{
				getProductByIDFunc: func(
					ctx context.Context,
					id int64,
				) (Product, error) {
					called = true

					if id != 1 && tt.serviceErr != ErrProductNotFound {
						t.Fatalf(
							"GetProductByID() id = %d, want 1",
							id,
						)
					}

					return tt.product, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodGet,
				tt.path,
				nil,
			)

			req.SetPathValue(
				"id",
				strings.TrimPrefix(tt.path, "/products/"),
			)

			rec := httptest.NewRecorder()

			handler.GetProductByID(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"GetProductByID() status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}

			if rec.Body.String() != tt.wantBody &&
				tt.wantBody != "" {
				var got any
				var want any

				if json.Unmarshal(
					rec.Body.Bytes(),
					&got,
				) == nil &&
					json.Unmarshal(
						[]byte(tt.wantBody),
						&want,
					) == nil {

					gotJSON, _ := json.Marshal(got)
					wantJSON, _ := json.Marshal(want)

					if string(gotJSON) != string(wantJSON) {
						t.Fatalf(
							"GetProductByID() body = %s, want %s",
							gotJSON,
							wantJSON,
						)
					}
				} else if rec.Body.String() != tt.wantBody {
					t.Fatalf(
						"GetProductByID() body = %q, want %q",
						rec.Body.String(),
						tt.wantBody,
					)
				}
			}

			if tt.wantStatus == http.StatusBadRequest &&
				(strings.HasPrefix(tt.path, "/products/abc") ||
					tt.path == "/products/0" ||
					tt.path == "/products/-1") &&
				called {
				t.Fatal("GetProductByID() service was called for invalid ID")
			}
		})
	}
}

func TestHandler_CreateProduct(t *testing.T) {
	t.Parallel()

	internalErr := errors.New("database error")

	request := CreateProductRequest{
		Name:        "Product",
		Description: "Description",
		Price:       100,
		Currency:    "EUR",
		Stock:       10,
	}

	product := Product{
		ID:          1,
		Name:        "Product",
		Description: "Description",
		Price:       100,
		Currency:    "EUR",
		Stock:       10,
	}

	tests := []struct {
		name string

		body string

		product    Product
		serviceErr error

		wantStatus int
		wantBody   string
	}{
		{
			name: "success",

			body: `{
				"name": "Product",
				"description": "Description",
				"price": 100,
				"currency": "EUR",
				"stock": 10
			}`,

			product: product,

			wantStatus: http.StatusCreated,
			wantBody: `{
				"id": 1,
				"name": "Product",
				"description": "Description",
				"price": 100,
				"currency": "EUR",
				"stock": 10
			}`,
		},

		{
			name: "invalid json",

			body: `{"name":`,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid request body\n",
		},

		{
			name: "invalid product",

			body: `{
				"name": "",
				"description": "Description",
				"price": 100,
				"currency": "EUR",
				"stock": 10
			}`,

			serviceErr: ErrInvalidProduct,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product\n",
		},

		{
			name: "service error",

			body: `{
				"name": "Product",
				"description": "Description",
				"price": 100,
				"currency": "EUR",
				"stock": 10
			}`,

			serviceErr: internalErr,

			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			service := &mockProductService{
				createProductFunc: func(
					ctx context.Context,
					gotRequest CreateProductRequest,
				) (Product, error) {
					called = true

					if tt.name == "success" {
						if gotRequest != request {
							t.Fatalf(
								"CreateProduct() request = %+v, want %+v",
								gotRequest,
								request,
							)
						}
					}

					return tt.product, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodPost,
				"/products",
				strings.NewReader(tt.body),
			)

			rec := httptest.NewRecorder()

			handler.CreateProduct(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"CreateProduct() status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}

			if tt.wantBody != "" {
				if tt.wantStatus >= 400 {
					if rec.Body.String() != tt.wantBody {
						t.Fatalf(
							"CreateProduct() body = %q, want %q",
							rec.Body.String(),
							tt.wantBody,
						)
					}
				} else {
					var got any
					var want any

					if err := json.Unmarshal(
						rec.Body.Bytes(),
						&got,
					); err != nil {
						t.Fatalf(
							"invalid response JSON: %v",
							err,
						)
					}

					if err := json.Unmarshal(
						[]byte(tt.wantBody),
						&want,
					); err != nil {
						t.Fatalf(
							"invalid expected JSON: %v",
							err,
						)
					}

					gotJSON, _ := json.Marshal(got)
					wantJSON, _ := json.Marshal(want)

					if string(gotJSON) != string(wantJSON) {
						t.Fatalf(
							"CreateProduct() body = %s, want %s",
							gotJSON,
							wantJSON,
						)
					}
				}
			}

			if tt.name == "invalid json" && called {
				t.Fatal("CreateProduct() service was called for invalid JSON")
			}
		})
	}
}

func TestHandler_UpdateProduct(t *testing.T) {
	t.Parallel()

	internalErr := errors.New("database error")

	request := UpdateProductRequest{
		Name:        "Updated",
		Description: "Updated description",
		Price:       200,
		Currency:    "EUR",
		Stock:       20,
	}

	product := Product{
		ID:          1,
		Name:        "Updated",
		Description: "Updated description",
		Price:       200,
		Currency:    "EUR",
		Stock:       20,
	}

	tests := []struct {
		name string

		path string
		body string

		product    Product
		serviceErr error

		wantStatus int
		wantBody   string
	}{
		{
			name: "success",

			path: "/products/1",

			body: `{
				"name": "Updated",
				"description": "Updated description",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,

			product: product,

			wantStatus: http.StatusOK,
			wantBody: `{
				"id": 1,
				"name": "Updated",
				"description": "Updated description",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,
		},

		{
			name: "invalid id",

			path: "/products/abc",

			body: `{
				"name": "Updated",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "zero id",

			path: "/products/0",

			body: `{
				"name": "Updated",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "invalid body",

			path: "/products/1",

			body: `{"name":`,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid request body\n",
		},

		{
			name: "invalid product",

			path: "/products/1",

			body: `{
				"name": "",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,

			serviceErr: ErrInvalidProduct,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product\n",
		},

		{
			name: "not found",

			path: "/products/999",

			body: `{
				"name": "Updated",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,

			serviceErr: ErrProductNotFound,

			wantStatus: http.StatusNotFound,
			wantBody:   "product not found\n",
		},

		{
			name: "service error",

			path: "/products/1",

			body: `{
				"name": "Updated",
				"price": 200,
				"currency": "EUR",
				"stock": 20
			}`,

			serviceErr: internalErr,

			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			service := &mockProductService{
				updateProductFunc: func(
					ctx context.Context,
					id int64,
					gotRequest UpdateProductRequest,
				) (Product, error) {
					called = true

					if id != 1 &&
						tt.name != "not found" {
						t.Fatalf(
							"UpdateProduct() id = %d, want 1",
							id,
						)
					}

					if tt.name == "success" &&
						gotRequest != request {
						t.Fatalf(
							"UpdateProduct() request = %+v, want %+v",
							gotRequest,
							request,
						)
					}

					return tt.product, tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodPut,
				tt.path,
				strings.NewReader(tt.body),
			)

			req.SetPathValue(
				"id",
				strings.TrimPrefix(tt.path, "/products/"),
			)

			rec := httptest.NewRecorder()

			handler.UpdateProduct(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"UpdateProduct() status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}

			if tt.wantBody != "" {
				if tt.wantStatus >= 400 {
					if rec.Body.String() != tt.wantBody {
						t.Fatalf(
							"UpdateProduct() body = %q, want %q",
							rec.Body.String(),
							tt.wantBody,
						)
					}
				} else {
					var got any
					var want any

					if err := json.Unmarshal(
						rec.Body.Bytes(),
						&got,
					); err != nil {
						t.Fatalf(
							"invalid response JSON: %v",
							err,
						)
					}

					if err := json.Unmarshal(
						[]byte(tt.wantBody),
						&want,
					); err != nil {
						t.Fatalf(
							"invalid expected JSON: %v",
							err,
						)
					}

					gotJSON, _ := json.Marshal(got)
					wantJSON, _ := json.Marshal(want)

					if string(gotJSON) != string(wantJSON) {
						t.Fatalf(
							"UpdateProduct() body = %s, want %s",
							gotJSON,
							wantJSON,
						)
					}
				}
			}

			if (tt.name == "invalid id" ||
				tt.name == "zero id" ||
				tt.name == "invalid body") &&
				called {
				t.Fatal("UpdateProduct() service was called for invalid request")
			}
		})
	}
}

func TestHandler_DeleteProduct(t *testing.T) {
	t.Parallel()

	internalErr := errors.New("database error")

	tests := []struct {
		name string

		path string

		serviceErr error

		wantStatus int
		wantBody   string
	}{
		{
			name: "success",

			path: "/products/1",

			wantStatus: http.StatusNoContent,
		},

		{
			name: "invalid id",

			path: "/products/abc",

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "zero id",

			path: "/products/0",

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "negative id",

			path: "/products/-1",

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "invalid product",

			path: "/products/1",

			serviceErr: ErrInvalidProduct,

			wantStatus: http.StatusBadRequest,
			wantBody:   "invalid product id\n",
		},

		{
			name: "not found",

			path: "/products/999",

			serviceErr: ErrProductNotFound,

			wantStatus: http.StatusNotFound,
			wantBody:   "product not found\n",
		},

		{
			name: "service error",

			path: "/products/1",

			serviceErr: internalErr,

			wantStatus: http.StatusInternalServerError,
			wantBody:   "internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			called := false

			service := &mockProductService{
				deleteProductFunc: func(
					ctx context.Context,
					id int64,
				) error {
					called = true

					if id != 1 &&
						tt.name != "not found" {
						t.Fatalf(
							"DeleteProduct() id = %d, want 1",
							id,
						)
					}

					return tt.serviceErr
				},
			}

			handler := NewHandler(service)

			req := httptest.NewRequest(
				http.MethodDelete,
				tt.path,
				nil,
			)

			req.SetPathValue(
				"id",
				strings.TrimPrefix(tt.path, "/products/"),
			)

			rec := httptest.NewRecorder()

			handler.DeleteProduct(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"DeleteProduct() status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}

			if tt.wantBody != "" &&
				rec.Body.String() != tt.wantBody {
				t.Fatalf(
					"DeleteProduct() body = %q, want %q",
					rec.Body.String(),
					tt.wantBody,
				)
			}

			if tt.wantStatus == http.StatusNoContent &&
				rec.Body.Len() != 0 {
				t.Fatalf(
					"DeleteProduct() body = %q, want empty body",
					rec.Body.String(),
				)
			}

			if (tt.name == "invalid id" ||
				tt.name == "zero id" ||
				tt.name == "negative id") &&
				called {
				t.Fatal("DeleteProduct() service was called for invalid ID")
			}
		})
	}
}
