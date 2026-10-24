# Build player + roster management screens

Type: task
Status: open
Blocked by: 06

## Question

Build the player and roster-management screens:

- Players: create / list / edit / delete within a team (names/aliases, display name first) via the
  `players` service.
- Rosters: add/remove players on the season roster and the event roster (two independent levels, per
  ADR-0005), via the roster-entry sets on the `seasons` and `events` services.

No starting/position columns — the on-field lineup is derived from MatchEvents, never stored.
