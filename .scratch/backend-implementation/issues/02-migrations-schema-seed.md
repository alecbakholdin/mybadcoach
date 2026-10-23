# 02: migrations, schema, and dev seed

Status: resolved

Blocked by: 01

Land the approved schema as migration 0001 and the embedded runner.

- Move `.scratch/backend-api/assets/0001-initial-schema.sql` to
  `server/internal/db/migrations/0001_initial.sql` unchanged (STRICT tables, json_valid CHECKs,
  ON DELETE CASCADE, no storage tombstones).
- Implement the forward-only embedded runner (`docs/testing.md` + backend-api ticket 02): SQL
  files via `go:embed`, `schema_migrations` tracking, one transaction per migration, applied on
  boot before serving. DSN pragmas: `foreign_keys=ON`, `busy_timeout=5000`, `journal_mode=WAL`.
- Dev seed (env-gated, dev builds only, idempotent): admin `admin@mybadcoach.dev` / `password`,
  bcrypt-hashed at seed time, logging in through the normal AuthService RPCs.
- Tests: runner behavior against a real temp-file SQLite database (no mocks, per `docs/testing.md`),
  including re-run idempotency and failure mid-migration.

## Comments

Resolved by branch `ticket/02-migrations-schema-seed` (af0cf8a, 4c7c8b3, 339db45, 94f9976),
merged to integration. Conventions ticket 03 must match:

- Seed gate: env `MBC_DEV_SEED=1` (`db.SeedEnvVar`); admin `admin@mybadcoach.dev` / `password`
  (`db.DevAdminEmail`, `db.DevAdminPassword`, name "Dev Admin").
- Password identity: `auth_identities.provider="password"` (`db.PasswordProvider`),
  `provider_subject` = account email (lowercase), `auth_identities.email` = same email.
- Bcrypt cost 12; hash stored as TEXT (not []byte — STRICT TEXT rejects BLOB).
- Ids: `crypto/rand` 16 bytes → 32 lowercase hex chars (no uuid dependency).
- Migration version = filename stem; `applied_at` = UTC RFC3339Nano.
- API: `db.Open(dsn)` (applies embedded migrations + DSN pragmas), `db.Migrate(ctx, *sql.DB, fs.FS)`
  (injectable seam), `db.Seed(ctx, *sql.DB)`.
