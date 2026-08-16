package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if err := createMigrationsTable(ctx, db); err != nil {
		log.Fatalf("failed to create migrations table: %v", err)
	}

	if err := runMigrations(ctx, db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("migrations completed successfully")
}

func createMigrationsTable(
	ctx context.Context,
	db *sql.DB,
) error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`

	_, err := db.ExecContext(ctx, query)

	return err
}

func runMigrations(
	ctx context.Context,
	db *sql.DB,
) error {
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return err
	}

	sort.Strings(files)

	for _, file := range files {
		version, name, err := migrationInfo(file)
		if err != nil {
			return err
		}

		applied, err := isMigrationApplied(
			ctx,
			db,
			version,
		)
		if err != nil {
			return err
		}

		if applied {
			log.Printf(
				"migration %s already applied",
				name,
			)

			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf(
				"read migration %s: %w",
				name,
				err,
			)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(
			ctx,
			string(content),
		); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf(
				"execute migration %s: %w",
				name,
				err,
			)
		}

		if _, err := tx.ExecContext(
			ctx,
			`
			INSERT INTO schema_migrations (
				version,
				name
			)
			VALUES ($1, $2)
			`,
			version,
			name,
		); err != nil {
			_ = tx.Rollback()

			return fmt.Errorf(
				"record migration %s: %w",
				name,
				err,
			)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf(
				"commit migration %s: %w",
				name,
				err,
			)
		}

		log.Printf(
			"migration %s applied",
			name,
		)
	}

	return nil
}

func migrationInfo(
	file string,
) (int64, string, error) {
	name := filepath.Base(file)

	parts := strings.SplitN(
		name,
		"_",
		2,
	)

	if len(parts) != 2 {
		return 0, "", fmt.Errorf(
			"invalid migration filename: %s",
			name,
		)
	}

	version, err := strconv.ParseInt(
		parts[0],
		10,
		64,
	)
	if err != nil {
		return 0, "", fmt.Errorf(
			"invalid migration version: %s",
			name,
		)
	}

	return version, name, nil
}

func isMigrationApplied(
	ctx context.Context,
	db *sql.DB,
	version int64,
) (bool, error) {
	var exists bool

	err := db.QueryRowContext(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM schema_migrations
			WHERE version = $1
		)
		`,
		version,
	).Scan(&exists)

	return exists, err
}
