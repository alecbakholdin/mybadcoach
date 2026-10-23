# Backend API wayfinding map

**Complete** — all seven tickets resolved (2026-10-22). The spec is handed to
`.scratch/backend-implementation/` for implementation.

## Destination

A settled backend API design handed to implementation: the `.proto` service definitions for CRUD
(structure + gameplay) and authentication, the finalized SQLite schema (DDL + embedded migrations +
dev seed admin), and the settled auth/identity + data-access architecture — no open decisions left.

## Notes

- Domain: Team → Season → Event → Match, roster levels, event-sourced gameplay. Consult
  `GLOSSARY.md`, `docs/architecture.md`, and `docs/adr/` when working tickets. Grilling tickets are
  HITL — worked live with Alec, never answered for him.
- Settled in grilling (rounds 1–3): spec-first; CRUD for all structure entities + gameplay
  append/read; **Connect RPC transport** (protobuf binary over HTTP, browser-native, no proxy —
  Alec's call: mobile-web, binary encoding, no intermediate layer); schema of record = `.proto`
  generated into Go and TS via buf; **parity fixtures stay hand-written, human-readable JSON** under
  the existing canonical-JSON equality rules (Alec constraint); auth = email+password (bcrypt) +
  opaque cookie sessions, hand-rolled (hashed at rest, 30-day absolute expiry); hand-rolled embedded
  migrations; sqlc data access (SQLite engine is beta — accepted); `modernc.org/sqlite` (pure Go);
  open signup; owner-gated membership/destructive ops; offset/limit pagination; Connect status codes
  + stable machine-code error details; dev seed: default admin email/password (dev only).
- Standards: `docs/testing.md` (strict lint, mock-minimized, three integration seams, dependency
  vetting by Alec). Go 1.27. Tickets list proposed dependencies up front.

## Decisions so far

- [Approve the backend dependency set](issues/01-approve-dependency-set.md): all six core deps
  approved by Alec — x/crypto (bcrypt), modernc.org/sqlite, connect-go, connect-web, sqlc, buf;
  the test-boundary fake rides with ticket 03.
- [Finalize the schema DDL](issues/02-finalize-schema-ddl.md): DDL approved — STRICT tables, TEXT
  ids, json_valid CHECKs, TEXT ISO-8601 dates, embedded forward-only runner, dev seed admin.
- [Auth service RPC design](issues/04-auth-service-design.md): AuthService = Register/Login/
  Logout/Me, all unary; cookie session + custom-header CSRF; User = {id,email,name}; bcrypt cost
  12, 30-day absolute expiry; stable machine-code error details.
- [Structure CRUD RPC design](issues/05-structure-crud-design.md): five family services,
  nested-slug addressing, hard deletes everywhere with FK cascades (match_events.deleted stays —
  domain tombstone), owner-only team/season/player deletes, CopySeasonRoster, ListMyTeams.
- [Gameplay endpoints RPC design](issues/06-gameplay-endpoints-design.md): oneof MatchEvent
  payloads, batch append as the only write path, tombstone+append edits (no edit RPC), five
  reads, no streaming yet, snake_case→camelCase protojson parity.
- [Assemble the proto contract](issues/07-assemble-proto-contract.md): full proto tree approved —
  6 services, MatchEvent oneof, buf lint/breaking, ErrorDetail machine codes; parity harness maps
  fixture JSON into generated types.
- Ticket 03: fake the backend in frontend-only Playwright tests with a **stub Connect server** (`@connectrpc/connect-node` adapter + in-memory fake store), not MSW or transport doubles — it is the only network-boundary option covering unary + streaming + binary Protobuf + real CORS/HttpOnly cookies/SSR. [Findings](https://github.com/alecbakholdin/mybadcoach/blob/research/connect-test-boundary/docs/research/connect-test-boundary.md)

## Not yet specified

- *(nothing beyond the live tickets)*

## Out of scope

- Implementing the endpoints on the server — the next effort, ticketed from the resolved spec
  (ticket 07).
- Live sync (MatchActor) — future Connect bidirectional-stream effort.
- Social SSO providers (Google/Apple) — eventual; the identity model accommodates them via
  `auth_identities`.
- Email verification, password reset, member invites by email — deferred in grilling (need an
  email-delivery service dependency + infra).
- Slug renames + old-slug permanent redirects — "not yet" (Q15).
- Observability and metrics — backlogged; shape unknown.
- SQLite performance revisit — if it becomes a bottleneck.
- Match-point lifecycle (time cap, forfeit), full UI.
- SearchPlayer with fuzzy search — backlogged; SQLite FTS5 is the likely tool when it returns.
- Storage tombstones for entities — hard deletes chosen for now; a possible later migration.
