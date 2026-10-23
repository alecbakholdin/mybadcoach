/**
 * The MatchEvent schema: atomic, time-ordered, idempotent-by-id gameplay
 * events. Gameplay state is derived by folding the non-deleted events in
 * effective-time order (ADR-0001).
 *
 * This array is the canonical event-type order. The reducer switches over the
 * same order in TypeScript and Go so the two ports stay reviewable side by
 * side.
 */
export const MATCH_EVENT_TYPES = [
  'TOUCH',
  'DROP',
  'TURNOVER',
  'DEFENSIVE_BLOCK',
  'OPPONENT_TURNOVER',
  'SCORE',
  'SET_LINEUP',
  'SUB_PLAYER',
  'PULL',
] as const;

/** One of the nine atomic MatchEvent types. */
export type MatchEventType = (typeof MATCH_EVENT_TYPES)[number];

/** A player catches or possesses the disc. */
export interface TouchPayload {
  playerId: string;
}

/** The receiver drops a catchable throw. */
export interface DropPayload {
  playerId: string;
}

/** The thrower commits a turnover (throwaway, stall, or miscellaneous throw). */
export interface TurnoverPayload {
  playerId: string;
}

/** Our player blocks an opponent throw. */
export interface DefensiveBlockPayload {
  playerId: string;
}

/**
 * The faceless opponent turns the disc over (ADR-0003). There is no opponent
 * player to attribute, so the payload carries no fields.
 */
export type OpponentTurnoverPayload = Record<string, never>;

/** A goal: the scorer, plus the assister ("" when the goal was unassisted). */
export interface ScorePayload {
  playerId: string;
  assistPlayerId: string;
}

/** The up-to-seven players on the field. */
export interface SetLineupPayload {
  playerIds: string[];
}

/** One player leaves the field, another enters. */
export interface SubPlayerPayload {
  playerOutId: string;
  playerInId: string;
}

/** The player who pulled. */
export interface PullPayload {
  playerId: string;
}

/** Payload shape for each MatchEvent type, keyed by the event type. */
export interface MatchEventPayloadMap {
  TOUCH: TouchPayload;
  DROP: DropPayload;
  TURNOVER: TurnoverPayload;
  DEFENSIVE_BLOCK: DefensiveBlockPayload;
  OPPONENT_TURNOVER: OpponentTurnoverPayload;
  SCORE: ScorePayload;
  SET_LINEUP: SetLineupPayload;
  SUB_PLAYER: SubPlayerPayload;
  PULL: PullPayload;
}

/** Fields shared by every MatchEvent. */
export interface MatchEventBase {
  /** Unique event id; replayed events are idempotent by this id. */
  id: string;
  matchId: string;
  pointNumber: number;
  /**
   * Effective time as epoch milliseconds. Ordering is ascending `effectiveTs`,
   * with `id` as the deterministic tie-breaker.
   */
  effectiveTs: number;
  /** Wall-clock creation time (ISO-8601 UTC); informational, not used for ordering. */
  createdAt: string;
  /** Tombstone flag: deleted events are filtered out before reduction. */
  deleted: boolean;
}

type MatchEventFor<K extends MatchEventType> = MatchEventBase & {
  type: K;
  payload: MatchEventPayloadMap[K];
};

/** An atomic gameplay event, discriminated by `type` and `payload`. */
export type MatchEvent = { [K in MatchEventType]: MatchEventFor<K> }[MatchEventType];
