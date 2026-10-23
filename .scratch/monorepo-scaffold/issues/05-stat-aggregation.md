# 05: Stat aggregation

**What to build:** The upward fold from reduced points to per-match, per-event, and career stats,
including the three derived stats, so a stat table can be rendered for any player, match, event, or team.

**Blocked by:** 04 — Full reducer + 24-fixture parity.

**Status:** resolved

- [x] `aggregateMatchStats` sums the seven base stats correctly over reduced points
- [x] `receptions`, `throwAttempts`, and `plusMinus` are computed from the base stats
- [x] `aggregateEventStats` and `aggregateCareerStats` roll up matches, and `TeamRecord` computes W/L/T
- [x] Empty input returns zeroed shapes, never null
- [x] `aggregateStats.test.ts` passes

## Comments

Resolved by branch `ticket/05-stat-aggregation` (cc80f40), merged as 7549113. Receptions = touches (rationale in `aggregateStats.ts`); W/L/T computed from caller-fed `MatchScore` inputs (`teamRecord.ts`).
