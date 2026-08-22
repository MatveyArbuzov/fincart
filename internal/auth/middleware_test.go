package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClaimsFromContext(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		want := Claims{
			UserID: 1,
			Role:   "admin",
		}

		ctx := httptest.NewRequest(
			http.MethodGet,
			"/",
			nil,
		).Context()

		ctx = contextWithClaims(ctx, want)

		got, ok := ClaimsFromContext(ctx)

		if !ok {
			t.Fatal("ClaimsFromContext() ok = false")
		}

		if got.UserID != want.UserID {
			t.Fatalf(
				"UserID = %d, want %d",
				got.UserID,
				want.UserID,
			)
		}

		if got.Role != want.Role {
			t.Fatalf(
				"Role = %q, want %q",
				got.Role,
				want.Role,
			)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()

		_, ok := ClaimsFromContext(
			httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			).Context(),
		)

		if ok {
			t.Fatal("ClaimsFromContext() ok = true, want false")
		}
	})
}

func contextWithClaims(
	ctx context.Context,
	claims Claims,
) context.Context {
	return context.WithValue(
		ctx,
		claimsContextKey{},
		claims,
	)
}

func TestRequireRole(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		claims         *Claims
		requiredRole   string
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name: "admin allowed",
			claims: &Claims{
				UserID: 1,
				Role:   "admin",
			},
			requiredRole:   "admin",
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name: "wrong role forbidden",
			claims: &Claims{
				UserID: 1,
				Role:   "user",
			},
			requiredRole:   "admin",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "missing claims unauthorized",
			requiredRole:   "admin",
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			nextCalled := false

			next := http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireRole(tt.requiredRole)(next)

			req := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)

			if tt.claims != nil {
				req = req.WithContext(
					contextWithClaims(
						req.Context(),
						*tt.claims,
					),
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

			if nextCalled != tt.wantNextCalled {
				t.Fatalf(
					"nextCalled = %v, want %v",
					nextCalled,
					tt.wantNextCalled,
				)
			}
		})
	}
}
