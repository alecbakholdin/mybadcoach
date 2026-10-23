# Backend implementation spec

Implements the settled backend API design from `.scratch/backend-api/` (map + resolved tickets).
All design decisions are closed; these tickets are execution.

## References

- API contract: `.scratch/backend-api/assets/proto/` (approved proto tree) — moves to repo-root
  `proto/` in ticket 01.
- Schema: `.scratch/backend-api/assets/0001-initial-schema.sql` (approved DDL) — moves to
  `server/internal/db/migrations/` in ticket 02.
- `docs/adr/0008-connect-rpc-proto-first.md` — transport decision.
- `docs/testing.md` — strict lint (ESLint, golangci-lint, buf), mock-minimized tests, three
  integration seams, dependency vetting (all deps below are pre-approved in the backend-api map's
  ticket 01).
- `docs/architecture.md`, `GLOSSARY.md` — domain model and contracts.

## Tickets

01 toolchain, codegen, Go lint gates → 02 migrations + schema + seed → 03 auth → 04 structure
CRUD → 05 gameplay + parity → 06 container + full-stack e2e + CI gates.

## Deferred (separate efforts)

- Client implementation: SvelteKit consuming the API, Playwright per-action e2e tests + screen
  snapshots (when screens exist), the frontend-only stub Connect server seam
  (`@connectrpc/connect-node`, per backend-api ticket 03).
- ESLint strict config + Playwright infra + devcontainer tooling (`docs/testing.md` implementation).
- Live sync (MatchActor) streaming, SSO providers, email verification/reset/invites, SearchPlayer
  (FTS5), slug renames + redirects, observability/metrics.
