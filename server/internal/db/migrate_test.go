package db_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
)

// openTestDB opens a real temp-file SQLite database through the production
// entry point and cleans it up when the test ends.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})
	return sqlDB
}

func TestOpenAppliesEmbeddedMigrations(t *testing.T) {
	sqlDB := openTestDB(t)

	rows, err := sqlDB.Query(
		"SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name",
	)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Errorf("close rows: %v", closeErr)
		}
	}()

	var tables []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			t.Fatalf("scan table name: %v", scanErr)
		}
		tables = append(tables, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("iterate tables: %v", rowsErr)
	}

	want := []string{
		"auth_identities",
		"event_roster_entries",
		"events",
		"match_events",
		"matches",
		"password_credentials",
		"players",
		"schema_migrations",
		"season_roster_entries",
		"seasons",
		"sessions",
		"team_memberships",
		"teams",
		"users",
	}
	if !slices.Equal(tables, want) {
		t.Fatalf("tables = %v, want %v", tables, want)
	}

	var (
		version   string
		appliedAt string
	)
	err = sqlDB.QueryRow("SELECT version, applied_at FROM schema_migrations").Scan(
		&version,
		&appliedAt,
	)
	if err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if version != "0001_initial" {
		t.Fatalf("version = %q, want %q", version, "0001_initial")
	}
	if appliedAt == "" {
		t.Fatal("applied_at must not be empty")
	}
}

func TestOpenConfiguresDSNPragmas(t *testing.T) {
	sqlDB := openTestDB(t)

	var foreignKeys int
	if err := sqlDB.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	var busyTimeout int
	if err := sqlDB.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("PRAGMA busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}

	var journalMode string
	if err := sqlDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want %q", journalMode, "wal")
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "app.db")

	first, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if closeErr := first.Close(); closeErr != nil {
		t.Fatalf("close first Open: %v", closeErr)
	}

	second, err := db.Open(dsn)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer func() {
		if closeErr := second.Close(); closeErr != nil {
			t.Errorf("close second Open: %v", closeErr)
		}
	}()

	var count int
	if err := second.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("schema_migrations rows = %d, want 1", count)
	}
}

func TestMigrateRollsBackFailedMigration(t *testing.T) {
	sqlDB := openTestDB(t)

	failing := fstest.MapFS{
		"0002_broken.sql": &fstest.MapFile{
			Data: []byte(
				"CREATE TABLE broken_partial (id TEXT PRIMARY KEY) STRICT;\n" +
					"THIS IS NOT VALID SQL;\n",
			),
		},
		// A valid migration after the failure must not run.
		"0003_after_broken.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE after_broken (id TEXT PRIMARY KEY) STRICT;"),
		},
	}
	if err := db.Migrate(context.Background(), sqlDB, failing); err == nil {
		t.Fatal("Migrate with a broken migration returned nil, want an error")
	}

	var partialTables int
	err := sqlDB.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'broken_partial'",
	).Scan(&partialTables)
	if err != nil {
		t.Fatalf("check broken_partial: %v", err)
	}
	if partialTables != 0 {
		t.Fatalf("broken_partial tables = %d, want 0 (migration must roll back)", partialTables)
	}

	var recorded int
	err = sqlDB.QueryRow(
		"SELECT COUNT(*) FROM schema_migrations WHERE version = '0002_broken'",
	).Scan(&recorded)
	if err != nil {
		t.Fatalf("check recorded version: %v", err)
	}
	if recorded != 0 {
		t.Fatalf("recorded 0002_broken rows = %d, want 0", recorded)
	}

	var afterBroken int
	err = sqlDB.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'after_broken'",
	).Scan(&afterBroken)
	if err != nil {
		t.Fatalf("check after_broken: %v", err)
	}
	if afterBroken != 0 {
		t.Fatalf("after_broken tables = %d, want 0 (run must stop at the failure)", afterBroken)
	}

	var total int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&total); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if total != 1 {
		t.Fatalf("schema_migrations rows = %d, want only the initial migration", total)
	}
}

func TestMigrateAppliesMigrationsInLexicographicOrder(t *testing.T) {
	sqlDB := openTestDB(t)

	migrations := fstest.MapFS{
		"0002_create_ordered.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE ordered (id TEXT PRIMARY KEY) STRICT;"),
		},
		// This fails unless 0002 ran first.
		"0003_insert_ordered.sql": &fstest.MapFile{
			Data: []byte("INSERT INTO ordered (id) VALUES ('first');"),
		},
	}
	if err := db.Migrate(context.Background(), sqlDB, migrations); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var id string
	if err := sqlDB.QueryRow("SELECT id FROM ordered").Scan(&id); err != nil {
		t.Fatalf("read ordered: %v", err)
	}
	if id != "first" {
		t.Fatalf("ordered.id = %q, want %q", id, "first")
	}

	rows, err := sqlDB.Query("SELECT version FROM schema_migrations ORDER BY rowid")
	if err != nil {
		t.Fatalf("read versions: %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Errorf("close rows: %v", closeErr)
		}
	}()

	var versions []string
	for rows.Next() {
		var version string
		if scanErr := rows.Scan(&version); scanErr != nil {
			t.Fatalf("scan version: %v", scanErr)
		}
		versions = append(versions, version)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("iterate versions: %v", rowsErr)
	}

	want := []string{"0001_initial", "0002_create_ordered", "0003_insert_ordered"}
	if !slices.Equal(versions, want) {
		t.Fatalf("versions = %v, want %v", versions, want)
	}
}

// TestMigrateBootstrapsWithoutSchemaMigrations pins the fresh-database path:
// Migrate must not create schema_migrations itself, so a migration that begins
// with CREATE TABLE schema_migrations (as 0001 does) applies cleanly.
func TestMigrateBootstrapsWithoutSchemaMigrations(t *testing.T) {
	raw, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "raw.db"))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() {
		if closeErr := raw.Close(); closeErr != nil {
			t.Errorf("close raw database: %v", closeErr)
		}
	}()

	migrations := fstest.MapFS{
		"0001_initial.sql": &fstest.MapFile{
			Data: []byte(
				"CREATE TABLE schema_migrations (" +
					"version TEXT PRIMARY KEY, applied_at TEXT NOT NULL) STRICT;\n" +
					"CREATE TABLE boot (id TEXT PRIMARY KEY) STRICT;",
			),
		},
	}
	err = db.Migrate(context.Background(), raw, migrations)
	if err != nil {
		t.Fatalf("Migrate on a fresh database: %v", err)
	}

	var version string
	err = raw.QueryRow("SELECT version FROM schema_migrations").Scan(&version)
	if err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if version != "0001_initial" {
		t.Fatalf("version = %q, want %q", version, "0001_initial")
	}

	var bootTables int
	err = raw.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'boot'",
	).Scan(&bootTables)
	if err != nil {
		t.Fatalf("check boot table: %v", err)
	}
	if bootTables != 1 {
		t.Fatalf("boot tables = %d, want 1", bootTables)
	}
}
