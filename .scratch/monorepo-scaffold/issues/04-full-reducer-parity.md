# 04: Full reducer + 24-fixture parity

**What to build:** The complete `reducePoint` handling all nine `MatchEvent` types identically in
TypeScript and Go, proven by the full 24-fixture parity suite — the "single most important guarantee"
that stats are computed the same everywhere.

**Blocked by:** 03 — Domain types + event schema + parity harness.

**Status:** resolved

- [x] `reducePoint` switches on all nine payload types in the same case order in TS and Go
- [x] 24 fixtures cover every event type and the edge cases (deleted, lineup-full, callahan guard, sub-of-thrower, dedupe, out-of-order)
- [x] The seven spec-gap defaults (PULL, SUB_PLAYER, …) are pinned in fixtures
- [x] `npm run test:shared` and `go test ./...` are both green on all 24 fixtures

## Comments

Resolved by branch `ticket/04-full-reducer-parity` (e0105a0), merged as f4654ab. The seven spec-gap defaults were designed and pinned in fixtures; recorded in `fixtures/README.md` and `docs/architecture.md`.
