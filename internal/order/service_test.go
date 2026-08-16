package order

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func TestCreateOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			name,
			description,
			price,
			currency,
			stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				10,
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`)).
		WithArgs(
			int64(10),
			"pending",
			int64(300000),
			"EUR",
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				int64(100),
				time.Now(),
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`)).
		WithArgs(
			int64(100),
			int64(1),
			2,
			int64(150000),
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
			}).AddRow(
				int64(1),
			),
		)

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE products
		SET stock = stock - $1
		WHERE id = $2
	`)).
		WithArgs(
			2,
			int64(1),
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	mock.ExpectCommit()

	result, err := service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != 100 {
		t.Fatalf(
			"expected order ID 100, got %d",
			result.ID,
		)
	}

	if result.UserID != 10 {
		t.Fatalf(
			"expected user ID 10, got %d",
			result.UserID,
		)
	}

	if result.Status != "pending" {
		t.Fatalf(
			"expected status pending, got %s",
			result.Status,
		)
	}

	if result.TotalAmount != 300000 {
		t.Fatalf(
			"expected total amount 300000, got %d",
			result.TotalAmount,
		)
	}

	if result.Currency != "EUR" {
		t.Fatalf(
			"expected currency EUR, got %s",
			result.Currency,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			name,
			description,
			price,
			currency,
			stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				1,
			),
		)

	mock.ExpectRollback()

	_, err = service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		},
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf(
			"expected ErrInsufficientStock, got %v",
			err,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}

func TestCreateOrder_OrderItemCreationFailed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	productRepository := product.NewPostgresTransactionRepository()
	orderRepository := NewPostgresRepository()

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	mock.ExpectBegin()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			id,
			name,
			description,
			price,
			currency,
			stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`)).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"name",
				"description",
				"price",
				"currency",
				"stock",
			}).AddRow(
				int64(1),
				"MacBook",
				"Laptop",
				int64(150000),
				"EUR",
				10,
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`)).
		WithArgs(
			int64(10),
			"pending",
			int64(300000),
			"EUR",
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				int64(100),
				time.Now(),
			),
		)

	mock.ExpectQuery(regexp.QuoteMeta(`
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`)).
		WithArgs(
			int64(100),
			int64(1),
			2,
			int64(150000),
		).
		WillReturnError(
			errors.New("failed to create order item"),
		)

	mock.ExpectRollback()

	// Act
	_, err = service.CreateOrder(
		context.Background(),
		10,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		},
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create order item") {
		t.Fatalf(
			"expected order item error, got %v",
			err,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"there were unmet SQL expectations: %v",
			err,
		)
	}
}

func TestService_CreateOrder_DifferentCurrencies(t *testing.T) {
	productRepository := &fakeMultiProductRepository{
		products: map[int64]product.Product{
			1: {
				ID:       1,
				Name:     "MacBook",
				Price:    150000,
				Currency: "EUR",
				Stock:    10,
			},
			2: {
				ID:       2,
				Name:     "iPhone",
				Price:    100000,
				Currency: "USD",
				Stock:    10,
			},
		},
	}

	orderRepository := &fakeOrderRepository{}
	transactionManager := &fakeTransactionManager{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
	)

	request := CreateOrderRequest{
		Items: []CreateOrderItem{
			{
				ProductID: 1,
				Quantity:  1,
			},
			{
				ProductID: 2,
				Quantity:  1,
			},
		},
	}

	_, err := service.CreateOrder(
		context.Background(),
		10,
		request,
	)

	if !errors.Is(err, ErrDifferentCurrencies) {
		t.Fatalf(
			"expected ErrDifferentCurrencies, got %v",
			err,
		)
	}
}

type cancelTestProductRepository struct {
	increaseStockErr error

	increasedProducts map[int64]int
}

func (f *cancelTestProductRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (product.Product, error) {
	return product.Product{}, nil
}

func (f *cancelTestProductRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	return nil
}

func (f *cancelTestProductRepository) IncreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	if f.increaseStockErr != nil {
		return f.increaseStockErr
	}

	if f.increasedProducts == nil {
		f.increasedProducts = make(map[int64]int)
	}

	f.increasedProducts[id] += quantity

	return nil
}

type cancelTestOrderRepository struct {
	order       Order
	items       []OrderItem
	getOrderErr error
	getItemsErr error
	updateErr   error

	updatedOrderID int64
	updatedStatus  string
}

func (f *cancelTestOrderRepository) GetByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	if f.getOrderErr != nil {
		return Order{}, f.getOrderErr
	}

	if f.order.ID != id {
		return Order{}, ErrOrderNotFound
	}

	return f.order, nil
}

func (f *cancelTestOrderRepository) Create(
	ctx context.Context,
	tx database.Tx,
	order Order,
) (Order, error) {
	return order, nil
}

func (f *cancelTestOrderRepository) CreateItem(
	ctx context.Context,
	tx database.Tx,
	item OrderItem,
) (OrderItem, error) {
	return item, nil
}

func (f *cancelTestOrderRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	if f.getOrderErr != nil {
		return Order{}, f.getOrderErr
	}

	return f.order, nil
}

func (f *cancelTestOrderRepository) GetItems(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) ([]OrderItem, error) {
	if f.getItemsErr != nil {
		return nil, f.getItemsErr
	}

	return f.items, nil
}

func (f *cancelTestOrderRepository) UpdateStatus(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	status string,
) error {
	if f.updateErr != nil {
		return f.updateErr
	}

	f.updatedOrderID = orderID
	f.updatedStatus = status

	return nil
}

func TestCancelOrder_Success(t *testing.T) {
	productRepository := &cancelTestProductRepository{}

	orderRepository := &cancelTestOrderRepository{
		order: Order{
			ID:     100,
			UserID: 10,
			Status: "pending",
		},
		items: []OrderItem{
			{
				ID:        1,
				OrderID:   100,
				ProductID: 1,
				Quantity:  2,
				UnitPrice: 150000,
			},
			{
				ID:        2,
				OrderID:   100,
				ProductID: 2,
				Quantity:  3,
				UnitPrice: 12000,
			},
		},
	}

	service := NewService(
		&fakeTransactionManager{},
		productRepository,
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if productRepository.increasedProducts[1] != 2 {
		t.Fatalf(
			"expected product 1 stock increase by 2, got %d",
			productRepository.increasedProducts[1],
		)
	}

	if productRepository.increasedProducts[2] != 3 {
		t.Fatalf(
			"expected product 2 stock increase by 3, got %d",
			productRepository.increasedProducts[2],
		)
	}

	if orderRepository.updatedOrderID != 100 {
		t.Fatalf(
			"expected updated order ID 100, got %d",
			orderRepository.updatedOrderID,
		)
	}

	if orderRepository.updatedStatus != "cancelled" {
		t.Fatalf(
			"expected status cancelled, got %s",
			orderRepository.updatedStatus,
		)
	}
}

func TestCancelOrder_InvalidOrderID(t *testing.T) {
	service := NewService(
		&fakeTransactionManager{},
		&cancelTestProductRepository{},
		&cancelTestOrderRepository{},
	)

	err := service.CancelOrder(
		context.Background(),
		0,
	)

	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf(
			"expected ErrInvalidOrder, got %v",
			err,
		)
	}
}

func TestCancelOrder_NegativeOrderID(t *testing.T) {
	service := NewService(
		&fakeTransactionManager{},
		&cancelTestProductRepository{},
		&cancelTestOrderRepository{},
	)

	err := service.CancelOrder(
		context.Background(),
		-1,
	)

	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf(
			"expected ErrInvalidOrder, got %v",
			err,
		)
	}
}

func TestCancelOrder_OrderNotFound(t *testing.T) {
	expectedErr := ErrOrderNotFound

	orderRepository := &cancelTestOrderRepository{
		getOrderErr: expectedErr,
	}

	service := NewService(
		&fakeTransactionManager{},
		&cancelTestProductRepository{},
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected ErrOrderNotFound, got %v",
			err,
		)
	}
}

func TestCancelOrder_GetOrderError(t *testing.T) {
	expectedErr := errors.New("database error")

	orderRepository := &cancelTestOrderRepository{
		getOrderErr: expectedErr,
	}

	service := NewService(
		&fakeTransactionManager{},
		&cancelTestProductRepository{},
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected database error, got %v",
			err,
		)
	}
}

func TestCancelOrder_InvalidOrderState(t *testing.T) {
	orderRepository := &cancelTestOrderRepository{
		order: Order{
			ID:     100,
			UserID: 10,
			Status: "completed",
		},
	}

	productRepository := &cancelTestProductRepository{}

	service := NewService(
		&fakeTransactionManager{},
		productRepository,
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, ErrInvalidOrderState) {
		t.Fatalf(
			"expected ErrInvalidOrderState, got %v",
			err,
		)
	}

	if len(productRepository.increasedProducts) != 0 {
		t.Fatal("stock must not be increased for invalid order state")
	}

	if orderRepository.updatedOrderID != 0 {
		t.Fatal("order status must not be updated")
	}
}

func TestCancelOrder_GetItemsError(t *testing.T) {
	expectedErr := errors.New("failed to get order items")

	orderRepository := &cancelTestOrderRepository{
		order: Order{
			ID:     100,
			UserID: 10,
			Status: "pending",
		},
		getItemsErr: expectedErr,
	}

	productRepository := &cancelTestProductRepository{}

	service := NewService(
		&fakeTransactionManager{},
		productRepository,
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected items error, got %v",
			err,
		)
	}

	if len(productRepository.increasedProducts) != 0 {
		t.Fatal("stock must not be increased when getting items fails")
	}
}

func TestCancelOrder_IncreaseStockError(t *testing.T) {
	expectedErr := errors.New("failed to increase stock")

	productRepository := &cancelTestProductRepository{
		increaseStockErr: expectedErr,
	}

	orderRepository := &cancelTestOrderRepository{
		order: Order{
			ID:     100,
			UserID: 10,
			Status: "pending",
		},
		items: []OrderItem{
			{
				ID:        1,
				OrderID:   100,
				ProductID: 1,
				Quantity:  2,
				UnitPrice: 150000,
			},
		},
	}

	service := NewService(
		&fakeTransactionManager{},
		productRepository,
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected increase stock error, got %v",
			err,
		)
	}

	if orderRepository.updatedOrderID != 0 {
		t.Fatal("order status must not be updated when stock restoration fails")
	}
}

func TestCancelOrder_UpdateStatusError(t *testing.T) {
	expectedErr := errors.New("failed to update order status")

	productRepository := &cancelTestProductRepository{}

	orderRepository := &cancelTestOrderRepository{
		order: Order{
			ID:     100,
			UserID: 10,
			Status: "pending",
		},
		items: []OrderItem{
			{
				ID:        1,
				OrderID:   100,
				ProductID: 1,
				Quantity:  2,
				UnitPrice: 150000,
			},
		},
		updateErr: expectedErr,
	}

	service := NewService(
		&fakeTransactionManager{},
		productRepository,
		orderRepository,
	)

	err := service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected update status error, got %v",
			err,
		)
	}

	if productRepository.increasedProducts[1] != 2 {
		t.Fatalf(
			"expected product stock increase by 2, got %d",
			productRepository.increasedProducts[1],
		)
	}
}
