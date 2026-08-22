package order

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

func newMockTx(t *testing.T) (sqlmock.Sqlmock, *sql.Tx) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	mock.ExpectBegin()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin() error = %v", err)
	}

	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	return mock, tx
}

func TestPostgresRepository_Create(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(
		2026,
		time.August,
		22,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name    string
		order   Order
		rows    *sqlmock.Rows
		want    Order
		wantErr bool
	}{
		{
			name: "success",
			order: Order{
				UserID:      42,
				Status:      string(OrderStatusPending),
				TotalAmount: 1500,
				Currency:    "EUR",
			},
			rows: sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				int64(100),
				createdAt,
			),
			want: Order{
				ID:          100,
				UserID:      42,
				Status:      string(OrderStatusPending),
				TotalAmount: 1500,
				Currency:    "EUR",
				CreatedAt:   createdAt,
			},
		},
		{
			name: "scan error",
			order: Order{
				UserID:      42,
				Status:      string(OrderStatusPending),
				TotalAmount: 1500,
				Currency:    "EUR",
			},
			rows: sqlmock.NewRows([]string{
				"id",
				"created_at",
			}).AddRow(
				"invalid-id",
				createdAt,
			),
			want:    Order{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		INSERT INTO orders (
			user_id,
			status,
			total_amount,
			currency
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

			mock.ExpectQuery(regexp.QuoteMeta(query)).
				WithArgs(
					tt.order.UserID,
					tt.order.Status,
					tt.order.TotalAmount,
					tt.order.Currency,
				).
				WillReturnRows(tt.rows)

			repo := NewPostgresRepository()

			got, err := repo.Create(
				context.Background(),
				tx,
				tt.order,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Create() error = nil, want error")
				}
			} else if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Create() = %+v, want %+v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

func TestPostgresRepository_CreateItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		item    OrderItem
		rows    *sqlmock.Rows
		want    OrderItem
		wantErr bool
	}{
		{
			name: "success",
			item: OrderItem{
				OrderID:   100,
				ProductID: 10,
				Quantity:  3,
				UnitPrice: 500,
			},
			rows: sqlmock.NewRows([]string{"id"}).
				AddRow(int64(200)),
			want: OrderItem{
				ID:        200,
				OrderID:   100,
				ProductID: 10,
				Quantity:  3,
				UnitPrice: 500,
			},
		},
		{
			name: "scan error",
			item: OrderItem{
				OrderID:   100,
				ProductID: 10,
				Quantity:  3,
				UnitPrice: 500,
			},
			rows: sqlmock.NewRows([]string{"id"}).
				AddRow("invalid-id"),
			want:    OrderItem{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

			mock.ExpectQuery(regexp.QuoteMeta(query)).
				WithArgs(
					tt.item.OrderID,
					tt.item.ProductID,
					tt.item.Quantity,
					tt.item.UnitPrice,
				).
				WillReturnRows(tt.rows)

			repo := NewPostgresRepository()

			got, err := repo.CreateItem(
				context.Background(),
				tx,
				tt.item,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("CreateItem() error = nil, want error")
				}
			} else if err != nil {
				t.Fatalf("CreateItem() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("CreateItem() = %+v, want %+v", got, tt.want)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

func TestPostgresRepository_GetByIDForUpdate(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(
		2026,
		time.August,
		22,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name    string
		id      int64
		rows    *sqlmock.Rows
		dbErr   error
		want    Order
		wantErr error
	}{
		{
			name: "success",
			id:   100,
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				int64(100),
				int64(42),
				string(OrderStatusPending),
				int64(1500),
				"EUR",
				createdAt,
			),
			want: Order{
				ID:          100,
				UserID:      42,
				Status:      string(OrderStatusPending),
				TotalAmount: 1500,
				Currency:    "EUR",
				CreatedAt:   createdAt,
			},
		},
		{
			name: "not found",
			id:   999,
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}),
			want:    Order{},
			wantErr: ErrOrderNotFound,
		},
		{
			name:  "database error",
			id:    100,
			dbErr: errors.New("database unavailable"),
			want:  Order{},
		},
		{
			name: "scan error",
			id:   100,
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				"invalid-id",
				42,
				string(OrderStatusPending),
				1500,
				"EUR",
				createdAt,
			),
			want: Order{},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE id = $1
		FOR UPDATE
	`

			expect := mock.ExpectQuery(regexp.QuoteMeta(query)).
				WithArgs(tt.id)

			switch {
			case tt.dbErr != nil:
				expect.WillReturnError(tt.dbErr)
			default:
				expect.WillReturnRows(tt.rows)
			}

			repo := NewPostgresRepository()

			got, err := repo.GetByIDForUpdate(
				context.Background(),
				tx,
				tt.id,
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"GetByIDForUpdate() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}
			} else if tt.dbErr != nil {
				if err == nil {
					t.Fatal("GetByIDForUpdate() error = nil, want error")
				}
			} else if tt.name == "scan error" {
				if err == nil {
					t.Fatal("GetByIDForUpdate() error = nil, want error")
				}
			} else if err != nil {
				t.Fatalf("GetByIDForUpdate() error = %v", err)
			}

			if got != tt.want {
				t.Errorf(
					"GetByIDForUpdate() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

func TestPostgresRepository_GetItems(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		orderID  int64
		rows     *sqlmock.Rows
		queryErr error
		want     []OrderItem
		wantErr  bool
	}{
		{
			name:    "success",
			orderID: 100,
			rows: sqlmock.NewRows([]string{
				"id",
				"order_id",
				"product_id",
				"quantity",
				"unit_price",
			}).
				AddRow(1, 100, 10, 2, 500).
				AddRow(2, 100, 20, 1, 700),
			want: []OrderItem{
				{
					ID:        1,
					OrderID:   100,
					ProductID: 10,
					Quantity:  2,
					UnitPrice: 500,
				},
				{
					ID:        2,
					OrderID:   100,
					ProductID: 20,
					Quantity:  1,
					UnitPrice: 700,
				},
			},
		},
		{
			name:    "empty result",
			orderID: 100,
			rows: sqlmock.NewRows([]string{
				"id",
				"order_id",
				"product_id",
				"quantity",
				"unit_price",
			}),
			want: []OrderItem{},
		},
		{
			name:     "query error",
			orderID:  100,
			queryErr: errors.New("query failed"),
			wantErr:  true,
		},
		{
			name:    "scan error",
			orderID: 100,
			rows: sqlmock.NewRows([]string{
				"id",
				"order_id",
				"product_id",
				"quantity",
				"unit_price",
			}).AddRow(
				"invalid-id",
				100,
				10,
				2,
				500,
			),
			wantErr: true,
		},
		{
			name:    "rows error",
			orderID: 100,
			rows: sqlmock.NewRows([]string{
				"id",
				"order_id",
				"product_id",
				"quantity",
				"unit_price",
			}).
				AddRow(1, 100, 10, 2, 500).
				RowError(0, errors.New("row iteration failed")),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

			expect := mock.ExpectQuery(regexp.QuoteMeta(query)).
				WithArgs(tt.orderID)

			if tt.queryErr != nil {
				expect.WillReturnError(tt.queryErr)
			} else {
				expect.WillReturnRows(tt.rows)
			}

			repo := NewPostgresRepository()

			got, err := repo.GetItems(
				context.Background(),
				tx,
				tt.orderID,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("GetItems() error = nil, want error")
				}

				if got != nil {
					t.Errorf("GetItems() = %+v, want nil", got)
				}
			} else {
				if err != nil {
					t.Fatalf("GetItems() error = %v", err)
				}

				if len(got) != len(tt.want) {
					t.Fatalf(
						"GetItems() len = %d, want %d",
						len(got),
						len(tt.want),
					)
				}

				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf(
							"GetItems()[%d] = %+v, want %+v",
							i,
							got[i],
							tt.want[i],
						)
					}
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

func TestPostgresRepository_UpdateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		orderID int64
		status  string
		dbErr   error
	}{
		{
			name:    "success",
			orderID: 100,
			status:  string(OrderStatusPaid),
		},
		{
			name:    "database error",
			orderID: 100,
			status:  string(OrderStatusPaid),
			dbErr:   errors.New("update failed"),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`

			expect := mock.ExpectExec(regexp.QuoteMeta(query)).
				WithArgs(tt.status, tt.orderID)

			if tt.dbErr != nil {
				expect.WillReturnError(tt.dbErr)
			} else {
				expect.WillReturnResult(sqlmock.NewResult(0, 1))
			}

			repo := NewPostgresRepository()

			err := repo.UpdateStatus(
				context.Background(),
				tx,
				tt.orderID,
				tt.status,
			)

			if tt.dbErr != nil {
				if err == nil {
					t.Fatal("UpdateStatus() error = nil, want error")
				}
			} else if err != nil {
				t.Fatalf("UpdateStatus() error = %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

func TestPostgresRepository_GetByID(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(
		2026,
		time.August,
		22,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name    string
		id      int64
		rows    *sqlmock.Rows
		dbErr   error
		want    Order
		wantErr error
	}{
		{
			name: "success",
			id:   100,
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				100,
				42,
				string(OrderStatusPaid),
				1500,
				"EUR",
				createdAt,
			),
			want: Order{
				ID:          100,
				UserID:      42,
				Status:      string(OrderStatusPaid),
				TotalAmount: 1500,
				Currency:    "EUR",
				CreatedAt:   createdAt,
			},
		},
		{
			name: "not found",
			id:   999,
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}),
			want:    Order{},
			wantErr: ErrOrderNotFound,
		},
		{
			name:  "database error",
			id:    100,
			dbErr: errors.New("database unavailable"),
			want:  Order{},
		},
		{
			name: "scan error",
			id:   100,
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				"invalid-id",
				42,
				string(OrderStatusPaid),
				1500,
				"EUR",
				createdAt,
			),
			want: Order{},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		WHERE id = $1
	`

			expect := mock.ExpectQuery(regexp.QuoteMeta(query)).
				WithArgs(tt.id)

			if tt.dbErr != nil {
				expect.WillReturnError(tt.dbErr)
			} else {
				expect.WillReturnRows(tt.rows)
			}

			repo := NewPostgresRepository()

			got, err := repo.GetByID(
				context.Background(),
				tx,
				tt.id,
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"GetByID() error = %v, want %v",
						err,
						tt.wantErr,
					)
				}
			} else if tt.dbErr != nil {
				if err == nil {
					t.Fatal("GetByID() error = nil, want error")
				}
			} else if tt.name == "scan error" {
				if err == nil {
					t.Fatal("GetByID() error = nil, want error")
				}
			} else if err != nil {
				t.Fatalf("GetByID() error = %v", err)
			}

			if got != tt.want {
				t.Errorf(
					"GetByID() = %+v, want %+v",
					got,
					tt.want,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

func TestPostgresRepository_List(t *testing.T) {
	t.Parallel()

	createdAt1 := time.Date(
		2026,
		time.August,
		22,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	createdAt2 := time.Date(
		2026,
		time.August,
		21,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	tests := []struct {
		name     string
		rows     *sqlmock.Rows
		queryErr error
		want     []Order
		wantErr  bool
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
			}).
				AddRow(
					100,
					42,
					string(OrderStatusPaid),
					1500,
					"EUR",
					createdAt1,
				).
				AddRow(
					99,
					43,
					string(OrderStatusPending),
					2500,
					"USD",
					createdAt2,
				),
			want: []Order{
				{
					ID:          100,
					UserID:      42,
					Status:      string(OrderStatusPaid),
					TotalAmount: 1500,
					Currency:    "EUR",
					CreatedAt:   createdAt1,
				},
				{
					ID:          99,
					UserID:      43,
					Status:      string(OrderStatusPending),
					TotalAmount: 2500,
					Currency:    "USD",
					CreatedAt:   createdAt2,
				},
			},
		},
		{
			name: "empty result",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}),
			want: []Order{},
		},
		{
			name:     "query error",
			queryErr: errors.New("query failed"),
			wantErr:  true,
		},
		{
			name: "scan error",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).AddRow(
				"invalid-id",
				42,
				string(OrderStatusPaid),
				1500,
				"EUR",
				createdAt1,
			),
			wantErr: true,
		},
		{
			name: "rows error",
			rows: sqlmock.NewRows([]string{
				"id",
				"user_id",
				"status",
				"total_amount",
				"currency",
				"created_at",
			}).
				AddRow(
					100,
					42,
					string(OrderStatusPaid),
					1500,
					"EUR",
					createdAt1,
				).
				RowError(
					0,
					errors.New("row iteration failed"),
				),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock, tx := newMockTx(t)

			const query = `
		SELECT
			id,
			user_id,
			status,
			total_amount,
			currency,
			created_at
		FROM orders
		ORDER BY id DESC
	`

			expect := mock.ExpectQuery(regexp.QuoteMeta(query))

			if tt.queryErr != nil {
				expect.WillReturnError(tt.queryErr)
			} else {
				expect.WillReturnRows(tt.rows)
			}

			repo := NewPostgresRepository()

			got, err := repo.List(
				context.Background(),
				tx,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatal("List() error = nil, want error")
				}

				if got != nil {
					t.Errorf("List() = %+v, want nil", got)
				}
			} else {
				if err != nil {
					t.Fatalf("List() error = %v", err)
				}

				if len(got) != len(tt.want) {
					t.Fatalf(
						"List() len = %d, want %d",
						len(got),
						len(tt.want),
					)
				}

				for i := range tt.want {
					if got[i] != tt.want[i] {
						t.Errorf(
							"List()[%d] = %+v, want %+v",
							i,
							got[i],
							tt.want[i],
						)
					}
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("ExpectationsWereMet() error = %v", err)
			}
		})
	}
}

var _ Repository = (*PostgresRepository)(nil)

var _ database.Tx = (*sql.Tx)(nil)
