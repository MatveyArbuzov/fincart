package httpserver

import (
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/product"
	"github.com/MatveyArbuzov/fincart/internal/user"
)

func NewRouter(
	productHandler *product.Handler,
	orderHandler *order.Handler,
	userHandler *user.Handler,
	jwtManager *auth.JWTManager,
) http.Handler {
	mux := http.NewServeMux()

	authMiddleware := jwtManager.Middleware
	adminMiddleware := auth.RequireRole("admin")

	mux.HandleFunc(
		"GET /api/v1/products",
		productHandler.GetProducts,
	)

	mux.HandleFunc(
		"GET /api/v1/products/{id}",
		productHandler.GetProductByID,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/register",
		userHandler.Register,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/login",
		userHandler.Login,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/refresh",
		userHandler.Refresh,
	)

	mux.HandleFunc(
		"POST /api/v1/auth/logout",
		userHandler.Logout,
	)

	mux.Handle(
		"POST /api/v1/orders",
		authMiddleware(
			http.HandlerFunc(
				orderHandler.CreateOrder,
			),
		),
	)

	mux.Handle(
		"GET /api/v1/orders/{id}",
		authMiddleware(
			http.HandlerFunc(
				orderHandler.GetOrder,
			),
		),
	)

	mux.Handle(
		"POST /api/v1/orders/{id}/cancel",
		authMiddleware(
			http.HandlerFunc(
				orderHandler.CancelOrder,
			),
		),
	)

	mux.Handle(
		"POST /api/v1/orders/{id}/pay",
		authMiddleware(
			http.HandlerFunc(
				orderHandler.PayOrder,
			),
		),
	)

	mux.Handle(
		"GET /api/v1/admin/orders",
		authMiddleware(
			adminMiddleware(
				http.HandlerFunc(
					orderHandler.ListOrders,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/admin/orders/{id}/status",
		authMiddleware(
			adminMiddleware(
				http.HandlerFunc(
					orderHandler.UpdateOrderStatus,
				),
			),
		),
	)

	return mux
}
