package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/auth"
)

type Handler struct {
	service    *Service
	jwtManager *auth.JWTManager
}

func NewHandler(
	service *Service,
	jwtManager *auth.JWTManager,
) *Handler {
	return &Handler{
		service:    service,
		jwtManager: jwtManager,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type LoginResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
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

func (h *Handler) Register(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	user, err := h.service.Register(
		r.Context(),
		request.Email,
		request.Password,
	)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidUser):
			writeError(
				w,
				http.StatusBadRequest,
				"invalid_user",
			)

		default:
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

	_ = json.NewEncoder(w).Encode(user)
}

func (h *Handler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	user, err := h.service.Login(
		r.Context(),
		request.Email,
		request.Password,
	)

	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			writeError(
				w,
				http.StatusUnauthorized,
				"invalid_credentials",
			)

		default:
			writeError(
				w,
				http.StatusInternalServerError,
				"internal_server_error",
			)
		}

		return
	}

	token, err := h.jwtManager.GenerateToken(
		user.ID,
		string(user.Role),
	)
	if err != nil {
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

	_ = json.NewEncoder(w).Encode(
		LoginResponse{
			User:  user,
			Token: token,
		},
	)
}
