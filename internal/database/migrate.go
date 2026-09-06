package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

const migrationLockID int64 = 362627341046204981

type transactionStarter interface {
	Begin(context.Context) (pgx.Tx, error)
}

// Migrate applies embedded, forward-only schema migrations in one transaction.
// The baseline is idempotent so an existing EF-created database can adopt the
// Go migration history without recreating or deleting data.
func Migrate(ctx context.Context, database transactionStarter) error {
	tx, err := database.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return fmt.Errorf("lock schema migration: %w", err)
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS "FlareSchemaMigrations" (
        "Version" bigint PRIMARY KEY,
        "Name" text NOT NULL,
        "AppliedAt" timestamptz NOT NULL DEFAULT now()
    )`); err != nil {
		return fmt.Errorf("create schema migration history: %w", err)
	}

	names, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list schema migrations: %w", err)
	}
	sort.Strings(names)
	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		var applied bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "FlareSchemaMigrations" WHERE "Version"=$1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("read schema migration %d: %w", version, err)
		}
		if applied {
			continue
		}
		script, err := migrationFiles.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read schema migration %d: %w", version, err)
		}
		if _, err = tx.Exec(ctx, string(script)); err != nil {
			return fmt.Errorf("apply schema migration %d: %w", version, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO "FlareSchemaMigrations" ("Version", "Name") VALUES ($1,$2)`, version, filepath.Base(name)); err != nil {
			return fmt.Errorf("record schema migration %d: %w", version, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit schema migrations: %w", err)
	}
	return nil
}

func migrationVersion(name string) (int64, error) {
	prefix, _, found := strings.Cut(filepath.Base(name), "_")
	if !found {
		return 0, fmt.Errorf("invalid schema migration name %q", name)
	}
	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil || version < 1 {
		return 0, fmt.Errorf("invalid schema migration name %q", name)
	}
	return version, nil
}
