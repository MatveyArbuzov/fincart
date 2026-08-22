package auth

import (
	"context"
	"testing"
)

func TestUserIDContext(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := WithUserID(context.Background(), 42)

		got, ok := UserIDFromContext(ctx)

		if !ok {
			t.Fatal("UserIDFromContext() ok = false, want true")
		}

		if got != 42 {
			t.Fatalf("UserIDFromContext() = %d, want 42", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()

		got, ok := UserIDFromContext(context.Background())

		if ok {
			t.Fatal("UserIDFromContext() ok = true, want false")
		}

		if got != 0 {
			t.Fatalf("UserIDFromContext() = %d, want 0", got)
		}
	})
}

func TestRoleContext(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ctx := WithRole(context.Background(), "admin")

		got, ok := RoleFromContext(ctx)

		if !ok {
			t.Fatal("RoleFromContext() ok = false, want true")
		}

		if got != "admin" {
			t.Fatalf("RoleFromContext() = %q, want %q", got, "admin")
		}
	})

	t.Run("missing", func(t *testing.T) {
		t.Parallel()

		got, ok := RoleFromContext(context.Background())

		if ok {
			t.Fatal("RoleFromContext() ok = true, want false")
		}

		if got != "" {
			t.Fatalf("RoleFromContext() = %q, want empty string", got)
		}
	})
}
