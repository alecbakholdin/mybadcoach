# 05: Stat aggregation

**What to build:** The upward fold from reduced points to per-match, per-event, and career stats,
including the three derived stats, so a stat table can be rendered for any player, match, event, or team.

**Blocked by:** 04 — Full reducer + 24-fixture parity.

**Status:** ready-for-agent

- [ ] `aggregateMatchStats` sums the seven base stats correctly over reduced points
- [ ] `receptions`, `throwAttempts`, and `plusMinus` are computed from the base stats
- [ ] `aggregateEventStats` and `aggregateCareerStats` roll up matches, and `TeamRecord` computes W/L/T
- [ ] Empty input returns zeroed shapes, never null
- [ ] `aggregateStats.test.ts` passes
