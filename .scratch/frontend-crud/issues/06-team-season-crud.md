# Build team + season CRUD screens

Type: task
Status: resolved
Blocked by: 05

## Question

Build the team and season CRUD screens:

- Teams: list-my-teams, create, rename, delete (owner-gated) via the `teams` service.
- Seasons: create / list / edit / delete within a team via the `seasons` service, including the season
  roster screen.

Follow the structure CRUD RPC contract in `.scratch/backend-api/issues/05` (nested-slug addressing,
owner-only destructive ops).

## Answer

Resolved 2026-10-24.

**Routes (nested-slug addressing; ids never appear in requests):**

- `/teams` — `ListMyTeams` list with each team's name and an Owner/Member badge, plus a
  `CreateTeam({ name })` form. Creating navigates to `/teams/<team_slug>`.
- `/teams/[team_slug]` — team detail: name, `ListSeasons` (offset/limit) with links, a
  `CreateSeason({ team_slug, name, start_date?, end_date? })` form, and — **owner only** — a
  `UpdateTeam` rename form and a confirm-then-`DeleteTeam` action (hard delete, returns to `/teams`).
- `/teams/[team_slug]/seasons/[season_slug]` — season detail: `UpdateSeason` (name/dates; any
  member), **owner-only** confirm-then-`DeleteSeason` (hard delete, returns to the team), and a
  read-only `ListSeasonRoster` panel showing the count/list. Players remain empty until ticket 08;
  no add/remove here.

**Owner gating:** each route load calls `ListMyTeams` and carries the caller's `role`; a team the
caller is not a member of renders a "not found" state (no second permission-denied request). The
rename/delete-team controls render only when `role === 'owner'`, and delete-season only on the
season screen when the role is owner. The stub enforces the same rules server-side, so the gating is
testable at the seam as well as merely hidden in the UI. Members still create/edit seasons.

**Stub server:** `apps/client/test-server/store.ts` gained in-memory teams, memberships, and
seasons; slugs are generated from names (`slugify` + numeric-suffix collisions) and are stable across
renames — team slugs globally unique, season slugs unique within a team (ADR-0006). `structure.ts`
implements `TeamService` (Create/Get/Update/Delete/ListMyTeams, plus `AddMembership` so tests can
seed a non-owner; no membership screen) and `SeasonService` (Create/Get/Update/List/Delete/
ListSeasonRoster returning an empty list). `/__test/reset` now also clears structure data, and the
structure mutating RPCs joined the CSRF-header set.

**Tests:** `e2e/tests/frontend/structure.spec.ts` covers create→list→rename (stable slug), duplicate
name slug suffixes, season create/edit, roster rendering, owner delete-season/delete-team, and
non-owner destructive controls hidden with member edits still allowed. Ticket 05's `/teams`
placeholder assertions in `auth.spec.ts` were updated for the real screen. No Vitest unit tests were
added: the client gained no pure logic (slugs are server-side only per ADR-0006), and the client has
no Vitest workspace.

**Decision for later tickets:** the route load functions wait on the memoized `session.load()` and
skip their RPC when signed out. Page loads run in parallel with the root layout guard, so without
this the first RPC would race the guard and turn a clean `/login` redirect into
`/login?notice=session-expired`. Tickets 07–08 should follow the same pattern.

**Verification:** `npm run lint`, `npm run test`, `npm run test:integration:frontend` (15 specs),
`npm run build:client` all pass.
