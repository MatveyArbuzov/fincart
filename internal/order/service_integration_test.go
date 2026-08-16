package order

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

type mockProductRepository struct {
	increaseStockCalls []struct {
		productID int64
		quantity  int
	}

	increaseStockErr error
}

func (m *mockProductRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (product.Product, error) {
	return product.Product{}, nil
}

func (m *mockProductRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	return nil
}

func (m *mockProductRepository) IncreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	if m.increaseStockErr != nil {
		return m.increaseStockErr
	}

	m.increaseStockCalls = append(
		m.increaseStockCalls,
		struct {
			productID int64
			quantity  int
		}{
			productID: id,
			quantity:  quantity,
		},
	)

	return nil
}

type mockOrderRepository struct {
	getByIDForUpdateResult Order
	getItemsResult         []OrderItem

	updateStatusOrderID int64
	updateStatusValue   string
}

func (m *mockOrderRepository) Create(
	ctx context.Context,
	tx database.Tx,
	order Order,
) (Order, error) {
	return order, nil
}

func (m *mockOrderRepository) CreateItem(
	ctx context.Context,
	tx database.Tx,
	item OrderItem,
) (OrderItem, error) {
	return item, nil
}

func (m *mockOrderRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	return m.getByIDForUpdateResult, nil
}

func (m *mockOrderRepository) GetItems(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) ([]OrderItem, error) {
	return m.getItemsResult, nil
}

func (m *mockOrderRepository) UpdateStatus(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	status string,
) error {
	m.updateStatusOrderID = orderID
	m.updateStatusValue = status

	return nil
}

func (m *mockOrderRepository) GetByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	return Order{}, nil
}

func TestCancelOrder_Transaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)
	productRepository := &mockProductRepository{}
	orderRepository := &mockOrderRepository{}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
		payment.NewFakeService(payment.ResultSuccess),
	)

	mock.ExpectBegin()

	orderRepository.getByIDForUpdateResult = Order{
		ID:          100,
		UserID:      10,
		Status:      "pending",
		TotalAmount: 300000,
		Currency:    "EUR",
	}

	orderRepository.getItemsResult = []OrderItem{
		{
			ID:        1,
			OrderID:   100,
			ProductID: 1,
			Quantity:  2,
			UnitPrice: 150000,
		},
	}

	mock.ExpectCommit()

	err = service.CancelOrder(
		context.Background(),
		100,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(productRepository.increaseStockCalls) != 1 {
		t.Fatalf(
			"expected 1 IncreaseStock call, got %d",
			len(productRepository.increaseStockCalls),
		)
	}

	call := productRepository.increaseStockCalls[0]

	if call.productID != 1 {
		t.Fatalf(
			"expected product ID 1, got %d",
			call.productID,
		)
	}

	if call.quantity != 2 {
		t.Fatalf(
			"expected quantity 2, got %d",
			call.quantity,
		)
	}

	if orderRepository.updateStatusOrderID != 100 {
		t.Fatalf(
			"expected order ID 100, got %d",
			orderRepository.updateStatusOrderID,
		)
	}

	if orderRepository.updateStatusValue != "cancelled" {
		t.Fatalf(
			"expected status cancelled, got %s",
			orderRepository.updateStatusValue,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
func TestCancelOrder_RollbackOnIncreaseStockError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	expectedErr := errors.New("increase stock failed")

	productRepository := &mockProductRepository{
		increaseStockErr: expectedErr,
	}

	orderRepository := &mockOrderRepository{
		getByIDForUpdateResult: Order{
			ID:     100,
			UserID: 10,
			Status: "pending",
		},
		getItemsResult: []OrderItem{
			{
				ID:        1,
				OrderID:   100,
				ProductID: 1,
				Quantity:  2,
			},
		},
	}

	service := NewService(
		transactionManager,
		productRepository,
		orderRepository,
		payment.NewFakeService(payment.ResultSuccess),
	)

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = service.CancelOrder(
		context.Background(),
		100,
	)

	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"expected %v, got %v",
			expectedErr,
			err,
		)
	}

	if orderRepository.updateStatusValue != "" {
		t.Fatalf(
			"expected status not to be updated, got %s",
			orderRepository.updateStatusValue,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}
