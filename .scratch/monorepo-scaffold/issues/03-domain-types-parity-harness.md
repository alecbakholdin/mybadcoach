# 03: Domain types + event schema + parity harness

**What to build:** The shared package's domain types and `MatchEvent` schema, plus a minimal `reducePoint`
and the cross-language test harness, so a stat-keeper's event log can be reduced to initial point state
identically in TypeScript and Go. This is the tracer bullet: it proves the parity harness (one `fixtures/`
dir run by both Vitest and `go test`) before any real reducer logic exists.

**Blocked by:** 01 — Toolchain + monorepo skeleton.

**Status:** ready-for-agent

- [ ] `entities.ts` models Team → Season → Event → Match, `Player.names[]`, and season + event rosters per GLOSSARY/ADRs
- [ ] `events.ts` defines `MatchEvent` and all nine payload variants
- [ ] `state.ts` defines `PointDerivedState`/`PlayerStats` with zero-filled, never-null invariants
- [ ] `helpers.ts` implements filter → sort → dedupe
- [ ] `fixtures/` holds a README + three fixtures (empty point, out-of-order sort, duplicate-id dedupe)
- [ ] `npm run test:shared` and `go test ./...` both pass the same three fixtures
- [ ] `reducePoint([], 'OFFENSE')` returns `LINEUP_SETUP` initial state in both languages
