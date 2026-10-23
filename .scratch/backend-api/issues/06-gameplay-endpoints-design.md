# Gameplay endpoints RPC design

Type: grilling
Status: resolved
Blocked by: 02, 05

## Question

Walk the gameplay RPCs with Alec: MatchEvent append — batch append for offline queue replay (unary
with repeated events vs client-streaming; idempotency by event id; deleted tombstones) — and reads:
folded per-point state for a match, match summary, aggregate stats (TeamRecord, MatchScore). Settle
the read shapes against the derived-stats rules (ADR-0002, ADR-0004). Output: settled gameplay RPCs.

## Answer

Resolved 2026-10-22. Q1: oneof payloads. Q2 (revised): batch append is the **only** gameplay
write path — live taps flush through it too, because the client is offline-first by design;
edits are tombstone + corrected event composed client-side (no separate edit RPC); storage
upserts by event id, the `deleted` flag is last-writer-wins, and re-using an id with different
content is rejected (`invalid_argument`, `id_conflict`). Q3: batch cap 1000; malformed events →
`invalid_argument` with the offending index; duplicates are not errors
(`skipped_duplicate_ids`). Q4: five reads — GetMatch, ListMatchEvents, GetMatchState (folded
per point), GetMatchStats (match aggregation), GetTeamRecord; event/career aggregation and
MatchScore backlogged. Q5: no streaming in this effort (live sync is the backlogged MatchActor
effort). Q6: snake_case proto fields so protojson emits the fixtures' camelCase keys.
