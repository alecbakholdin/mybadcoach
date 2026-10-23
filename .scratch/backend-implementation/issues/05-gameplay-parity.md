# 05: gameplay endpoints and generated-type parity

Status: ready-for-agent

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
