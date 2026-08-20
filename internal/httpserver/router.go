package httpserver

import (
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/cart"
	"github.com/MatveyArbuzov/fincart/internal/order"
	"github.com/MatveyArbuzov/fincart/internal/product"
	"github.com/MatveyArbuzov/fincart/internal/user"
)

func NewRouter(
	productHandler *product.Handler,
	orderHandler *order.Handler,
	userHandler *user.Handler,
	cartHandler *cart.Handler,
	jwtManager *auth.JWTManager,
) http.Handler {
	mux := http.NewServeMux()

	authMiddleware := jwtManager.Middleware
	adminMiddleware := auth.RequireRole("admin")

	// Public products

	mux.HandleFunc(
		"GET /api/v1/products",
		productHandler.GetProducts,
	)

	mux.HandleFunc(
		"GET /api/v1/products/{id}",
		productHandler.GetProductByID,
	)

	// Auth

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

	// Admin products

	mux.Handle(
		"POST /api/v1/admin/products",
		authMiddleware(
			adminMiddleware(
				http.HandlerFunc(
					productHandler.CreateProduct,
				),
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/admin/products/{id}",
		authMiddleware(
			adminMiddleware(
				http.HandlerFunc(
					productHandler.UpdateProduct,
				),
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/admin/products/{id}",
		authMiddleware(
			adminMiddleware(
				http.HandlerFunc(
					productHandler.DeleteProduct,
				),
			),
		),
	)

	// Orders

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

	// Admin orders

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

	mux.Handle(
		"GET /api/v1/cart",
		authMiddleware(
			http.HandlerFunc(
				cartHandler.GetCart,
			),
		),
	)

	mux.Handle(
		"POST /api/v1/cart/items",
		authMiddleware(
			http.HandlerFunc(
				cartHandler.AddItem,
			),
		),
	)

	mux.Handle(
		"PATCH /api/v1/cart/items/{product_id}",
		authMiddleware(
			http.HandlerFunc(
				cartHandler.UpdateItem,
			),
		),
	)

	mux.Handle(
		"DELETE /api/v1/cart/items/{product_id}",
		authMiddleware(
			http.HandlerFunc(
				cartHandler.DeleteItem,
			),
		),
	)

	mux.Handle(
		"POST /api/v1/cart/checkout",
		authMiddleware(
			http.HandlerFunc(
				cartHandler.Checkout,
			),
		),
	)

	return mux
}
