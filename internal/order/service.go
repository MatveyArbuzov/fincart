package order

import (
	"context"
	"errors"
	"sort"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

var (
	ErrInvalidOrder        = errors.New("invalid order")
	ErrInsufficientStock   = errors.New("insufficient stock")
	ErrProductNotFound     = errors.New("product not found")
	ErrDifferentCurrencies = errors.New("products have different currencies")
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
}

func NewService(
	transactions TransactionManager,
	products ProductRepository,
	orders Repository,
) *Service {
	return &Service{
		transactions: transactions,
		products:     products,
		orders:       orders,
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
			Status:      "pending",
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
