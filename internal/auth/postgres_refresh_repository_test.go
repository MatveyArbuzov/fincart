package auth

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newAuthSQLMockDB(
	t *testing.T,
) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, mock
}

func TestPostgresRefreshTokenRepository_Create(t *testing.T) {
	t.Parallel()

	db, mock := newAuthSQLMockDB(t)

	mock.ExpectBegin()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin() error = %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	repository := NewPostgresRefreshTokenRepository()

	expiresAt := time.Now().Add(time.Hour)

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`)).
		WithArgs(
			int64(42),
			"hash",
			expiresAt,
		).
		WillReturnResult(
			sqlmock.NewResult(1, 1),
		)

	err = repository.Create(
		context.Background(),
		tx,
		RefreshToken{
			UserID:    42,
			TokenHash: "hash",
			ExpiresAt: expiresAt,
		},
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations: %v", err)
	}
}

func TestPostgresRefreshTokenRepository_GetByHash(
	t *testing.T,
) {
	t.Parallel()

	createdAt := time.Now()
	expiresAt := createdAt.Add(time.Hour)

	tests := []struct {
		name    string
		setup   func(sqlmock.Sqlmock)
		want    RefreshToken
		wantErr error
	}{
		{
			name: "success",

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"user_id",
					"token_hash",
					"expires_at",
					"created_at",
					"revoked_at",
				}).AddRow(
					int64(1),
					int64(42),
					"hash",
					expiresAt,
					createdAt,
					nil,
				)

				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						user_id,
						token_hash,
						expires_at,
						created_at,
						revoked_at
					FROM refresh_tokens
					WHERE token_hash = $1
				`)).
					WithArgs("hash").
					WillReturnRows(rows)
			},

			want: RefreshToken{
				ID:        1,
				UserID:    42,
				TokenHash: "hash",
				ExpiresAt: expiresAt,
				CreatedAt: createdAt,
			},
		},

		{
			name: "not found",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						user_id,
						token_hash,
						expires_at,
						created_at,
						revoked_at
					FROM refresh_tokens
					WHERE token_hash = $1
				`)).
					WithArgs("missing").
					WillReturnError(sql.ErrNoRows)
			},

			wantErr: ErrRefreshTokenNotFound,
		},

		{
			name: "query error",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						user_id,
						token_hash,
						expires_at,
						created_at,
						revoked_at
					FROM refresh_tokens
					WHERE token_hash = $1
				`)).
					WithArgs("missing").
					WillReturnError(sql.ErrConnDone)
			},

			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newAuthSQLMockDB(t)

			mock.ExpectBegin()

			tx, err := db.Begin()
			if err != nil {
				t.Fatalf("db.Begin() error = %v", err)
			}

			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			tt.setup(mock)

			repository := NewPostgresRefreshTokenRepository()

			hash := "hash"
			if tt.name != "success" {
				hash = "missing"
			}

			got, err := repository.GetByHash(
				context.Background(),
				tx,
				hash,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetByHash() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got.ID != tt.want.ID {
				t.Fatalf(
					"GetByHash() ID = %d, want %d",
					got.ID,
					tt.want.ID,
				)
			}

			if got.UserID != tt.want.UserID {
				t.Fatalf(
					"GetByHash() UserID = %d, want %d",
					got.UserID,
					tt.want.UserID,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"sqlmock expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresRefreshTokenRepository_Revoke(t *testing.T) {
	t.Parallel()

	db, mock := newAuthSQLMockDB(t)

	mock.ExpectBegin()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin() error = %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	repository := NewPostgresRefreshTokenRepository()

	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
	`)).
		WithArgs(int64(42)).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	err = repository.Revoke(
		context.Background(),
		tx,
		42,
	)
	if err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"sqlmock expectations: %v",
			err,
		)
	}
}
