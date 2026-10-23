# Parity fixtures

One cross-language contract, executed unchanged by both implementations:

- TypeScript: `packages/shared/src/parity.test.ts` (Vitest) reads this directory via
  `new URL('../../../fixtures', import.meta.url)` from `packages/shared/src/`.
- Go: `server/internal/reduce/parity_test.go` (`go test ./...` from `server/`) reads
  `../../../fixtures` relative to the test's working directory
  (`server/internal/reduce/`).

Both resolve to the repo-root `fixtures/` directory. A fixture is language-neutral JSON: it
describes input `MatchEvent`s, an `initialRole`, and the exact derived state expected after
`reducePoint`.

## File format

Files are named `NN-<slug>.json` and executed in lexicographic filename order. Every `*.json`
file in this directory is a fixture; do not put other JSON here.

```jsonc
{
  "name": "human-readable fixture description",
  "initialRole": "OFFENSE",          // "OFFENSE" | "DEFENSE"
  "events": [ /* MatchEvent objects, in raw input order */ ],
  "expected": { /* PointDerivedState */ }
}
```

All four fields are required.

### `events`

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

Payloads, in the canonical event-type order (`MATCH_EVENT_TYPES` in `events.ts` and the Go
`EventType` constants mirror this order):

| `type`              | `payload`                                                    |
| ------------------- | ------------------------------------------------------------ |
| `TOUCH`             | `{ "playerId": string }`                                     |
| `DROP`              | `{ "playerId": string }`                                     |
| `TURNOVER`          | `{ "playerId": string }`                                     |
| `DEFENSIVE_BLOCK`   | `{ "playerId": string }`                                     |
| `OPPONENT_TURNOVER` | `{}` (faceless opponent: no player attribution)              |
| `SCORE`             | `{ "playerId": string, "assistPlayerId": string }` (`""` = unassisted) |
| `SET_LINEUP`        | `{ "playerIds": string[] }` (up to seven)                    |
| `SUB_PLAYER`        | `{ "playerOutId": string, "playerInId": string }`            |
| `PULL`              | `{ "playerId": string }`                                     |

### `expected`

A complete `PointDerivedState`:

```jsonc
{
  "view": "LINEUP_SETUP",            // LINEUP_SETUP | PULL_WAIT | DEFENSE_ACTIVE | OFFENSE_POSSESSION | SCORED
  "initialRole": "OFFENSE",          // echo of the fixture's initialRole
  "lineup": ["player-a"],            // player ids on the field, in field order
  "thrower": "",                     // player id in possession; "" means nobody
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

Every field is required. `lineup` is always an array (`[]` when empty), `thrower` is always a
string (`""` when absent), `playerStats` is always an object (`{}` when empty), and every
player-stats entry carries all seven base stats as non-negative integers. There are no `null`
or `undefined` values anywhere in derived state.

## Ordering and idempotence rules

Before reducing, both implementations run the same `prepareEvents` pipeline:

1. **Filter**: drop every event with `"deleted": true`.
2. **Sort**: ascending by `effectiveTs` (integer epoch milliseconds). Equal `effectiveTs`
   values are ordered by ascending `id` (code-unit/byte comparison). The order events appear
   in the fixture's `events` array is therefore irrelevant.
3. **Dedupe**: collapse equal `id`s, keeping the first occurrence in the sorted order. Replays
   are idempotent by id, so an event applied twice counts once.

## Equality rules

A fixture passes when the runner's `reducePoint(events, initialRole)` output is **identically
equal** to `expected` under canonical JSON:

- Object keys are sorted lexicographically at every level; key order in the fixture file is
  irrelevant.
- Array order is significant (lineups, player-id lists).
- All values are strings, booleans, integers, arrays, or objects. There are no nulls, no
  `undefined`, and no non-integer numbers.
- All state fields and all seven `PlayerStats` fields are required. A missing field or an
  unknown extra field is a mismatch: both runners compare the canonical key sets exactly.
- The comparison is byte-for-byte on the canonical JSON strings, so the two ports must
  produce the same values, not merely equivalent ones.

## Initial state

`reducePoint([], role)` returns the empty-point state in both languages: `view` is
`"LINEUP_SETUP"`, `initialRole` echoes the argument, `lineup` is `[]`, `thrower` is `""`, and
`playerStats` is `{}`.

## Current fixtures

| File                            | Pins                                                        |
| ------------------------------- | ----------------------------------------------------------- |
| `01-empty-point.json`           | `reducePoint([], "OFFENSE")` returns initial `LINEUP_SETUP`. |
| `02-out-of-order-sort.json`     | Raw input order is ignored; `effectiveTs` decides.          |
| `03-duplicate-id-dedupe.json`   | Duplicate ids reduce once (one TOUCH, not two).             |

Ticket 04 extends this suite to 24 fixtures covering every event type and the documented edge
cases. The three fixtures above remain part of that suite.

## Running

```sh
npm run test:shared      # repo root; Vitest executes every fixture
cd server && go test ./...  # Go executes the same fixtures
```
