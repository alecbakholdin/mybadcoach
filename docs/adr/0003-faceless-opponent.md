# Single-team match tracking with a faceless opponent

A `Match` tracks one team — ours — and records the opponent as a name string only (no opponent
roster, lineup, or events).

We chose this over modeling both teams because the product tracks our team's stats; requiring an
opponent roster would add input friction without producing stats we display. The multi-team platform
(many teams, each tracking its own matches) is a user-system concern, not a per-match concern.
