import type { PlayerStats, PointDerivedState, View } from './state';

type Expect<T extends true> = T;

const zeroStats: PlayerStats = {
  touches: 0,
  completions: 0,
  drops: 0,
  turnovers: 0,
  blocks: 0,
  goals: 0,
  assists: 0,
};

const initialState: PointDerivedState = {
  view: 'LINEUP_SETUP',
  initialRole: 'OFFENSE',
  lineup: [],
  thrower: '',
  playerStats: {},
};

export const stateSample = { zeroStats, initialState } as const;

// All seven base stats are required and numeric; there are no optional fields (ADR-0002).
type _AllStatsPresent = Expect<
  'touches' | 'completions' | 'drops' | 'turnovers' | 'blocks' | 'goals' | 'assists' extends keyof PlayerStats
    ? keyof PlayerStats extends 'touches' | 'completions' | 'drops' | 'turnovers' | 'blocks' | 'goals' | 'assists'
      ? true
      : false
    : false
>;
type _StatsAreNumbers = Expect<PlayerStats[keyof PlayerStats] extends number ? true : false>;

// The view vocabulary is exactly the five named phases in GLOSSARY.md.
type _Views = Expect<
  View extends 'LINEUP_SETUP' | 'PULL_WAIT' | 'DEFENSE_ACTIVE' | 'OFFENSE_POSSESSION' | 'SCORED'
    ? ('LINEUP_SETUP' | 'PULL_WAIT' | 'DEFENSE_ACTIVE' | 'OFFENSE_POSSESSION' | 'SCORED') extends View
      ? true
      : false
    : false
>;

// State is zero-filled and never null: collections are empty, thrower is the empty string.
type _LineupNeverNull = Expect<PointDerivedState['lineup'] extends string[] ? true : false>;
type _ThrowerNeverNull = Expect<PointDerivedState['thrower'] extends string ? true : false>;
type _StatsNeverNull = Expect<
  PointDerivedState['playerStats'] extends Record<string, PlayerStats> ? true : false
>;

// @ts-expect-error a PlayerStats literal must include every base stat
const incompleteStats: PlayerStats = { touches: 0, goals: 1 };

// @ts-expect-error thrower uses "" for "nobody", never null
const nullThrower: PointDerivedState = { ...initialState, thrower: null };

// @ts-expect-error lineup is always an array, never null
const nullLineup: PointDerivedState = { ...initialState, lineup: null };

export const rejectedStateSamples = { incompleteStats, nullThrower, nullLineup } as const;
