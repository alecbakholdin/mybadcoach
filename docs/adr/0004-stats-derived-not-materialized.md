# Statistics are derived, not materialized

`match_events` is the single source of truth for gameplay; every statistic is computed from it by the
pure reducer (`reducePoint` → `aggregateStats`) and is never stored in SQLite. There is no stats table,
no `stat_credits`, and no per-player stats projection.

We chose derive-only over materializing a stats projection because a projection duplicates state the log
already determines, and under offline-first sync (out-of-order events, deletions, replayed events) it
would drift unless carefully rebuilt. The client already folds its local log in memory during live
scoring, so the reducer must exist and stay fast on-device regardless — a materialized table only helps
server reads. That one real cost (folding a team's full history on every read) is deferred: the server
may later cache folded aggregates in memory — the per-match actors already hold them — invalidated on
sync. The read path serves the small per-player aggregate over the wire, never the unbounded log; that
payload win is encoding-agnostic and does not require a stored stats table.
