package main

import (
	"log"
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/httpserver"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func main() {
	productRepository := product.NewMemoryRepository()
	productService := product.NewService(productRepository)
	productHandler := product.NewHandler(productService)

	router := httpserver.NewRouter(productHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("server started on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
