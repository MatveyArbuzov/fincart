package cart

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

var (
	errDatabase    = errors.New("database error")
	errTransaction = errors.New("transaction error")
)

type fakeTx struct{}

func (fakeTx) Commit() error {
	return nil
}

func (fakeTx) Rollback() error {
	return nil
}

func (fakeTx) ExecContext(
	context.Context,
	string,
	...any,
) (sql.Result, error) {
	return nil, nil
}

func (fakeTx) QueryContext(
	context.Context,
	string,
	...any,
) (*sql.Rows, error) {
	return nil, nil
}

func (fakeTx) QueryRowContext(
	context.Context,
	string,
	...any,
) *sql.Row {
	return nil
}

type fakeTransactionManager struct {
	err error

	calls int
	tx    database.Tx
}

func (m *fakeTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	m.calls++

	if m.err != nil {
		return m.err
	}

	if m.tx == nil {
		m.tx = fakeTx{}
	}

	return fn(m.tx)
}

type fakeCartRepository struct {
	getDraftFn     func(context.Context, database.Tx, int64) (Cart, error)
	createDraftFn  func(context.Context, database.Tx, int64) (Cart, error)
	getItemFn      func(context.Context, database.Tx, int64, int64) (Item, error)
	addItemFn      func(context.Context, database.Tx, int64, int64, int, int64) (Item, error)
	updateItemFn   func(context.Context, database.Tx, int64, int64, int) error
	deleteItemFn   func(context.Context, database.Tx, int64, int64) error
	getItemsFn     func(context.Context, database.Tx, int64) ([]Item, error)
	updateTotalFn  func(context.Context, database.Tx, int64, int64, string) error
	checkoutFn     func(context.Context, database.Tx, int64) error
	getDraftByIDFn func(context.Context, database.Tx, int64) (Cart, error)
	updatePriceFn  func(context.Context, database.Tx, int64, int64, int64) error
}

func (r *fakeCartRepository) GetDraft(
	ctx context.Context,
	tx database.Tx,
	userID int64,
) (Cart, error) {
	if r.getDraftFn != nil {
		return r.getDraftFn(ctx, tx, userID)
	}

	return Cart{}, nil
}

func (r *fakeCartRepository) CreateDraft(
	ctx context.Context,
	tx database.Tx,
	userID int64,
) (Cart, error) {
	if r.createDraftFn != nil {
		return r.createDraftFn(ctx, tx, userID)
	}

	return Cart{}, nil
}

func (r *fakeCartRepository) GetItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
) (Item, error) {
	if r.getItemFn != nil {
		return r.getItemFn(ctx, tx, orderID, productID)
	}

	return Item{}, sql.ErrNoRows
}

func (r *fakeCartRepository) AddItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
	quantity int,
	unitPrice int64,
) (Item, error) {
	if r.addItemFn != nil {
		return r.addItemFn(
			ctx,
			tx,
			orderID,
			productID,
			quantity,
			unitPrice,
		)
	}

	return Item{}, nil
}

func (r *fakeCartRepository) UpdateItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
	quantity int,
) error {
	if r.updateItemFn != nil {
		return r.updateItemFn(
			ctx,
			tx,
			orderID,
			productID,
			quantity,
		)
	}

	return nil
}

func (r *fakeCartRepository) DeleteItem(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
) error {
	if r.deleteItemFn != nil {
		return r.deleteItemFn(
			ctx,
			tx,
			orderID,
			productID,
		)
	}

	return nil
}

func (r *fakeCartRepository) GetItems(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) ([]Item, error) {
	if r.getItemsFn != nil {
		return r.getItemsFn(ctx, tx, orderID)
	}

	return []Item{}, nil
}

func (r *fakeCartRepository) UpdateTotal(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	totalAmount int64,
	currency string,
) error {
	if r.updateTotalFn != nil {
		return r.updateTotalFn(
			ctx,
			tx,
			orderID,
			totalAmount,
			currency,
		)
	}

	return nil
}

func (r *fakeCartRepository) Checkout(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) error {
	if r.checkoutFn != nil {
		return r.checkoutFn(ctx, tx, orderID)
	}

	return nil
}

func (r *fakeCartRepository) GetDraftByID(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) (Cart, error) {
	if r.getDraftByIDFn != nil {
		return r.getDraftByIDFn(ctx, tx, orderID)
	}

	return Cart{
		ID:     orderID,
		Status: string(order.OrderStatusDraft),
	}, nil
}

func (r *fakeCartRepository) UpdateItemPrice(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	productID int64,
	unitPrice int64,
) error {
	if r.updatePriceFn != nil {
		return r.updatePriceFn(
			ctx,
			tx,
			orderID,
			productID,
			unitPrice,
		)
	}

	return nil
}

type fakeProductRepository struct {
	getByIDFn       func(context.Context, database.Tx, int64) (product.Product, error)
	decreaseStockFn func(context.Context, database.Tx, int64, int) error
}

func (r *fakeProductRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (product.Product, error) {
	if r.getByIDFn != nil {
		return r.getByIDFn(ctx, tx, id)
	}

	return product.Product{}, nil
}

func (r *fakeProductRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	if r.decreaseStockFn != nil {
		return r.decreaseStockFn(ctx, tx, id, quantity)
	}

	return nil
}

func newTestService(
	repository *fakeCartRepository,
	products *fakeProductRepository,
	manager *fakeTransactionManager,
) *Service {
	return NewService(
		manager,
		repository,
		products,
	)
}

func TestService_GetCart(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     int64
		repository *fakeCartRepository
		managerErr error
		want       Cart
		wantErr    error
	}{
		{
			name:    "invalid user id",
			userID:  0,
			wantErr: ErrInvalidCart,
		},
		{
			name:    "negative user id",
			userID:  -1,
			wantErr: ErrInvalidCart,
		},
		{
			name:   "returns empty cart when draft does not exist",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{}, nil
				},
			},
			want: Cart{
				UserID: 1,
				Status: string(order.OrderStatusDraft),
				Items:  []Item{},
			},
		},
		{
			name:   "returns existing cart with items",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{
						ID:     10,
						UserID: 1,
						Status: string(order.OrderStatusDraft),
					}, nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{
						{
							ID:        100,
							ProductID: 5,
							Quantity:  2,
						},
					}, nil
				},
			},
			want: Cart{
				ID:     10,
				UserID: 1,
				Status: string(order.OrderStatusDraft),
				Items: []Item{
					{
						ID:        100,
						ProductID: 5,
						Quantity:  2,
					},
				},
			},
		},
		{
			name:   "repository error",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{}, errDatabase
				},
			},
			wantErr: errDatabase,
		},
		{
			name:       "transaction error",
			userID:     1,
			managerErr: errTransaction,
			wantErr:    errTransaction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &fakeTransactionManager{
				err: tt.managerErr,
			}

			repository := tt.repository
			if repository == nil {
				repository = &fakeCartRepository{}
			}

			service := newTestService(
				repository,
				&fakeProductRepository{},
				manager,
			)

			got, err := service.GetCart(
				context.Background(),
				tt.userID,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetCart() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr == nil && !equalCart(got, tt.want) {
				t.Fatalf(
					"GetCart() = %+v, want %+v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestService_AddItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		userID       int64
		request      AddItemRequest
		repository   *fakeCartRepository
		products     *fakeProductRepository
		wantErr      error
		wantAdd      bool
		wantUpdate   bool
		wantQuantity int
		wantPrice    int64
	}{
		{
			name:   "invalid user",
			userID: 0,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  1,
			},
			wantErr: ErrInvalidCart,
		},
		{
			name:   "invalid product",
			userID: 1,
			request: AddItemRequest{
				ProductID: 0,
				Quantity:  1,
			},
			wantErr: ErrInvalidCart,
		},
		{
			name:   "invalid quantity",
			userID: 1,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  0,
			},
			wantErr: ErrInvalidQuantity,
		},
		{
			name:   "product not found",
			userID: 1,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  1,
			},
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{}, product.ErrProductNotFound
				},
			},
			wantErr: ErrProductNotFound,
		},
		{
			name:   "insufficient stock for new item",
			userID: 1,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  5,
			},
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{
						ID:       1,
						Price:    100,
						Currency: "EUR",
						Stock:    3,
					}, nil
				},
			},
			wantErr: ErrInsufficientStock,
		},
		{
			name:   "adds new item",
			userID: 1,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  2,
			},
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
				getItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
				) (Item, error) {
					return Item{}, sql.ErrNoRows
				},
				addItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
					int,
					int64,
				) (Item, error) {
					return Item{
						ProductID: 1,
						Quantity:  2,
						UnitPrice: 100,
					}, nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{
						{
							ProductID: 1,
							Quantity:  2,
							UnitPrice: 100,
							Currency:  "EUR",
						},
					}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{
						ID:       1,
						Price:    100,
						Currency: "EUR",
						Stock:    5,
					}, nil
				},
			},
			wantAdd:      true,
			wantQuantity: 2,
			wantPrice:    100,
		},
		{
			name:   "increases existing item quantity",
			userID: 1,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  2,
			},
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
				getItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
				) (Item, error) {
					return Item{
						ProductID: 1,
						Quantity:  3,
					}, nil
				},
				updateItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
					int,
				) error {
					return nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{
						{
							ProductID: 1,
							Quantity:  5,
							UnitPrice: 100,
							Currency:  "EUR",
						},
					}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{
						ID:       1,
						Price:    100,
						Currency: "EUR",
						Stock:    5,
					}, nil
				},
			},
			wantUpdate:   true,
			wantQuantity: 5,
		},
		{
			name:   "existing item exceeds stock",
			userID: 1,
			request: AddItemRequest{
				ProductID: 1,
				Quantity:  3,
			},
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
				getItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
				) (Item, error) {
					return Item{
						ProductID: 1,
						Quantity:  3,
					}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{
						ID:    1,
						Stock: 5,
					}, nil
				},
			},
			wantErr: ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			if repository == nil {
				repository = &fakeCartRepository{}
			}

			products := tt.products
			if products == nil {
				products = &fakeProductRepository{}
			}

			service := newTestService(
				repository,
				products,
				&fakeTransactionManager{},
			)

			_, err := service.AddItem(
				context.Background(),
				tt.userID,
				tt.request,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"AddItem() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestService_UpdateItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     int64
		productID  int64
		quantity   int
		repository *fakeCartRepository
		products   *fakeProductRepository
		wantErr    error
	}{
		{
			name:      "invalid user",
			userID:    0,
			productID: 1,
			quantity:  1,
			wantErr:   ErrInvalidCart,
		},
		{
			name:      "invalid product",
			userID:    1,
			productID: 0,
			quantity:  1,
			wantErr:   ErrInvalidCart,
		},
		{
			name:      "invalid quantity",
			userID:    1,
			productID: 1,
			quantity:  0,
			wantErr:   ErrInvalidQuantity,
		},
		{
			name:      "cart not found",
			userID:    1,
			productID: 1,
			quantity:  1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{}, nil
				},
			},
			wantErr: ErrCartNotFound,
		},
		{
			name:      "product not found",
			userID:    1,
			productID: 1,
			quantity:  1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{}, product.ErrProductNotFound
				},
			},
			wantErr: ErrProductNotFound,
		},
		{
			name:      "insufficient stock",
			userID:    1,
			productID: 1,
			quantity:  5,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{Stock: 2}, nil
				},
			},
			wantErr: ErrInsufficientStock,
		},
		{
			name:      "item not found",
			userID:    1,
			productID: 1,
			quantity:  1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
				updateItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
					int,
				) error {
					return sql.ErrNoRows
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{Stock: 10}, nil
				},
			},
			wantErr: ErrCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			if repository == nil {
				repository = &fakeCartRepository{}
			}

			products := tt.products
			if products == nil {
				products = &fakeProductRepository{}
			}

			service := newTestService(
				repository,
				products,
				&fakeTransactionManager{},
			)

			_, err := service.UpdateItem(
				context.Background(),
				tt.userID,
				tt.productID,
				tt.quantity,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"UpdateItem() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestService_DeleteItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		userID     int64
		productID  int64
		repository *fakeCartRepository
		wantErr    error
	}{
		{
			name:      "invalid user",
			userID:    0,
			productID: 1,
			wantErr:   ErrInvalidCart,
		},
		{
			name:      "invalid product",
			userID:    1,
			productID: 0,
			wantErr:   ErrInvalidCart,
		},
		{
			name:      "cart not found",
			userID:    1,
			productID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{}, nil
				},
			},
			wantErr: ErrCartNotFound,
		},
		{
			name:      "item not found",
			userID:    1,
			productID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{ID: 10}, nil
				},
				deleteItemFn: func(
					context.Context,
					database.Tx,
					int64,
					int64,
				) error {
					return sql.ErrNoRows
				},
			},
			wantErr: ErrCartItemNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			if repository == nil {
				repository = &fakeCartRepository{}
			}

			service := newTestService(
				repository,
				&fakeProductRepository{},
				&fakeTransactionManager{},
			)

			_, err := service.DeleteItem(
				context.Background(),
				tt.userID,
				tt.productID,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"DeleteItem() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestService_Checkout(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(
		2026,
		8,
		22,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name       string
		userID     int64
		repository *fakeCartRepository
		products   *fakeProductRepository
		want       order.Order
		wantErr    error
	}{
		{
			name:    "invalid user",
			userID:  0,
			wantErr: ErrInvalidCart,
		},
		{
			name:   "cart not found",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{}, nil
				},
			},
			wantErr: ErrCartNotFound,
		},
		{
			name:   "empty cart",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{
						ID:        10,
						UserID:    1,
						CreatedAt: createdAt,
					}, nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{}, nil
				},
			},
			wantErr: ErrEmptyCart,
		},
		{
			name:   "insufficient stock",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{
						ID:        10,
						UserID:    1,
						CreatedAt: createdAt,
					}, nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{
						{
							ProductID: 2,
							Quantity:  5,
						},
					}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					context.Context,
					database.Tx,
					int64,
				) (product.Product, error) {
					return product.Product{
						ID:       2,
						Price:    100,
						Currency: "EUR",
						Stock:    2,
					}, nil
				},
			},
			wantErr: ErrInsufficientStock,
		},
		{
			name:   "different currencies",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{
						ID:        10,
						UserID:    1,
						CreatedAt: createdAt,
					}, nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{
						{
							ProductID: 1,
							Quantity:  1,
						},
						{
							ProductID: 2,
							Quantity:  1,
						},
					}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					_ context.Context,
					_ database.Tx,
					id int64,
				) (product.Product, error) {
					if id == 1 {
						return product.Product{
							ID:       1,
							Price:    100,
							Currency: "EUR",
							Stock:    10,
						}, nil
					}

					return product.Product{
						ID:       2,
						Price:    200,
						Currency: "USD",
						Stock:    10,
					}, nil
				},
			},
			wantErr: ErrDifferentCurrency,
		},
		{
			name:   "successful checkout",
			userID: 1,
			repository: &fakeCartRepository{
				getDraftFn: func(
					context.Context,
					database.Tx,
					int64,
				) (Cart, error) {
					return Cart{
						ID:        10,
						UserID:    1,
						CreatedAt: createdAt,
					}, nil
				},
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return []Item{
						{
							ProductID: 2,
							Quantity:  2,
							UnitPrice: 50,
						},
						{
							ProductID: 1,
							Quantity:  3,
							UnitPrice: 100,
						},
					}, nil
				},
			},
			products: &fakeProductRepository{
				getByIDFn: func(
					_ context.Context,
					_ database.Tx,
					id int64,
				) (product.Product, error) {
					switch id {
					case 1:
						return product.Product{
							ID:       1,
							Price:    120,
							Currency: "EUR",
							Stock:    10,
						}, nil
					case 2:
						return product.Product{
							ID:       2,
							Price:    70,
							Currency: "EUR",
							Stock:    10,
						}, nil
					default:
						return product.Product{}, product.ErrProductNotFound
					}
				},
			},
			want: order.Order{
				ID:          10,
				UserID:      1,
				Status:      string(order.OrderStatusPending),
				TotalAmount: 500,
				Currency:    "EUR",
				CreatedAt:   createdAt,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := tt.repository
			if repository == nil {
				repository = &fakeCartRepository{}
			}

			products := tt.products
			if products == nil {
				products = &fakeProductRepository{}
			}

			service := newTestService(
				repository,
				products,
				&fakeTransactionManager{},
			)

			got, err := service.Checkout(
				context.Background(),
				tt.userID,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Checkout() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr == nil {
				if got != tt.want {
					t.Fatalf(
						"Checkout() = %+v, want %+v",
						got,
						tt.want,
					)
				}
			}
		})
	}
}

func TestService_Recalculate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		items        []Item
		itemsErr     error
		wantTotal    int64
		wantCurrency string
		wantErr      error
	}{
		{
			name:         "empty cart",
			items:        []Item{},
			wantTotal:    0,
			wantCurrency: "",
		},
		{
			name: "single currency",
			items: []Item{
				{
					UnitPrice: 100,
					Quantity:  2,
					Currency:  "EUR",
				},
				{
					UnitPrice: 50,
					Quantity:  3,
					Currency:  "EUR",
				},
			},
			wantTotal:    350,
			wantCurrency: "EUR",
		},
		{
			name: "different currencies",
			items: []Item{
				{
					UnitPrice: 100,
					Quantity:  1,
					Currency:  "EUR",
				},
				{
					UnitPrice: 100,
					Quantity:  1,
					Currency:  "USD",
				},
			},
			wantErr: ErrDifferentCurrency,
		},
		{
			name:     "repository error",
			itemsErr: errDatabase,
			wantErr:  errDatabase,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTotal int64
			var gotCurrency string

			repository := &fakeCartRepository{
				getItemsFn: func(
					context.Context,
					database.Tx,
					int64,
				) ([]Item, error) {
					return tt.items, tt.itemsErr
				},
				updateTotalFn: func(
					_ context.Context,
					_ database.Tx,
					_ int64,
					total int64,
					currency string,
				) error {
					gotTotal = total
					gotCurrency = currency
					return nil
				},
			}

			service := newTestService(
				repository,
				&fakeProductRepository{},
				&fakeTransactionManager{},
			)

			err := service.recalculate(
				context.Background(),
				fakeTx{},
				10,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"recalculate() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr == nil {
				if gotTotal != tt.wantTotal {
					t.Fatalf(
						"total = %d, want %d",
						gotTotal,
						tt.wantTotal,
					)
				}

				if gotCurrency != tt.wantCurrency {
					t.Fatalf(
						"currency = %q, want %q",
						gotCurrency,
						tt.wantCurrency,
					)
				}
			}
		})
	}
}

// func TestService_GetOrCreateDraft(t *testing.T) {
// 	t.Parallel()

// 	tests := []struct {
// 		name      string
// 		existing  Cart
// 		getErr    error
// 		create    Cart
// 		createErr error
// 		want      Cart
// 		wantErr   error
// 	}{
// 		{
// 			name: "returns existing draft",
// 			existing: Cart{
// 				ID:     10,
// 				UserID: 1,
// 			},
// 			want: Cart{
// 				ID:     10,
// 				UserID: 1,
// 			},
// 		},
// 		{
// 			name: "creates draft when missing",
// 			create: Cart{
// 				ID:     20,
// 				UserID: 1,
// 			},
// 			want: Cart{
// 				ID:     20,
// 				UserID: 1,
// 			},
// 		},
// 		{
// 			name:    "get draft error",
// 			getErr:  errors.New("get error"),
// 			wantErr: errors.New("get error"),
// 		},
// 		{
// 			name:      "create draft error",
// 			createErr: errors.New("create error"),
// 			wantErr:   errors.New("create error"),
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			repository := &fakeCartRepository{
// 				getDraftFn: func(
// 					context.Context,
// 					database.Tx,
// 					int64,
// 				) (Cart, error) {
// 					return tt.existing, tt.getErr
// 				},
// 				createDraftFn: func(
// 					context.Context,
// 					database.Tx,
// 					int64,
// 				) (Cart, error) {
// 					return tt.create, tt.createErr
// 				},
// 			}

// 			service := newTestService(
// 				repository,
// 				&fakeProductRepository{},
// 				&fakeTransactionManager{},
// 			)

// 			got, err := service.getOrCreateDraft(
// 				context.Background(),
// 				fakeTx{},
// 				1,
// 			)

// 			if !errors.Is(err, tt.wantErr) {
// 				t.Fatalf(
// 					"getOrCreateDraft() error = %v, want %v",
// 					err,
// 					tt.wantErr,
// 				)
// 			}

// 			if tt.wantErr == nil && got != tt.want {
// 				t.Fatalf(
// 					"getOrCreateDraft() = %+v, want %+v",
// 					got,
// 					tt.want,
// 				)
// 			}
// 		})
// 	}
// }

func TestService_LoadCart(t *testing.T) {
	t.Parallel()

	repository := &fakeCartRepository{
		getDraftByIDFn: func(
			context.Context,
			database.Tx,
			int64,
		) (Cart, error) {
			return Cart{
				ID: 10,
			}, nil
		},
		getItemsFn: func(
			context.Context,
			database.Tx,
			int64,
		) ([]Item, error) {
			return []Item{
				{
					ID:        1,
					ProductID: 2,
					Quantity:  3,
				},
			}, nil
		},
	}

	service := newTestService(
		repository,
		&fakeProductRepository{},
		&fakeTransactionManager{},
	)

	got, err := service.loadCart(
		context.Background(),
		fakeTx{},
		10,
	)

	if err != nil {
		t.Fatalf("loadCart() error = %v", err)
	}

	if len(got.Items) != 1 {
		t.Fatalf("len(Items) = %d, want 1", len(got.Items))
	}

	if got.Items[0].ProductID != 2 {
		t.Fatalf(
			"ProductID = %d, want 2",
			got.Items[0].ProductID,
		)
	}
}

func equalCart(a, b Cart) bool {
	if a.ID != b.ID ||
		a.UserID != b.UserID ||
		a.Status != b.Status ||
		a.TotalAmount != b.TotalAmount ||
		a.Currency != b.Currency {
		return false
	}

	if len(a.Items) != len(b.Items) {
		return false
	}

	for i := range a.Items {
		if a.Items[i] != b.Items[i] {
			return false
		}
	}

	return true
}
