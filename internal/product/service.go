package product

import (
	"context"
	"errors"
	"strings"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

var (
	ErrInvalidProduct = errors.New("invalid product")
)

type Service struct {
	transactions TransactionManager
	repository   Repository
	transaction  TransactionRepository
}

type TransactionManager interface {
	WithinTransaction(
		ctx context.Context,
		fn func(tx database.Tx) error,
	) error
}

func NewService(
	transactions TransactionManager,
	repository Repository,
	transaction TransactionRepository,
) *Service {
	return &Service{
		transactions: transactions,
		repository:   repository,
		transaction:  transaction,
	}
}

func (s *Service) GetProductByID(
	ctx context.Context,
	id int64,
) (Product, error) {
	if id <= 0 {
		return Product{}, ErrInvalidProduct
	}

	return s.repository.GetByID(ctx, id)
}

func (s *Service) GetProducts(
	ctx context.Context,
) ([]Product, error) {
	return s.repository.List(ctx)
}

func (s *Service) CreateProduct(
	ctx context.Context,
	request CreateProductRequest,
) (Product, error) {
	product := Product{
		Name:        strings.TrimSpace(request.Name),
		Description: strings.TrimSpace(request.Description),
		Price:       request.Price,
		Currency:    strings.ToUpper(strings.TrimSpace(request.Currency)),
		Stock:       request.Stock,
	}

	if err := validateProduct(product); err != nil {
		return Product{}, err
	}

	var created Product

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			var err error

			created, err = s.transaction.Create(
				ctx,
				tx,
				product,
			)

			return err
		},
	)
	if err != nil {
		return Product{}, err
	}

	return created, nil
}

func (s *Service) UpdateProduct(
	ctx context.Context,
	id int64,
	request UpdateProductRequest,
) (Product, error) {
	if id <= 0 {
		return Product{}, ErrInvalidProduct
	}

	product := Product{
		ID:          id,
		Name:        strings.TrimSpace(request.Name),
		Description: strings.TrimSpace(request.Description),
		Price:       request.Price,
		Currency:    strings.ToUpper(strings.TrimSpace(request.Currency)),
		Stock:       request.Stock,
	}

	if err := validateProduct(product); err != nil {
		return Product{}, err
	}

	var updated Product

	err := s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			current, err := s.transaction.GetByIDForUpdate(
				ctx,
				tx,
				id,
			)
			if err != nil {
				return err
			}

			product.ID = current.ID

			if err := s.transaction.Update(
				ctx,
				tx,
				product,
			); err != nil {
				return err
			}

			updated = product

			return nil
		},
	)
	if err != nil {
		return Product{}, err
	}

	return updated, nil
}

func (s *Service) DeleteProduct(
	ctx context.Context,
	id int64,
) error {
	if id <= 0 {
		return ErrInvalidProduct
	}

	return s.transactions.WithinTransaction(
		ctx,
		func(tx database.Tx) error {
			return s.transaction.Delete(
				ctx,
				tx,
				id,
			)
		},
	)
}

func validateProduct(product Product) error {
	if product.Name == "" {
		return ErrInvalidProduct
	}

	if len(product.Name) > 255 {
		return ErrInvalidProduct
	}

	if product.Price < 0 {
		return ErrInvalidProduct
	}

	if product.Stock < 0 {
		return ErrInvalidProduct
	}

	if product.Currency == "" {
		return ErrInvalidProduct
	}

	if len(product.Currency) != 3 {
		return ErrInvalidProduct
	}

	return nil
}
