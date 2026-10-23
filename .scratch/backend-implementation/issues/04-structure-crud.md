# 04: structure CRUD services

Status: ready-for-agent

Blocked by: 02, 03

Implement Team/Season/Event/Player/Match services per the proto tree (backend-api ticket 05).

- Nested-slug addressing everywhere; ids never in requests. Slug generation reuses
  `server/internal/slugify` with scoped-uniqueness suffixes (ADR-0006).
- Partial updates via proto3 optional presence; `Player.names` replaces wholesale; slugs immutable.
- Lists: `offset`/`limit` → `{ items, total }`; `ListMyTeams` returns `{ team, role }` pairs.
- Hard deletes with FK cascades. Gating: membership grants reads/mutations; owner-only for
  membership management, team rename, `DeleteTeam`, `DeleteSeason`, `DeletePlayer`; member for
  `DeleteEvent`, `DeleteMatch`, roster add/remove. Last-owner guard errors `last_owner`.
- `CopySeasonRoster` skips already-registered players.
- Data access via sqlc; no hand-written SQL outside the migrations.
- Tests: backend integration per resource — CRUD, cascade deletes, gating (member vs owner vs
  anonymous), last-owner guard, slug stability across renames.
