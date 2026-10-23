# Finalize the schema DDL

Type: task
Status: resolved

## Question

Commit the planned schema (`docs/architecture.md`) as actual DDL plus the auth additions:

- `users` — as planned, no slug (slugs are Team/Season/Event/Match/Player only, ADR-0006)
- `auth_identities` — provider, provider_subject, `user_id` FK, per-provider email
- `sessions` — token_hash, `user_id` FK, expires_at, created_at
- `schema_migrations` — applied-version tracking
- Dev seed: default admin user (default email + password), dev builds only

Deliverables: `schema.sql` (migration 0001) plus the embedded forward-only migration runner design
(`go:embed`, transaction-per-migration, applied on boot) and the seed plan. Review the DDL against
the ADRs: tombstones vs hard deletes per ADR-0005, slugs per ADR-0006, no stats tables per ADR-0004.

## Answer

Resolved 2026-10-22. DDL approved as drafted, with review amendments:

- Asset: `assets/0001-initial-schema.sql` — STRICT tables; TEXT ids (client-generated for
  `match_events` so offline replays are idempotent, server-generated elsewhere); tombstones per
  ADR-0005; slugs per ADR-0006; no stats tables per ADR-0004.
- `CHECK (json_valid(...))` on `players.names` and `match_events.payload` — engine-level JSON
  validation (SQLite ≥3.38 strictness; modernc bundles 3.45+).
- Dates: calendar/timestamp columns (`start_date`, `end_date`, `created_at`, `scheduled_at`) are
  TEXT ISO-8601; INTEGER epoch-ms only where the contract demands it (`effective_ts`,
  `sessions.expires_at`). Alec's preference for SQLite-native types applied wherever a fitting
  type exists.
- `users.name` and `auth_identities.email` are NOT NULL.
- Migration runner: `go:embed`-ed SQL, forward-only, transaction-per-migration, applied on boot
  before serving; DSN pragmas `foreign_keys=ON`, `busy_timeout=5000`, `journal_mode=WAL`.
- Dev seed (dev builds only, env-gated, idempotent): admin `admin@mybadcoach.dev` / `password`,
  bcrypt-hashed at seed time, logging in through the same AuthService RPCs.
