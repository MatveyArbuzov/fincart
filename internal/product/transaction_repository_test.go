package product

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresTransactionRepository_GetByIDForUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		id int64

		setup func(mock sqlmock.Sqlmock)

		want    Product
		wantErr error
	}{
		{
			name: "success",

			id: 1,

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"name",
					"description",
					"price",
					"currency",
					"stock",
				}).AddRow(
					int64(1),
					"Product",
					"Description",
					int64(100),
					"EUR",
					10,
				)

				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						name,
						description,
						price,
						currency,
						stock
					FROM products
					WHERE id = $1
					FOR UPDATE
				`)).
					WithArgs(1).
					WillReturnRows(rows)
			},

			want: Product{
				ID:          1,
				Name:        "Product",
				Description: "Description",
				Price:       100,
				Currency:    "EUR",
				Stock:       10,
			},
		},

		{
			name: "not found",

			id: 999,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						name,
						description,
						price,
						currency,
						stock
					FROM products
					WHERE id = $1
					FOR UPDATE
				`)).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},

			wantErr: ErrProductNotFound,
		},

		{
			name: "query error",

			id: 1,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						name,
						description,
						price,
						currency,
						stock
					FROM products
					WHERE id = $1
					FOR UPDATE
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

			repository := NewPostgresTransactionRepository()

			tt.setup(mock)

			got, err := repository.GetByIDForUpdate(
				context.Background(),
				tx,
				tt.id,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetByIDForUpdate() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"GetByIDForUpdate() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

func TestPostgresTransactionRepository_DecreaseStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		id       int64
		quantity int

		setup func(mock sqlmock.Sqlmock)

		wantErr error
	}{
		{
			name:     "success",
			id:       1,
			quantity: 3,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET stock = stock - $1
					WHERE id = $2
				`)).
					WithArgs(3, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},

		{
			name:     "exec error",
			id:       1,
			quantity: 3,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET stock = stock - $1
					WHERE id = $2
				`)).
					WithArgs(3, 1).
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

			repository := NewPostgresTransactionRepository()

			tt.setup(mock)

			err = repository.DecreaseStock(
				context.Background(),
				tx,
				tt.id,
				tt.quantity,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"DecreaseStock() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

func TestPostgresTransactionRepository_IncreaseStock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		id       int64
		quantity int

		setup func(mock sqlmock.Sqlmock)

		wantErr error
	}{
		{
			name:     "success",
			id:       1,
			quantity: 3,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET stock = stock + $1
					WHERE id = $2
				`)).
					WithArgs(3, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},

		{
			name:     "exec error",
			id:       1,
			quantity: 3,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET stock = stock + $1
					WHERE id = $2
				`)).
					WithArgs(3, 1).
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

			repository := NewPostgresTransactionRepository()

			tt.setup(mock)

			err = repository.IncreaseStock(
				context.Background(),
				tx,
				tt.id,
				tt.quantity,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"IncreaseStock() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

func TestPostgresTransactionRepository_Create(t *testing.T) {
	t.Parallel()

	product := Product{
		Name:        "Product",
		Description: "Description",
		Price:       100,
		Currency:    "EUR",
		Stock:       10,
	}

	tests := []struct {
		name string

		setup func(mock sqlmock.Sqlmock)

		want    Product
		wantErr error
	}{
		{
			name: "success",

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"name",
					"description",
					"price",
					"currency",
					"stock",
				}).AddRow(
					int64(1),
					"Product",
					"Description",
					int64(100),
					"EUR",
					10,
				)

				mock.ExpectQuery(regexp.QuoteMeta(`
					INSERT INTO products (
						name,
						description,
						price,
						currency,
						stock
					)
					VALUES ($1, $2, $3, $4, $5)
					RETURNING
						id,
						name,
						description,
						price,
						currency,
						stock
				`)).
					WithArgs(
						"Product",
						"Description",
						int64(100),
						"EUR",
						10,
					).
					WillReturnRows(rows)
			},

			want: Product{
				ID:          1,
				Name:        "Product",
				Description: "Description",
				Price:       100,
				Currency:    "EUR",
				Stock:       10,
			},
		},

		{
			name: "query error",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta(`
					INSERT INTO products (
						name,
						description,
						price,
						currency,
						stock
					)
					VALUES ($1, $2, $3, $4, $5)
					RETURNING
						id,
						name,
						description,
						price,
						currency,
						stock
				`)).
					WithArgs(
						"Product",
						"Description",
						int64(100),
						"EUR",
						10,
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

			repository := NewPostgresTransactionRepository()

			tt.setup(mock)

			got, err := repository.Create(
				context.Background(),
				tx,
				product,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Create() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"Create() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

func TestPostgresTransactionRepository_Update(t *testing.T) {
	t.Parallel()

	product := Product{
		ID:          1,
		Name:        "Updated",
		Description: "Updated description",
		Price:       200,
		Currency:    "EUR",
		Stock:       20,
	}

	tests := []struct {
		name string

		setup func(mock sqlmock.Sqlmock)

		wantErr error
	}{
		{
			name: "success",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET
						name = $1,
						description = $2,
						price = $3,
						currency = $4,
						stock = $5
					WHERE id = $6
				`)).
					WithArgs(
						"Updated",
						"Updated description",
						int64(200),
						"EUR",
						20,
						int64(1),
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},

		{
			name: "not found",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET
						name = $1,
						description = $2,
						price = $3,
						currency = $4,
						stock = $5
					WHERE id = $6
				`)).
					WithArgs(
						"Updated",
						"Updated description",
						int64(200),
						"EUR",
						20,
						int64(1),
					).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},

			wantErr: ErrProductNotFound,
		},

		{
			name: "exec error",

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					UPDATE products
					SET
						name = $1,
						description = $2,
						price = $3,
						currency = $4,
						stock = $5
					WHERE id = $6
				`)).
					WithArgs(
						"Updated",
						"Updated description",
						int64(200),
						"EUR",
						20,
						int64(1),
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

			repository := NewPostgresTransactionRepository()

			tt.setup(mock)

			err = repository.Update(
				context.Background(),
				tx,
				product,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Update() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}

func TestPostgresTransactionRepository_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		id int64

		setup func(mock sqlmock.Sqlmock)

		wantErr error
	}{
		{
			name: "success",

			id: 1,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					DELETE FROM products
					WHERE id = $1
				`)).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},

		{
			name: "not found",

			id: 999,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					DELETE FROM products
					WHERE id = $1
				`)).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},

			wantErr: ErrProductNotFound,
		},

		{
			name: "exec error",

			id: 1,

			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(regexp.QuoteMeta(`
					DELETE FROM products
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

			repository := NewPostgresTransactionRepository()

			tt.setup(mock)

			err = repository.Delete(
				context.Background(),
				tx,
				tt.id,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Delete() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("sqlmock expectations: %v", err)
			}
		})
	}
}
