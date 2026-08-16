package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/httpserver"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")

	db, err := database.NewPostgres(ctx, dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Product
	productRepository := product.NewPostgresRepository(db)
	productService := product.NewService(productRepository)
	productHandler := product.NewHandler(productService)

	// Order
	transactionManager := database.NewManager(db)
	paymentService := payment.NewFakeService(payment.ResultSuccess)

	productTransactionRepository :=
		product.NewPostgresTransactionRepository()

	orderRepository :=
		order.NewPostgresRepository()

	orderService := order.NewService(
		transactionManager,
		productTransactionRepository,
		orderRepository,
		paymentService,
	)

	orderHandler := order.NewHandler(orderService)

	// HTTP
	router := httpserver.NewRouter(
		productHandler,
		orderHandler,
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	log.Println("server started on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
