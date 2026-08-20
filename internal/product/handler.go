package product

import (
	"encoding/json"
	"errors"
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

func (h *Handler) GetProducts(
	w http.ResponseWriter,
	r *http.Request,
) {
	products, err := h.service.GetProducts(r.Context())
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	response := struct {
		Items []Product `json:"items"`
	}{
		Items: products,
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func (h *Handler) GetProductByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid product id",
			http.StatusBadRequest,
		)
		return
	}

	product, err := h.service.GetProductByID(
		r.Context(),
		id,
	)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			http.Error(
				w,
				"product not found",
				http.StatusNotFound,
			)
			return
		}

		if errors.Is(err, ErrInvalidProduct) {
			http.Error(
				w,
				"invalid product id",
				http.StatusBadRequest,
			)
			return
		}

		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	if err := json.NewEncoder(w).Encode(product); err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func (h *Handler) CreateProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	product, err := h.service.CreateProduct(
		r.Context(),
		request,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidProduct) {
			http.Error(
				w,
				"invalid product",
				http.StatusBadRequest,
			)
			return
		}

		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(product); err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func (h *Handler) UpdateProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid product id",
			http.StatusBadRequest,
		)
		return
	}

	var request UpdateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	product, err := h.service.UpdateProduct(
		r.Context(),
		id,
		request,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidProduct):
			http.Error(
				w,
				"invalid product",
				http.StatusBadRequest,
			)

		case errors.Is(err, ErrProductNotFound):
			http.Error(
				w,
				"product not found",
				http.StatusNotFound,
			)

		default:
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

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(product); err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func (h *Handler) DeleteProduct(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid product id",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.DeleteProduct(
		r.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidProduct):
			http.Error(
				w,
				"invalid product id",
				http.StatusBadRequest,
			)

		case errors.Is(err, ErrProductNotFound):
			http.Error(
				w,
				"product not found",
				http.StatusNotFound,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
