import { describe, expect, it } from 'vitest';

import type { MatchEvent } from './events';
import { reducePoint } from './reducer';

const createdAt = '2026-03-14T09:00:00Z';

function touch(id: string, effectiveTs: number, playerId: string): MatchEvent {
  return {
    id,
    matchId: 'match-1',
    pointNumber: 1,
    effectiveTs,
    createdAt,
    deleted: false,
    type: 'TOUCH',
    payload: { playerId },
  };
}

function setLineup(id: string, effectiveTs: number, playerIds: string[]): MatchEvent {
  return {
    id,
    matchId: 'match-1',
    pointNumber: 1,
    effectiveTs,
    createdAt,
    deleted: false,
    type: 'SET_LINEUP',
    payload: { playerIds },
  };
}

describe('reducePoint', () => {
  it('starts an empty point in lineup setup with zero-filled state', () => {
    const state = reducePoint([], 'OFFENSE');

    expect(state).toEqual({
      view: 'LINEUP_SETUP',
      initialRole: 'OFFENSE',
      lineup: [],
      thrower: '',
      playerStats: {},
    });
  });

  it('records a defensive starting role without changing the initial view', () => {
    const state = reducePoint([], 'DEFENSE');

    expect(state.view).toBe('LINEUP_SETUP');
    expect(state.initialRole).toBe('DEFENSE');
  });

  it('applies SET_LINEUP events in effectiveTs order, then waits for the pull', () => {
    const state = reducePoint(
      [
        setLineup('evt-lineup-late', 2_000, ['player-c', 'player-d']),
        setLineup('evt-lineup-early', 1_000, ['player-a', 'player-b']),
      ],
      'OFFENSE',
    );

    expect(state.lineup).toEqual(['player-c', 'player-d']);
    expect(state.view).toBe('PULL_WAIT');
  });

  it('credits a TOUCH and records the thrower in offense possession', () => {
    const state = reducePoint([touch('evt-touch', 1_000, 'player-a')], 'OFFENSE');

    expect(state.thrower).toBe('player-a');
    expect(state.view).toBe('OFFENSE_POSSESSION');
    expect(state.playerStats['player-a']).toEqual({
      touches: 1,
      completions: 0,
      drops: 0,
      turnovers: 0,
      blocks: 0,
      goals: 0,
      assists: 0,
    });
  });
});
