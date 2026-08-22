package product

import (
	"context"
	"errors"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type mockRepository struct {
	getByIDFunc func(
		ctx context.Context,
		id int64,
	) (Product, error)

	listFunc func(
		ctx context.Context,
	) ([]Product, error)
}

func (m *mockRepository) GetByID(
	ctx context.Context,
	id int64,
) (Product, error) {
	if m.getByIDFunc == nil {
		panic("mockRepository.GetByID called without mock implementation")
	}

	return m.getByIDFunc(ctx, id)
}

func (m *mockRepository) List(
	ctx context.Context,
) ([]Product, error) {
	if m.listFunc == nil {
		panic("mockRepository.List called without mock implementation")
	}

	return m.listFunc(ctx)
}

type mockTransactionManager struct {
	withinTransactionFunc func(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error

	called bool
}

func (m *mockTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	m.called = true

	if m.withinTransactionFunc == nil {
		panic(
			"mockTransactionManager.WithinTransaction " +
				"called without mock implementation",
		)
	}

	return m.withinTransactionFunc(ctx, fn)
}

type mockTransactionRepository struct {
	getByIDForUpdateFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (Product, error)

	decreaseStockFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
		quantity int,
	) error

	increaseStockFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
		quantity int,
	) error

	createFunc func(
		ctx context.Context,
		tx database.Tx,
		product Product,
	) (Product, error)

	updateFunc func(
		ctx context.Context,
		tx database.Tx,
		product Product,
	) error

	deleteFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) error
}

func (m *mockTransactionRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Product, error) {
	if m.getByIDForUpdateFunc == nil {
		panic(
			"mockTransactionRepository.GetByIDForUpdate " +
				"called without mock implementation",
		)
	}

	return m.getByIDForUpdateFunc(ctx, tx, id)
}

func (m *mockTransactionRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	if m.decreaseStockFunc == nil {
		panic(
			"mockTransactionRepository.DecreaseStock " +
				"called without mock implementation",
		)
	}

	return m.decreaseStockFunc(
		ctx,
		tx,
		id,
		quantity,
	)
}

func (m *mockTransactionRepository) IncreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	if m.increaseStockFunc == nil {
		panic(
			"mockTransactionRepository.IncreaseStock " +
				"called without mock implementation",
		)
	}

	return m.increaseStockFunc(
		ctx,
		tx,
		id,
		quantity,
	)
}

func (m *mockTransactionRepository) Create(
	ctx context.Context,
	tx database.Tx,
	product Product,
) (Product, error) {
	if m.createFunc == nil {
		panic(
			"mockTransactionRepository.Create " +
				"called without mock implementation",
		)
	}

	return m.createFunc(ctx, tx, product)
}

func (m *mockTransactionRepository) Update(
	ctx context.Context,
	tx database.Tx,
	product Product,
) error {
	if m.updateFunc == nil {
		panic(
			"mockTransactionRepository.Update " +
				"called without mock implementation",
		)
	}

	return m.updateFunc(ctx, tx, product)
}

func (m *mockTransactionRepository) Delete(
	ctx context.Context,
	tx database.Tx,
	id int64,
) error {
	if m.deleteFunc == nil {
		panic(
			"mockTransactionRepository.Delete " +
				"called without mock implementation",
		)
	}

	return m.deleteFunc(ctx, tx, id)
}

func TestService_GetProductByID(t *testing.T) {
	t.Parallel()

	expected := Product{
		ID:          1,
		Name:        "MacBook Pro",
		Description: "Apple laptop",
		Price:       150000,
		Currency:    "EUR",
		Stock:       10,
	}

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name string

		id int64

		repositoryProduct Product
		repositoryErr     error

		want    Product
		wantErr error

		wantRepositoryCall bool
	}{
		{
			name: "success",

			id: 1,

			repositoryProduct: expected,

			want: expected,

			wantRepositoryCall: true,
		},
		{
			name: "invalid id zero",

			id: 0,

			wantErr: ErrInvalidProduct,

			wantRepositoryCall: false,
		},
		{
			name: "invalid negative id",

			id: -1,

			wantErr: ErrInvalidProduct,

			wantRepositoryCall: false,
		},
		{
			name: "repository error",

			id: 1,

			repositoryErr: repositoryErr,

			wantErr: repositoryErr,

			wantRepositoryCall: true,
		},
		{
			name: "product not found",

			id: 999,

			repositoryErr: ErrProductNotFound,

			wantErr: ErrProductNotFound,

			wantRepositoryCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var repositoryCalled bool

			repository := &mockRepository{
				getByIDFunc: func(
					ctx context.Context,
					id int64,
				) (Product, error) {
					repositoryCalled = true

					if id != tt.id {
						t.Fatalf(
							"repository received id %d, want %d",
							id,
							tt.id,
						)
					}

					return tt.repositoryProduct, tt.repositoryErr
				},
			}

			service := NewService(
				nil,
				repository,
				nil,
			)

			got, err := service.GetProductByID(
				context.Background(),
				tt.id,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetProductByID() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"GetProductByID() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if repositoryCalled != tt.wantRepositoryCall {
				t.Fatalf(
					"repositoryCalled = %v, want %v",
					repositoryCalled,
					tt.wantRepositoryCall,
				)
			}
		})
	}
}

func TestService_GetProducts(t *testing.T) {
	t.Parallel()

	expected := []Product{
		{
			ID:       1,
			Name:     "MacBook Pro",
			Price:    150000,
			Currency: "EUR",
			Stock:    10,
		},
		{
			ID:       2,
			Name:     "Keyboard",
			Price:    12000,
			Currency: "EUR",
			Stock:    50,
		},
	}

	repositoryErr := errors.New("repository error")

	tests := []struct {
		name string

		products []Product
		err      error

		want    []Product
		wantErr error
	}{
		{
			name:     "success",
			products: expected,
			want:     expected,
		},
		{
			name:     "empty result",
			products: []Product{},
			want:     []Product{},
		},
		{
			name:    "repository error",
			err:     repositoryErr,
			wantErr: repositoryErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &mockRepository{
				listFunc: func(
					ctx context.Context,
				) ([]Product, error) {
					return tt.products, tt.err
				},
			}

			service := NewService(
				nil,
				repository,
				nil,
			)

			got, err := service.GetProducts(
				context.Background(),
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetProducts() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if len(got) != len(tt.want) {
				t.Fatalf(
					"GetProducts() length = %d, want %d",
					len(got),
					len(tt.want),
				)
			}

			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf(
						"GetProducts()[%d] = %+v, want %+v",
						i,
						got[i],
						tt.want[i],
					)
				}
			}
		})
	}
}

func successfulTransactionManager(
	t *testing.T,
	tx database.Tx,
) *mockTransactionManager {
	t.Helper()

	return &mockTransactionManager{
		withinTransactionFunc: func(
			ctx context.Context,
			fn func(tx database.Tx) error,
		) error {
			return fn(tx)
		},
	}
}

func TestService_CreateProduct(t *testing.T) {
	t.Parallel()

	repositoryErr := errors.New("create error")

	tests := []struct {
		name string

		request CreateProductRequest

		createProduct Product
		createErr     error

		wantProduct Product
		wantErr     error

		wantTransaction bool
	}{
		{
			name: "success",

			request: CreateProductRequest{
				Name:        "  MacBook Pro  ",
				Description: "  Apple laptop  ",
				Price:       150000,
				Currency:    " eur ",
				Stock:       10,
			},

			createProduct: Product{
				ID:          1,
				Name:        "MacBook Pro",
				Description: "Apple laptop",
				Price:       150000,
				Currency:    "EUR",
				Stock:       10,
			},

			wantProduct: Product{
				ID:          1,
				Name:        "MacBook Pro",
				Description: "Apple laptop",
				Price:       150000,
				Currency:    "EUR",
				Stock:       10,
			},

			wantTransaction: true,
		},

		{
			name: "empty name",

			request: CreateProductRequest{
				Name:        "   ",
				Description: "description",
				Price:       100,
				Currency:    "EUR",
				Stock:       1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "name too long",

			request: CreateProductRequest{
				Name:        string(make([]byte, 256)),
				Description: "description",
				Price:       100,
				Currency:    "EUR",
				Stock:       1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "negative price",

			request: CreateProductRequest{
				Name:     "Product",
				Price:    -1,
				Currency: "EUR",
				Stock:    1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "negative stock",

			request: CreateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EUR",
				Stock:    -1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "empty currency",

			request: CreateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "",
				Stock:    1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "currency too short",

			request: CreateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EU",
				Stock:    1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "currency too long",

			request: CreateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EURO",
				Stock:    1,
			},

			wantErr:         ErrInvalidProduct,
			wantTransaction: false,
		},

		{
			name: "transaction repository error",

			request: CreateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EUR",
				Stock:    1,
			},

			createErr: repositoryErr,

			wantErr:         repositoryErr,
			wantTransaction: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			transactionManager := &mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					return fn(nil)
				},
			}

			transactionRepository := &mockTransactionRepository{
				createFunc: func(
					ctx context.Context,
					tx database.Tx,
					product Product,
				) (Product, error) {
					if product.Name == "" {
						t.Fatal("Create received empty product name")
					}

					return tt.createProduct, tt.createErr
				},
			}

			service := NewService(
				transactionManager,
				nil,
				transactionRepository,
			)

			got, err := service.CreateProduct(
				context.Background(),
				tt.request,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"CreateProduct() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.wantProduct {
				t.Fatalf(
					"CreateProduct() = %+v, want %+v",
					got,
					tt.wantProduct,
				)
			}

			if transactionManager.called != tt.wantTransaction {
				t.Fatalf(
					"transaction called = %v, want %v",
					transactionManager.called,
					tt.wantTransaction,
				)
			}
		})
	}
}

func TestService_UpdateProduct(t *testing.T) {
	t.Parallel()

	getErr := errors.New("get error")
	updateErr := errors.New("update error")

	tests := []struct {
		name string

		id      int64
		request UpdateProductRequest

		current Product

		getErr    error
		updateErr error

		want       Product
		wantErr    error
		wantGet    bool
		wantUpdate bool
	}{
		{
			name: "success",

			id: 1,

			request: UpdateProductRequest{
				Name:        "  New Product  ",
				Description: "  New description  ",
				Price:       200,
				Currency:    " eur ",
				Stock:       20,
			},

			current: Product{
				ID: 1,
			},

			want: Product{
				ID:          1,
				Name:        "New Product",
				Description: "New description",
				Price:       200,
				Currency:    "EUR",
				Stock:       20,
			},

			wantGet:    true,
			wantUpdate: true,
		},

		{
			name: "invalid id",

			id: 0,

			request: UpdateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EUR",
				Stock:    1,
			},

			wantErr:    ErrInvalidProduct,
			wantGet:    false,
			wantUpdate: false,
		},

		{
			name: "invalid product",

			id: 1,

			request: UpdateProductRequest{
				Name:     "",
				Price:    100,
				Currency: "EUR",
				Stock:    1,
			},

			wantErr:    ErrInvalidProduct,
			wantGet:    false,
			wantUpdate: false,
		},

		{
			name: "product not found",

			id: 999,

			request: UpdateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EUR",
				Stock:    1,
			},

			getErr: ErrProductNotFound,

			wantErr:    ErrProductNotFound,
			wantGet:    true,
			wantUpdate: false,
		},

		{
			name: "get error",

			id: 1,

			request: UpdateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EUR",
				Stock:    1,
			},

			getErr: getErr,

			wantErr:    getErr,
			wantGet:    true,
			wantUpdate: false,
		},

		{
			name: "update error",

			id: 1,

			request: UpdateProductRequest{
				Name:     "Product",
				Price:    100,
				Currency: "EUR",
				Stock:    1,
			},

			current: Product{
				ID: 1,
			},

			updateErr: updateErr,

			wantErr:    updateErr,
			wantGet:    true,
			wantUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var getCalled bool
			var updateCalled bool

			transactionManager := &mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					return fn(nil)
				},
			}

			transactionRepository := &mockTransactionRepository{
				getByIDForUpdateFunc: func(
					ctx context.Context,
					tx database.Tx,
					id int64,
				) (Product, error) {
					getCalled = true

					if id != tt.id {
						t.Fatalf(
							"GetByIDForUpdate id = %d, want %d",
							id,
							tt.id,
						)
					}

					return tt.current, tt.getErr
				},

				updateFunc: func(
					ctx context.Context,
					tx database.Tx,
					product Product,
				) error {
					updateCalled = true

					if product.ID != tt.id {
						t.Fatalf(
							"Update product ID = %d, want %d",
							product.ID,
							tt.id,
						)
					}

					return tt.updateErr
				},
			}

			service := NewService(
				transactionManager,
				nil,
				transactionRepository,
			)

			got, err := service.UpdateProduct(
				context.Background(),
				tt.id,
				tt.request,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"UpdateProduct() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"UpdateProduct() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if getCalled != tt.wantGet {
				t.Fatalf(
					"GetByIDForUpdate called = %v, want %v",
					getCalled,
					tt.wantGet,
				)
			}

			if updateCalled != tt.wantUpdate {
				t.Fatalf(
					"Update called = %v, want %v",
					updateCalled,
					tt.wantUpdate,
				)
			}
		})
	}
}

func TestService_DeleteProduct(t *testing.T) {
	t.Parallel()

	deleteErr := errors.New("delete error")

	tests := []struct {
		name string

		id int64

		deleteErr error

		wantErr          error
		wantTransaction  bool
		wantDeleteCalled bool
	}{
		{
			name: "success",

			id: 1,

			wantTransaction:  true,
			wantDeleteCalled: true,
		},
		{
			name: "invalid id zero",

			id: 0,

			wantErr:          ErrInvalidProduct,
			wantTransaction:  false,
			wantDeleteCalled: false,
		},
		{
			name: "invalid negative id",

			id: -1,

			wantErr:          ErrInvalidProduct,
			wantTransaction:  false,
			wantDeleteCalled: false,
		},
		{
			name: "delete error",

			id: 1,

			deleteErr: deleteErr,

			wantErr:          deleteErr,
			wantTransaction:  true,
			wantDeleteCalled: true,
		},
		{
			name: "product not found",

			id: 999,

			deleteErr: ErrProductNotFound,

			wantErr:          ErrProductNotFound,
			wantTransaction:  true,
			wantDeleteCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var deleteCalled bool

			transactionManager := &mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					return fn(nil)
				},
			}

			transactionRepository := &mockTransactionRepository{
				deleteFunc: func(
					ctx context.Context,
					tx database.Tx,
					id int64,
				) error {
					deleteCalled = true

					if id != tt.id {
						t.Fatalf(
							"Delete id = %d, want %d",
							id,
							tt.id,
						)
					}

					return tt.deleteErr
				},
			}

			service := NewService(
				transactionManager,
				nil,
				transactionRepository,
			)

			err := service.DeleteProduct(
				context.Background(),
				tt.id,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"DeleteProduct() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if transactionManager.called != tt.wantTransaction {
				t.Fatalf(
					"transaction called = %v, want %v",
					transactionManager.called,
					tt.wantTransaction,
				)
			}

			if deleteCalled != tt.wantDeleteCalled {
				t.Fatalf(
					"Delete called = %v, want %v",
					deleteCalled,
					tt.wantDeleteCalled,
				)
			}
		})
	}
}

func TestService_UpdateProduct_UsesLockedProductID(t *testing.T) {
	t.Parallel()

	transactionManager := &mockTransactionManager{
		withinTransactionFunc: func(
			ctx context.Context,
			fn func(tx database.Tx) error,
		) error {
			return fn(nil)
		},
	}

	transactionRepository := &mockTransactionRepository{
		getByIDForUpdateFunc: func(
			ctx context.Context,
			tx database.Tx,
			id int64,
		) (Product, error) {
			return Product{
				ID: 42,
			}, nil
		},

		updateFunc: func(
			ctx context.Context,
			tx database.Tx,
			product Product,
		) error {
			if product.ID != 42 {
				t.Fatalf(
					"Update received product ID %d, want 42",
					product.ID,
				)
			}

			return nil
		},
	}

	service := NewService(
		transactionManager,
		nil,
		transactionRepository,
	)

	_, err := service.UpdateProduct(
		context.Background(),
		1,
		UpdateProductRequest{
			Name:     "Product",
			Price:    100,
			Currency: "EUR",
			Stock:    1,
		},
	)
	if err != nil {
		t.Fatalf(
			"UpdateProduct() error = %v",
			err,
		)
	}
}
