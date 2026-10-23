export * from './aggregateStats';
export * from './entities';
export * from './helpers';
export * from './mapping';
export * from './reducer';
export * from './teamRecord';

// Generated gameplay types replace the former hand-written events.ts/state.ts.
// `Match` is intentionally not re-exported here: entities.ts already owns that
// name for the domain entity (the generated Match is a wire-only shape).
export {
  DefensiveBlockPayloadSchema,
  DerivedStatsSchema,
  DropPayloadSchema,
  MatchEventSchema,
  MatchEventType,
  MatchPointStateSchema,
  OpponentTurnoverPayloadSchema,
  PointDerivedStateSchema,
  PlayerStatsSchema,
  PullPayloadSchema,
  ScorePayloadSchema,
  SetLineupPayloadSchema,
  StartingRole,
  SubPlayerPayloadSchema,
  TouchPayloadSchema,
  TurnoverPayloadSchema,
  View,
} from './gen/mybadcoach/matches/v1/matches_pb';
export type {
  AggregatedPlayerStats,
  DefensiveBlockPayload,
  DerivedStats,
  DropPayload,
  MatchEvent,
  MatchPointState,
  OpponentTurnoverPayload,
  PlayerStats,
  PointDerivedState,
  PullPayload,
  ScorePayload,
  SetLineupPayload,
  SubPlayerPayload,
  TouchPayload,
  TurnoverPayload,
} from './gen/mybadcoach/matches/v1/matches_pb';
