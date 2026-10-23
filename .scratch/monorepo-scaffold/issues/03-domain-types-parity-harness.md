# 03: Domain types + event schema + parity harness

**What to build:** The shared package's domain types and `MatchEvent` schema, plus a minimal `reducePoint`
and the cross-language test harness, so a stat-keeper's event log can be reduced to initial point state
identically in TypeScript and Go. This is the tracer bullet: it proves the parity harness (one `fixtures/`
dir run by both Vitest and `go test`) before any real reducer logic exists.

**Blocked by:** 01 — Toolchain + monorepo skeleton.

**Status:** resolved

- [x] `entities.ts` models Team → Season → Event → Match, `Player.names[]`, and season + event rosters per GLOSSARY/ADRs
- [x] `events.ts` defines `MatchEvent` and all nine payload variants
- [x] `state.ts` defines `PointDerivedState`/`PlayerStats` with zero-filled, never-null invariants
- [x] `helpers.ts` implements filter → sort → dedupe
- [x] `fixtures/` holds a README + three fixtures (empty point, out-of-order sort, duplicate-id dedupe)
- [x] `npm run test:shared` and `go test ./...` both pass the same three fixtures
- [x] `reducePoint([], 'OFFENSE')` returns `LINEUP_SETUP` initial state in both languages

## Comments

Resolved by branch `ticket/03-domain-types-parity-harness` (dd17d9d), merged as e8f8d1b.
