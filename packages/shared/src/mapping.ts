import { create } from '@bufbuild/protobuf';

import {
  DefensiveBlockPayloadSchema,
  DropPayloadSchema,
  MatchEventSchema,
  MatchEventType,
  OpponentTurnoverPayloadSchema,
  PullPayloadSchema,
  ScorePayloadSchema,
  SetLineupPayloadSchema,
  StartingRole,
  SubPlayerPayloadSchema,
  TouchPayloadSchema,
  TurnoverPayloadSchema,
  View,
} from './gen/mybadcoach/matches/v1/matches_pb';
import type {
  MatchEvent,
  PlayerStats,
  PointDerivedState,
} from './gen/mybadcoach/matches/v1/matches_pb';

/**
 * Bridge between the language-neutral parity fixtures (hand-written JSON) and
 * the generated protobuf types.
 *
 * The fixtures write short enum names ("TOUCH", "LINEUP_SETUP", "OFFENSE") and
 * a single `payload` object whose shape is implied by `type`. The generated
 * types name enums after their proto values ("MATCH_EVENT_TYPE_TOUCH", ...)
 * and expose the payload as a oneof keyed by member name ("touch", ...).
 * Everything else (camelCase field names) already matches protojson.
 */

/** A player's seven base stats in the fixture's number form. */
export interface FixturePlayerStats {
  touches: number;
  completions: number;
  drops: number;
  turnovers: number;
  blocks: number;
  goals: number;
  assists: number;
}

/** A complete point state in the fixture's short-enum, number form. */
export interface FixturePointState {
  view: string;
  initialRole: string;
  lineup: string[];
  thrower: string;
  playerStats: Record<string, FixturePlayerStats>;
}

/** A MatchEvent in the fixture's language-neutral JSON shape. */
export interface FixtureEvent {
  id: string;
  matchId: string;
  pointNumber: number;
  effectiveTs: number;
  createdAt: string;
  deleted: boolean;
  type: string;
  payload: Record<string, unknown>;
}

const EVENT_TYPE_BY_SHORT_NAME: Readonly<Record<string, MatchEventType>> = {
  TOUCH: MatchEventType.TOUCH,
  DROP: MatchEventType.DROP,
  TURNOVER: MatchEventType.TURNOVER,
  DEFENSIVE_BLOCK: MatchEventType.DEFENSIVE_BLOCK,
  OPPONENT_TURNOVER: MatchEventType.OPPONENT_TURNOVER,
  SCORE: MatchEventType.SCORE,
  SET_LINEUP: MatchEventType.SET_LINEUP,
  SUB_PLAYER: MatchEventType.SUB_PLAYER,
  PULL: MatchEventType.PULL,
};

const VIEW_BY_SHORT_NAME: Readonly<Record<string, View>> = {
  LINEUP_SETUP: View.LINEUP_SETUP,
  PULL_WAIT: View.PULL_WAIT,
  DEFENSE_ACTIVE: View.DEFENSE_ACTIVE,
  OFFENSE_POSSESSION: View.OFFENSE_POSSESSION,
  SCORED: View.SCORED,
};

const STARTING_ROLE_BY_SHORT_NAME: Readonly<Record<string, StartingRole>> = {
  OFFENSE: StartingRole.OFFENSE,
  DEFENSE: StartingRole.DEFENSE,
};

/** Resolve a fixture's short event-type name to the generated enum value. */
export function matchEventTypeFromShortName(name: string): MatchEventType {
  const type = EVENT_TYPE_BY_SHORT_NAME[name];
  if (type === undefined) {
    throw new Error(`unknown MatchEvent type: ${JSON.stringify(name)}`);
  }
  return type;
}

/** Render a generated event type as the fixture's short name. */
export function matchEventTypeShortName(type: MatchEventType): string {
  for (const [shortName, value] of Object.entries(EVENT_TYPE_BY_SHORT_NAME)) {
    if (value === type) {
      return shortName;
    }
  }
  throw new Error(`unsupported MatchEvent type: ${type}`);
}

/** Resolve a fixture's short view name to the generated enum value. */
export function viewFromShortName(name: string): View {
  const view = VIEW_BY_SHORT_NAME[name];
  if (view === undefined) {
    throw new Error(`unknown View: ${JSON.stringify(name)}`);
  }
  return view;
}

/** Render a generated view as the fixture's short name. */
export function viewShortName(view: View): string {
  for (const [shortName, value] of Object.entries(VIEW_BY_SHORT_NAME)) {
    if (value === view) {
      return shortName;
    }
  }
  throw new Error(`unsupported View: ${view}`);
}

/** Resolve a fixture's short starting-role name to the generated enum value. */
export function startingRoleFromShortName(name: string): StartingRole {
  const role = STARTING_ROLE_BY_SHORT_NAME[name];
  if (role === undefined) {
    throw new Error(`unknown StartingRole: ${JSON.stringify(name)}`);
  }
  return role;
}

/** Render a generated starting role as the fixture's short name. */
export function startingRoleShortName(role: StartingRole): string {
  for (const [shortName, value] of Object.entries(STARTING_ROLE_BY_SHORT_NAME)) {
    if (value === role) {
      return shortName;
    }
  }
  throw new Error(`unsupported StartingRole: ${role}`);
}

/** Map fixture JSON into a generated MatchEvent, including the oneof payload. */
export function matchEventFromJSON(raw: unknown): MatchEvent {
  if (!isRecord(raw)) {
    throw new Error('MatchEvent JSON must be an object');
  }
  const type = matchEventTypeFromShortName(requiredString(raw, 'type'));
  const payload = isRecord(raw.payload) ? raw.payload : {};

  return create(MatchEventSchema, {
    id: requiredString(raw, 'id'),
    matchId: requiredString(raw, 'matchId'),
    pointNumber: BigInt(requiredInteger(raw, 'pointNumber')),
    effectiveTs: BigInt(requiredInteger(raw, 'effectiveTs')),
    createdAt: requiredString(raw, 'createdAt'),
    deleted: raw.deleted === true,
    type,
    payload: payloadFor(type, payload),
  });
}

/** Render a reduced point state in the fixture's short-enum, number form. */
export function pointStateToJSON(state: PointDerivedState): FixturePointState {
  const playerStats: Record<string, FixturePlayerStats> = {};
  for (const [playerId, stats] of Object.entries(state.playerStats)) {
    playerStats[playerId] = playerStatsToJSON(stats);
  }
  return {
    view: viewShortName(state.view),
    initialRole: startingRoleShortName(state.initialRole),
    lineup: [...state.lineup],
    thrower: state.thrower,
    playerStats,
  };
}

function playerStatsToJSON(stats: PlayerStats): FixturePlayerStats {
  return {
    touches: Number(stats.touches),
    completions: Number(stats.completions),
    drops: Number(stats.drops),
    turnovers: Number(stats.turnovers),
    blocks: Number(stats.blocks),
    goals: Number(stats.goals),
    assists: Number(stats.assists),
  };
}

function payloadFor(
  type: MatchEventType,
  payload: Record<string, unknown>,
): MatchEvent['payload'] {
  switch (type) {
    case MatchEventType.TOUCH: {
      const result: MatchEvent['payload'] = {
        case: 'touch',
        value: create(TouchPayloadSchema, {
          playerId: requiredString(payload, 'playerId'),
        }),
      };
      return result;
    }
    case MatchEventType.DROP: {
      const result: MatchEvent['payload'] = {
        case: 'drop',
        value: create(DropPayloadSchema, {
          playerId: requiredString(payload, 'playerId'),
        }),
      };
      return result;
    }
    case MatchEventType.TURNOVER: {
      const result: MatchEvent['payload'] = {
        case: 'turnover',
        value: create(TurnoverPayloadSchema, {
          playerId: requiredString(payload, 'playerId'),
        }),
      };
      return result;
    }
    case MatchEventType.DEFENSIVE_BLOCK: {
      const result: MatchEvent['payload'] = {
        case: 'defensiveBlock',
        value: create(DefensiveBlockPayloadSchema, {
          playerId: requiredString(payload, 'playerId'),
        }),
      };
      return result;
    }
    case MatchEventType.OPPONENT_TURNOVER: {
      const result: MatchEvent['payload'] = {
        case: 'opponentTurnover',
        value: create(OpponentTurnoverPayloadSchema, {}),
      };
      return result;
    }
    case MatchEventType.SCORE: {
      const result: MatchEvent['payload'] = {
        case: 'score',
        value: create(ScorePayloadSchema, {
          playerId: requiredString(payload, 'playerId'),
          assistPlayerId: optionalString(payload, 'assistPlayerId'),
        }),
      };
      return result;
    }
    case MatchEventType.SET_LINEUP: {
      const result: MatchEvent['payload'] = {
        case: 'setLineup',
        value: create(SetLineupPayloadSchema, {
          playerIds: stringArray(payload, 'playerIds'),
        }),
      };
      return result;
    }
    case MatchEventType.SUB_PLAYER: {
      const result: MatchEvent['payload'] = {
        case: 'subPlayer',
        value: create(SubPlayerPayloadSchema, {
          playerOutId: requiredString(payload, 'playerOutId'),
          playerInId: requiredString(payload, 'playerInId'),
        }),
      };
      return result;
    }
    case MatchEventType.PULL: {
      const result: MatchEvent['payload'] = {
        case: 'pull',
        value: create(PullPayloadSchema, {
          playerId: requiredString(payload, 'playerId'),
        }),
      };
      return result;
    }
    default:
      throw new Error(`unsupported MatchEvent type: ${type}`);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function requiredString(record: Record<string, unknown>, field: string): string {
  const value = record[field];
  if (typeof value !== 'string') {
    throw new Error(`${field} must be a string`);
  }
  return value;
}

function optionalString(record: Record<string, unknown>, field: string): string {
  const value = record[field];
  if (value === undefined) {
    return '';
  }
  if (typeof value !== 'string') {
    throw new Error(`${field} must be a string when present`);
  }
  return value;
}

function requiredInteger(record: Record<string, unknown>, field: string): number {
  const value = record[field];
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(`${field} must be an integer`);
  }
  return value;
}

function stringArray(record: Record<string, unknown>, field: string): string[] {
  const value = record[field];
  if (!Array.isArray(value) || !value.every((entry) => typeof entry === 'string')) {
    throw new Error(`${field} must be an array of strings`);
  }
  return [...value];
}
