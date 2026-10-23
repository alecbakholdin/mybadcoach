# My Bad, Coach

Ultimate Frisbee live stat tracking — real-time, collaborative, offline-first. Gameplay is
event-sourced: every stat tap emits an atomic MatchEvent, and a pure reducer folds the time-sorted
log into point state and statistics.

## Structure

**Team**:
The top-level entity: the team whose matches and stats we track.
_Avoid_: Club, squad, org

**User**:
An app login in the (future) multi-team platform. A user may belong to many teams.
_Avoid_: Account

**TeamMembership**:
The join between a User and a Team, carrying a role (`owner` or `member`).
_Avoid_: Membership

**Season**:
A period during which a Team plays (e.g. "2026 Spring"). Groups the season's Events and carries the
team's season roster.
_Avoid_: Year, cycle

**Event**:
A grouping of matches within a Season — most commonly a tournament, but also a league night or
scrimmage. Belongs to a Season; has a name, dates, and location.
_Avoid_: Tournament, meet

**Match**:
A single game. Belongs to an Event within a Season and tracks one team against a faceless opponent.
_Avoid_: Game

**Opponent**:
The other team in a Match, recorded as a name string only — no roster, lineup, or events.
_Avoid_: Away team, rival

**Player**:
A person who plays, scoped to a Team. Identified by one or more names: the first is the
primary display name, the rest are addressable aliases (nicknames, alternate spellings).
_Avoid_: Athlete

**SeasonRosterEntry**:
A player's registration on a Team's roster for a Season.
_Avoid_: Member

**EventRosterEntry**:
A player's registration for a specific Event (tournament) — the set of players eligible to field in
that event's matches. The on-field lineup is derived from MatchEvents, not stored here.
_Avoid_: Line, slot

**Slug**:
The stable, human-readable identifier by which a Team, Season, Event, Match, or Player is addressed.
_Avoid_: Handle

## Gameplay

**MatchEvent**:
An atomic "intent" event a stat-keeper emits (TOUCH, DROP, TURNOVER, DEFENSIVE_BLOCK,
OPPONENT_TURNOVER, SCORE, SET_LINEUP, SUB_PLAYER, PULL). Time-ordered and idempotent by id.
_Avoid_: Action, command, log entry

**Point**:
A single scoring unit; events are grouped per (matchId, pointNumber).
_Avoid_: Possession, rally

**Lineup**:
The up-to-seven players currently on the field for our team.
_Avoid_: On-field set

**Thrower**:
The player currently in possession of the disc.
_Avoid_: Handler, carrier

**View**:
The current phase of a point: lineup setup, pull wait, defense active, offense possession, or scored.
_Avoid_: State

**StartingRole**:
Whether we start a point on offense or defense.
_Avoid_: Side

**Callahan**:
A goal scored by the defense (the offense is scored on).

## Statistics

**Touch**:
A player catching or possessing the disc. A base stat credited by the reducer.
_Avoid_: Reception, catch

**Completion**:
A throw completed to a teammate. A base stat.
_Avoid_: Completed pass

**Drop**:
Dropping a catchable throw — a receiver fault. A base stat.
_Avoid_: Bobble

**Turnover**:
A thrower turnover (throwaway, stall, or misc. throw). A base stat.
_Avoid_: Throwaway, turn

**Block**:
A defensive block. A base stat.
_Avoid_: Deflection, D

**Goal**:
A goal scored. A base stat.

**Assist**:
A throw that led directly to a goal. A base stat.
_Avoid_: Helper pass

**Throw attempt**:
A derived stat: touches + completions − goals − turnovers. Computed, never stored.

**Plus-minus**:
A derived individual stat: goals + assists + blocks − turnovers − drops. Computed, never stored.
_Avoid_: On-field plus/minus

**TeamRecord**:
A team's wins, losses, and ties across a set of matches.
_Avoid_: Standings

**MatchScore**:
The final score of a completed match: our score and the opponent's score.
_Avoid_: Result, final

## Reference

- [2026–27 USAU Rules of Ultimate, Markdown](docs/reference/usau-rules-2026-27.md) — grep-able
  conversion of the official rulebook. Official PDF: `docs/reference/usau-rules-2026-27.pdf`.
