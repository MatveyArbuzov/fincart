package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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
	userID, err := strconv.ParseInt(
		r.Header.Get("X-User-ID"),
		10,
		64,
	)
	if err != nil || userID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_user_id",
		)
		return
	}

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
