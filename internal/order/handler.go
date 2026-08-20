package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/MatveyArbuzov/fincart/internal/auth"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(
		errorResponse{
			Error: message,
		},
	)
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	userID := claims.UserID

	var request CreateOrderRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	order, err := h.service.CreateOrder(
		r.Context(),
		userID,
		request,
	)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrder):
			writeError(
				w,
				http.StatusBadRequest,
				"invalid_order",
			)

		case errors.Is(err, ErrInsufficientStock):
			writeError(
				w,
				http.StatusConflict,
				"insufficient_stock",
			)
		case errors.Is(err, ErrProductNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"product_not_found",
			)
		case errors.Is(err, ErrDifferentCurrencies):
			writeError(
				w,
				http.StatusBadRequest,
				"different_currencies",
			)
		default:
			log.Printf(
				"create order error: %v",
				err,
			)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal_server_error",
			)
		}

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf(
			"failed to encode order response: %v",
			err,
		)
	}
}

func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderIDString := r.PathValue("id")

	if orderIDString == "" {
		const prefix = "/api/v1/orders/"
		orderIDString = strings.TrimPrefix(r.URL.Path, prefix)
		orderIDString = strings.TrimSuffix(orderIDString, "/cancel")
	}

	orderID, err := strconv.ParseInt(orderIDString, 10, 64)
	if err != nil || orderID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_order_id",
		)
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	userID := claims.UserID

	err = h.service.CancelOrder(
		r.Context(),
		userID,
		orderID,
	)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrder):
			writeError(
				w,
				http.StatusBadRequest,
				"invalid_order",
			)

		case errors.Is(err, ErrOrderNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"order_not_found",
			)

		case errors.Is(err, ErrInvalidOrderState):
			writeError(
				w,
				http.StatusConflict,
				"invalid_order_state",
			)

		case errors.Is(err, ErrOrderForbidden):
			writeError(
				w,
				http.StatusForbidden,
				"order_forbidden",
			)

		default:
			log.Printf(
				"cancel order error: %v",
				err,
			)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal_server_error",
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetOrder(
	w http.ResponseWriter,
	r *http.Request,
) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	userID := claims.UserID

	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_order_id",
		)
		return
	}

	order, items, err := h.service.GetOrder(
		r.Context(),
		userID,
		id,
	)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrder):
			writeError(
				w,
				http.StatusBadRequest,
				"invalid_order_id",
			)

		case errors.Is(err, ErrOrderNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"order_not_found",
			)
		case errors.Is(err, ErrOrderForbidden):
			writeError(
				w,
				http.StatusForbidden,
				"order_forbidden",
			)

		default:
			log.Printf(
				"get order error: %v",
				err,
			)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal_server_error",
			)
		}

		return
	}

	response := struct {
		Order Order       `json:"order"`
		Items []OrderItem `json:"items"`
	}{
		Order: order,
		Items: items,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf(
			"failed to encode order response: %v",
			err,
		)
	}
}

func (h *Handler) PayOrder(w http.ResponseWriter, r *http.Request) {
	orderIDString := r.PathValue("id")

	orderID, err := strconv.ParseInt(orderIDString, 10, 64)
	if err != nil || orderID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_order_id",
		)
		return
	}

	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	userID := claims.UserID

	err = h.service.PayOrder(
		r.Context(),
		userID,
		orderID,
	)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrder):
			writeError(
				w,
				http.StatusBadRequest,
				"invalid_order_id",
			)

		case errors.Is(err, ErrOrderNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"order_not_found",
			)
		case errors.Is(err, ErrOrderForbidden):
			writeError(
				w,
				http.StatusForbidden,
				"order_forbidden",
			)
		case errors.Is(err, ErrInvalidOrderState):
			writeError(
				w,
				http.StatusConflict,
				"invalid_order_state",
			)

		case errors.Is(err, ErrPaymentFailed):
			writeError(
				w,
				http.StatusPaymentRequired,
				"payment_failed",
			)

		case errors.Is(err, ErrPaymentTimeout):
			writeError(
				w,
				http.StatusGatewayTimeout,
				"payment_timeout",
			)

		default:
			log.Printf(
				"pay order error: %v",
				err,
			)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal_server_error",
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListOrders(
	w http.ResponseWriter,
	r *http.Request,
) {
	orders, err := h.service.ListOrders(
		r.Context(),
	)
	if err != nil {
		log.Printf(
			"list orders error: %v",
			err,
		)

		writeError(
			w,
			http.StatusInternalServerError,
			"internal_server_error",
		)

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	response := struct {
		Orders []Order `json:"orders"`
	}{
		Orders: orders,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf(
			"failed to encode orders response: %v",
			err,
		)
	}
}

func (h *Handler) UpdateOrderStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	orderID, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || orderID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_order_id",
		)
		return
	}

	var request UpdateOrderStatusRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	if err := h.service.UpdateOrderStatus(
		r.Context(),
		orderID,
		request.Status,
	); err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrder):
			writeError(
				w,
				http.StatusBadRequest,
				"invalid_order_id",
			)

		case errors.Is(err, ErrOrderNotFound):
			writeError(
				w,
				http.StatusNotFound,
				"order_not_found",
			)

		case errors.Is(err, ErrInvalidOrderState):
			writeError(
				w,
				http.StatusConflict,
				"invalid_order_state",
			)

		default:
			log.Printf(
				"update order status error: %v",
				err,
			)

			writeError(
				w,
				http.StatusInternalServerError,
				"internal_server_error",
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
