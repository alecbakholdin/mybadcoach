import type { MatchEvent, MatchEventType } from './events';

type Expect<T extends true> = T;

const base = {
  matchId: 'match-1',
  pointNumber: 1,
  effectiveTs: 1_000,
  createdAt: '2026-03-14T09:00:00Z',
  deleted: false,
} as const;

const touch: MatchEvent = {
  ...base,
  id: 'evt-touch',
  type: 'TOUCH',
  payload: { playerId: 'player-1' },
};

const drop: MatchEvent = {
  ...base,
  id: 'evt-drop',
  type: 'DROP',
  payload: { playerId: 'player-1' },
};

const turnover: MatchEvent = {
  ...base,
  id: 'evt-turnover',
  type: 'TURNOVER',
  payload: { playerId: 'player-1' },
};

const defensiveBlock: MatchEvent = {
  ...base,
  id: 'evt-block',
  type: 'DEFENSIVE_BLOCK',
  payload: { playerId: 'player-1' },
};

const opponentTurnover: MatchEvent = {
  ...base,
  id: 'evt-opponent-turnover',
  type: 'OPPONENT_TURNOVER',
  payload: {},
};

const score: MatchEvent = {
  ...base,
  id: 'evt-score',
  type: 'SCORE',
  payload: { playerId: 'player-1', assistPlayerId: 'player-2' },
};

const unassistedScore: MatchEvent = {
  ...base,
  id: 'evt-score-unassisted',
  type: 'SCORE',
  payload: { playerId: 'player-1', assistPlayerId: '' },
};

const setLineup: MatchEvent = {
  ...base,
  id: 'evt-lineup',
  type: 'SET_LINEUP',
  payload: { playerIds: ['player-1', 'player-2'] },
};

const subPlayer: MatchEvent = {
  ...base,
  id: 'evt-sub',
  type: 'SUB_PLAYER',
  payload: { playerOutId: 'player-1', playerInId: 'player-2' },
};

const pull: MatchEvent = {
  ...base,
  id: 'evt-pull',
  type: 'PULL',
  payload: { playerId: 'player-1' },
};

export const eventSchemaSample = {
  touch,
  drop,
  turnover,
  defensiveBlock,
  opponentTurnover,
  score,
  unassistedScore,
  setLineup,
  subPlayer,
  pull,
} as const;

// The union covers exactly the nine MatchEvent types, no more and no fewer.
type _AllTypesCovered = Expect<
  MatchEvent['type'] extends MatchEventType
    ? MatchEventType extends MatchEvent['type']
      ? true
      : false
    : false
>;

// @ts-expect-error TOUCH requires a playerId
const touchWithoutPlayer: MatchEvent = { ...base, id: 'evt-bad', type: 'TOUCH', payload: {} };

// @ts-expect-error OPPONENT_TURNOVER is faceless and carries no player attribution
const opponentTurnoverWithPlayer: MatchEvent = { ...base, id: 'evt-bad', type: 'OPPONENT_TURNOVER', payload: { playerId: 'player-1' } };

// @ts-expect-error SET_LINEUP payload uses playerIds, not playerId
const lineupWithWrongPayload: MatchEvent = { ...base, id: 'evt-bad', type: 'SET_LINEUP', payload: { playerId: 'player-1' } };

// @ts-expect-error SCORE must carry an assistPlayerId ("" when unassisted)
const scoreWithoutAssistField: MatchEvent = { ...base, id: 'evt-bad', type: 'SCORE', payload: { playerId: 'player-1' } };

export const rejectedSamples = {
  touchWithoutPlayer,
  opponentTurnoverWithPlayer,
  lineupWithWrongPayload,
  scoreWithoutAssistField,
} as const;
