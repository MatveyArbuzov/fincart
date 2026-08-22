package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManager_GenerateAndParseToken(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("test-secret")

	token, err := manager.GenerateToken(123, "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	userID, role, err := manager.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}

	if userID != 123 {
		t.Fatalf("userID = %d, want 123", userID)
	}

	if role != "user" {
		t.Fatalf("role = %q, want user", role)
	}
}

func TestJWTManager_ParseToken_Invalid(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("test-secret")

	otherManager := NewJWTManager("other-secret")

	validToken, err := manager.GenerateToken(123, "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	otherToken, err := otherManager.GenerateToken(123, "user")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty",
			token: "",
		},
		{
			name:  "garbage",
			token: "not-a-jwt",
		},
		{
			name:  "wrong secret",
			token: otherToken,
		},
		{
			name:  "valid token with extra garbage",
			token: validToken + "garbage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := manager.ParseToken(tt.token)

			if err != ErrInvalidToken {
				t.Fatalf(
					"ParseToken() error = %v, want %v",
					err,
					ErrInvalidToken,
				)
			}
		})
	}
}

func TestJWTManager_ParseToken_InvalidClaims(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("test-secret")

	tests := []struct {
		name   string
		userID int64
		role   string
	}{
		{
			name:   "zero user id",
			userID: 0,
			role:   "user",
		},
		{
			name:   "negative user id",
			userID: -1,
			role:   "user",
		},
		{
			name:   "empty role",
			userID: 1,
			role:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := jwt.NewWithClaims(
				jwt.SigningMethodHS256,
				Claims{
					UserID: tt.userID,
					Role:   tt.role,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(
							time.Now().Add(time.Hour),
						),
					},
				},
			)

			tokenString, err := token.SignedString(
				[]byte("test-secret"),
			)
			if err != nil {
				t.Fatalf("SignedString() error = %v", err)
			}

			_, _, err = manager.ParseToken(tokenString)

			if err != ErrInvalidToken {
				t.Fatalf(
					"ParseToken() error = %v, want %v",
					err,
					ErrInvalidToken,
				)
			}
		})
	}
}

func TestJWTManager_ParseToken_Expired(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("test-secret")

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		Claims{
			UserID: 1,
			Role:   "user",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(-time.Hour),
				),
			},
		},
	)

	tokenString, err := token.SignedString(
		[]byte("test-secret"),
	)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, _, err = manager.ParseToken(tokenString)

	if err != ErrInvalidToken {
		t.Fatalf(
			"ParseToken() error = %v, want %v",
			err,
			ErrInvalidToken,
		)
	}
}

func TestJWTManager_ParseToken_WrongAlgorithm(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("test-secret")

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS384,
		Claims{
			UserID: 1,
			Role:   "user",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(time.Hour),
				),
			},
		},
	)

	tokenString, err := token.SignedString(
		[]byte("test-secret"),
	)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	_, _, err = manager.ParseToken(tokenString)

	if err != ErrInvalidToken {
		t.Fatalf(
			"ParseToken() error = %v, want %v",
			err,
			ErrInvalidToken,
		)
	}
}

func TestJWTManager_Middleware(t *testing.T) {
	t.Parallel()

	manager := NewJWTManager("test-secret")

	validToken, err := manager.GenerateToken(42, "admin")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantClaims bool
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantClaims: true,
		},
		{
			name:       "bearer lowercase",
			authHeader: "bearer " + validToken,
			wantStatus: http.StatusOK,
			wantClaims: true,
		},
		{
			name:       "missing header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing token",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "too many parts",
			authHeader: "Bearer " + validToken + " extra",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong scheme",
			authHeader: "Basic " + validToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				if tt.wantClaims {
					claims, ok := ClaimsFromContext(
						r.Context(),
					)

					if !ok {
						t.Fatal("ClaimsFromContext() ok = false")
					}

					if claims.UserID != 42 {
						t.Fatalf(
							"claims.UserID = %d, want 42",
							claims.UserID,
						)
					}

					if claims.Role != "admin" {
						t.Fatalf(
							"claims.Role = %q, want admin",
							claims.Role,
						)
					}
				}

				w.WriteHeader(http.StatusOK)
			})

			handler := manager.Middleware(next)

			req := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)

			if tt.authHeader != "" {
				req.Header.Set(
					"Authorization",
					tt.authHeader,
				)
			}

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf(
					"status = %d, want %d",
					rec.Code,
					tt.wantStatus,
				)
			}
		})
	}
}
