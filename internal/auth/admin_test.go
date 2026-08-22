package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminMiddleware(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		role           string
		withRole       bool
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name:           "admin allowed",
			role:           "admin",
			withRole:       true,
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
		{
			name:           "user forbidden",
			role:           "user",
			withRole:       true,
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "missing role forbidden",
			withRole:       false,
			wantStatus:     http.StatusForbidden,
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

			handler := AdminMiddleware(next)

			req := httptest.NewRequest(
				http.MethodGet,
				"/admin",
				nil,
			)

			if tt.withRole {
				req = req.WithContext(
					WithRole(req.Context(), tt.role),
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
