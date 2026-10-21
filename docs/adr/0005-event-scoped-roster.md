# Roster is season- and event-scoped; lineup is derived

A `Player` belongs to a `Team`, and rosters register players at two levels: a `SeasonRosterEntry`
registers a player for a `Season` (the team's roster that season), and an `EventRosterEntry` registers a
player for a specific `Event` (a tournament's registered players). Neither is per-`Match`; the on-field
lineup (starting seven plus substitutions) is derived state, produced by the reducer from `SET_LINEUP`
and `SUB_PLAYER` events, and deliberately not stored as roster columns (no `is_starting`/`position`).

We chose season- plus event-scoped over match-scoped rosters because a team registers once per season and
fields a subset per tournament; a per-match roster would re-enter the same list per game. The two levels
are independent, so a player can be added to an event roster and then carried into the season roster or
the following season as an application-level copy rather than a schema constraint. Roster rows are
hard-deleted (removing a player deletes the entry) and carry a composite unique index —
`UNIQUE(season_id, player_id)` and `UNIQUE(event_id, player_id)` — so a player can't be registered twice
on the same roster.
