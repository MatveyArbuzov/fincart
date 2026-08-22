package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/auth"
	"github.com/MatveyArbuzov/fincart/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func newTestRefreshService(
	repository *testRefreshRepository,
) *auth.RefreshService {
	return auth.NewRefreshService(
		&testRefreshTransactionManager{},
		repository,
		&testUserRoleProvider{},
		auth.NewJWTManager("test-secret"),
	)
}

type testRefreshTransactionManager struct{}

func (m *testRefreshTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	return fn(nil)
}

type testRefreshRepository struct {
	token auth.RefreshToken
	err   error
}

func (r *testRefreshRepository) Create(
	ctx context.Context,
	tx database.Tx,
	token auth.RefreshToken,
) error {
	r.token = token
	return nil
}

func (r *testRefreshRepository) GetByHash(
	ctx context.Context,
	tx database.Tx,
	tokenHash string,
) (auth.RefreshToken, error) {
	if r.err != nil {
		return auth.RefreshToken{}, r.err
	}

	return r.token, nil
}

func (r *testRefreshRepository) Revoke(
	ctx context.Context,
	tx database.Tx,
	id int64,
) error {
	return nil
}

type testUserRoleProvider struct{}

func (p *testUserRoleProvider) GetRoleByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (string, error) {
	return "user", nil
}

func TestHandler_Register(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		transactionManager := &mockTransactionManager{
			withinTransactionFunc: func(
				ctx context.Context,
				fn func(tx database.Tx) error,
			) error {
				return fn(nil)
			},
		}

		repository := &mockRepository{
			createFunc: func(
				ctx context.Context,
				tx database.Tx,
				email string,
				passwordHash string,
				role Role,
			) (User, error) {
				return User{
					ID:    1,
					Email: email,
					Role:  role,
				}, nil
			},
		}

		service := NewService(
			transactionManager,
			repository,
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("secret"),
			nil,
		)

		body := `{
			"email": "test@example.com",
			"password": "password123"
		}`

		req := httptest.NewRequest(
			http.MethodPost,
			"/register",
			bytes.NewBufferString(body),
		)

		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusCreated,
			)
		}

		var got User

		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if got.ID != 1 {
			t.Fatalf("ID = %d, want 1", got.ID)
		}

		if got.Email != "test@example.com" {
			t.Fatalf(
				"Email = %q, want test@example.com",
				got.Email,
			)
		}

		if got.PasswordHash != "" {
			t.Fatal("PasswordHash must not be exposed in JSON")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		service := NewService(
			&mockTransactionManager{},
			&mockRepository{},
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/register",
			bytes.NewBufferString("{invalid"),
		)

		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusBadRequest,
			)
		}

		assertJSONError(
			t,
			rec,
			"invalid_request_body",
		)
	})

	t.Run("invalid user", func(t *testing.T) {
		t.Parallel()

		transactionManager := &mockTransactionManager{
			withinTransactionFunc: func(
				ctx context.Context,
				fn func(tx database.Tx) error,
			) error {
				return nil
			},
		}

		service := NewService(
			transactionManager,
			&mockRepository{},
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/register",
			bytes.NewBufferString(`{
				"email": "",
				"password": "password"
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusBadRequest,
			)
		}

		assertJSONError(
			t,
			rec,
			"invalid_user",
		)
	})

	t.Run("internal error", func(t *testing.T) {
		t.Parallel()

		repositoryErr := errors.New("database error")

		transactionManager := &mockTransactionManager{
			withinTransactionFunc: func(
				ctx context.Context,
				fn func(tx database.Tx) error,
			) error {
				return fn(nil)
			},
		}

		repository := &mockRepository{
			createFunc: func(
				ctx context.Context,
				tx database.Tx,
				email string,
				passwordHash string,
				role Role,
			) (User, error) {
				return User{}, repositoryErr
			},
		}

		service := NewService(
			transactionManager,
			repository,
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/register",
			bytes.NewBufferString(`{
				"email": "test@example.com",
				"password": "password"
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Register(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusInternalServerError,
			)
		}

		assertJSONError(
			t,
			rec,
			"internal_server_error",
		)
	})
}

func TestHandler_Login(t *testing.T) {
	t.Parallel()

	password := "password123"

	passwordHash := mustHashPassword(t, password)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		repository := &mockRepository{
			getByEmailFunc: func(
				ctx context.Context,
				tx database.Tx,
				email string,
			) (User, string, error) {
				return User{
						ID:    42,
						Email: email,
						Role:  RoleUser,
					},
					passwordHash,
					nil
			},
		}

		service := NewService(
			&mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					return fn(nil)
				},
			},
			repository,
		)

		refreshRepository := &testRefreshRepository{}

		refreshService := newTestRefreshService(
			refreshRepository,
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("test-secret"),
			refreshService,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/login",
			bytes.NewBufferString(`{
				"email": "TEST@EXAMPLE.COM",
				"password": "password123"
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusOK,
			)
		}

		var response LoginResponse

		if err := json.NewDecoder(
			rec.Body,
		).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}

		if response.User.ID != 42 {
			t.Fatalf(
				"User.ID = %d, want 42",
				response.User.ID,
			)
		}

		if response.AccessToken == "" {
			t.Fatal("AccessToken is empty")
		}

		if response.RefreshToken == "" {
			t.Fatal("RefreshToken is empty")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		service := NewService(
			&mockTransactionManager{},
			&mockRepository{},
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/login",
			bytes.NewBufferString("{invalid"),
		)

		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusBadRequest,
			)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		t.Parallel()

		repository := &mockRepository{
			getByEmailFunc: func(
				ctx context.Context,
				tx database.Tx,
				email string,
			) (User, string, error) {
				return User{}, "", ErrUserNotFound
			},
		}

		service := NewService(
			&mockTransactionManager{
				withinTransactionFunc: func(
					ctx context.Context,
					fn func(tx database.Tx) error,
				) error {
					return fn(nil)
				},
			},
			repository,
		)

		handler := NewHandler(
			service,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/login",
			bytes.NewBufferString(`{
				"email": "missing@example.com",
				"password": "password"
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Login(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusUnauthorized,
			)
		}

		assertJSONError(
			t,
			rec,
			"invalid_credentials",
		)
	})
}

func TestHandler_Refresh(t *testing.T) {
	t.Parallel()

	t.Run("empty refresh token", func(t *testing.T) {
		t.Parallel()

		handler := NewHandler(
			nil,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/refresh",
			bytes.NewBufferString(`{
				"refresh_token": ""
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Refresh(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusUnauthorized,
			)
		}

		assertJSONError(
			t,
			rec,
			"invalid_refresh_token",
		)
	})

	t.Run("invalid token", func(t *testing.T) {
		t.Parallel()

		repository := &testRefreshRepository{
			err: auth.ErrRefreshTokenNotFound,
		}

		refreshService := newTestRefreshService(
			repository,
		)

		handler := NewHandler(
			nil,
			auth.NewJWTManager("secret"),
			refreshService,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/refresh",
			bytes.NewBufferString(`{
				"refresh_token": "invalid"
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Refresh(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusUnauthorized,
			)
		}

		assertJSONError(
			t,
			rec,
			"invalid_refresh_token",
		)
	})
}

func TestHandler_Logout(t *testing.T) {
	t.Parallel()

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()

		handler := NewHandler(
			nil,
			auth.NewJWTManager("secret"),
			nil,
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/logout",
			bytes.NewBufferString(`{
				"refresh_token": ""
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Logout(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusBadRequest,
			)
		}

		assertJSONError(
			t,
			rec,
			"invalid_refresh_token",
		)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		refreshRepository := &testRefreshRepository{
			token: auth.RefreshToken{
				ID:        1,
				UserID:    42,
				ExpiresAt: time.Now().Add(time.Hour),
			},
		}

		handler := NewHandler(
			nil,
			auth.NewJWTManager("secret"),
			newTestRefreshService(refreshRepository),
		)

		req := httptest.NewRequest(
			http.MethodPost,
			"/logout",
			bytes.NewBufferString(`{
				"refresh_token": "test-token"
			}`),
		)

		rec := httptest.NewRecorder()

		handler.Logout(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf(
				"status = %d, want %d",
				rec.Code,
				http.StatusNoContent,
			)
		}
	})
}

func mustHashPassword(
	t *testing.T,
	password string,
) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf(
			"bcrypt.GenerateFromPassword() error = %v",
			err,
		)
	}

	return string(hash)
}

func assertJSONError(
	t *testing.T,
	rec *httptest.ResponseRecorder,
	want string,
) {
	t.Helper()

	var response errorResponse

	if err := json.NewDecoder(
		rec.Body,
	).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if response.Error != want {
		t.Fatalf(
			"error = %q, want %q",
			response.Error,
			want,
		)
	}
}
