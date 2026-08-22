package database

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestManager_WithinTransaction(t *testing.T) {
	t.Parallel()

	errBegin := errors.New("begin failed")
	errCallback := errors.New("callback failed")
	errRollback := errors.New("rollback failed")
	errCommit := errors.New("commit failed")

	tests := []struct {
		name string

		setupDB func(t *testing.T, mock sqlmock.Sqlmock)

		callback func(tx Tx) error

		wantErr       error
		wantErrString string

		wantCallbackCalled bool
	}{
		{
			name: "begin error",
			setupDB: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin().
					WillReturnError(errBegin)
			},
			callback: func(tx Tx) error {
				t.Fatal("callback must not be called")
				return nil
			},
			wantErr:            errBegin,
			wantCallbackCalled: false,
		},
		{
			name: "successful transaction",
			setupDB: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			callback: func(tx Tx) error {
				return nil
			},
			wantErr:            nil,
			wantCallbackCalled: true,
		},
		{
			name: "callback error causes rollback",
			setupDB: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			callback: func(tx Tx) error {
				return errCallback
			},
			wantErr:            errCallback,
			wantCallbackCalled: true,
		},
		{
			name: "callback error and rollback error",
			setupDB: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback().
					WillReturnError(errRollback)
			},
			callback: func(tx Tx) error {
				return errCallback
			},
			wantErrString:      "rollback failed",
			wantCallbackCalled: true,
		},
		{
			name: "commit error",
			setupDB: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().
					WillReturnError(errCommit)
			},
			callback: func(tx Tx) error {
				return nil
			},
			wantErr:            errCommit,
			wantCallbackCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New() error = %v", err)
			}
			t.Cleanup(func() {
				_ = db.Close()
			})

			tt.setupDB(t, mock)

			manager := NewManager(db)

			callbackCalled := false

			err = manager.WithinTransaction(
				context.Background(),
				func(tx Tx) error {
					callbackCalled = true

					if tt.callback == nil {
						return nil
					}

					return tt.callback(tx)
				},
			)

			if tt.wantCallbackCalled != callbackCalled {
				t.Fatalf(
					"callbackCalled = %v, want %v",
					callbackCalled,
					tt.wantCallbackCalled,
				)
			}

			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf(
						"error = %v, want errors.Is(..., %v)",
						err,
						tt.wantErr,
					)
				}

			case tt.wantErrString != "":
				if err == nil {
					t.Fatalf(
						"error = nil, want error containing %q",
						tt.wantErrString,
					)
				}

				if !strings.Contains(err.Error(), tt.wantErrString) {
					t.Fatalf(
						"error = %q, want substring %q",
						err.Error(),
						tt.wantErrString,
					)
				}

			default:
				if err != nil {
					t.Fatalf(
						"error = %v, want nil",
						err,
					)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"mock expectations were not met: %v",
					err,
				)
			}
		})
	}
}
