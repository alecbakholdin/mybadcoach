# 04: structure CRUD services

Status: resolved

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

## Comments

Resolved on branch `ticket/04-structure-crud` (3099ac4, 212e463, d8a14ec), merged to integration.

Layout for ticket 05: services live in `server/internal/structure/` (`core.go` holds the
authorization + nested-slug resolution; `procedures.go` lists the mutating/authenticated
procedure paths wired into `api.NewHandler`). sqlc queries live in
`server/internal/db/queries/*.sql`, generated into `server/internal/db/*.sql.go`.

Left `connect.CodeUnimplemented` for ticket 05: `TeamService.GetTeamRecord` and
`MatchService.AppendMatchEvents`, `ListMatchEvents`, `GetMatchState`, `GetMatchStats`.

Judgment calls (Alec to confirm): AddMembership on an existing membership → `already_exists`;
owner-only ops resolve authorization before slug lookup (non-owner sees `permission_denied`
even for a nonexistent slug); roster ops accept only players of the addressing team (else
`not_found`); `offset`/`limit` clamped (offset ≥ 0, limit ≥ 1, default 50); dates stored as
provided with empty → NULL; `Player.names` empty = unchanged / non-empty = replace wholesale
(blank entries rejected).
