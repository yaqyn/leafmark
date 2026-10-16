package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Migrations are additive and tracked atomically; the original course data is preserved.
//
//go:generate sqlc generate
func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS leafmark_migrations (name TEXT PRIMARY KEY)`); err != nil {
		return err
	}
	entries, err := schemaFiles.ReadDir("sql/schema")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		var count int
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM leafmark_migrations WHERE name = ?`, name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		// Recognize migrations already applied by the course's Goose workflow.
		var exists int
		table := "users"
		if strings.HasPrefix(name, "002") {
			table = "notes"
		}
		if strings.HasPrefix(name, "003") {
			table = "accounts"
		}
		if err := db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&exists); err != nil {
			return err
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if exists == 0 {
			data, readErr := schemaFiles.ReadFile("sql/schema/" + name)
			if readErr != nil {
				_ = tx.Rollback()
				return readErr
			}
			up := strings.Split(strings.TrimPrefix(string(data), "-- +goose Up"), "-- +goose Down")[0]
			for _, statement := range strings.Split(up, ";") {
				if strings.TrimSpace(statement) == "" {
					continue
				}
				if _, err = tx.ExecContext(ctx, statement); err != nil {
					_ = tx.Rollback()
					return fmt.Errorf("migration %s: %w", name, err)
				}
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO leafmark_migrations(name) VALUES (?)`, name); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func timestamp() string { return time.Now().UTC().Format(time.RFC3339Nano) }
