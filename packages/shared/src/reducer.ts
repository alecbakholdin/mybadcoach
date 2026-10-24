import { create } from '@bufbuild/protobuf';

import {
	MatchEventType,
	PlayerStatsSchema,
	PointDerivedStateSchema,
	View,
} from './gen/mybadcoach/matches/v1/matches_pb';
import type {
	MatchEvent,
	PlayerStats,
	PointDerivedState,
	StartingRole,
} from './gen/mybadcoach/matches/v1/matches_pb';
import { prepareEvents } from './helpers';

/** One of the seven base stats the reducer credits (ADR-0002). */
type BaseStatField =
	| 'touches'
	| 'completions'
	| 'drops'
	| 'turnovers'
	| 'blocks'
	| 'goals'
	| 'assists';

/** A zero-filled PlayerStats message. */
export function emptyPlayerStats(): PlayerStats {
	return create(PlayerStatsSchema);
}

/**
 * The state every point starts in: no lineup, no thrower, no credited stats.
 * The first event of the point (typically SET_LINEUP) moves it forward.
 */
export function initialPointState(initialRole: StartingRole): PointDerivedState {
	return create(PointDerivedStateSchema, {
		view: View.LINEUP_SETUP,
		initialRole,
		lineup: [],
		thrower: '',
		playerStats: {},
	});
}

/**
 * Credit one base stat to a player, zero-filling their entry on first credit.
 * Mirrors CreditBaseStat in server/internal/reduce/helpers.go.
 */
function creditBaseStat(state: PointDerivedState, playerId: string, field: BaseStatField): void {
	const stats = state.playerStats[playerId] ?? emptyPlayerStats();
	stats[field] += 1n;
	state.playerStats[playerId] = stats;
}

/**
 * Fold a point's MatchEvent log into derived state.
 *
 * The log is filtered (deleted tombstones removed), sorted by effective time,
 * and deduplicated by id before any event is applied (ADR-0001).
 *
 * All nine MatchEvent types are implemented in the canonical
 * MATCH_EVENT_TYPES order, mirroring the Go switch in
 * server/internal/reduce/reduce.go. An event whose oneof payload does not
 * match its type is ignored, matching the Go port.
 */
export function reducePoint(
	events: readonly MatchEvent[],
	initialRole: StartingRole,
): PointDerivedState {
	const state = initialPointState(initialRole);

	for (const event of prepareEvents(events)) {
		switch (event.type) {
			case MatchEventType.TOUCH: {
				if (event.payload.case !== 'touch') {
					break;
				}
				const receiverId = event.payload.value.playerId;
				// A touch by a different player completes the previous throw. This is
				// the only way completions enter the log: there is no PASS event.
				// A same-player touch (e.g. regaining a tipped disc) is not a pass.
				if (state.thrower !== '' && state.thrower !== receiverId) {
					creditBaseStat(state, state.thrower, 'completions');
				}
				creditBaseStat(state, receiverId, 'touches');
				state.thrower = receiverId;
				state.view = View.OFFENSE_POSSESSION;
				break;
			}
			case MatchEventType.DROP: {
				if (event.payload.case !== 'drop') {
					break;
				}
				// A drop is a receiver fault: credit only the receiver, never the
				// thrower (no completion, no turnover). We no longer have the disc.
				creditBaseStat(state, event.payload.value.playerId, 'drops');
				state.thrower = '';
				state.view = View.DEFENSE_ACTIVE;
				break;
			}
			case MatchEventType.TURNOVER: {
				if (event.payload.case !== 'turnover') {
					break;
				}
				// The event names the thrower responsible for the throwaway, stall,
				// or other thrower fault. The throw itself is not a completion.
				creditBaseStat(state, event.payload.value.playerId, 'turnovers');
				state.thrower = '';
				state.view = View.DEFENSE_ACTIVE;
				break;
			}
			case MatchEventType.DEFENSIVE_BLOCK: {
				if (event.payload.case !== 'defensiveBlock') {
					break;
				}
				// Our team gains the disc. A block is not a completed throw, so clear
				// the thrower: the next TOUCH must not credit the blocker a
				// completion. If the blocker caught it, that TOUCH sets them as
				// thrower.
				creditBaseStat(state, event.payload.value.playerId, 'blocks');
				state.thrower = '';
				state.view = View.OFFENSE_POSSESSION;
				break;
			}
			case MatchEventType.OPPONENT_TURNOVER: {
				if (event.payload.case !== 'opponentTurnover') {
					break;
				}
				// Faceless opponent (ADR-0003): no player stats. Our team gains the
				// disc, and clearing the thrower prevents a stale player from being
				// credited a completion on the next touch.
				state.thrower = '';
				state.view = View.OFFENSE_POSSESSION;
				break;
			}
			case MatchEventType.SCORE: {
				if (event.payload.case !== 'score') {
					break;
				}
				// A goal credits only the goal and, when present, the assist. The
				// scorer's touch/completion, if recorded, came from a preceding TOUCH
				// event. The event is terminal for the point.
				creditBaseStat(state, event.payload.value.playerId, 'goals');

				// Callahan guard: "" means a defensive score (GLOSSARY: Callahan).
				// Never fall back to the current thrower for an assist or completion,
				// and never charge a turnover to anybody.
				if (event.payload.value.assistPlayerId !== '') {
					creditBaseStat(state, event.payload.value.assistPlayerId, 'assists');
				}

				state.thrower = '';
				state.view = View.SCORED;
				break;
			}
			case MatchEventType.SET_LINEUP: {
				if (event.payload.case !== 'setLineup') {
					break;
				}
				state.lineup = [...event.payload.value.playerIds];
				state.view = View.PULL_WAIT;
				break;
			}
			case MatchEventType.SUB_PLAYER: {
				if (event.payload.case !== 'subPlayer') {
					break;
				}
				// Swap in place so the lineup keeps its size and field order. A
				// substitution credits no base stat. If the thrower leaves the
				// field, possession passes to the incoming player (USAU 7.C.3.a).
				const { playerInId, playerOutId } = event.payload.value;
				const index = state.lineup.indexOf(playerOutId);
				if (index !== -1) {
					state.lineup[index] = playerInId;
				}
				if (state.thrower === playerOutId) {
					state.thrower = playerInId;
				}
				break;
			}
			case MatchEventType.PULL: {
				if (event.payload.case !== 'pull') {
					break;
				}
				// A pull is not one of the seven base stats. Our player pulled, so
				// this point is being played on defense. A pull is not a completed
				// pass: clear the thrower so the receiving touch credits no
				// completion.
				state.thrower = '';
				state.view = View.DEFENSE_ACTIVE;
				break;
			}
			default:
				break;
		}
	}

	return state;
}
