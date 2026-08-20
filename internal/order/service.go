package order

import (
	"context"
	"errors"
	"sort"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

var (
	ErrInvalidOrder        = errors.New("invalid order")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrProductNotFound     = errors.New("product not found")
	ErrDifferentCurrencies = errors.New("products have different currencies")
	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidOrderState   = errors.New("invalid order state")
	ErrPaymentFailed       = errors.New("payment failed")
	ErrPaymentTimeout      = errors.New("payment timeout")
	ErrOrderForbidden      = errors.New("order forbidden")
)

type ProductRepository interface {
	GetByIDForUpdate(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (product.Product, error)

	DecreaseStock(
		ctx context.Context,
		tx database.Tx,
		id int64,
		quantity int,
	) error

	IncreaseStock(
		ctx context.Context,
		tx database.Tx,
		id int64,
		quantity int,
	) error
}

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error
}

type Service struct {
	transactions TransactionManager
	products     ProductRepository
	orders       Repository
	payment      payment.Service
}

type PaymentService interface {
	Pay(ctx context.Context, order Order) error
}

func NewService(
	transactions TransactionManager,
	products ProductRepository,
	orders Repository,
	paymentService payment.Service,
) *Service {
	return &Service{
		transactions: transactions,
		products:     products,
		orders:       orders,
		payment:      paymentService,
	}
}

func (s *Service) CreateOrder(
	ctx context.Context,
	userID int64,
	request CreateOrderRequest,
) (Order, error) {
	if userID <= 0 {
		return Order{}, ErrInvalidOrder
	}

	if len(request.Items) == 0 {
		return Order{}, ErrInvalidOrder
	}

	quantities := make(map[int64]int)

	for _, item := range request.Items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return Order{}, ErrInvalidOrder
		}

		quantities[item.ProductID] += item.Quantity
	}

	productIDs := make([]int64, 0, len(quantities))

	for productID := range quantities {
		productIDs = append(productIDs, productID)
	}

	sort.Slice(productIDs, func(i, j int) bool {
		return productIDs[i] < productIDs[j]
	})

	var createdOrder Order

	err := s.transactions.WithinTransaction(ctx, func(tx database.Tx) error {
		products := make([]product.Product, 0, len(productIDs))

		var totalAmount int64
		var currency string

		for _, productID := range productIDs {
			currentProduct, err := s.products.GetByIDForUpdate(
				ctx,
				tx,
				productID,
			)
			if err != nil {
				if errors.Is(err, product.ErrProductNotFound) {
					return ErrProductNotFound
				}

				return err
			}

			quantity := quantities[productID]

			if currentProduct.Stock < quantity {
				return ErrInsufficientStock
			}

			if currency == "" {
				currency = currentProduct.Currency
			}

			if currentProduct.Currency != currency {
				return ErrDifferentCurrencies
			}

			totalAmount += currentProduct.Price * int64(quantity)

			products = append(products, currentProduct)
		}

		order := Order{
			UserID:      userID,
			Status:      string(OrderStatusPending),
			TotalAmount: totalAmount,
			Currency:    currency,
		}

		var err error

		createdOrder, err = s.orders.Create(ctx, tx, order)
		if err != nil {
			return err
		}

		for _, currentProduct := range products {
			quantity := quantities[currentProduct.ID]

			_, err := s.orders.CreateItem(
				ctx,
				tx,
				OrderItem{
					OrderID:   createdOrder.ID,
					ProductID: currentProduct.ID,
					Quantity:  quantity,
					UnitPrice: currentProduct.Price,
				},
			)
			if err != nil {
				return err
			}

			if err := s.products.DecreaseStock(
				ctx,
				tx,
				currentProduct.ID,
				quantity,
			); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return Order{}, err
	}

	return createdOrder, nil
}

func (s *Service) CancelOrder(
	ctx context.Context,
	userID int64,
	orderID int64,
) error {
	if userID <= 0 || orderID <= 0 {
		return ErrInvalidOrder
	}

	return s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			currentOrder, err := s.orders.GetByIDForUpdate(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			if currentOrder.UserID != userID {
				return ErrOrderForbidden
			}

			if OrderStatus(currentOrder.Status) != OrderStatusPending {
				return ErrInvalidOrderState
			}

			items, err := s.orders.GetItems(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			for _, item := range items {
				if err := s.products.IncreaseStock(
					ctx,
					tx,
					item.ProductID,
					item.Quantity,
				); err != nil {
					return err
				}
			}

			return s.orders.UpdateStatus(
				ctx,
				tx,
				orderID,
				string(OrderStatusCancelled),
			)
		},
	)
}

func (s *Service) GetOrder(
	ctx context.Context,
	userID int64,
	orderID int64,
) (Order, []OrderItem, error) {
	if userID <= 0 || orderID <= 0 {
		return Order{}, nil, ErrInvalidOrder
	}

	var result Order
	var items []OrderItem

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			order, err := s.orders.GetByID(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			if order.UserID != userID {
				return ErrOrderForbidden
			}

			orderItems, err := s.orders.GetItems(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			result = order
			items = orderItems

			return nil
		},
	)

	if err != nil {
		return Order{}, nil, err
	}

	return result, items, nil
}

func (s *Service) ListOrders(
	ctx context.Context,
) ([]Order, error) {
	var orders []Order

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			var err error

			orders, err = s.orders.List(
				ctx,
				tx,
			)

			return err
		},
	)

	if err != nil {
		return nil, err
	}

	return orders, nil
}

func (s *Service) UpdateOrderStatus(
	ctx context.Context,
	orderID int64,
	status string,
) error {
	if orderID <= 0 {
		return ErrInvalidOrder
	}

	if !OrderStatus(status).IsValid() {
		return ErrInvalidOrderState
	}

	return s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			currentOrder, err := s.orders.GetByIDForUpdate(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			currentStatus := OrderStatus(currentOrder.Status)
			newStatus := OrderStatus(status)

			if !CanTransition(currentStatus, newStatus) {
				return ErrInvalidOrderState
			}

			return s.orders.UpdateStatus(
				ctx,
				tx,
				orderID,
				string(newStatus),
			)
		},
	)
}

func (s *Service) PayOrder(
	ctx context.Context,
	userID int64,
	orderID int64,
) error {
	if userID <= 0 || orderID <= 0 {
		return ErrInvalidOrder
	}

	return s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			currentOrder, err := s.orders.GetByIDForUpdate(
				ctx,
				tx,
				orderID,
			)
			if err != nil {
				return err
			}

			if currentOrder.UserID != userID {
				return ErrOrderForbidden
			}

			if OrderStatus(currentOrder.Status) != OrderStatusPending {
				return ErrInvalidOrderState
			}

			result, err := s.payment.Pay(
				ctx,
				currentOrder.ID,
				currentOrder.TotalAmount,
				currentOrder.Currency,
			)
			if err != nil {
				return err
			}

			switch result {
			case payment.ResultSuccess:
				return s.orders.UpdateStatus(
					ctx,
					tx,
					orderID,
					string(OrderStatusPaid),
				)

			case payment.ResultFailed:
				return ErrPaymentFailed

			case payment.ResultTimeout:
				return ErrPaymentTimeout

			default:
				return ErrPaymentFailed
			}
		},
	)
}
