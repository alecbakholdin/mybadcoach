# Structure CRUD RPC design

Type: grilling
Status: resolved
Blocked by: 02

## Question

Walk the structure-entity CRUD RPCs with Alec: Team, Season, Event, Match, Player, TeamMembership,
SeasonRosterEntry, EventRosterEntry. Settle service grouping, method-by-method request/response
messages, slug-addressed requests (ADR-0006), pagination fields (offset/limit), deletion semantics
(tombstones vs hard deletes per ADRs), Player.names editing (slug immutable), and owner-role gating
on membership + destructive operations. Output: settled service/method list for structure entities.

## Answer

Resolved 2026-10-22. Round-1 recommendations accepted (Q1, Q2, Q4, Q5, Q6, Q7); Q3 amended by
Alec to hard deletes, Q8 gains CopySeasonRoster:

- **Five services by family**: TeamService (teams + memberships), SeasonService (seasons +
  season rosters), EventService (events + event rosters), MatchService, PlayerService.
  Packages `mybadcoach.{teams,seasons,events,matches,players}.v1`.
- **Addressing**: nested slugs (`team_slug` [+ `season_slug` [+ `event_slug`]]); ids never in
  requests. **Gating**: membership grants reads + structure mutations. Owner-only: membership
  management, team rename, `DeleteTeam`, `DeleteSeason`, `DeletePlayer` (Alec: player deletion
  is owner territory). Member: `DeleteEvent`, `DeleteMatch`, roster add/remove.
- **Method set**: Create/Get/Update/List for all five resources; `Delete` (hard) for all five;
  `Remove` (hard) for memberships and roster entries; `CopySeasonRoster { team_slug,
  from_season_slug, to_season_slug } → { copied }` skipping existing entries (Alec's addition).
- **Hard delete everywhere** (Alec: simpler and testable now; tombstones are a possible later
  migration). FK `ON DELETE CASCADE` throughout; DDL asset and `docs/architecture.md` updated.
  Exception: `match_events.deleted` stays — event-log tombstone of the parity contract. Note:
  deleting a Player removes the player + roster rows but never the `match_events` log, so
  historical stats stay derivable from the log.
- **Messages**: `Team { id, slug, name }`; `Season { id, slug, name, start_date, end_date }`;
  `Event { id, slug, name, start_date, end_date, location }`; `Match { id, slug, opponent_name,
  scheduled_at, point_cap }`; `Player { id, slug, names }` (`repeated string`, display name
  first). Membership rows `{ user, role }`; roster rows `{ player }`.
- **Updates**: partial via proto3 optional presence; `names` replaces wholesale when present;
  slug never updatable.
- **Lists**: `offset`/`limit` → `{ items, total }`; `ListMyTeams → { memberships: [{ team,
  role }] }` for the client boot call.
- **Memberships**: `CreateTeam` makes the caller owner; `AddMembership` by email of an
  existing user (`user_not_found` otherwise — invites backlogged); `UpdateMembership` role;
  `RemoveMembership`; last-owner guard errors `last_owner`.
- **Rosters**: Add/Remove/List per season and per event; hard deletes (ADR-0005).
- **Backlogged**: SearchPlayer fuzzy search (SQLite FTS5 is the likely future tool).
