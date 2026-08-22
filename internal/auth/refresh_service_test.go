package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MatveyArbuzov/fincart/internal/database"
)

type mockRefreshTransactionManager struct {
	err error
}

func (m *mockRefreshTransactionManager) WithinTransaction(
	ctx context.Context,
	fn func(tx database.Tx) error,
) error {
	if m.err != nil {
		return m.err
	}

	return fn(nil)
}

type mockRefreshRepository struct {
	createFunc func(
		ctx context.Context,
		tx database.Tx,
		token RefreshToken,
	) error

	getByHashFunc func(
		ctx context.Context,
		tx database.Tx,
		tokenHash string,
	) (RefreshToken, error)

	revokeFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) error
}

func (m *mockRefreshRepository) Create(
	ctx context.Context,
	tx database.Tx,
	token RefreshToken,
) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, tx, token)
	}

	return nil
}

func (m *mockRefreshRepository) GetByHash(
	ctx context.Context,
	tx database.Tx,
	tokenHash string,
) (RefreshToken, error) {
	if m.getByHashFunc != nil {
		return m.getByHashFunc(ctx, tx, tokenHash)
	}

	return RefreshToken{}, nil
}

func (m *mockRefreshRepository) Revoke(
	ctx context.Context,
	tx database.Tx,
	id int64,
) error {
	if m.revokeFunc != nil {
		return m.revokeFunc(ctx, tx, id)
	}

	return nil
}

type mockUserRoleProvider struct {
	getRoleByIDFunc func(
		ctx context.Context,
		tx database.Tx,
		id int64,
	) (string, error)
}

func (m *mockUserRoleProvider) GetRoleByID(
	ctx context.Context,
	tx database.Tx,
	id int64,
) (string, error) {
	if m.getRoleByIDFunc != nil {
		return m.getRoleByIDFunc(ctx, tx, id)
	}

	return "", nil
}

func TestRefreshService_Create(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var created RefreshToken

		repository := &mockRefreshRepository{
			createFunc: func(
				ctx context.Context,
				tx database.Tx,
				token RefreshToken,
			) error {
				created = token
				return nil
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		accessToken, refreshToken, err := service.Create(
			context.Background(),
			42,
			"user",
		)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if accessToken == "" {
			t.Fatal("access token is empty")
		}

		if refreshToken == "" {
			t.Fatal("refresh token is empty")
		}

		if created.UserID != 42 {
			t.Fatalf(
				"created.UserID = %d, want 42",
				created.UserID,
			)
		}

		if created.TokenHash != hashRefreshToken(refreshToken) {
			t.Fatal("stored token hash does not match refresh token")
		}

		if !created.ExpiresAt.After(time.Now()) {
			t.Fatal("refresh token expiration is not in the future")
		}

		userID, role, err := NewJWTManager(
			"secret",
		).ParseToken(accessToken)
		if err != nil {
			t.Fatalf("ParseToken() error = %v", err)
		}

		if userID != 42 {
			t.Fatalf("userID = %d, want 42", userID)
		}

		if role != "user" {
			t.Fatalf("role = %q, want user", role)
		}
	})

	t.Run("invalid user id", func(t *testing.T) {
		t.Parallel()

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			&mockRefreshRepository{},
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		_, _, err := service.Create(
			context.Background(),
			0,
			"user",
		)

		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrInvalidRefreshToken,
			)
		}
	})

	t.Run("empty role", func(t *testing.T) {
		t.Parallel()

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			&mockRefreshRepository{},
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		_, _, err := service.Create(
			context.Background(),
			1,
			"",
		)

		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf(
				"Create() error = %v, want %v",
				err,
				ErrInvalidRefreshToken,
			)
		}
	})
}

func TestRefreshService_Refresh(t *testing.T) {
	t.Parallel()

	const oldToken = "old-refresh-token"

	t.Run("success rotates token", func(t *testing.T) {
		t.Parallel()

		oldHash := hashRefreshToken(oldToken)

		var revokedID int64
		var created RefreshToken

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				if tokenHash != oldHash {
					t.Fatalf("hash = %q, want %q", tokenHash, oldHash)
				}

				return RefreshToken{
					ID:        10,
					UserID:    42,
					TokenHash: oldHash,
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},

			revokeFunc: func(
				ctx context.Context,
				tx database.Tx,
				id int64,
			) error {
				revokedID = id
				return nil
			},

			createFunc: func(
				ctx context.Context,
				tx database.Tx,
				token RefreshToken,
			) error {
				created = token
				return nil
			},
		}

		users := &mockUserRoleProvider{
			getRoleByIDFunc: func(
				ctx context.Context,
				tx database.Tx,
				id int64,
			) (string, error) {
				if id != 42 {
					t.Fatalf("user id = %d, want 42", id)
				}

				return "admin", nil
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			users,
			NewJWTManager("secret"),
		)

		accessToken, newRefreshToken, err := service.Refresh(
			context.Background(),
			oldToken,
		)
		if err != nil {
			t.Fatalf("Refresh() error = %v", err)
		}

		if accessToken == "" {
			t.Fatal("access token is empty")
		}

		if newRefreshToken == "" {
			t.Fatal("new refresh token is empty")
		}

		if newRefreshToken == oldToken {
			t.Fatal("refresh token was not rotated")
		}

		if revokedID != 10 {
			t.Fatalf("revokedID = %d, want 10", revokedID)
		}

		if created.UserID != 42 {
			t.Fatalf("created.UserID = %d, want 42", created.UserID)
		}

		if created.TokenHash != hashRefreshToken(newRefreshToken) {
			t.Fatal("new token hash is incorrect")
		}

		userID, role, err := NewJWTManager(
			"secret",
		).ParseToken(accessToken)
		if err != nil {
			t.Fatalf("ParseToken() error = %v", err)
		}

		if userID != 42 {
			t.Fatalf("userID = %d, want 42", userID)
		}

		if role != "admin" {
			t.Fatalf("role = %q, want admin", role)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			&mockRefreshRepository{},
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		_, _, err := service.Refresh(
			context.Background(),
			"",
		)

		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf(
				"Refresh() error = %v, want %v",
				err,
				ErrInvalidRefreshToken,
			)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				return RefreshToken{}, ErrRefreshTokenNotFound
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		_, _, err := service.Refresh(
			context.Background(),
			oldToken,
		)

		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf(
				"Refresh() error = %v, want %v",
				err,
				ErrInvalidRefreshToken,
			)
		}
	})

	t.Run("revoked", func(t *testing.T) {
		t.Parallel()

		revokedAt := time.Now()

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				return RefreshToken{
					ID:        1,
					UserID:    42,
					ExpiresAt: time.Now().Add(time.Hour),
					RevokedAt: &revokedAt,
				}, nil
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		_, _, err := service.Refresh(
			context.Background(),
			oldToken,
		)

		if !errors.Is(err, ErrRefreshTokenRevoked) {
			t.Fatalf(
				"Refresh() error = %v, want %v",
				err,
				ErrRefreshTokenRevoked,
			)
		}
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				return RefreshToken{
					ID:        1,
					UserID:    42,
					ExpiresAt: time.Now().Add(-time.Hour),
				}, nil
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		_, _, err := service.Refresh(
			context.Background(),
			oldToken,
		)

		if !errors.Is(err, ErrRefreshTokenExpired) {
			t.Fatalf(
				"Refresh() error = %v, want %v",
				err,
				ErrRefreshTokenExpired,
			)
		}
	})
}

func TestRefreshService_Revoke(t *testing.T) {
	t.Parallel()

	const refreshToken = "refresh-token"

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		var revokedID int64

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				return RefreshToken{
					ID:        55,
					UserID:    1,
					ExpiresAt: time.Now().Add(time.Hour),
				}, nil
			},

			revokeFunc: func(
				ctx context.Context,
				tx database.Tx,
				id int64,
			) error {
				revokedID = id
				return nil
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		err := service.Revoke(
			context.Background(),
			refreshToken,
		)
		if err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}

		if revokedID != 55 {
			t.Fatalf("revokedID = %d, want 55", revokedID)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		t.Parallel()

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			&mockRefreshRepository{},
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		err := service.Revoke(
			context.Background(),
			"",
		)

		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf(
				"Revoke() error = %v, want %v",
				err,
				ErrInvalidRefreshToken,
			)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				return RefreshToken{}, ErrRefreshTokenNotFound
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		err := service.Revoke(
			context.Background(),
			refreshToken,
		)

		if !errors.Is(err, ErrInvalidRefreshToken) {
			t.Fatalf(
				"Revoke() error = %v, want %v",
				err,
				ErrInvalidRefreshToken,
			)
		}
	})

	t.Run("already revoked is idempotent", func(t *testing.T) {
		t.Parallel()

		revokedAt := time.Now()

		revokeCalled := false

		repository := &mockRefreshRepository{
			getByHashFunc: func(
				ctx context.Context,
				tx database.Tx,
				tokenHash string,
			) (RefreshToken, error) {
				return RefreshToken{
					ID:        55,
					UserID:    1,
					RevokedAt: &revokedAt,
				}, nil
			},

			revokeFunc: func(
				ctx context.Context,
				tx database.Tx,
				id int64,
			) error {
				revokeCalled = true
				return nil
			},
		}

		service := NewRefreshService(
			&mockRefreshTransactionManager{},
			repository,
			&mockUserRoleProvider{},
			NewJWTManager("secret"),
		)

		err := service.Revoke(
			context.Background(),
			refreshToken,
		)
		if err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}

		if revokeCalled {
			t.Fatal("Revoke repository method was called for already revoked token")
		}
	})
}
