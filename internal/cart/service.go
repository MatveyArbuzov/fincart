package cart

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

var (
	ErrInvalidCart       = errors.New("invalid cart")
	ErrCartNotFound      = errors.New("cart not found")
	ErrCartItemNotFound  = errors.New("cart item not found")
	ErrInvalidQuantity   = errors.New("invalid quantity")
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrProductNotFound   = errors.New("product not found")
	ErrDifferentCurrency = errors.New("products have different currencies")
	ErrEmptyCart         = errors.New("cart is empty")
)

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error
}

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

type Service struct {
	transactions TransactionManager
	repository   Repository
	products     ProductRepository
}

func NewService(
	transactions TransactionManager,
	repository Repository,
	products ProductRepository,
) *Service {
	return &Service{
		transactions: transactions,
		repository:   repository,
		products:     products,
	}
}

func (s *Service) GetCart(
	ctx context.Context,
	userID int64,
) (Cart, error) {
	if userID <= 0 {
		return Cart{}, ErrInvalidCart
	}

	var result Cart

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			cart, err := s.repository.GetDraft(
				ctx,
				tx,
				userID,
			)
			if err != nil {
				return err
			}

			if cart.ID == 0 {
				cart = Cart{
					UserID: userID,
					Status: string(order.OrderStatusDraft),
					Items:  make([]Item, 0),
				}
			}

			if cart.ID != 0 {
				items, err := s.repository.GetItems(
					ctx,
					tx,
					cart.ID,
				)
				if err != nil {
					return err
				}

				cart.Items = items
			}

			result = cart

			return nil
		},
	)

	if err != nil {
		return Cart{}, err
	}

	return result, nil
}

func (s *Service) AddItem(
	ctx context.Context,
	userID int64,
	request AddItemRequest,
) (Cart, error) {
	if userID <= 0 {
		return Cart{}, ErrInvalidCart
	}

	if request.ProductID <= 0 {
		return Cart{}, ErrInvalidCart
	}

	if request.Quantity <= 0 {
		return Cart{}, ErrInvalidQuantity
	}

	var result Cart

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			cart, err := s.getOrCreateDraft(
				ctx,
				tx,
				userID,
			)
			if err != nil {
				return err
			}

			currentProduct, err := s.products.GetByIDForUpdate(
				ctx,
				tx,
				request.ProductID,
			)
			if err != nil {
				if errors.Is(err, product.ErrProductNotFound) {
					return ErrProductNotFound
				}

				return err
			}

			existing, err := s.repository.GetItem(
				ctx,
				tx,
				cart.ID,
				request.ProductID,
			)

			newQuantity := request.Quantity

			if err == nil {
				newQuantity += existing.Quantity
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			if currentProduct.Stock < newQuantity {
				return ErrInsufficientStock
			}

			if err == nil {
				if err := s.repository.UpdateItem(
					ctx,
					tx,
					cart.ID,
					request.ProductID,
					newQuantity,
				); err != nil {
					return err
				}
			} else {
				if _, err := s.repository.AddItem(
					ctx,
					tx,
					cart.ID,
					request.ProductID,
					request.Quantity,
					currentProduct.Price,
				); err != nil {
					return err
				}
			}

			if err := s.recalculate(
				ctx,
				tx,
				cart.ID,
			); err != nil {
				return err
			}

			result, err = s.loadCart(
				ctx,
				tx,
				cart.ID,
			)

			return err
		},
	)

	if err != nil {
		return Cart{}, err
	}

	return result, nil
}

func (s *Service) UpdateItem(
	ctx context.Context,
	userID int64,
	productID int64,
	quantity int,
) (Cart, error) {
	if userID <= 0 || productID <= 0 {
		return Cart{}, ErrInvalidCart
	}

	if quantity <= 0 {
		return Cart{}, ErrInvalidQuantity
	}

	var result Cart

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			cart, err := s.repository.GetDraft(
				ctx,
				tx,
				userID,
			)
			if err != nil {
				return err
			}

			if cart.ID == 0 {
				return ErrCartNotFound
			}

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

			if currentProduct.Stock < quantity {
				return ErrInsufficientStock
			}

			if err := s.repository.UpdateItem(
				ctx,
				tx,
				cart.ID,
				productID,
				quantity,
			); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrCartItemNotFound
				}

				return err
			}

			if err := s.recalculate(
				ctx,
				tx,
				cart.ID,
			); err != nil {
				return err
			}

			result, err = s.loadCart(
				ctx,
				tx,
				cart.ID,
			)

			return err
		},
	)

	if err != nil {
		return Cart{}, err
	}

	return result, nil
}

func (s *Service) DeleteItem(
	ctx context.Context,
	userID int64,
	productID int64,
) (Cart, error) {
	if userID <= 0 || productID <= 0 {
		return Cart{}, ErrInvalidCart
	}

	var result Cart

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			cart, err := s.repository.GetDraft(
				ctx,
				tx,
				userID,
			)
			if err != nil {
				return err
			}

			if cart.ID == 0 {
				return ErrCartNotFound
			}

			if err := s.repository.DeleteItem(
				ctx,
				tx,
				cart.ID,
				productID,
			); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrCartItemNotFound
				}

				return err
			}

			if err := s.recalculate(
				ctx,
				tx,
				cart.ID,
			); err != nil {
				return err
			}

			result, err = s.loadCart(
				ctx,
				tx,
				cart.ID,
			)

			return err
		},
	)

	if err != nil {
		return Cart{}, err
	}

	return result, nil
}

func (s *Service) Checkout(
	ctx context.Context,
	userID int64,
) (order.Order, error) {
	if userID <= 0 {
		return order.Order{}, ErrInvalidCart
	}

	var result order.Order

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			cart, err := s.repository.GetDraft(
				ctx,
				tx,
				userID,
			)
			if err != nil {
				return err
			}

			if cart.ID == 0 {
				return ErrCartNotFound
			}

			items, err := s.repository.GetItems(
				ctx,
				tx,
				cart.ID,
			)
			if err != nil {
				return err
			}

			if len(items) == 0 {
				return ErrEmptyCart
			}

			/*
				Заново блокируем товары и проверяем stock.
				Это важно: stock мог измениться после добавления
				товара в корзину.
			*/
			productIDs := make([]int64, 0, len(items))

			quantities := make(map[int64]int)

			for _, item := range items {
				productIDs = append(productIDs, item.ProductID)
				quantities[item.ProductID] = item.Quantity
			}

			sort.Slice(
				productIDs,
				func(i, j int) bool {
					return productIDs[i] < productIDs[j]
				},
			)

			var (
				totalAmount int64
				currency    string
			)

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
					return ErrDifferentCurrency
				}

				totalAmount += currentProduct.Price * int64(quantity)
			}

			/*
				Цена фиксируется на момент checkout.
			*/
			for _, item := range items {
				currentProduct, err := s.products.GetByIDForUpdate(
					ctx,
					tx,
					item.ProductID,
				)
				if err != nil {
					return err
				}

				item.UnitPrice = currentProduct.Price

				if err := s.repository.UpdateItemPrice(
					ctx,
					tx,
					cart.ID,
					item.ProductID,
					currentProduct.Price,
				); err != nil {
					return err
				}
			}

			if err := s.repository.UpdateTotal(
				ctx,
				tx,
				cart.ID,
				totalAmount,
				currency,
			); err != nil {
				return err
			}

			for _, productID := range productIDs {
				if err := s.products.DecreaseStock(
					ctx,
					tx,
					productID,
					quantities[productID],
				); err != nil {
					return err
				}
			}

			if err := s.repository.Checkout(
				ctx,
				tx,
				cart.ID,
			); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrCartNotFound
				}

				return err
			}

			result = order.Order{
				ID:          cart.ID,
				UserID:      userID,
				Status:      string(order.OrderStatusPending),
				TotalAmount: totalAmount,
				Currency:    currency,
				CreatedAt:   cart.CreatedAt,
			}

			return nil
		},
	)

	if err != nil {
		return order.Order{}, err
	}

	return result, nil
}

func (s *Service) getOrCreateDraft(
	ctx context.Context,
	tx database.Tx,
	userID int64,
) (Cart, error) {
	cart, err := s.repository.GetDraft(
		ctx,
		tx,
		userID,
	)
	if err != nil {
		return Cart{}, err
	}

	if cart.ID != 0 {
		return cart, nil
	}

	return s.repository.CreateDraft(
		ctx,
		tx,
		userID,
	)
}

func (s *Service) loadCart(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) (Cart, error) {
	cart, err := s.repository.GetDraftByID(
		ctx,
		tx,
		orderID,
	)
	if err != nil {
		return Cart{}, err
	}

	items, err := s.repository.GetItems(
		ctx,
		tx,
		orderID,
	)
	if err != nil {
		return Cart{}, err
	}

	cart.Items = items

	return cart, nil
}

func (s *Service) recalculate(
	ctx context.Context,
	tx database.Tx,
	orderID int64,
) error {
	items, err := s.repository.GetItems(
		ctx,
		tx,
		orderID,
	)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return s.repository.UpdateTotal(
			ctx,
			tx,
			orderID,
			0,
			"",
		)
	}

	var (
		total    int64
		currency string
	)

	for _, item := range items {
		if currency == "" {
			currency = item.Currency
		}

		if item.Currency != currency {
			return ErrDifferentCurrency
		}

		total += item.UnitPrice * int64(item.Quantity)
	}

	return s.repository.UpdateTotal(
		ctx,
		tx,
		orderID,
		total,
		currency,
	)
}
