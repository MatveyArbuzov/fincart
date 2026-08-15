package database

import (
	"context"
	"database/sql"
	"fmt"
)

type Tx interface {
	Commit() error
	Rollback() error

	ExecContext(
		ctx context.Context,
		query string,
		args ...any,
	) (sql.Result, error)

	QueryContext(
		ctx context.Context,
		query string,
		args ...any,
	) (*sql.Rows, error)

	QueryRowContext(
		ctx context.Context,
		query string,
		args ...any,
	) *sql.Row
}

type Manager struct {
	db *sql.DB
}

func NewManager(db *sql.DB) *Manager {
	return &Manager{
		db: db,
	}
}

func (m *Manager) WithinTransaction(
	ctx context.Context,
	fn func(tx Tx) error,
) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("%w: rollback failed: %v", err, rollbackErr)
		}

		return err
	}

	return tx.Commit()
}
