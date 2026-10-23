# Architecture

My Bad, Coach is a real-time, collaborative, offline-first Ultimate Frisbee stat tracker. This
document records the architecture that the settled decisions in `docs/adr/` imply, and the contracts
that the TypeScript client and the Go server must both uphold. Read it alongside `GLOSSARY.md` for
domain vocabulary and `fixtures/README.md` for the normative parity contract.

## Event sourcing and cross-language parity

Gameplay is event-sourced (ADR-0001). Every stat-keeper tap emits an atomic `MatchEvent`; gameplay is
never stored as mutable state. A pure reducer folds the time-sorted event log for one point:

```ts
reducePoint(events: MatchEvent[], initialRole: StartingRole): PointDerivedState
```

The reducer is implemented twice — TypeScript for the client and Go for the server. The TypeScript
port lives in `packages/shared/src/` (the offline-first client recomputes state locally from the
same log); the Go port lives in `server/internal/reduce/` (the server folds the log for reads and
future synchronization). Because the client and server must agree exactly, the two
ports are held to one parity contract rather than allowed to drift independently: a shared,
hand-authored fixture suite is executed unchanged by Vitest (`packages/shared/src/parity.test.ts`) and
by `go test` (`server/internal/reduce/parity_test.go`). Both runners resolve to the repo-root
`fixtures/` directory. Parity fixtures are the only guarantee that the ports cannot silently diverge;
a behaviour change lands with a fixture that pins it in both languages.

Before reducing, both implementations run the same `prepareEvents` pipeline:

1. **Filter** — drop every event with `"deleted": true`.
2. **Sort** — ascending by `effectiveTs` (integer epoch milliseconds). Equal `effectiveTs` values are
   ordered by ascending `id` (code-unit/byte comparison). The order events appear in the fixture's
   `events` array is therefore irrelevant.
3. **Dedupe** — collapse equal `id`s, keeping the first occurrence in the sorted order. Replays are
   idempotent by id, so an event applied twice counts once.

The client is offline-first: it folds its local event log in memory during live scoring, and the
server can recompute the same state from the same events. Statistics are likewise derived, never
stored (ADR-0004); `match_events` is the single source of truth.

## Canonical PointDerivedState

`PointDerivedState` is the complete derived state for one point. Its shape is identical in
`packages/shared/src/state.ts` and `server/internal/reduce/state.go`, and it is the `expected` value of
every parity fixture:

```jsonc
{
  "view": "LINEUP_SETUP",       // LINEUP_SETUP | PULL_WAIT | DEFENSE_ACTIVE | OFFENSE_POSSESSION | SCORED
  "initialRole": "OFFENSE",     // echo of the point's starting role
  "lineup": ["player-a"],       // player ids on the field, in field order
  "thrower": "",                // player id in possession; "" means nobody
  "playerStats": {
    "player-a": {
      "touches": 0,
      "completions": 0,
      "drops": 0,
      "turnovers": 0,
      "blocks": 0,
      "goals": 0,
      "assists": 0
    }
  }
}
```

The five state fields:

| Field         | Type                                                                                         | Meaning                                                    |
| ------------- | -------------------------------------------------------------------------------------------- | ---------------------------------------------------------- |
| `view`        | `LINEUP_SETUP`, `PULL_WAIT`, `DEFENSE_ACTIVE`, `OFFENSE_POSSESSION`, or `SCORED`              | The current phase of the point.                            |
| `initialRole` | `OFFENSE` or `DEFENSE`                                                                        | Whether we started the point on offense or defense; echo.  |
| `lineup`      | `string[]`                                                                                    | Up-to-seven player ids on the field, in field order.       |
| `thrower`     | `string`                                                                                      | Player id currently in possession; `""` when absent.       |
| `playerStats` | `Record<string, PlayerStats>`                                                                 | Zero-filled seven base stats keyed by player id.           |

`PlayerStats` carries the seven base stats credited by the reducer (ADR-0002): `touches`,
`completions`, `drops`, `turnovers`, `blocks`, `goals`, and `assists`. All seven are required,
non-negative integers. Receptions, throw attempts, and plus-minus are derived during aggregation and
never appear in derived point state.

The no-null invariants are part of the contract: every field is required; `lineup` is always an array
(`[]` when empty); `thrower` is always a string (`""` when absent); `playerStats` is always an object
(`{}` when empty); and every stats entry is a complete, zero-filled `PlayerStats`. There are no null
or undefined values anywhere in derived state, and all numbers are integers. These invariants are what
make the TypeScript and Go encodings byte-identical under canonical JSON.

The empty-point state, returned by `reducePoint([], role)`, has `view` `"LINEUP_SETUP"`, `initialRole`
echoing `role`, `lineup` `[]`, `thrower` `""`, and `playerStats` `{}`.

## Parity fixtures: format and equality rules

One cross-language contract, executed unchanged by both implementations:

- TypeScript: `packages/shared/src/parity.test.ts` (Vitest) reads the directory via
  `new URL('../../../fixtures', import.meta.url)` from `packages/shared/src/`.
- Go: `server/internal/reduce/parity_test.go` (`go test ./...` from `server/`) reads
  `../../../fixtures` relative to the test's working directory (`server/internal/reduce/`).

Both resolve to the repo-root `fixtures/` directory. A fixture is language-neutral JSON: it describes
input `MatchEvent`s, an `initialRole`, and the exact derived state expected after `reducePoint`.

### File format

Files are named `NN-<slug>.json` and executed in lexicographic filename order. Every `*.json` file in
`fixtures/` is a fixture; do not put other JSON there.

```jsonc
{
  "name": "human-readable fixture description",
  "initialRole": "OFFENSE",          // "OFFENSE" | "DEFENSE"
  "events": [ /* MatchEvent objects, in raw input order */ ],
  "expected": { /* PointDerivedState */ }
}
```

All four fields — `name`, `initialRole`, `events`, and `expected` — are required.

Each event matches the `MatchEvent` schema in `packages/shared/src/events.ts`:

```jsonc
{
  "id": "evt-unique-id",             // string; replay is idempotent by this id
  "matchId": "match-1",              // string
  "pointNumber": 1,                  // integer
  "effectiveTs": 1000,               // integer epoch milliseconds
  "createdAt": "2026-03-14T09:00:00Z", // ISO-8601 UTC; informational, never used for ordering
  "deleted": false,                  // boolean tombstone flag
  "type": "SET_LINEUP",              // one of the nine event types below
  "payload": { "playerIds": ["player-a"] }
}
```

Payloads, in the canonical event-type order (`MATCH_EVENT_TYPES` in `events.ts` and the Go `EventType`
constants mirror this order):

| `type`              | `payload`                                                               |
| ------------------- | ----------------------------------------------------------------------- |
| `TOUCH`             | `{ "playerId": string }`                                                |
| `DROP`              | `{ "playerId": string }`                                                |
| `TURNOVER`          | `{ "playerId": string }`                                                |
| `DEFENSIVE_BLOCK`   | `{ "playerId": string }`                                                |
| `OPPONENT_TURNOVER` | `{}` (faceless opponent: no player attribution)                         |
| `SCORE`             | `{ "playerId": string, "assistPlayerId": string }` (`""` = unassisted) |
| `SET_LINEUP`        | `{ "playerIds": string[] }` (up to seven)                               |
| `SUB_PLAYER`        | `{ "playerOutId": string, "playerInId": string }`                       |
| `PULL`              | `{ "playerId": string }`                                                |

### Ordering and idempotence

Before reducing, both implementations run the same `prepareEvents` pipeline:

1. **Filter**: drop every event with `"deleted": true`.
2. **Sort**: ascending by `effectiveTs` (integer epoch milliseconds). Equal `effectiveTs` values are
   ordered by ascending `id` (code-unit/byte comparison). The order events appear in the fixture's
   `events` array is therefore irrelevant.
3. **Dedupe**: collapse equal `id`s, keeping the first occurrence in the sorted order. Replays are
   idempotent by id, so an event applied twice counts once.

### Equality rules

A fixture passes when the runner's `reducePoint(events, initialRole)` output is **identically equal**
to `expected` under canonical JSON:

- Object keys are sorted lexicographically at every level; key order in the fixture file is
  irrelevant.
- Array order is significant (lineups, player-id lists).
- All values are strings, booleans, integers, arrays, or objects. There are no nulls, no `undefined`,
  and no non-integer numbers.
- All state fields and all seven `PlayerStats` fields are required. A missing field or an unknown
  extra field is a mismatch: both runners compare the canonical key sets exactly.
- The comparison is byte-for-byte on the canonical JSON strings, so the two ports must produce the
  same values, not merely equivalent ones.

### Current suite

The suite is fixed at 24 fixtures; both runners assert that exact count. The fixture table in
`fixtures/README.md` lists what each one pins. The spec-gap defaults below are part of the same
contract and are each pinned by at least one fixture.

## Domain model

The model has two layers.

**Grouping layer.** `Team → Season → Event → Match` (ADR-0007). A `Season` sits between a team and its
events and is required: every event belongs to a season, and every match belongs to an event. The app
requires creating a season before any event, so a null season never arises. Non-tournament play (a
league night or a scrimmage) is modeled as a one-off `Event`, not as a match hanging directly off a
season. A `Match` tracks one team — ours — and records the opponent as `opponent_name`, a text field
only: the opponent is faceless, with no roster, lineup, or events (ADR-0003).

**People and roster layer.** A `User` joins a `Team` through `TeamMembership` (role `owner` or
`member`). A `Player` belongs to a `Team`. Rosters register players at two independent levels
(ADR-0005): a `SeasonRosterEntry` registers a player for a `Season`, and an `EventRosterEntry`
registers a player for a specific `Event`. Neither is per-`Match`. The on-field lineup (starting seven
plus substitutions) is derived by the reducer from `SET_LINEUP` and `SUB_PLAYER` events and is
deliberately never stored: there are no `is_starting` or `position` columns.

**Statistics.** Seven base stats are credited by the reducer; three (receptions, throw attempts,
plus-minus) are computed upward during aggregation and never stored (ADR-0002, ADR-0004).

**Addressing.** `Team`, `Season`, `Event`, `Match`, and `Player` each carry a stable slug (ADR-0006),
generated once server-side from a name and unchanged by later renames. Uniqueness is scoped to the URL
hierarchy: team slugs are globally unique; season and player slugs are unique within their team; event
slugs are unique within their season; match slugs are unique within their event. Collisions append a
numeric suffix.

## SQLite schema

The schema of record is `server/internal/db/migrations/0001_initial.sql`, applied by the embedded
migration runner when the server opens the database. It mirrors `docs/er-diagram.mmd` (the ER diagram
is the source for this table). Gameplay is the `match_events` log; the other tables are pre-game
structure and authentication.

| Table | Purpose | Keys and constraints |
| --- | --- | --- |
| `users` | Platform logins; one user may belong to many teams. | `id` PK; `email` unique; `name`; `created_at`. |
| `auth_identities` | A login identity linked to a user (currently `password`). | `id` PK; `user_id` FK cascade; `provider`; `provider_subject`; `email`; `created_at`; `UNIQUE(provider, provider_subject)`. |
| `password_credentials` | The bcrypt hash for a password identity. | `user_id` PK/FK cascade; `password_hash`; `updated_at`. |
| `sessions` | Cookie-backed session tokens, stored hashed. | `token_hash` PK; `user_id` FK cascade; `expires_at` (epoch seconds); `created_at`. |
| `teams` | The top-level tracked team. | `id` PK; `slug` globally unique; `name`; `created_at`. |
| `team_memberships` | User ↔ team join carrying `owner`/`member`. | `id` PK; `user_id` FK; `team_id` FK; `role`; `created_at`. |
| `seasons` | Required grouping between a team and its events (ADR-0007). | `id` PK; `team_id` FK; `name`; `slug` unique within team; `start_date`; `end_date`; `created_at`. |
| `events` | Tournament, league night, or scrimmage. | `id` PK; `season_id` FK required; `name`; `slug` unique within season; `start_date`; `end_date`; `location`. |
| `matches` | One game against a faceless opponent. | `id` PK; `event_id` FK required; `our_team_id` FK; `opponent_name` text only (ADR-0003); `slug` unique within event; `scheduled_at`; `point_cap`; `created_at`. |
| `players` | Person scoped to a team; `names` is JSON with the display name first. | `id` PK; `team_id` FK; `names` JSON; `slug` unique within team; `created_at`. |
| `season_roster_entries` | A player's registration for a season. | `id` PK; `season_id` FK; `player_id` FK; `created_at`; `UNIQUE(season_id, player_id)`; hard-deleted. |
| `event_roster_entries` | A player's registration for an event. | `id` PK; `event_id` FK; `player_id` FK; `created_at`; `UNIQUE(event_id, player_id)`; hard-deleted. |
| `match_events` | Append-only source of truth for gameplay. | `id` PK (idempotent by id); `match_id` FK; `point_number`; `effective_ts`; `type`; `payload` JSON; `deleted`; `created_at`. |

Additional schema rules:

- **No stats tables, ever** (ADR-0004). There is no stats table, no `stat_credits`, and no per-player
  stats projection. Every statistic is computed from `match_events` by `reducePoint` →
  `aggregateStats` and is never materialized. The server may later cache folded aggregates in memory,
  invalidated on sync; that is a cache, not a source of truth.
- **Faceless opponent** (ADR-0003). `matches.opponent_name` is the only opponent data; there is no
  opponent table and no opponent-side events or stats.
- **Two roster levels, hard deletes** (ADR-0005). `season_roster_entries` and
  `event_roster_entries` are independent; a player may be registered at either or both. Removing a
  player deletes the roster row (hard delete, no tombstone). Both tables carry a composite unique index
  — `UNIQUE(season_id, player_id)` and `UNIQUE(event_id, player_id)` — so the same player cannot be
  registered twice on the same roster. The on-field lineup is derived state, so neither table carries
  starting or position columns.
- **Slug columns** (ADR-0006). `teams.slug`, `seasons.slug`, `events.slug`, `matches.slug`, and
  `players.slug` are generated server-side and stable across renames; scoped uniqueness is listed per
  table above. Clients consume slugs; they never predict them.
- **Seasons are required** (ADR-0007). `seasons.team_id` is required, `events.season_id` is required,
  and `matches.event_id` is required.
- **Hard deletes with cascades.** Entities are removed by hard delete with FK `ON DELETE
  CASCADE`; there are no storage tombstones. (`match_events.deleted` is the event-log tombstone
  of the parity contract, not delete semantics.) Deleting a player removes the player and its
  roster rows but never the gameplay log, so their historical stats stay derivable.

## Spec-gap defaults

The original spec named the nine event types but left the effects of seven of them unspecified. TOUCH
and SET_LINEUP were pinned by the three tracer fixtures; the remaining seven are decided here. These
defaults are part of the parity contract: both ports implement them identically, and every decision
below is pinned by at least one of the fixtures in `fixtures/README.md`.

Shared conventions:

- Only our team's events are logged; the opponent is faceless (ADR-0003). Opponent actions never
  produce `playerStats` entries.
- `initialRole` is an echo of how the point started. It never gates an event: every event applies its
  own effect, and the view transitions below are absolute.
- The reducer does not validate rosters or lineup membership. Any player id named by an event may be
  credited even if it is not in `lineup`.
- `playerStats` entries are created only when a stat is credited, and are always complete, zero-filled
  `PlayerStats` values. `thrower` is `""` when nobody on our team has the disc.
- The canonical case order is TOUCH, DROP, TURNOVER, DEFENSIVE_BLOCK, OPPONENT_TURNOVER, SCORE,
  SET_LINEUP, SUB_PLAYER, PULL in both languages.

1. **DROP** — Credits `drops + 1` to the named receiver and nothing else. A drop is a receiver fault
   (GLOSSARY), so it is neither a completion nor a thrower turnover. Clears `thrower` and moves `view`
   to `DEFENSE_ACTIVE`: the disc is dead and the opponent takes possession (USAU 14.A.1, "a pass is
   incomplete").
2. **TURNOVER** — Credits `turnovers + 1` to the named thrower. The throw is not a completion. Clears
   `thrower` and moves `view` to `DEFENSE_ACTIVE` (USAU 14.A: throwaway, stall, or miscellaneous
   thrower fault).
3. **DEFENSIVE_BLOCK** — Credits `blocks + 1` to the named defender, clears `thrower`, and moves `view`
   to `OFFENSE_POSSESSION`: our team gains the disc. A block is not a completed throw, so clearing
   `thrower` ensures the next TOUCH does not credit the blocker a completion. If the blocker caught the
   disc, that TOUCH establishes them as thrower (USAU 14.C: an interception is defensive possession,
   and the throwing team loses it).
4. **OPPONENT_TURNOVER** — Credits no player stats: the opponent is faceless (ADR-0003). Clears
   `thrower` and moves `view` to `OFFENSE_POSSESSION`. Clearing `thrower` prevents a stale id from
   being credited a completion on the next TOUCH.
5. **SCORE** — Credits `goals + 1` to the scorer and, when `assistPlayerId` is non-empty,
   `assists + 1` to the assister. Clears `thrower` and moves `view` to `SCORED`. SCORE is terminal and
   credits no touch or completion: those belong to TOUCH events, so a scoring play recorded as TOUCH
   then SCORE counts both parts, while one recorded as SCORE alone counts only the goal/assist.
   **Callahan guard:** `assistPlayerId: ""` means a defensive score (GLOSSARY: Callahan). The reducer
   must not fall back to the current `thrower` for an assist or completion, and must not charge a
   turnover to any player; a SCORE event changes no `turnovers` value (USAU 13: a goal is scored by
   catching the disc in the end zone, not by throwing it).
6. **SUB_PLAYER** — Credits no stats. Replaces the first occurrence of `playerOutId` in `lineup` with
   `playerInId`, preserving field order and lineup size, including a full seven-player lineup (USAU
   9.C: at most seven players on the field). If `playerOutId` is the current `thrower`, the incoming
   player becomes the `thrower` (USAU 7.C.3.a: a replacement takes possession when the player in
   possession leaves injured). If `playerOutId` is not in `lineup`, `lineup` is unchanged; the thrower
   swap still applies if the id matches. `view` is unchanged. Malformed substitutions (a player not on
   the field, a player already on the field) are not validated.
7. **PULL** — Credits no base stat (a pull is not one of the seven tracked stats, ADR-0002). Clears
   `thrower` and moves `view` to `DEFENSE_ACTIVE`: the event names our puller, so this point is being
   played on defense (USAU 9.B: after a goal the scoring team pulls). A pull is not a completed pass,
   so the first TOUCH after receiving it credits no completion.

### Cross-cutting TOUCH and SET_LINEUP behavior

These extend the two event types that already had tracer fixtures; they are pinned by the full suite
as well.

- **TOUCH** — Credits `touches + 1` to the receiver. If a different player was the current `thrower`,
  that thrower also gets `completions + 1`: the touch is the only evidence of a completed pass, since
  there is no PASS event (USAU 3.N.5: a throw is complete when the team retains possession). A
  same-player TOUCH never credits a self-completion. Sets `thrower` to the receiver and `view` to
  `OFFENSE_POSSESSION`.
- **SET_LINEUP** — Replaces `lineup` wholesale with the payload ids in field order and sets `view` to
  `PULL_WAIT`. A full seven-player lineup is accepted verbatim; the reducer does not enforce the
  seven-player maximum because the payload is contractually "up to seven".

## Out of scope

This milestone deliberately excludes:

- The WebSocket `MatchActor` and live synchronization.
- The full UI.
- Match-level point lifecycle: time cap, forfeit, and related match-end rules.

## References

- `GLOSSARY.md` — domain vocabulary.
- `docs/adr/` — settled decisions: [0001](adr/0001-event-sourced-reducer-parity.md),
  [0002](adr/0002-base-and-derived-stats.md), [0003](adr/0003-faceless-opponent.md),
  [0004](adr/0004-stats-derived-not-materialized.md), [0005](adr/0005-event-scoped-roster.md),
  [0006](adr/0006-stable-scoped-slugs.md), [0007](adr/0007-seasons-grouping-level.md).
- `fixtures/README.md` — normative fixture format, equality rules, and spec-gap defaults.
- `docs/er-diagram.mmd` — the SQLite entity-relationship diagram.
- `docs/reference/usau-rules-2026-27.md` — grep-able 2026–27 USAU rules; the official PDF sits
  alongside it.
