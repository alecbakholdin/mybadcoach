# 02: migrations, schema, and dev seed

Status: ready-for-agent

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
