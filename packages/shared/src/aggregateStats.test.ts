import { describe, expect, it } from 'vitest';

import {
  aggregateCareerStats,
  aggregateEventStats,
  aggregateMatchStats,
  computeTeamRecord,
} from './aggregateStats';
import type { PlayerStatLine } from './aggregateStats';
import type {
  MatchEvent,
  MatchEventPayloadMap,
  MatchEventType,
} from './events';
import { reducePoint } from './reducer';
import type { PlayerStats, PointDerivedState } from './state';
import { emptyPlayerStats } from './state';

/** Build a MatchEvent for the end-to-end aggregation test. */
function event<K extends MatchEventType>(
  id: string,
  effectiveTs: number,
  pointNumber: number,
  type: K,
  payload: MatchEventPayloadMap[K],
): MatchEvent {
  return {
    id,
    matchId: 'match-1',
    pointNumber,
    effectiveTs,
    createdAt: '2026-03-14T09:00:00Z',
    deleted: false,
    type,
    payload,
  } as MatchEvent;
}

/**
 * Build a reduced point for aggregation tests. Each player entry starts
 * zero-filled so tests only spell out the stats that matter to them.
 */
function reducedPoint(
  playerStats: Record<string, Partial<PlayerStats>>,
  overrides: Partial<PointDerivedState> = {},
): PointDerivedState {
  const complete: Record<string, PlayerStats> = {};
  for (const [playerId, stats] of Object.entries(playerStats)) {
    complete[playerId] = { ...emptyPlayerStats(), ...stats };
  }
  return {
    view: 'SCORED',
    initialRole: 'OFFENSE',
    lineup: [],
    thrower: '',
    playerStats: complete,
    ...overrides,
  };
}

describe('aggregateMatchStats', () => {
  it('returns an empty, never-null map for a match with no reduced points', () => {
    expect(aggregateMatchStats([])).toEqual({});
  });

  it('sums the seven base stats across points', () => {
    const stats = aggregateMatchStats([
      reducedPoint({ 'player-a': { touches: 2, completions: 1, turnovers: 1, goals: 1 } }),
      reducedPoint({ 'player-a': { touches: 1, drops: 1, blocks: 2, assists: 1 } }),
    ]);

    expect(stats['player-a']).toMatchObject({
      touches: 3,
      completions: 1,
      drops: 1,
      turnovers: 1,
      blocks: 2,
      goals: 1,
      assists: 1,
    });
  });

  it('zero-fills the base stats of a player who is absent from some points', () => {
    const stats = aggregateMatchStats([
      reducedPoint({ 'player-a': { touches: 2 }, 'player-b': { goals: 1 } }),
      reducedPoint({ 'player-a': { touches: 1 } }),
    ]);

    expect(stats['player-b']).toMatchObject({
      touches: 0,
      completions: 0,
      drops: 0,
      turnovers: 0,
      blocks: 0,
      goals: 1,
      assists: 0,
    });
  });

  it('includes credited players who were never in the lineup', () => {
    const stats = aggregateMatchStats([
      reducedPoint({ 'player-z': { touches: 1 } }, { lineup: ['player-a'] }),
    ]);

    expect(stats['player-z']).toMatchObject({ touches: 1 });
    expect(stats['player-a']).toBeUndefined();
  });

  it('derives receptions from touches, so a callahan goal is not a reception', () => {
    const stats = aggregateMatchStats([
      reducedPoint({ 'player-a': { touches: 3, completions: 2, goals: 1 } }),
      reducedPoint({ 'player-callahan': { goals: 1, blocks: 1 } }),
    ]);

    expect(stats['player-a']).toEqual({
      touches: 3,
      completions: 2,
      drops: 0,
      turnovers: 0,
      blocks: 0,
      goals: 1,
      assists: 0,
      receptions: 3,
      throwAttempts: 4,
      plusMinus: 1,
    });
    // A callahan is a goal with no logged TOUCH: goal only, no reception.
    expect(stats['player-callahan']).toEqual({
      touches: 0,
      completions: 0,
      drops: 0,
      turnovers: 0,
      blocks: 1,
      goals: 1,
      assists: 0,
      receptions: 0,
      throwAttempts: -1,
      plusMinus: 2,
    });
  });

  it('derives throw attempts as touches + completions - goals - turnovers', () => {
    const stats = aggregateMatchStats([
      reducedPoint({
        'player-a': { touches: 5, completions: 3, goals: 2, turnovers: 1 },
      }),
    ]);

    expect(stats['player-a']?.throwAttempts).toBe(5);
  });

  it('derives plus-minus as goals + assists + blocks - turnovers - drops', () => {
    const stats = aggregateMatchStats([
      reducedPoint({
        'player-a': {
          goals: 2,
          assists: 1,
          blocks: 3,
          turnovers: 2,
          drops: 1,
        },
      }),
    ]);

    expect(stats['player-a']?.plusMinus).toBe(3);
  });

  it('does not mutate the reduced points it aggregates', () => {
    const points = [
      reducedPoint({ 'player-a': { touches: 2, goals: 1 } }),
      reducedPoint({ 'player-a': { blocks: 1 } }),
    ];
    const snapshot = structuredClone(points);

    aggregateMatchStats(points);

    expect(points).toEqual(snapshot);
  });
});

describe('aggregateEventStats', () => {
  it('returns an empty, never-null map for an event with no matches', () => {
    expect(aggregateEventStats([])).toEqual({});
  });

  it('sums match stat lines across the matches in the event', () => {
    const eventStats = aggregateEventStats([
      aggregateMatchStats([
        reducedPoint({ 'player-a': { touches: 2, completions: 1, goals: 1 } }),
      ]),
      aggregateMatchStats([reducedPoint({ 'player-a': { touches: 1, assists: 1 } })]),
    ]);

    expect(eventStats['player-a']).toEqual({
      touches: 3,
      completions: 1,
      drops: 0,
      turnovers: 0,
      blocks: 0,
      goals: 1,
      assists: 1,
      receptions: 3,
      throwAttempts: 3,
      plusMinus: 2,
    });
  });

  it('recomputes derived stats from base totals instead of summing stored values', () => {
    // A caller-supplied line whose derived fields disagree with its base stats.
    const staleLine: PlayerStatLine = {
      touches: 2,
      completions: 1,
      drops: 0,
      turnovers: 0,
      blocks: 0,
      goals: 0,
      assists: 0,
      receptions: 0,
      throwAttempts: 0,
      plusMinus: 0,
    };

    const eventStats = aggregateEventStats([{ 'player-a': staleLine }]);

    expect(eventStats['player-a']?.receptions).toBe(2);
    expect(eventStats['player-a']?.throwAttempts).toBe(3);
    expect(eventStats['player-a']?.plusMinus).toBe(0);
  });
});

describe('aggregateCareerStats', () => {
  it('returns an empty, never-null map for a career with no matches', () => {
    expect(aggregateCareerStats([])).toEqual({});
  });

  it('sums match stat lines across every match and zero-fills absent players', () => {
    const career = aggregateCareerStats([
      aggregateMatchStats([reducedPoint({ 'player-a': { touches: 4 } })]),
      aggregateMatchStats([
        reducedPoint({ 'player-a': { touches: 2 }, 'player-b': { goals: 3 } }),
      ]),
    ]);

    expect(career['player-a']).toMatchObject({ touches: 6, receptions: 6 });
    expect(career['player-b']).toMatchObject({
      touches: 0,
      completions: 0,
      drops: 0,
      turnovers: 0,
      blocks: 0,
      goals: 3,
      assists: 0,
      receptions: 0,
      throwAttempts: -3,
      plusMinus: 3,
    });
  });
});

describe('aggregateMatchStats over reducer output', () => {
  it('rolls up a full match exactly as reducePoint credits it', () => {
    // Point 1: a throws to b, who scores with a's assist. The completion is
    // credited on b's touch; the assist on the SCORE.
    const point1 = [
      event('p1-lineup', 1_000, 1, 'SET_LINEUP', {
        playerIds: ['player-a', 'player-b'],
      }),
      event('p1-touch-a', 1_100, 1, 'TOUCH', { playerId: 'player-a' }),
      event('p1-touch-b', 1_200, 1, 'TOUCH', { playerId: 'player-b' }),
      event('p1-score', 1_300, 1, 'SCORE', {
        playerId: 'player-b',
        assistPlayerId: 'player-a',
      }),
    ];

    // Point 2: c blocks and scores unassisted — a callahan: goal only, no touch.
    const point2 = [
      event('p2-lineup', 2_000, 2, 'SET_LINEUP', { playerIds: ['player-c'] }),
      event('p2-block', 2_100, 2, 'DEFENSIVE_BLOCK', { playerId: 'player-c' }),
      event('p2-score', 2_200, 2, 'SCORE', {
        playerId: 'player-c',
        assistPlayerId: '',
      }),
    ];

    // Point 3: a catches an opponent turnover, then throws it away. The
    // cleared thrower means the catch credits no completion.
    const point3 = [
      event('p3-lineup', 3_000, 3, 'SET_LINEUP', { playerIds: ['player-a'] }),
      event('p3-opponent-turnover', 3_100, 3, 'OPPONENT_TURNOVER', {}),
      event('p3-touch-a', 3_200, 3, 'TOUCH', { playerId: 'player-a' }),
      event('p3-turnover-a', 3_300, 3, 'TURNOVER', { playerId: 'player-a' }),
    ];

    const stats = aggregateMatchStats([
      reducePoint(point1, 'OFFENSE'),
      reducePoint(point2, 'DEFENSE'),
      reducePoint(point3, 'OFFENSE'),
    ]);

    expect(stats).toEqual({
      'player-a': {
        touches: 2,
        completions: 1,
        drops: 0,
        turnovers: 1,
        blocks: 0,
        goals: 0,
        assists: 1,
        receptions: 2,
        throwAttempts: 2,
        plusMinus: 0,
      },
      'player-b': {
        touches: 1,
        completions: 0,
        drops: 0,
        turnovers: 0,
        blocks: 0,
        goals: 1,
        assists: 0,
        receptions: 1,
        throwAttempts: 0,
        plusMinus: 1,
      },
      'player-c': {
        touches: 0,
        completions: 0,
        drops: 0,
        turnovers: 0,
        blocks: 1,
        goals: 1,
        assists: 0,
        receptions: 0,
        // GLOSSARY's literal formula; a callahan has no throw attempts.
        throwAttempts: -1,
        plusMinus: 2,
      },
    });
  });
});

describe('computeTeamRecord', () => {
  it('returns a zeroed, never-null record for no completed matches', () => {
    expect(computeTeamRecord([])).toEqual({ wins: 0, losses: 0, ties: 0 });
  });

  it('counts wins, losses, and ties from final scores', () => {
    expect(
      computeTeamRecord([
        { ourScore: 15, opponentScore: 12 },
        { ourScore: 10, opponentScore: 15 },
        { ourScore: 13, opponentScore: 13 },
        { ourScore: 15, opponentScore: 14 },
      ]),
    ).toEqual({ wins: 2, losses: 1, ties: 1 });
  });
});
