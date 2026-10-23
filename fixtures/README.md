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

The suite is fixed at 24 fixtures; both runners assert that exact count.

| File                                    | Pins                                                                       |
| --------------------------------------- | -------------------------------------------------------------------------- |
| `01-empty-point.json`                   | `reducePoint([], "OFFENSE")` returns initial `LINEUP_SETUP`.               |
| `02-out-of-order-sort.json`             | Raw input order is ignored; `effectiveTs` decides.                          |
| `03-duplicate-id-dedupe.json`           | Duplicate ids reduce once (one TOUCH, not two).                             |
| `04-set-lineup-full-seven.json`         | A full seven-player lineup is accepted in field order.                      |
| `05-set-lineup-replaces-lineup.json`    | A later SET_LINEUP replaces the lineup wholesale.                           |
| `06-touch-credits-completion.json`      | A touch credits the previous thrower with a completion.                     |
| `07-pull.json`                          | PULL advances to `DEFENSE_ACTIVE`, credits no stat, clears thrower.         |
| `08-pull-then-touch.json`               | Receiving the pull is a touch, never a completion.                          |
| `09-drop.json`                          | DROP credits the receiver, clears thrower, flips to defense.                |
| `10-turnover.json`                      | TURNOVER credits the thrower, clears thrower, flips to defense.             |
| `11-defensive-block.json`               | A block credits the defender and gives our team possession.                 |
| `12-block-then-touch.json`              | A block is not a throw: the next touch credits no completion.               |
| `13-opponent-turnover.json`             | Opponent turnover clears any thrower and credits no player stats.           |
| `14-score-assisted.json`                | SCORE credits a goal and an assist, then marks the point scored.            |
| `15-score-callahan-guard.json`          | Empty assist never falls back to the prior thrower; no turnover credited.   |
| `16-score-callahan-sequence.json`       | Block then unassisted score is a callahan: goal only.                       |
| `17-sub-player.json`                    | SUB_PLAYER swaps the outgoing player in place.                              |
| `18-sub-of-thrower.json`                | Substituting the thrower transfers possession to the incoming player.       |
| `19-sub-full-lineup.json`               | A substitution in a full seven-player lineup keeps the lineup at seven.     |
| `20-deleted-events-ignored.json`        | Deleted tombstone events never affect derived state.                        |
| `21-out-of-order-full-sequence.json`    | A whole lineup/touch/touch/score log reduces in `effectiveTs` order.        |
| `22-duplicate-score-idempotent.json`    | A replayed SCORE id counts once, keeping the first after sorting.           |
| `23-turnover-recovery-cycle.json`       | Turnovers clear the thrower, so the next touch is not a completion.         |
| `24-touch-same-player-no-self-completion.json` | A repeated TOUCH by one player never credits a self-completion.      |

## Spec-gap defaults

The original spec named the nine event types but left the effects of seven of them unspecified.
TOUCH and SET_LINEUP were pinned by the three tracer fixtures; the remaining seven are decided
here. These defaults are part of the parity contract: both ports implement them identically, and
every decision below is pinned by at least one of the fixtures in the table above.

Shared conventions:

- Only our team's events are logged; the opponent is faceless (ADR-0003). Opponent actions never
  produce `playerStats` entries.
- `initialRole` is an echo of how the point started. It never gates an event: every event applies
  its own effect, and the view transitions below are absolute.
- The reducer does not validate rosters or lineup membership. Any player id named by an event may
  be credited even if it is not in `lineup`.
- `playerStats` entries are created only when a stat is credited, and are always complete,
  zero-filled `PlayerStats` values. `thrower` is `""` when nobody on our team has the disc.
- The canonical case order is TOUCH, DROP, TURNOVER, DEFENSIVE_BLOCK, OPPONENT_TURNOVER, SCORE,
  SET_LINEUP, SUB_PLAYER, PULL in both languages.

1. **DROP** — Credits `drops + 1` to the named receiver and nothing else. A drop is a receiver
   fault (GLOSSARY), so it is neither a completion nor a thrower turnover. Clears `thrower` and
   moves `view` to `DEFENSE_ACTIVE`: the disc is dead and the opponent takes possession
   (USAU 14.A.1, "a pass is incomplete").
2. **TURNOVER** — Credits `turnovers + 1` to the named thrower. The throw is not a completion.
   Clears `thrower` and moves `view` to `DEFENSE_ACTIVE` (USAU 14.A: throwaway, stall, or
   miscellaneous thrower fault).
3. **DEFENSIVE_BLOCK** — Credits `blocks + 1` to the named defender, clears `thrower`, and moves
   `view` to `OFFENSE_POSSESSION`: our team gains the disc. A block is not a completed throw, so
   clearing `thrower` ensures the next TOUCH does not credit the blocker a completion. If the
   blocker caught the disc, that TOUCH establishes them as thrower (USAU 14.C: an interception is
   defensive possession, and the throwing team loses it).
4. **OPPONENT_TURNOVER** — Credits no player stats: the opponent is faceless (ADR-0003). Clears
   `thrower` and moves `view` to `OFFENSE_POSSESSION`. Clearing `thrower` prevents a stale id from
   being credited a completion on the next TOUCH.
5. **SCORE** — Credits `goals + 1` to the scorer and, when `assistPlayerId` is non-empty,
   `assists + 1` to the assister. Clears `thrower` and moves `view` to `SCORED`. SCORE is
   terminal and credits no touch or completion: those belong to TOUCH events, so a scoring play
   recorded as TOUCH then SCORE counts both parts, while one recorded as SCORE alone counts only
   the goal/assist. **Callahan guard:** `assistPlayerId: ""` means a defensive score (GLOSSARY:
   Callahan). The reducer must not fall back to the current `thrower` for an assist or completion,
   and must not charge a turnover to any player; a SCORE event changes no `turnovers` value
   (USAU 13: a goal is scored by catching the disc in the end zone, not by throwing it).
6. **SUB_PLAYER** — Credits no stats. Replaces the first occurrence of `playerOutId` in `lineup`
   with `playerInId`, preserving field order and lineup size, including a full seven-player
   lineup (USAU 9.C: at most seven players on the field). If `playerOutId` is the current
   `thrower`, the incoming player becomes the `thrower` (USAU 7.C.3.a: a replacement takes
   possession when the player in possession leaves injured). If `playerOutId` is not in `lineup`,
   `lineup` is unchanged; the thrower swap still applies if the id matches. `view` is unchanged.
   Malformed substitutions (a player not on the field, a player already on the field) are not
   validated.
7. **PULL** — Credits no base stat (a pull is not one of the seven tracked stats, ADR-0002).
   Clears `thrower` and moves `view` to `DEFENSE_ACTIVE`: the event names our puller, so this
   point is being played on defense (USAU 9.B: after a goal the scoring team pulls). A pull is
   not a completed pass, so the first TOUCH after receiving it credits no completion.

### Cross-cutting TOUCH and SET_LINEUP behavior

These extend the two event types that already had tracer fixtures; they are pinned by the full
suite as well.

- **TOUCH** — Credits `touches + 1` to the receiver. If a different player was the current
  `thrower`, that thrower also gets `completions + 1`: the touch is the only evidence of a
  completed pass, since there is no PASS event (USAU 3.N.5: a throw is complete when the team
  retains possession). A same-player TOUCH never credits a self-completion. Sets `thrower` to the
  receiver and `view` to `OFFENSE_POSSESSION`.
- **SET_LINEUP** — Replaces `lineup` wholesale with the payload ids in field order and sets
  `view` to `PULL_WAIT`. A full seven-player lineup is accepted verbatim; the reducer does not
  enforce the seven-player maximum because the payload is contractually "up to seven".


## Running

```sh
npm run test:shared      # repo root; Vitest executes every fixture
cd server && go test ./...  # Go executes the same fixtures
```
