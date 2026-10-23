# 05: gameplay endpoints and generated-type parity

Status: resolved

Blocked by: 01, 04

Implement the gameplay path and migrate both reducers to generated types (backend-api tickets 06
+ 07).

- `AppendMatchEvents`: upsert by event id; replays land in `skipped_duplicate_ids`; the
  `deleted` flag is last-writer-wins; re-using an id with different content → `invalid_argument`
  (`id_conflict`); batch cap 1000; malformed events rejected with the offending index.
- `ListMatchEvents` (optional point filter), `GetMatchState` (folded per point via
  `reducePoint`), `GetMatchStats` (match-scoped `aggregateStats`), `GetTeamRecord`.
- Migrate both reducers (`packages/shared/src/`, `server/internal/reduce/`) to the generated
  `MatchEvent`/`PointDerivedState` types; delete the hand-written `events.ts`/`state.go` mirrors.
- Implement the parity harness mapping in both languages: fixture JSON → generated types,
  including short enum names (`TOUCH` ↔ `MATCH_EVENT_TYPE_TOUCH`, …) and the oneof member per
  event type.
- Parity proof: the existing 24-fixture suite green under Vitest and `go test`; add fixtures for
  any new behavior. Fixtures stay hand-written, human-readable JSON.
- Tests: backend integration for append/replay/idempotency/correction (tombstone + append)
  against real temp-file SQLite.

## Comments

Resolved on branch `ticket/05-gameplay-parity` (ceaf590, 37138e0, 7ad231f), merged to integration.
Parity: 24/24 fixtures green in both Vitest and `go test`; the hand-written `events.ts`/`state.ts`
and `events.go`/`state.go` mirrors are deleted in favour of the generated types. Mapping lives in
`packages/shared/src/mapping.ts` and `server/internal/gameplay/mapping.go` (short↔full enum names +
type↔oneof-member), reused by both the parity harnesses and `match_events` persistence.

Review-worthy judgment calls (Alec to confirm):
- **GetTeamRecord** derives `ourScore` from the match's live `SCORE` events and sets
  `opponentScore = 0` — the faceless opponent (ADR-0003) logs no events, so only our score is
  derivable. Only matches with ≥1 live SCORE count as completed, so ties are structurally
  impossible and unplayed matches don't read as 0-0. (Match-score/opponent modeling is still out
  of scope.)
- `GetMatchState`/`GetMatchStats` fold every point from `STARTING_ROLE_OFFENSE` (the log records
  no starting role; reads need a deterministic default).
- `match_events.type` = short enum name, `payload` = compacted camelCase protojson; append
  identity = (match_id, point_number, effective_ts, type, payload) with `deleted` last-writer-wins.
- `ListMatchEvents` includes tombstones (the offline-sync contract) and returns the unpaginated
  total.
