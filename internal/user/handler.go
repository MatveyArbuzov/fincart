package user

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MatveyArbuzov/fincart/internal/auth"
)

type Handler struct {
	service        *Service
	jwtManager     *auth.JWTManager
	refreshService *auth.RefreshService
}

func NewHandler(
	service *Service,
	jwtManager *auth.JWTManager,
	refreshService *auth.RefreshService,
) *Handler {
	return &Handler{
		service:        service,
		jwtManager:     jwtManager,
		refreshService: refreshService,
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

type LoginResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func writeError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

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

	accessToken, refreshToken, err := h.refreshService.Create(
		r.Context(),
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
			User:         user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	)
}

func (h *Handler) Refresh(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request RefreshRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	if request.RefreshToken == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			"invalid_refresh_token",
		)
		return
	}

	accessToken, refreshToken, err := h.refreshService.Refresh(
		r.Context(),
		request.RefreshToken,
	)

	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidRefreshToken),
			errors.Is(err, auth.ErrRefreshTokenExpired),
			errors.Is(err, auth.ErrRefreshTokenRevoked):

			writeError(
				w,
				http.StatusUnauthorized,
				"invalid_refresh_token",
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

	_ = json.NewEncoder(w).Encode(
		RefreshResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
	)
}

func (h *Handler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request LogoutRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_request_body",
		)
		return
	}

	if request.RefreshToken == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_refresh_token",
		)
		return
	}

	err := h.refreshService.Revoke(
		r.Context(),
		request.RefreshToken,
	)

	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidRefreshToken):
			writeError(
				w,
				http.StatusUnauthorized,
				"invalid_refresh_token",
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

	w.WriteHeader(http.StatusNoContent)
}
