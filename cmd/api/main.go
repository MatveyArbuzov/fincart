package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/database"
	"github.com/MatveyArbuzov/fincart/internal/httpserver"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/payment"
	"github.com/MatveyArbuzov/fincart/internal/product"
	"github.com/MatveyArbuzov/fincart/internal/user"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")

	db, err := database.NewPostgres(ctx, dsn)
	if err != nil {
		log.Fatalf(
			"failed to connect to database: %v",
			err,
		)
	}
	defer db.Close()

	transactionManager := database.NewManager(db)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	jwtManager := auth.NewJWTManager(
		jwtSecret,
	)

	refreshRepository :=
		auth.NewPostgresRefreshTokenRepository()

	userRepository :=
		user.NewPostgresRepository()

	refreshService := auth.NewRefreshService(
		transactionManager,
		refreshRepository,
		userRepository,
		jwtManager,
	)

	// User

	userService := user.NewService(
		transactionManager,
		userRepository,
	)

	userHandler := user.NewHandler(
		userService,
		jwtManager,
		refreshService,
	)

	// Product

	productRepository :=
		product.NewPostgresRepository(db)

	productTransactionRepository :=
		product.NewPostgresTransactionRepository()

	productService := product.NewService(
		transactionManager,
		productRepository,
		productTransactionRepository,
	)

	productHandler :=
		product.NewHandler(productService)

	// Order

	paymentService :=
		payment.NewFakeService(
			payment.ResultSuccess,
		)

	orderRepository :=
		order.NewPostgresRepository()

	orderService := order.NewService(
		transactionManager,
		productTransactionRepository,
		orderRepository,
		paymentService,
	)

	orderHandler :=
		order.NewHandler(orderService)

	// HTTP

	router := httpserver.NewRouter(
		productHandler,
		orderHandler,
		userHandler,
		jwtManager,
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
