/**
 * Derived point state.
 *
 * Everything here is computed from MatchEvents; nothing is persisted
 * (ADR-0004). State is zero-filled and never null: collections are always
 * present, numeric stats are always integers, and an absent Thrower is the
 * empty string rather than null/undefined. This keeps the TypeScript and Go
 * encodings byte-identical under canonical JSON.
 */

/** Whether we start a point on offense or defense. */
export const STARTING_ROLES = ['OFFENSE', 'DEFENSE'] as const;
export type StartingRole = (typeof STARTING_ROLES)[number];

/** The current phase of a point. */
export const VIEWS = [
  'LINEUP_SETUP',
  'PULL_WAIT',
  'DEFENSE_ACTIVE',
  'OFFENSE_POSSESSION',
  'SCORED',
] as const;
export type View = (typeof VIEWS)[number];

/**
 * The seven base stats credited by the reducer (ADR-0002). All fields are
 * required integers and must be zero-filled, never null.
 */
export interface PlayerStats {
  touches: number;
  completions: number;
  drops: number;
  turnovers: number;
  blocks: number;
  goals: number;
  assists: number;
}

/** Complete derived state for one point. */
export interface PointDerivedState {
  view: View;
  initialRole: StartingRole;
  /** Up-to-seven player ids on the field, in field order. */
  lineup: string[];
  /** Player id currently in possession; "" when nobody has the disc. */
  thrower: string;
  /** Per-player base stats keyed by player id; zero-filled entries only. */
  playerStats: Record<string, PlayerStats>;
}

/** A zero-filled PlayerStats value. */
export function emptyPlayerStats(): PlayerStats {
  return {
    touches: 0,
    completions: 0,
    drops: 0,
    turnovers: 0,
    blocks: 0,
    goals: 0,
    assists: 0,
  };
}

/**
 * The state every point starts in: no lineup, no thrower, no credited stats.
 * The first event of the point (typically SET_LINEUP) moves it forward.
 */
export function initialPointState(initialRole: StartingRole): PointDerivedState {
  return {
    view: 'LINEUP_SETUP',
    initialRole,
    lineup: [],
    thrower: '',
    playerStats: {},
  };
}
