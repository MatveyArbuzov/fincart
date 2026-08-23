package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"
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

type testApp struct {
	db     *sql.DB
	router http.Handler
}

func TestMain(m *testing.M) {
	code := runTests(m)
	os.Exit(code)
}

func runTests(m *testing.M) int {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	dsn := os.Getenv("DATABASE_URL")

	if dsn == "" {
		dsn = "postgresql://fincart:fincart@localhost:5432/fincart?sslmode=disable"
	}

	db, err := database.NewPostgres(ctx, dsn)
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	defer db.Close()

	app := newTestApp(db)

	testServer = app

	return m.Run()
}

var testServer *testApp

func newTestApp(db *sql.DB) *testApp {
	transactionManager := database.NewManager(db)

	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		jwtSecret = "integration-test-secret"
	}

	jwtManager := auth.NewJWTManager(jwtSecret)

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

	userService := user.NewService(
		transactionManager,
		userRepository,
	)

	userHandler := user.NewHandler(
		userService,
		jwtManager,
		refreshService,
	)

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

	cartRepository :=
		cart.NewPostgresRepository()

	cartService := cart.NewService(
		transactionManager,
		cartRepository,
		productTransactionRepository,
	)

	cartHandler :=
		cart.NewHandler(cartService)

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

	router := httpserver.NewRouter(
		productHandler,
		orderHandler,
		userHandler,
		cartHandler,
		jwtManager,
	)

	return &testApp{
		db:     db,
		router: router,
	}
}

func resetDatabase(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	_, err := testServer.db.ExecContext(
		ctx,
		`
		TRUNCATE TABLE
			refresh_tokens,
			order_items,
			orders,
			users
		RESTART IDENTITY CASCADE
		`,
	)
	if err != nil {
		t.Fatalf("failed to reset users/orders: %v", err)
	}

	_, err = testServer.db.ExecContext(
		ctx,
		`
    	TRUNCATE TABLE products RESTART IDENTITY CASCADE
    	`,
	)
	if err != nil {
		t.Fatalf("failed to reset products: %v", err)
	}

	_, err = testServer.db.ExecContext(
		ctx,
		`
		INSERT INTO products (
			name,
			description,
			price,
			currency,
			stock
		)
		VALUES
			(
				'MacBook Pro',
				'Apple laptop',
				150000,
				'EUR',
				10
			),
			(
				'Mechanical Keyboard',
				'Mechanical keyboard',
				12000,
				'EUR',
				50
			)
		`,
	)
	if err != nil {
		t.Fatalf("failed to seed products: %v", err)
	}
}

func decodeJSON[T any](
	t *testing.T,
	resp *http.Response,
) T {
	t.Helper()

	defer resp.Body.Close()

	var result T

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf(
			"failed to decode response JSON: %v",
			err,
		)
	}

	return result
}
