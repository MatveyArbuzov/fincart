package httpserver

import (
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func NewRouter(productHandler *product.Handler, orderHandler *order.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/products", productHandler.GetProducts)

	mux.HandleFunc("GET /api/v1/products/{id}", productHandler.GetProductByID)

	mux.HandleFunc("POST /api/v1/orders", orderHandler.CreateOrder)

	return mux
}
