package order

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

type fakeOrderTx struct{}

func (fakeOrderTx) Commit() error   { return nil }
func (fakeOrderTx) Rollback() error { return nil }

func (fakeOrderTx) ExecContext(
	context.Context,
	string,
	...any,
) (sql.Result, error) {
	return nil, nil
}

func (fakeOrderTx) QueryContext(
	context.Context,
	string,
	...any,
) (*sql.Rows, error) {
	return nil, nil
}

func (fakeOrderTx) QueryRowContext(
	context.Context,
	string,
	...any,
) *sql.Row {
	return nil
}

type fakeOrderTransactionManager struct {
	err error

	calls int
}

func (f *fakeOrderTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	f.calls++

	if f.err != nil {
		return f.err
	}

	return fn(fakeOrderTx{})
}

type fakeOrderProductRepository struct {
	products map[int64]product.Product

	getErr map[int64]error

	decreased []struct {
		id       int64
		quantity int
	}

	increased []struct {
		id       int64
		quantity int
	}
}

func (f *fakeOrderProductRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (product.Product, error) {
	if err := f.getErr[id]; err != nil {
		return product.Product{}, err
	}

	p, ok := f.products[id]
	if !ok {
		return product.Product{}, product.ErrProductNotFound
	}

	return p, nil
}

func (f *fakeOrderProductRepository) DecreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	f.decreased = append(
		f.decreased,
		struct {
			id       int64
			quantity int
		}{id, quantity},
	)

	return nil
}

func (f *fakeOrderProductRepository) IncreaseStock(
	ctx context.Context,
	tx database.Tx,
	id int64,
	quantity int,
) error {
	f.increased = append(
		f.increased,
		struct {
			id       int64
			quantity int
		}{id, quantity},
	)

	return nil
}

type fakeOrderRepository struct {
	order  Order
	orders []Order
	items  []OrderItem

	getByIDErr          error
	getByIDForUpdateErr error
	getItemsErr         error
	createErr           error
	createItemErr       error
	updateStatusErr     error
	listErr             error

	created      Order
	createdItems []OrderItem

	updatedOrderID int64
	updatedStatus  string
}

func (f *fakeOrderRepository) Create(
	ctx context.Context,
	tx database.Tx,
	order Order,
) (Order, error) {
	if f.createErr != nil {
		return Order{}, f.createErr
	}

	if order.ID == 0 {
		order.ID = 100
	}

	f.created = order
	return order, nil
}

func (f *fakeOrderRepository) CreateItem(
	ctx context.Context,
	tx database.Tx,
	item OrderItem,
) (OrderItem, error) {
	if f.createItemErr != nil {
		return OrderItem{}, f.createItemErr
	}

	if item.ID == 0 {
		item.ID = int64(len(f.createdItems) + 1)
	}

	f.createdItems = append(f.createdItems, item)

	return item, nil
}

func (f *fakeOrderRepository) GetByIDForUpdate(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	if f.getByIDForUpdateErr != nil {
		return Order{}, f.getByIDForUpdateErr
	}

	return f.order, nil
}

func (f *fakeOrderRepository) GetItems(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) ([]OrderItem, error) {
	if f.getItemsErr != nil {
		return nil, f.getItemsErr
	}

	return f.items, nil
}

func (f *fakeOrderRepository) UpdateStatus(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
	status string,
) error {
	if f.updateStatusErr != nil {
		return f.updateStatusErr
	}

	f.updatedOrderID = orderID
	f.updatedStatus = status

	return nil
}

func (f *fakeOrderRepository) GetByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (Order, error) {
	if f.getByIDErr != nil {
		return Order{}, f.getByIDErr
	}

	return f.order, nil
}

func (f *fakeOrderRepository) List(
	ctx context.Context,
	tx database.Tx,
) ([]Order, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	return f.orders, nil
}

type fakePaymentService struct {
	result payment.Result
	err    error

	orderID  int64
	amount   int64
	currency string
}

func (f *fakePaymentService) Pay(
	ctx context.Context,
	orderID int64,
	amount int64,
	currency string,
) (payment.Result, error) {
	f.orderID = orderID
	f.amount = amount
	f.currency = currency

	return f.result, f.err
}

func newOrderService(
	transactions TransactionManager,
	products ProductRepository,
	orders Repository,
	paymentService payment.Service,
) *Service {
	return NewService(
		transactions,
		products,
		orders,
		paymentService,
	)
}

func TestService_CreateOrder_InvalidUser(t *testing.T) {
	t.Parallel()

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	_, err := service.CreateOrder(
		context.Background(),
		0,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		},
	)

	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("error = %v, want ErrInvalidOrder", err)
	}
}

func TestService_CreateOrder_EmptyItems(t *testing.T) {
	t.Parallel()

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	_, err := service.CreateOrder(
		context.Background(),
		1,
		CreateOrderRequest{},
	)

	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("error = %v, want ErrInvalidOrder", err)
	}
}

func TestService_CreateOrder_InvalidItem(t *testing.T) {
	t.Parallel()

	tests := []CreateOrderItem{
		{
			ProductID: 0,
			Quantity:  1,
		},
		{
			ProductID: 1,
			Quantity:  0,
		},
		{
			ProductID: -1,
			Quantity:  1,
		},
		{
			ProductID: 1,
			Quantity:  -1,
		},
	}

	for _, item := range tests {
		item := item

		t.Run(
			"invalid",
			func(t *testing.T) {
				t.Parallel()

				service := newOrderService(
					&fakeOrderTransactionManager{},
					&fakeOrderProductRepository{},
					&fakeOrderRepository{},
					&fakePaymentService{},
				)

				_, err := service.CreateOrder(
					context.Background(),
					1,
					CreateOrderRequest{
						Items: []CreateOrderItem{item},
					},
				)

				if !errors.Is(err, ErrInvalidOrder) {
					t.Fatalf(
						"error = %v, want ErrInvalidOrder",
						err,
					)
				}
			},
		)
	}
}

func TestService_CreateOrder_Success(t *testing.T) {
	t.Parallel()

	products := &fakeOrderProductRepository{
		products: map[int64]product.Product{
			1: {
				ID:       1,
				Name:     "Phone",
				Price:    100,
				Currency: "USD",
				Stock:    10,
			},
			2: {
				ID:       2,
				Name:     "Case",
				Price:    50,
				Currency: "USD",
				Stock:    20,
			},
		},
	}

	orders := &fakeOrderRepository{}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		products,
		orders,
		&fakePaymentService{},
	)

	got, err := service.CreateOrder(
		context.Background(),
		7,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 2,
					Quantity:  2,
				},
				{
					ProductID: 1,
					Quantity:  3,
				},
			},
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.ID != 100 {
		t.Fatalf("ID = %d, want 100", got.ID)
	}

	if got.UserID != 7 {
		t.Fatalf("UserID = %d, want 7", got.UserID)
	}

	if got.Status != string(OrderStatusPending) {
		t.Fatalf(
			"Status = %q, want %q",
			got.Status,
			OrderStatusPending,
		)
	}

	// 3 * 100 + 2 * 50
	if got.TotalAmount != 400 {
		t.Fatalf(
			"TotalAmount = %d, want 400",
			got.TotalAmount,
		)
	}

	if got.Currency != "USD" {
		t.Fatalf(
			"Currency = %q, want USD",
			got.Currency,
		)
	}

	if len(orders.createdItems) != 2 {
		t.Fatalf(
			"created items = %d, want 2",
			len(orders.createdItems),
		)
	}

	if len(products.decreased) != 2 {
		t.Fatalf(
			"decreased calls = %d, want 2",
			len(products.decreased),
		)
	}
}

func TestService_CreateOrder_MergesDuplicateProducts(t *testing.T) {
	t.Parallel()

	products := &fakeOrderProductRepository{
		products: map[int64]product.Product{
			1: {
				ID:       1,
				Price:    100,
				Currency: "USD",
				Stock:    10,
			},
		},
	}

	orders := &fakeOrderRepository{}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		products,
		orders,
		&fakePaymentService{},
	)

	_, err := service.CreateOrder(
		context.Background(),
		1,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 1, Quantity: 3},
			},
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orders.createdItems) != 1 {
		t.Fatalf(
			"created items = %d, want 1",
			len(orders.createdItems),
		)
	}

	if orders.createdItems[0].Quantity != 5 {
		t.Fatalf(
			"quantity = %d, want 5",
			orders.createdItems[0].Quantity,
		)
	}
}

func TestService_CreateOrder_InsufficientStock(t *testing.T) {
	t.Parallel()

	products := &fakeOrderProductRepository{
		products: map[int64]product.Product{
			1: {
				ID:       1,
				Price:    100,
				Currency: "USD",
				Stock:    2,
			},
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		products,
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	_, err := service.CreateOrder(
		context.Background(),
		1,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 1,
					Quantity:  3,
				},
			},
		},
	)

	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf(
			"error = %v, want ErrInsufficientStock",
			err,
		)
	}
}

func TestService_CreateOrder_ProductNotFound(t *testing.T) {
	t.Parallel()

	products := &fakeOrderProductRepository{
		products: map[int64]product.Product{},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		products,
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	_, err := service.CreateOrder(
		context.Background(),
		1,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{
					ProductID: 999,
					Quantity:  1,
				},
			},
		},
	)

	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf(
			"error = %v, want ErrProductNotFound",
			err,
		)
	}
}

func TestService_CreateOrder_DifferentCurrencies(t *testing.T) {
	t.Parallel()

	products := &fakeOrderProductRepository{
		products: map[int64]product.Product{
			1: {
				ID:       1,
				Price:    100,
				Currency: "USD",
				Stock:    10,
			},
			2: {
				ID:       2,
				Price:    100,
				Currency: "EUR",
				Stock:    10,
			},
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		products,
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	_, err := service.CreateOrder(
		context.Background(),
		1,
		CreateOrderRequest{
			Items: []CreateOrderItem{
				{ProductID: 1, Quantity: 1},
				{ProductID: 2, Quantity: 1},
			},
		},
	)

	if !errors.Is(err, ErrDifferentCurrencies) {
		t.Fatalf(
			"error = %v, want ErrDifferentCurrencies",
			err,
		)
	}
}

func TestService_CancelOrder_Success(t *testing.T) {
	t.Parallel()

	products := &fakeOrderProductRepository{
		products: map[int64]product.Product{
			1: {ID: 1},
			2: {ID: 2},
		},
	}

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 42,
			Status: string(OrderStatusPending),
		},
		items: []OrderItem{
			{
				OrderID:   10,
				ProductID: 1,
				Quantity:  2,
			},
			{
				OrderID:   10,
				ProductID: 2,
				Quantity:  3,
			},
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		products,
		orders,
		&fakePaymentService{},
	)

	err := service.CancelOrder(
		context.Background(),
		42,
		10,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products.increased) != 2 {
		t.Fatalf(
			"increase calls = %d, want 2",
			len(products.increased),
		)
	}

	if orders.updatedOrderID != 10 {
		t.Fatalf(
			"updated order ID = %d, want 10",
			orders.updatedOrderID,
		)
	}

	if orders.updatedStatus != string(OrderStatusCancelled) {
		t.Fatalf(
			"status = %q, want cancelled",
			orders.updatedStatus,
		)
	}
}

func TestService_CancelOrder_InvalidArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		userID  int64
		orderID int64
	}{
		{
			name:    "invalid user",
			userID:  0,
			orderID: 1,
		},
		{
			name:    "invalid order",
			userID:  1,
			orderID: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service := newOrderService(
				&fakeOrderTransactionManager{},
				&fakeOrderProductRepository{},
				&fakeOrderRepository{},
				&fakePaymentService{},
			)

			err := service.CancelOrder(
				context.Background(),
				tt.userID,
				tt.orderID,
			)

			if !errors.Is(err, ErrInvalidOrder) {
				t.Fatalf(
					"error = %v, want ErrInvalidOrder",
					err,
				)
			}
		})
	}
}

func TestService_CancelOrder_Forbidden(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 100,
			Status: string(OrderStatusPending),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	err := service.CancelOrder(
		context.Background(),
		200,
		10,
	)

	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf(
			"error = %v, want ErrOrderForbidden",
			err,
		)
	}
}

func TestService_CancelOrder_InvalidState(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 100,
			Status: string(OrderStatusPaid),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	err := service.CancelOrder(
		context.Background(),
		100,
		10,
	)

	if !errors.Is(err, ErrInvalidOrderState) {
		t.Fatalf(
			"error = %v, want ErrInvalidOrderState",
			err,
		)
	}
}

func TestService_GetOrder_Success(t *testing.T) {
	t.Parallel()

	wantOrder := Order{
		ID:       10,
		UserID:   42,
		Status:   string(OrderStatusPending),
		Currency: "USD",
	}

	wantItems := []OrderItem{
		{
			ID:        1,
			OrderID:   10,
			ProductID: 5,
			Quantity:  2,
			UnitPrice: 100,
		},
	}

	orders := &fakeOrderRepository{
		order: wantOrder,
		items: wantItems,
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	gotOrder, gotItems, err := service.GetOrder(
		context.Background(),
		42,
		10,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotOrder != wantOrder {
		t.Fatalf("order = %+v, want %+v", gotOrder, wantOrder)
	}

	if len(gotItems) != 1 {
		t.Fatalf("items = %d, want 1", len(gotItems))
	}
}

func TestService_GetOrder_Forbidden(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 42,
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	_, _, err := service.GetOrder(
		context.Background(),
		99,
		10,
	)

	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf(
			"error = %v, want ErrOrderForbidden",
			err,
		)
	}
}

func TestService_GetOrder_InvalidArguments(t *testing.T) {
	t.Parallel()

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	_, _, err := service.GetOrder(
		context.Background(),
		0,
		1,
	)

	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf(
			"error = %v, want ErrInvalidOrder",
			err,
		)
	}
}

func TestService_ListOrders(t *testing.T) {
	t.Parallel()

	want := []Order{
		{ID: 3},
		{ID: 2},
		{ID: 1},
	}

	orders := &fakeOrderRepository{
		orders: want,
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	got, err := service.ListOrders(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf(
			"len = %d, want %d",
			len(got),
			len(want),
		)
	}
}

func TestService_UpdateOrderStatus(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			Status: string(OrderStatusPending),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	err := service.UpdateOrderStatus(
		context.Background(),
		10,
		string(OrderStatusPaid),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if orders.updatedOrderID != 10 {
		t.Fatalf(
			"order ID = %d, want 10",
			orders.updatedOrderID,
		)
	}

	if orders.updatedStatus != string(OrderStatusPaid) {
		t.Fatalf(
			"status = %q, want paid",
			orders.updatedStatus,
		)
	}
}

func TestService_UpdateOrderStatus_InvalidStatus(t *testing.T) {
	t.Parallel()

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		&fakeOrderRepository{},
		&fakePaymentService{},
	)

	err := service.UpdateOrderStatus(
		context.Background(),
		10,
		"invalid",
	)

	if !errors.Is(err, ErrInvalidOrderState) {
		t.Fatalf(
			"error = %v, want ErrInvalidOrderState",
			err,
		)
	}
}

func TestService_UpdateOrderStatus_InvalidTransition(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			Status: string(OrderStatusPending),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	err := service.UpdateOrderStatus(
		context.Background(),
		10,
		string(OrderStatusProcessing),
	)

	if !errors.Is(err, ErrInvalidOrderState) {
		t.Fatalf(
			"error = %v, want ErrInvalidOrderState",
			err,
		)
	}
}

func TestService_PayOrder_Success(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:          10,
			UserID:      42,
			Status:      string(OrderStatusPending),
			TotalAmount: 1000,
			Currency:    "USD",
		},
	}

	paymentService := &fakePaymentService{
		result: payment.ResultSuccess,
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		paymentService,
	)

	err := service.PayOrder(
		context.Background(),
		42,
		10,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if paymentService.orderID != 10 {
		t.Fatalf(
			"payment order ID = %d, want 10",
			paymentService.orderID,
		)
	}

	if paymentService.amount != 1000 {
		t.Fatalf(
			"payment amount = %d, want 1000",
			paymentService.amount,
		)
	}

	if paymentService.currency != "USD" {
		t.Fatalf(
			"payment currency = %q, want USD",
			paymentService.currency,
		)
	}

	if orders.updatedStatus != string(OrderStatusPaid) {
		t.Fatalf(
			"status = %q, want paid",
			orders.updatedStatus,
		)
	}
}

func TestService_PayOrder_Results(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		result  payment.Result
		wantErr error
	}{
		{
			name:    "failed",
			result:  payment.ResultFailed,
			wantErr: ErrPaymentFailed,
		},
		{
			name:    "timeout",
			result:  payment.ResultTimeout,
			wantErr: ErrPaymentTimeout,
		},
		{
			name:    "unknown",
			result:  payment.Result("unknown"),
			wantErr: ErrPaymentFailed,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orders := &fakeOrderRepository{
				order: Order{
					ID:          10,
					UserID:      42,
					Status:      string(OrderStatusPending),
					TotalAmount: 100,
					Currency:    "USD",
				},
			}

			service := newOrderService(
				&fakeOrderTransactionManager{},
				&fakeOrderProductRepository{},
				orders,
				&fakePaymentService{
					result: tt.result,
				},
			)

			err := service.PayOrder(
				context.Background(),
				42,
				10,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"error = %v, want %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}

func TestService_PayOrder_PaymentError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("payment service unavailable")

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 42,
			Status: string(OrderStatusPending),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{
			err: wantErr,
		},
	)

	err := service.PayOrder(
		context.Background(),
		42,
		10,
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf(
			"error = %v, want %v",
			err,
			wantErr,
		)
	}
}

func TestService_PayOrder_Forbidden(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 42,
			Status: string(OrderStatusPending),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{
			result: payment.ResultSuccess,
		},
	)

	err := service.PayOrder(
		context.Background(),
		99,
		10,
	)

	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf(
			"error = %v, want ErrOrderForbidden",
			err,
		)
	}
}

func TestService_PayOrder_InvalidState(t *testing.T) {
	t.Parallel()

	orders := &fakeOrderRepository{
		order: Order{
			ID:     10,
			UserID: 42,
			Status: string(OrderStatusPaid),
		},
	}

	service := newOrderService(
		&fakeOrderTransactionManager{},
		&fakeOrderProductRepository{},
		orders,
		&fakePaymentService{},
	)

	err := service.PayOrder(
		context.Background(),
		42,
		10,
	)

	if !errors.Is(err, ErrInvalidOrderState) {
		t.Fatalf(
			"error = %v, want ErrInvalidOrderState",
			err,
		)
	}
}
