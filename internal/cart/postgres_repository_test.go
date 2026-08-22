package cart

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/MatveyArbuzov/fincart/internal/database"
)

var (
	errExec = errors.New("exec error")
	errRows = errors.New("rows error")
)

func newMockTx(
	t *testing.T,
) (*sql.DB, sqlmock.Sqlmock, database.Tx) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf(
			"sqlmock.New() error = %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	mock.ExpectBegin()

	sqlTx, err := db.Begin()
	if err != nil {
		t.Fatalf(
			"db.Begin() error = %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = sqlTx.Rollback()
	})

	return db, mock, sqlTx
}

func TestPostgresRepository_GetDraft(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		queryError error
		rows       *sqlmock.Rows
		want       Cart
		wantErr    error
	}{
		{
			name: "success",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				10,
				1,
				"draft",
				500,
				"EUR",
				time.Now(),
			),
			want: Cart{
				ID:          10,
				UserID:      1,
				Status:      "draft",
				TotalAmount: 500,
				Currency:    "EUR",
			},
		},
		{
			name: "not found",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}),
			want: Cart{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, tx := newMockTx(t)

			query := regexp.QuoteMeta(`
        SELECT
            id,
            user_id,
            status,
            total_amount,
            currency,
            created_at
        FROM orders
        WHERE user_id = $1
          AND status = 'draft'
        ORDER BY id
        LIMIT 1
        FOR UPDATE
    `)

			mock.ExpectQuery(query).
				WithArgs(int64(1)).
				WillReturnRows(tt.rows)

			repository := NewPostgresRepository()

			got, err := repository.GetDraft(
				context.Background(),
				tx,
				1,
			)

			if tt.name == "not found" {
				if !errors.Is(err, nil) {
					t.Fatalf("error = %v", err)
				}
			} else if err != nil {
				t.Fatalf(
					"GetDraft() error = %v",
					err,
				)
			}

			if got.ID != tt.want.ID ||
				got.UserID != tt.want.UserID ||
				got.Status != tt.want.Status ||
				got.TotalAmount != tt.want.TotalAmount ||
				got.Currency != tt.want.Currency {
				t.Fatalf(
					"GetDraft() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"mock expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresRepository_CreateDraft(t *testing.T) {
	t.Parallel()

	_, mock, tx := newMockTx(t)

	createdAt := time.Now()

	mock.ExpectQuery(
		regexp.QuoteMeta(`
        INSERT INTO orders (
            user_id,
            status,
            total_amount,
            currency
        )
        VALUES ($1, 'draft', 0, '')
        RETURNING
            id,
            user_id,
            status,
            total_amount,
            currency,
            created_at
    `),
	).
		WithArgs(int64(1)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				10,
				1,
				"draft",
				0,
				"",
				createdAt,
			),
		)

	repository := NewPostgresRepository()

	got, err := repository.CreateDraft(
		context.Background(),
		tx,
		1,
	)

	if err != nil {
		t.Fatalf(
			"CreateDraft() error = %v",
			err,
		)
	}

	if got.ID != 10 ||
		got.UserID != 1 ||
		got.Status != "draft" ||
		got.TotalAmount != 0 ||
		got.Currency != "" {
		t.Fatalf(
			"CreateDraft() = %+v",
			got,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"mock expectations: %v",
			err,
		)
	}
}

func TestPostgresRepository_GetItem(t *testing.T) {
	t.Parallel()

	_, mock, tx := newMockTx(t)

	mock.ExpectQuery(
		regexp.QuoteMeta(`
        SELECT
            oi.id,
            oi.product_id,
            oi.quantity,
            oi.unit_price,
            p.name,
            p.description,
            p.currency,
            p.stock
        FROM order_items oi
        JOIN products p ON p.id = oi.product_id
        WHERE oi.order_id = $1
          AND oi.product_id = $2
    `),
	).
		WithArgs(int64(10), int64(20)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"product_id",
				"quantity",
				"unit_price",
				"name",
				"description",
				"currency",
				"stock",
			}).AddRow(
				1,
				20,
				3,
				100,
				"Phone",
				"Test phone",
				"EUR",
				10,
			),
		)

	repository := NewPostgresRepository()

	got, err := repository.GetItem(
		context.Background(),
		tx,
		10,
		20,
	)

	if err != nil {
		t.Fatalf(
			"GetItem() error = %v",
			err,
		)
	}

	if got.ID != 1 ||
		got.ProductID != 20 ||
		got.Quantity != 3 ||
		got.UnitPrice != 100 ||
		got.Name != "Phone" ||
		got.Currency != "EUR" ||
		got.Stock != 10 {
		t.Fatalf(
			"GetItem() = %+v",
			got,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"mock expectations: %v",
			err,
		)
	}
}

func TestPostgresRepository_AddItem(t *testing.T) {
	t.Parallel()

	_, mock, tx := newMockTx(t)

	mock.ExpectQuery(
		regexp.QuoteMeta(`
        INSERT INTO order_items (
            order_id,
            product_id,
            quantity,
            unit_price
        )
        VALUES ($1, $2, $3, $4)
        RETURNING id, product_id, quantity, unit_price
    `),
	).
		WithArgs(
			int64(10),
			int64(20),
			3,
			int64(100),
		).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"product_id",
				"quantity",
				"unit_price",
			}).AddRow(
				1,
				20,
				3,
				100,
			),
		)

	repository := NewPostgresRepository()

	got, err := repository.AddItem(
		context.Background(),
		tx,
		10,
		20,
		3,
		100,
	)

	if err != nil {
		t.Fatalf(
			"AddItem() error = %v",
			err,
		)
	}

	if got.ID != 1 ||
		got.ProductID != 20 ||
		got.Quantity != 3 ||
		got.UnitPrice != 100 {
		t.Fatalf(
			"AddItem() = %+v",
			got,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"mock expectations: %v",
			err,
		)
	}
}

func TestPostgresRepository_UpdateItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rows    int64
		execErr error
		rowsErr error
		wantErr error
	}{
		{
			name: "success",
			rows: 1,
		},
		{
			name:    "item not found",
			rows:    0,
			wantErr: sql.ErrNoRows,
		},
		{
			name:    "exec error",
			execErr: errExec,
			wantErr: errExec,
		},
		{
			name:    "rows affected error",
			rows:    1,
			rowsErr: errRows,
			wantErr: errRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, tx := newMockTx(t)

			result := sqlmock.NewResult(0, tt.rows)

			if tt.rowsErr != nil {
				result = sqlmock.NewErrorResult(tt.rowsErr)
			}

			expect := mock.ExpectExec(
				regexp.QuoteMeta(`
        UPDATE order_items
        SET quantity = $1
        WHERE order_id = $2
          AND product_id = $3
    `),
			).
				WithArgs(
					3,
					int64(10),
					int64(20),
				)

			if tt.execErr != nil {
				expect.WillReturnError(tt.execErr)
			} else {
				expect.WillReturnResult(result)
			}

			repository := NewPostgresRepository()

			err := repository.UpdateItem(
				context.Background(),
				tx,
				10,
				20,
				3,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"UpdateItem() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"mock expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresRepository_DeleteItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rows    int64
		execErr error
		rowsErr error
		wantErr error
	}{
		{
			name: "success",
			rows: 1,
		},
		{
			name:    "not found",
			rows:    0,
			wantErr: sql.ErrNoRows,
		},
		{
			name:    "exec error",
			execErr: errExec,
			wantErr: errExec,
		},
		{
			name:    "rows error",
			rows:    1,
			rowsErr: errRows,
			wantErr: errRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, tx := newMockTx(t)

			result := sqlmock.NewResult(0, tt.rows)

			if tt.rowsErr != nil {
				result = sqlmock.NewErrorResult(tt.rowsErr)
			}

			expect := mock.ExpectExec(
				regexp.QuoteMeta(`
        DELETE FROM order_items
        WHERE order_id = $1
          AND product_id = $2
    `),
			).
				WithArgs(
					int64(10),
					int64(20),
				)

			if tt.execErr != nil {
				expect.WillReturnError(tt.execErr)
			} else {
				expect.WillReturnResult(result)
			}

			repository := NewPostgresRepository()

			err := repository.DeleteItem(
				context.Background(),
				tx,
				10,
				20,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"DeleteItem() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"mock expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresRepository_GetItems(t *testing.T) {
	t.Parallel()

	_, mock, tx := newMockTx(t)

	mock.ExpectQuery(
		regexp.QuoteMeta(`
        SELECT
            oi.id,
            oi.product_id,
            oi.quantity,
            oi.unit_price,
            p.name,
            p.description,
            p.currency,
            p.stock
        FROM order_items oi
        JOIN products p ON p.id = oi.product_id
        WHERE oi.order_id = $1
        ORDER BY oi.id
    `),
	).
		WithArgs(int64(10)).
		WillReturnRows(
			sqlmock.NewRows([]string{
				"id",
				"product_id",
				"quantity",
				"unit_price",
				"name",
				"description",
				"currency",
				"stock",
			}).
				AddRow(
					1,
					20,
					2,
					100,
					"Phone",
					"Phone",
					"EUR",
					10,
				).
				AddRow(
					2,
					30,
					1,
					200,
					"Book",
					"Book",
					"EUR",
					5,
				),
		)

	repository := NewPostgresRepository()

	got, err := repository.GetItems(
		context.Background(),
		tx,
		10,
	)

	if err != nil {
		t.Fatalf(
			"GetItems() error = %v",
			err,
		)
	}

	if len(got) != 2 {
		t.Fatalf(
			"len(GetItems()) = %d, want 2",
			len(got),
		)
	}

	if got[0].ID != 1 ||
		got[0].ProductID != 20 ||
		got[1].ID != 2 ||
		got[1].ProductID != 30 {
		t.Fatalf(
			"unexpected items: %+v",
			got,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"mock expectations: %v",
			err,
		)
	}
}

func TestPostgresRepository_UpdateTotal(t *testing.T) {
	t.Parallel()

	_, mock, tx := newMockTx(t)

	mock.ExpectExec(
		regexp.QuoteMeta(`
        UPDATE orders
        SET
            total_amount = $1,
            currency = $2
        WHERE id = $3
          AND status = 'draft'
    `),
	).
		WithArgs(
			int64(500),
			"EUR",
			int64(10),
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	repository := NewPostgresRepository()

	err := repository.UpdateTotal(
		context.Background(),
		tx,
		10,
		500,
		"EUR",
	)

	if err != nil {
		t.Fatalf(
			"UpdateTotal() error = %v",
			err,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"mock expectations: %v",
			err,
		)
	}
}

func TestPostgresRepository_Checkout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rows    int64
		execErr error
		rowsErr error
		wantErr error
	}{
		{
			name: "success",
			rows: 1,
		},
		{
			name:    "not found",
			rows:    0,
			wantErr: sql.ErrNoRows,
		},
		{
			name:    "exec error",
			execErr: errExec,
			wantErr: errExec,
		},
		{
			name:    "rows error",
			rows:    1,
			rowsErr: errRows,
			wantErr: errRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, tx := newMockTx(t)

			result := sqlmock.NewResult(0, tt.rows)

			if tt.rowsErr != nil {
				result = sqlmock.NewErrorResult(tt.rowsErr)
			}

			expect := mock.ExpectExec(
				regexp.QuoteMeta(`
        UPDATE orders
        SET status = 'pending'
        WHERE id = $1
          AND status = 'draft'
    `),
			).
				WithArgs(int64(10))

			if tt.execErr != nil {
				expect.WillReturnError(tt.execErr)
			} else {
				expect.WillReturnResult(result)
			}

			repository := NewPostgresRepository()

			err := repository.Checkout(
				context.Background(),
				tx,
				10,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"Checkout() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"mock expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresRepository_GetDraftByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		wantID  int64
		wantErr error
	}{
		{
			name: "success",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				10,
				1,
				"draft",
				500,
				"EUR",
				time.Now(),
			),
			wantID: 10,
		},
		{
			name: "not found",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}),
			wantErr: ErrCartNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mock, tx := newMockTx(t)

			mock.ExpectQuery(
				regexp.QuoteMeta(`
        SELECT
            id,
            user_id,
            status,
            total_amount,
            currency,
            created_at
        FROM orders
        WHERE id = $1
          AND status = 'draft'
        FOR UPDATE
    `),
			).
				WithArgs(int64(10)).
				WillReturnRows(tt.rows)

			repository := NewPostgresRepository()

			got, err := repository.GetDraftByID(
				context.Background(),
				tx,
				10,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"GetDraftByID() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if tt.wantErr == nil && got.ID != tt.wantID {
				t.Fatalf(
					"ID = %d, want %d",
					got.ID,
					tt.wantID,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"mock expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresRepository_UpdateItemPrice(t *testing.T) {
	t.Parallel()

	_, mock, tx := newMockTx(t)

	mock.ExpectExec(
		regexp.QuoteMeta(`
        UPDATE order_items
        SET unit_price = $1
        WHERE order_id = $2
          AND product_id = $3
    `),
	).
		WithArgs(
			int64(150),
			int64(10),
			int64(20),
		).
		WillReturnResult(
			sqlmock.NewResult(0, 1),
		)

	repository := NewPostgresRepository()

	err := repository.UpdateItemPrice(
		context.Background(),
		tx,
		10,
		20,
		150,
	)

	if err != nil {
		t.Fatalf(
			"UpdateItemPrice() error = %v",
			err,
		)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf(
			"mock expectations: %v",
			err,
		)
	}
}
