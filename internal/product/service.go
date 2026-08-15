package product

import "errors"

type Service struct {
	products []Product
}

func NewService() *Service {
	return &Service{
		products: []Product{
			{
				ID:          1,
				Name:        "MacBook Pro",
				Description: "Apple laptop",
				Price:       150000,
				Currency:    "EUR",
				Stock:       10,
			},
			{
				ID:          2,
				Name:        "Mechanical Keyboard",
				Description: "Mechanical keyboard",
				Price:       12000,
				Currency:    "EUR",
				Stock:       50,
			},
		},
	}
}

func (s *Service) GetProducts() []Product {
	return s.products
}

func (s *Service) GetProductByID(id int64) (Product, error) {
	for _, product := range s.products {
		if product.ID == id {
			return product, nil
		}
	}

	return Product{}, errors.New("the product was not found")

}
