package product

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

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

func TestPostgresRepository_GetByID(t *testing.T) {
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
					"MacBook Pro",
					"Apple laptop",
					int64(150000),
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
				`)).
					WithArgs(1).
					WillReturnRows(rows)
			},

			want: Product{
				ID:          1,
				Name:        "MacBook Pro",
				Description: "Apple laptop",
				Price:       150000,
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
				`)).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},

			wantErr: sql.ErrConnDone,
		},

		{
			name: "scan error",

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
					"invalid-id",
					"Product",
					"Description",
					int64(100),
					"EUR",
					1,
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
				`)).
					WithArgs(1).
					WillReturnRows(rows)
			},

			wantErr: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newSQLMockDB(t)
			repository := NewPostgresRepository(db)

			tt.setup(mock)

			got, err := repository.GetByID(
				context.Background(),
				tt.id,
			)

			if tt.name == "scan error" {
				if err == nil {
					t.Fatal("GetByID() error = nil, want error")
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetByID() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if got != tt.want {
				t.Fatalf(
					"GetByID() = %+v, want %+v",
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

func TestPostgresRepository_List(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string

		setup func(mock sqlmock.Sqlmock)

		want    []Product
		wantErr bool
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
				}).
					AddRow(
						int64(1),
						"MacBook Pro",
						"Apple laptop",
						int64(150000),
						"EUR",
						10,
					).
					AddRow(
						int64(2),
						"Keyboard",
						"Mechanical keyboard",
						int64(12000),
						"EUR",
						50,
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
					ORDER BY id
				`)).
					WillReturnRows(rows)
			},

			want: []Product{
				{
					ID:          1,
					Name:        "MacBook Pro",
					Description: "Apple laptop",
					Price:       150000,
					Currency:    "EUR",
					Stock:       10,
				},
				{
					ID:          2,
					Name:        "Keyboard",
					Description: "Mechanical keyboard",
					Price:       12000,
					Currency:    "EUR",
					Stock:       50,
				},
			},
		},

		{
			name: "empty",

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"name",
					"description",
					"price",
					"currency",
					"stock",
				})

				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						name,
						description,
						price,
						currency,
						stock
					FROM products
					ORDER BY id
				`)).
					WillReturnRows(rows)
			},

			want: []Product(nil),
		},

		{
			name: "query error",

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
					ORDER BY id
				`)).
					WillReturnError(sql.ErrConnDone)
			},

			wantErr: true,
		},

		{
			name: "scan error",

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"name",
					"description",
					"price",
					"currency",
					"stock",
				}).AddRow(
					"invalid-id",
					"Product",
					"Description",
					int64(100),
					"EUR",
					1,
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
					ORDER BY id
				`)).
					WillReturnRows(rows)
			},

			wantErr: true,
		},

		{
			name: "rows error",

			setup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id",
					"name",
					"description",
					"price",
					"currency",
					"stock",
				}).
					AddRow(
						int64(1),
						"Product",
						"Description",
						int64(100),
						"EUR",
						1,
					).
					RowError(0, sql.ErrConnDone)

				mock.ExpectQuery(regexp.QuoteMeta(`
					SELECT
						id,
						name,
						description,
						price,
						currency,
						stock
					FROM products
					ORDER BY id
				`)).
					WillReturnRows(rows)
			},

			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock := newSQLMockDB(t)
			repository := NewPostgresRepository(db)

			tt.setup(mock)

			got, err := repository.List(context.Background())

			if tt.wantErr && err == nil {
				t.Fatal("List() error = nil, want error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("List() unexpected error = %v", err)
			}

			if !productsEqual(got, tt.want) {
				t.Fatalf(
					"List() = %+v, want %+v",
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

func productsEqual(a, b []Product) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
