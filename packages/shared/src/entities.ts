/**
 * Domain entities for My Bad, Coach.
 *
 * The hierarchy is Team -> Season -> Event -> Match (ADR-0007). A Player
 * belongs to a Team, and rosters register players at season and event scope
 * (ADR-0005). Lineup membership is derived from MatchEvents and is never a
 * roster column.
 *
 * IDs are opaque UUID strings. Dates are ISO-8601 calendar dates
 * (`YYYY-MM-DD`); timestamps are ISO-8601 datetimes in UTC.
 */

/** The top-level entity: the team whose matches and stats we track. */
export interface Team {
  id: string;
  name: string;
  /** Stable URL identifier, generated once from `name` (ADR-0006). */
  slug: string;
  createdAt: string;
}

/** A period during which a Team plays (e.g. "2026 Spring"). */
export interface Season {
  id: string;
  teamId: Team['id'];
  name: string;
  /** Unique within the team (ADR-0006). */
  slug: string;
  startDate: string;
  endDate: string;
  createdAt: string;
}

/** A grouping of matches within a Season — a tournament, league night, or scrimmage. */
export interface Event {
  id: string;
  seasonId: Season['id'];
  name: string;
  /** Unique within the season (ADR-0006). */
  slug: string;
  startDate: string;
  endDate: string;
  location: string;
}

/** A single game against one faceless opponent (ADR-0003). */
export interface Match {
  id: string;
  eventId: Event['id'];
  ourTeamId: Team['id'];
  /** The opponent is a name only: no roster, lineup, or events (ADR-0003). */
  opponentName: string;
  /** Unique within the event; derived from opponent name + scheduled date (ADR-0006). */
  slug: string;
  scheduledAt: string;
  pointCap: number;
  deleted: boolean;
  createdAt: string;
}

/** A person who plays, scoped to a Team. */
export interface Player {
  id: string;
  teamId: Team['id'];
  /**
   * One or more names: the first is the primary display name, the rest are
   * addressable aliases (nicknames, alternate spellings).
   */
  names: [string, ...string[]];
  /** Derived from `names[0]`; unique within the team (ADR-0006). */
  slug: string;
  deleted: boolean;
  createdAt: string;
}

/** A player's registration on a Team's roster for a Season (ADR-0005). */
export interface SeasonRosterEntry {
  id: string;
  seasonId: Season['id'];
  playerId: Player['id'];
  createdAt: string;
}

/**
 * A player's registration for a specific Event — the set eligible to field in
 * that event's matches (ADR-0005). The on-field lineup is derived, not stored.
 */
export interface EventRosterEntry {
  id: string;
  eventId: Event['id'];
  playerId: Player['id'];
  createdAt: string;
}
