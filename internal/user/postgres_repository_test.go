package user

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newSQLMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
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

func TestPostgresRepository_Create(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(
		2026,
		8,
		21,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name string

		email        string
		passwordHash string
		role         Role

		setup func(mock sqlmock.Sqlmock)

		want    User
		wantErr error
	}{
		{
			name: "success",

			email:        "test@example.com",
			passwordHash: "hashed-password",
			role:         RoleUser,

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"email",
					"role",
					"created_at",
				}).AddRow(
					int64(1),
					"test@example.com",
					"user",
					createdAt,
				)

				mock.ExpectQuery(regexp.QuoteMeta(`
					INSERT INTO users (
						email,
						password_hash,
						role
					)
					VALUES ($1, $2, $3)
					RETURNING id, email, role, created_at
				`)).
					WithArgs(
						"test@example.com",
						"hashed-password",
						RoleUser,
					).
					WillReturnRows(rows)
			},

			want: User{
				ID:        1,
				Email:     "test@example.com",
				Role:      RoleUser,
				CreatedAt: createdAt,
			},
		},

		{
			name: "query error",

			email:        "test@example.com",
			passwordHash: "hashed-password",
			role:         RoleUser,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					INSERT INTO users (
						email,
						password_hash,
						role
					)
					VALUES ($1, $2, $3)
					RETURNING id, email, role, created_at
				`)).
					WithArgs(
						"test@example.com",
						"hashed-password",
						RoleUser,
					).
					WillReturnError(sql.ErrConnDone)
			},

			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newSQLMockDB(t)

			mock.ExpectBegin()

			tx, err := db.Begin()
			if err != nil {
				t.Fatalf("db.Begin() error = %v", err)
			}

			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			repository := NewPostgresRepository()

			tt.setup(mock)

			got, err := repository.Create(
				context.Background(),
				tx,
				tt.email,
				tt.passwordHash,
				tt.role,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Create() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got.ID != tt.want.ID {
				t.Fatalf(
					"Create() ID = %d, want %d",
					got.ID,
					tt.want.ID,
				)
			}

			if got.Email != tt.want.Email {
				t.Fatalf(
					"Create() Email = %q, want %q",
					got.Email,
					tt.want.Email,
				)
			}

			if got.Role != tt.want.Role {
				t.Fatalf(
					"Create() Role = %q, want %q",
					got.Role,
					tt.want.Role,
				)
			}

			if !got.CreatedAt.Equal(tt.want.CreatedAt) {
				t.Fatalf(
					"Create() CreatedAt = %v, want %v",
					got.CreatedAt,
					tt.want.CreatedAt,
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

func TestPostgresRepository_GetByEmail(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(
		2026,
		8,
		21,
		10,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name string

		email string

		setup func(mock sqlmock.Sqlmock)

		wantUser         User
		wantPasswordHash string
		wantErr          error
	}{
		{
			name: "success",

			email: "test@example.com",

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"email",
					"password_hash",
					"role",
					"created_at",
				}).AddRow(
					int64(1),
					"test@example.com",
					"hashed-password",
					"user",
					createdAt,
				)

				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						email,
						password_hash,
						role,
						created_at
					FROM users
					WHERE email = $1
				`)).
					WithArgs("test@example.com").
					WillReturnRows(rows)
			},

			wantUser: User{
				ID:        1,
				Email:     "test@example.com",
				Role:      RoleUser,
				CreatedAt: createdAt,
			},

			wantPasswordHash: "hashed-password",
		},

		{
			name: "not found",

			email: "missing@example.com",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						email,
						password_hash,
						role,
						created_at
					FROM users
					WHERE email = $1
				`)).
					WithArgs("missing@example.com").
					WillReturnError(sql.ErrNoRows)
			},

			wantErr: ErrUserNotFound,
		},

		{
			name: "query error",

			email: "test@example.com",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						email,
						password_hash,
						role,
						created_at
					FROM users
					WHERE email = $1
				`)).
					WithArgs("test@example.com").
					WillReturnError(sql.ErrConnDone)
			},

			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newSQLMockDB(t)

			mock.ExpectBegin()

			tx, err := db.Begin()
			if err != nil {
				t.Fatalf("db.Begin() error = %v", err)
			}

			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			repository := NewPostgresRepository()

			tt.setup(mock)

			gotUser, gotPasswordHash, err := repository.GetByEmail(
				context.Background(),
				tx,
				tt.email,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetByEmail() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if gotUser.ID != tt.wantUser.ID {
				t.Fatalf(
					"GetByEmail() ID = %d, want %d",
					gotUser.ID,
					tt.wantUser.ID,
				)
			}

			if gotUser.Email != tt.wantUser.Email {
				t.Fatalf(
					"GetByEmail() Email = %q, want %q",
					gotUser.Email,
					tt.wantUser.Email,
				)
			}

			if gotUser.Role != tt.wantUser.Role {
				t.Fatalf(
					"GetByEmail() Role = %q, want %q",
					gotUser.Role,
					tt.wantUser.Role,
				)
			}

			if !gotUser.CreatedAt.Equal(tt.wantUser.CreatedAt) {
				t.Fatalf(
					"GetByEmail() CreatedAt = %v, want %v",
					gotUser.CreatedAt,
					tt.wantUser.CreatedAt,
				)
			}

			if gotPasswordHash != tt.wantPasswordHash {
				t.Fatalf(
					"GetByEmail() password hash = %q, want %q",
					gotPasswordHash,
					tt.wantPasswordHash,
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

func TestPostgresRepository_GetRoleByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		id int64

		setup func(mock sqlmock.Sqlmock)

		want    string
		wantErr error
	}{
		{
			name: "success",

			id: 1,

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"role",
				}).AddRow(
					"USER",
				)

				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT role
					FROM users
					WHERE id = $1
				`)).
					WithArgs(1).
					WillReturnRows(rows)
			},

			want: "USER",
		},

		{
			name: "not found",

			id: 999,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT role
					FROM users
					WHERE id = $1
				`)).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},

			wantErr: ErrUserNotFound,
		},

		{
			name: "query error",

			id: 1,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT role
					FROM users
					WHERE id = $1
				`)).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},

			wantErr: sql.ErrConnDone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newSQLMockDB(t)

			mock.ExpectBegin()

			tx, err := db.Begin()
			if err != nil {
				t.Fatalf("db.Begin() error = %v", err)
			}

			t.Cleanup(func() {
				_ = tx.Rollback()
			})

			repository := NewPostgresRepository()

			tt.setup(mock)

			got, err := repository.GetRoleByID(
				context.Background(),
				tx,
				tt.id,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetRoleByID() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"GetRoleByID() = %q, want %q",
					got,
					tt.want,
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
