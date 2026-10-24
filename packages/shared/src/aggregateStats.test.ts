import { create } from '@bufbuild/protobuf';
import { describe, expect, it } from 'vitest';

import { aggregateCareerStats, aggregateEventStats, aggregateMatchStats } from './aggregateStats';
import type { PlayerStatLine } from './aggregateStats';
import {
	DefensiveBlockPayloadSchema,
	MatchEventSchema,
	MatchEventType,
	OpponentTurnoverPayloadSchema,
	PlayerStatsSchema,
	PointDerivedStateSchema,
	ScorePayloadSchema,
	SetLineupPayloadSchema,
	StartingRole,
	TouchPayloadSchema,
	TurnoverPayloadSchema,
	View,
} from './gen/mybadcoach/matches/v1/matches_pb';
import type { MatchEvent, PointDerivedState } from './gen/mybadcoach/matches/v1/matches_pb';
import { reducePoint } from './reducer';
import { computeTeamRecord } from './teamRecord';

const createdAt = '2026-03-14T09:00:00Z';

type BaseStatName =
	| 'touches'
	| 'completions'
	| 'drops'
	| 'turnovers'
	| 'blocks'
	| 'goals'
	| 'assists';

/** Build a MatchEvent for the end-to-end aggregation test. */
function event(
	id: string,
	effectiveTs: number,
	pointNumber: number,
	type: MatchEventType,
	payload: MatchEvent['payload'],
): MatchEvent {
	return create(MatchEventSchema, {
		id,
		matchId: 'match-1',
		pointNumber: BigInt(pointNumber),
		effectiveTs: BigInt(effectiveTs),
		createdAt,
		deleted: false,
		type,
		payload,
	});
}

function lineupEvent(
	id: string,
	effectiveTs: number,
	pointNumber: number,
	playerIds: string[],
): MatchEvent {
	return event(id, effectiveTs, pointNumber, MatchEventType.SET_LINEUP, {
		case: 'setLineup',
		value: create(SetLineupPayloadSchema, { playerIds }),
	});
}

function touchEvent(
	id: string,
	effectiveTs: number,
	pointNumber: number,
	playerId: string,
): MatchEvent {
	return event(id, effectiveTs, pointNumber, MatchEventType.TOUCH, {
		case: 'touch',
		value: create(TouchPayloadSchema, { playerId }),
	});
}

function scoreEvent(
	id: string,
	effectiveTs: number,
	pointNumber: number,
	playerId: string,
	assistPlayerId: string,
): MatchEvent {
	return event(id, effectiveTs, pointNumber, MatchEventType.SCORE, {
		case: 'score',
		value: create(ScorePayloadSchema, { playerId, assistPlayerId }),
	});
}

function blockEvent(
	id: string,
	effectiveTs: number,
	pointNumber: number,
	playerId: string,
): MatchEvent {
	return event(id, effectiveTs, pointNumber, MatchEventType.DEFENSIVE_BLOCK, {
		case: 'defensiveBlock',
		value: create(DefensiveBlockPayloadSchema, { playerId }),
	});
}

function turnoverEvent(
	id: string,
	effectiveTs: number,
	pointNumber: number,
	playerId: string,
): MatchEvent {
	return event(id, effectiveTs, pointNumber, MatchEventType.TURNOVER, {
		case: 'turnover',
		value: create(TurnoverPayloadSchema, { playerId }),
	});
}

function opponentTurnoverEvent(id: string, effectiveTs: number, pointNumber: number): MatchEvent {
	return event(id, effectiveTs, pointNumber, MatchEventType.OPPONENT_TURNOVER, {
		case: 'opponentTurnover',
		value: create(OpponentTurnoverPayloadSchema, {}),
	});
}

/**
 * Build a reduced point for aggregation tests. Each player entry starts
 * zero-filled so tests only spell out the stats that matter to them.
 */
function reducedPoint(
	playerStats: Record<string, Partial<Record<BaseStatName, number>>>,
	overrides: {
		view?: View;
		initialRole?: StartingRole;
		lineup?: string[];
		thrower?: string;
	} = {},
): PointDerivedState {
	const complete: Record<string, ReturnType<typeof playerStatsMessage>> = {};
	for (const [playerId, stats] of Object.entries(playerStats)) {
		complete[playerId] = playerStatsMessage(stats);
	}
	return create(PointDerivedStateSchema, {
		view: overrides.view ?? View.SCORED,
		initialRole: overrides.initialRole ?? StartingRole.OFFENSE,
		lineup: overrides.lineup ?? [],
		thrower: overrides.thrower ?? '',
		playerStats: complete,
	});
}

function playerStatsMessage(stats: Partial<Record<BaseStatName, number>>) {
	return create(PlayerStatsSchema, {
		touches: BigInt(stats.touches ?? 0),
		completions: BigInt(stats.completions ?? 0),
		drops: BigInt(stats.drops ?? 0),
		turnovers: BigInt(stats.turnovers ?? 0),
		blocks: BigInt(stats.blocks ?? 0),
		goals: BigInt(stats.goals ?? 0),
		assists: BigInt(stats.assists ?? 0),
	});
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
			aggregateMatchStats([reducedPoint({ 'player-a': { touches: 2, completions: 1, goals: 1 } })]),
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
			aggregateMatchStats([reducedPoint({ 'player-a': { touches: 2 }, 'player-b': { goals: 3 } })]),
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
			lineupEvent('p1-lineup', 1_000, 1, ['player-a', 'player-b']),
			touchEvent('p1-touch-a', 1_100, 1, 'player-a'),
			touchEvent('p1-touch-b', 1_200, 1, 'player-b'),
			scoreEvent('p1-score', 1_300, 1, 'player-b', 'player-a'),
		];

		// Point 2: c blocks and scores unassisted — a callahan: goal only, no touch.
		const point2 = [
			lineupEvent('p2-lineup', 2_000, 2, ['player-c']),
			blockEvent('p2-block', 2_100, 2, 'player-c'),
			scoreEvent('p2-score', 2_200, 2, 'player-c', ''),
		];

		// Point 3: a catches an opponent turnover, then throws it away. The
		// cleared thrower means the touch credits no completion.
		const point3 = [
			lineupEvent('p3-lineup', 3_000, 3, ['player-a']),
			opponentTurnoverEvent('p3-opponent-turnover', 3_100, 3),
			touchEvent('p3-touch-a', 3_200, 3, 'player-a'),
			turnoverEvent('p3-turnover-a', 3_300, 3, 'player-a'),
		];

		const stats = aggregateMatchStats([
			reducePoint(point1, StartingRole.OFFENSE),
			reducePoint(point2, StartingRole.DEFENSE),
			reducePoint(point3, StartingRole.OFFENSE),
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
