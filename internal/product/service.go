package product

import "context"

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetProductByID(ctx context.Context, id int64) (Product, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) GetProducts(ctx context.Context) ([]Product, error) {
	return s.repository.List(ctx)
}
