import { create } from '@bufbuild/protobuf';
import { describe, expect, it } from 'vitest';

import {
	MatchEventSchema,
	MatchEventType,
	SetLineupPayloadSchema,
	StartingRole,
	TouchPayloadSchema,
	View,
} from './gen/mybadcoach/matches/v1/matches_pb';
import type { MatchEvent } from './gen/mybadcoach/matches/v1/matches_pb';
import { pointStateToJSON } from './mapping';
import { reducePoint } from './reducer';

const createdAt = '2026-03-14T09:00:00Z';

function touch(id: string, effectiveTs: number, playerId: string): MatchEvent {
	return create(MatchEventSchema, {
		id,
		matchId: 'match-1',
		pointNumber: 1n,
		effectiveTs: BigInt(effectiveTs),
		createdAt,
		deleted: false,
		type: MatchEventType.TOUCH,
		payload: { case: 'touch', value: create(TouchPayloadSchema, { playerId }) },
	});
}

function setLineup(id: string, effectiveTs: number, playerIds: string[]): MatchEvent {
	return create(MatchEventSchema, {
		id,
		matchId: 'match-1',
		pointNumber: 1n,
		effectiveTs: BigInt(effectiveTs),
		createdAt,
		deleted: false,
		type: MatchEventType.SET_LINEUP,
		payload: { case: 'setLineup', value: create(SetLineupPayloadSchema, { playerIds }) },
	});
}

describe('reducePoint', () => {
	it('starts an empty point in lineup setup with zero-filled state', () => {
		const state = reducePoint([], StartingRole.OFFENSE);

		expect(pointStateToJSON(state)).toEqual({
			view: 'LINEUP_SETUP',
			initialRole: 'OFFENSE',
			lineup: [],
			thrower: '',
			playerStats: {},
		});
		expect(state.view).toBe(View.LINEUP_SETUP);
	});

	it('records a defensive starting role without changing the initial view', () => {
		const state = reducePoint([], StartingRole.DEFENSE);

		expect(state.view).toBe(View.LINEUP_SETUP);
		expect(state.initialRole).toBe(StartingRole.DEFENSE);
	});

	it('applies SET_LINEUP events in effectiveTs order, then waits for the pull', () => {
		const state = reducePoint(
			[
				setLineup('evt-lineup-late', 2_000, ['player-c', 'player-d']),
				setLineup('evt-lineup-early', 1_000, ['player-a', 'player-b']),
			],
			StartingRole.OFFENSE,
		);

		expect(state.lineup).toEqual(['player-c', 'player-d']);
		expect(state.view).toBe(View.PULL_WAIT);
	});

	it('credits a TOUCH and records the thrower in offense possession', () => {
		const state = reducePoint([touch('evt-touch', 1_000, 'player-a')], StartingRole.OFFENSE);

		expect(state.thrower).toBe('player-a');
		expect(state.view).toBe(View.OFFENSE_POSSESSION);
		expect(state.playerStats['player-a']).toMatchObject({
			touches: 1n,
			completions: 0n,
			drops: 0n,
			turnovers: 0n,
			blocks: 0n,
			goals: 0n,
			assists: 0n,
		});
	});
});
