package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/cart"
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

	// Cart

	cartRepository := cart.NewPostgresRepository()

	cartService := cart.NewService(
		transactionManager,
		cartRepository,
		productTransactionRepository,
	)

	cartHandler := cart.NewHandler(
		cartService,
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
		cartHandler,
		jwtManager,
	)

	handler := httpserver.CORS(router)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Start server.

	serverErr := make(chan error, 1)

	go func() {
		log.Println("server started on :8080")

		if err := server.ListenAndServe(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for shutdown signal.

	shutdownSignal := make(chan os.Signal, 1)

	signal.Notify(
		shutdownSignal,
		os.Interrupt,
		syscall.SIGTERM,
	)

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf(
				"server failed: %v",
				err,
			)
		}

	case sig := <-shutdownSignal:
		log.Printf(
			"shutdown signal received: %v",
			sig,
		)
	}

	// Graceful shutdown.

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf(
			"graceful shutdown failed: %v",
			err,
		)

		return
	}

	log.Println("server stopped gracefully")
}
