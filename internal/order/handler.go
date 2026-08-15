package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(r.Header.Get("X-User-ID"), 10, 64)
	if err != nil || userID <= 0 {
		http.Error(
			w,
			"invalid user id",
			http.StatusBadRequest,
		)
		return
	}

	var request CreateOrderRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
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
			http.Error(
				w,
				"invalid order",
				http.StatusBadRequest,
			)

		case errors.Is(err, ErrInsufficientStock):
			http.Error(
				w,
				"insufficient stock",
				http.StatusConflict,
			)

		default:
			log.Printf("create order error: %v", err)

			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
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
		return
	}
}
