# Event-sourced gameplay with a cross-language pure reducer

Gameplay state and statistics are not stored — they are derived by a pure reducer (`reducePoint`)
that folds the time-sorted `MatchEvent` log. The reducer must behave identically in TypeScript
(client) and Go (server), proven by a shared hand-authored fixture suite run by both.

We chose this over a single server-authoritative implementation because the offline-first client
must be able to recompute the same state from the same events. Parity fixtures are the only way to
guarantee the two ports cannot drift.
