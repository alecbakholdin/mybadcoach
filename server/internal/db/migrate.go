// Package db owns the SQLite database, its embedded schema migrations, and the
// development seed.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	// The blank import registers the "sqlite" driver with database/sql.
	_ "modernc.org/sqlite"
)

// migrationFiles embeds the production migration set. Open applies them
// through Migrate.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// Open opens the SQLite database at dsn with the required connection pragmas
// (foreign_keys=ON, busy_timeout=5000, journal_mode=WAL) and applies every
// embedded migration before returning. A returned *sql.DB is ready to use; a
// migration failure closes the database and is reported to the caller.
func Open(dsn string) (*sql.DB, error) {
	sqlDB, err := sql.Open("sqlite", dsnWithPragmas(dsn))
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	migrations, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("load embedded migrations: %w", err)
	}
	if err := Migrate(context.Background(), sqlDB, migrations); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return sqlDB, nil
}

// Migrate applies every migration in fsys that has not been recorded in
// schema_migrations, in lexicographic filename order. fsys must be rooted at
// the directory holding the .sql files; the version recorded for a migration
// is its filename without the .sql suffix.
//
// Each migration and its bookkeeping row run in one transaction: if any
// statement fails, that migration's changes are rolled back, its version is not
// recorded, and Migrate stops without applying later migrations.
//
// A fresh database has no schema_migrations table. Rather than creating the
// table up front (which would collide with the CREATE TABLE in migration
// 0001), Migrate detects the missing table and treats the database as having no
// applied migrations.
func Migrate(ctx context.Context, sqlDB *sql.DB, fsys fs.FS) error {
	applied, err := appliedVersions(ctx, sqlDB)
	if err != nil {
		return err
	}

	names, err := migrationNames(fsys)
	if err != nil {
		return err
	}

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		if err := applyMigration(ctx, sqlDB, fsys, name, version); err != nil {
			return err
		}
	}
	return nil
}

// dsnWithPragmas appends the required connection pragmas to dsn. Modernc's
// driver applies _pragma query parameters to every connection it opens.
func dsnWithPragmas(dsn string) string {
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator +
		"_pragma=foreign_keys=ON&_pragma=busy_timeout=5000&_pragma=journal_mode=WAL"
}

// migrationNames lists the .sql files in fsys in lexicographic order.
func migrationNames(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

// appliedVersions reads the versions recorded in schema_migrations. When the
// table does not exist yet, it returns an empty set: migration 0001 creates it.
func appliedVersions(ctx context.Context, sqlDB *sql.DB) (map[string]bool, error) {
	var tableCount int
	err := sqlDB.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'schema_migrations'",
	).Scan(&tableCount)
	if err != nil {
		return nil, fmt.Errorf("check schema_migrations: %w", err)
	}

	applied := make(map[string]bool)
	if tableCount == 0 {
		return applied, nil
	}

	rows, err := sqlDB.QueryContext(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var version string
		if scanErr := rows.Scan(&version); scanErr != nil {
			return nil, fmt.Errorf("scan applied migration: %w", scanErr)
		}
		applied[version] = true
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", rowsErr)
	}
	return applied, nil
}

// applyMigration runs one migration file and records its version in a single
// transaction.
func applyMigration(ctx context.Context, sqlDB *sql.DB, fsys fs.FS, name, version string) error {
	contents, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, string(contents)); err != nil {
		return fmt.Errorf("apply migration %s: %w", version, err)
	}
	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
		version,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}
