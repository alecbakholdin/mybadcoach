# 04: Full reducer + 24-fixture parity

**What to build:** The complete `reducePoint` handling all nine `MatchEvent` types identically in
TypeScript and Go, proven by the full 24-fixture parity suite — the "single most important guarantee"
that stats are computed the same everywhere.

**Blocked by:** 03 — Domain types + event schema + parity harness.

**Status:** ready-for-agent

- [ ] `reducePoint` switches on all nine payload types in the same case order in TS and Go
- [ ] 24 fixtures cover every event type and the edge cases (deleted, lineup-full, callahan guard, sub-of-thrower, dedupe, out-of-order)
- [ ] The seven spec-gap defaults (PULL, SUB_PLAYER, …) are pinned in fixtures
- [ ] `npm run test:shared` and `go test ./...` are both green on all 24 fixtures
