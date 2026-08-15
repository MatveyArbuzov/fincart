package httpserver

import (
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/product"
)

func NewRouter(productHandler *product.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/products", productHandler.GetProducts)
	mux.HandleFunc("GET /api/v1/products/{id}", productHandler.GetProductByID)

	return mux
}
