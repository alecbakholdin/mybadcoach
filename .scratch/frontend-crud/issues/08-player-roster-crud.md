# Build player + roster management screens

Type: task
Status: resolved
Blocked by: 06

## Question

Build the player and roster-management screens:

- Players: create / list / edit / delete within a team (names/aliases, display name first) via the
  `players` service.
- Rosters: add/remove players on the season roster and the event roster (two independent levels, per
  ADR-0005), via the roster-entry sets on the `seasons` and `events` services.

No starting/position columns — the on-field lineup is derived from MatchEvents, never stored.

## Answer

Resolved 2026-10-24.

**Routes (nested-slug addressing; ids never appear in requests):**

- `/teams/[team_slug]/players` — team player list (`ListPlayers`) plus create (`CreatePlayer`),
  inline edit (`UpdatePlayer`), and owner-only confirm-then-`DeletePlayer` (hard). The create/edit
  form takes a display name and comma-separated aliases, builds `names` with the display name
  first, and sends the whole list on update (an empty list means "leave unchanged", per the proto
  comment). The slug is immutable and never edited. Linked from the team detail screen.
- `/teams/[team_slug]/seasons/[season_slug]` — the existing read-only season roster list gained
  add (`AddSeasonRosterEntry`, picked from the team's `ListPlayers`) and per-row remove
  (`RemoveSeasonRosterEntry`); `ListSeasonRoster` renders the registered players.
- `/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]` — new Event roster panel beside
  Matches: `ListEventRoster` with add/remove via `AddEventRosterEntry` /
  `RemoveEventRosterEntry`. `ListPlayers` joined the same route load for the picker.

The two roster levels are independent: a player can be on the season roster, the event roster,
both, or neither. No starting/position columns or lineup UI.

**Stub server:** `store.ts` gained `playersByTeamSlug` (team-scoped slug from `names[0]`,
`slugify` + numeric suffix collisions), `seasonRosterIds`, and `eventRosterIds` keyed by the
existing nested-slug keys. Player deletion is hard and cascades through both roster levels; team,
season, and event hard deletes also drop their rosters. `/__test/reset` clears all three.
`structure.ts` now exports `createPlayerImplementation` and fills the season/event roster RPCs;
`main.ts` registers `PlayerService` over the same cookie-session + membership guards and adds the
player and event-roster mutations to the CSRF set. `CopySeasonRoster` is deliberately still
`unimplemented` (out of this ticket's UI scope), as are the gameplay RPCs.

**Duplicate roster adds:** idempotent no-op that still returns the player, mirroring the server's
`INSERT OR IGNORE` and `Add*RosterEntry` handlers (not `already_exists`). Removing an absent entry
is likewise a no-op. `CreatePlayer`/`UpdatePlayer` reject empty or blank names with an
`invalid_argument` `names` field violation, and `UpdatePlayer` with an empty list leaves names
unchanged.

**Gating:** `DeletePlayer` is owner-only (owner-only button in the UI and a `requireOwner` check in
the stub). Player create/get/list/update and both roster levels are member-gated; roster controls
render for every member and are enforced server-side. The screens derive the caller's role from
`ListMyTeams` and skip RPCs when signed out, matching tickets 06–07.

**Tests:** new `e2e/tests/frontend/player-roster.spec.ts` covers create team → create player
(display name + aliases) → edit names → add to season roster → add to event roster → remove from
the event roster (season roster stays) → remove from the season roster → owner delete, plus a
seeded owner/member case proving a non-owner member can add/remove season and event roster entries,
cannot see the Delete control, and is rejected with `permission_denied` (403) when calling
`DeletePlayer` directly. No Vitest unit tests: no new pure client logic worth extracting.

**Verification:** `npm run lint` (golangci-lint, ESLint, buf), `npm run test` (49 shared + Go
suites), `npm run test:integration:frontend` (19 specs), and `npm run build:client` all pass.
