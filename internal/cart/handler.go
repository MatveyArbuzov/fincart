package cart

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/MatveyArbuzov/fincart/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetCart(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	cart, err := h.service.GetCart(
		r.Context(),
		userID,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		cart,
	)
}

func (h *Handler) AddItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	var request AddItemRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	cart, err := h.service.AddItem(
		r.Context(),
		userID,
		request,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		cart,
	)
}

func (h *Handler) UpdateItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	productID, err := strconv.ParseInt(
		r.PathValue("product_id"),
		10,
		64,
	)
	if err != nil || productID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_product_id",
		)
		return
	}

	var request UpdateItemRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	cart, err := h.service.UpdateItem(
		r.Context(),
		userID,
		productID,
		request.Quantity,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		cart,
	)
}

func (h *Handler) DeleteItem(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	productID, err := strconv.ParseInt(
		r.PathValue("product_id"),
		10,
		64,
	)
	if err != nil || productID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_product_id",
		)
		return
	}

	cart, err := h.service.DeleteItem(
		r.Context(),
		userID,
		productID,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		cart,
	)
}

func (h *Handler) Checkout(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := userIDFromRequest(r)
	if !ok {
		writeError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	order, err := h.service.Checkout(
		r.Context(),
		userID,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		order,
	)
}

func userIDFromRequest(r *http.Request) (int64, bool) {
	claims, ok := auth.ClaimsFromContext(
		r.Context(),
	)
	if !ok || claims.UserID <= 0 {
		return 0, false
	}

	return claims.UserID, true
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value interface{},
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	writeJSON(
		w,
		status,
		struct {
			Error string `json:"error"`
		}{
			Error: message,
		},
	)
}

func writeServiceError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, ErrInvalidCart),
		errors.Is(err, ErrInvalidQuantity):
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_cart",
		)

	case errors.Is(err, ErrProductNotFound):
		writeError(
			w,
			http.StatusNotFound,
			"product_not_found",
		)

	case errors.Is(err, ErrCartNotFound):
		writeError(
			w,
			http.StatusNotFound,
			"cart_not_found",
		)

	case errors.Is(err, ErrCartItemNotFound):
		writeError(
			w,
			http.StatusNotFound,
			"cart_item_not_found",
		)

	case errors.Is(err, ErrInsufficientStock):
		writeError(
			w,
			http.StatusConflict,
			"insufficient_stock",
		)

	case errors.Is(err, ErrDifferentCurrency):
		writeError(
			w,
			http.StatusBadRequest,
			"different_currencies",
		)

	case errors.Is(err, ErrEmptyCart):
		writeError(
			w,
			http.StatusConflict,
			"empty_cart",
		)

	default:
		writeError(
			w,
			http.StatusInternalServerError,
			"internal_server_error",
		)
	}
}
