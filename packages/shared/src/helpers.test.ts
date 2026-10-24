import { create } from '@bufbuild/protobuf';
import { describe, expect, it } from 'vitest';

import {
	MatchEventSchema,
	MatchEventType,
	TouchPayloadSchema,
} from './gen/mybadcoach/matches/v1/matches_pb';
import type { MatchEvent } from './gen/mybadcoach/matches/v1/matches_pb';
import { prepareEvents } from './helpers';

const createdAt = '2026-03-14T09:00:00Z';

function touch(id: string, effectiveTs: number, playerId = 'player-1'): MatchEvent {
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

function deletedTouch(id: string, effectiveTs: number): MatchEvent {
	return { ...touch(id, effectiveTs), deleted: true };
}

describe('prepareEvents', () => {
	it('sorts events by effectiveTs ascending', () => {
		const prepared = prepareEvents([
			touch('late', 2_000),
			touch('early', 1_000),
			touch('middle', 1_500),
		]);

		expect(prepared.map((event) => event.id)).toEqual(['early', 'middle', 'late']);
	});

	it('breaks equal effectiveTs ties by id', () => {
		const prepared = prepareEvents([touch('b', 1_000), touch('a', 1_000)]);

		expect(prepared.map((event) => event.id)).toEqual(['a', 'b']);
	});

	it('filters out deleted events', () => {
		const prepared = prepareEvents([deletedTouch('tombstone', 1_000), touch('live', 2_000)]);

		expect(prepared.map((event) => event.id)).toEqual(['live']);
	});

	it('deduplicates events by id, keeping the first after sorting', () => {
		const prepared = prepareEvents([touch('same-id', 2_000), touch('same-id', 1_000)]);

		expect(prepared).toHaveLength(1);
		expect(prepared[0]?.effectiveTs).toBe(1_000n);
	});
});
