import type { MatchEvent } from './events';
import { prepareEvents } from './helpers';
import type { PointDerivedState, StartingRole } from './state';
import { emptyPlayerStats, initialPointState } from './state';

/**
 * Fold a point's MatchEvent log into derived state.
 *
 * The log is filtered (deleted tombstones removed), sorted by effective time,
 * and deduplicated by id before any event is applied (ADR-0001).
 *
 * All nine MatchEvent types are implemented in the canonical
 * MATCH_EVENT_TYPES order, mirroring the Go switch in
 * server/internal/reduce/reduce.go.
 */
export function reducePoint(
  events: readonly MatchEvent[],
  initialRole: StartingRole,
): PointDerivedState {
  const state = initialPointState(initialRole);

  for (const event of prepareEvents(events)) {
    switch (event.type) {
      case 'TOUCH': {
        const receiverId = event.payload.playerId;
        // A catch by a different player completes the previous throw. This is
        // the only way completions enter the log: there is no PASS event.
        // A same-player touch (e.g. regaining a tipped disc) is not a pass.
        if (state.thrower !== '' && state.thrower !== receiverId) {
          const throwerStats = state.playerStats[state.thrower] ?? emptyPlayerStats();
          throwerStats.completions += 1;
          state.playerStats[state.thrower] = throwerStats;
        }
        const stats = state.playerStats[receiverId] ?? emptyPlayerStats();
        stats.touches += 1;
        state.playerStats[receiverId] = stats;
        state.thrower = receiverId;
        state.view = 'OFFENSE_POSSESSION';
        break;
      }
      case 'DROP': {
        // A drop is a receiver fault: credit only the receiver, never the
        // thrower (no completion, no turnover). We no longer have the disc.
        const stats = state.playerStats[event.payload.playerId] ?? emptyPlayerStats();
        stats.drops += 1;
        state.playerStats[event.payload.playerId] = stats;
        state.thrower = '';
        state.view = 'DEFENSE_ACTIVE';
        break;
      }
      case 'TURNOVER': {
        // The event names the thrower responsible for the throwaway, stall,
        // or other thrower fault. The throw itself is not a completion.
        const stats = state.playerStats[event.payload.playerId] ?? emptyPlayerStats();
        stats.turnovers += 1;
        state.playerStats[event.payload.playerId] = stats;
        state.thrower = '';
        state.view = 'DEFENSE_ACTIVE';
        break;
      }
      case 'DEFENSIVE_BLOCK': {
        // Our team gains the disc. A block is not a completed throw, so clear
        // the thrower: the next TOUCH must not credit the blocker a
        // completion. If the blocker caught it, that TOUCH sets them as
        // thrower.
        const stats = state.playerStats[event.payload.playerId] ?? emptyPlayerStats();
        stats.blocks += 1;
        state.playerStats[event.payload.playerId] = stats;
        state.thrower = '';
        state.view = 'OFFENSE_POSSESSION';
        break;
      }
      case 'OPPONENT_TURNOVER': {
        // Faceless opponent (ADR-0003): no player stats. Our team gains the
        // disc, and clearing the thrower prevents a stale player from being
        // credited a completion on the next touch.
        state.thrower = '';
        state.view = 'OFFENSE_POSSESSION';
        break;
      }
      case 'SCORE': {
        // A goal credits only the goal and, when present, the assist. The
        // scoring catch's touch/completion, if recorded, came from a
        // preceding TOUCH event. The event is terminal for the point.
        const scorerStats = state.playerStats[event.payload.playerId] ?? emptyPlayerStats();
        scorerStats.goals += 1;
        state.playerStats[event.payload.playerId] = scorerStats;

        // Callahan guard: "" means a defensive score (GLOSSARY: Callahan).
        // Never fall back to the current thrower for an assist or completion,
        // and never charge a turnover to anybody.
        if (event.payload.assistPlayerId !== '') {
          const assistStats = state.playerStats[event.payload.assistPlayerId] ?? emptyPlayerStats();
          assistStats.assists += 1;
          state.playerStats[event.payload.assistPlayerId] = assistStats;
        }

        state.thrower = '';
        state.view = 'SCORED';
        break;
      }
      case 'SET_LINEUP': {
        state.lineup = [...event.payload.playerIds];
        state.view = 'PULL_WAIT';
        break;
      }
      case 'SUB_PLAYER': {
        // Swap in place so the lineup keeps its size and field order. A
        // substitution credits no base stat. If the thrower leaves the
        // field, possession passes to the incoming player (USAU 7.C.3.a).
        const index = state.lineup.indexOf(event.payload.playerOutId);
        if (index !== -1) {
          state.lineup[index] = event.payload.playerInId;
        }
        if (state.thrower === event.payload.playerOutId) {
          state.thrower = event.payload.playerInId;
        }
        break;
      }
      case 'PULL': {
        // A pull is not one of the seven base stats. Our player pulled, so
        // this point is being played on defense. A pull is not a completed
        // pass: clear the thrower so the receiving touch credits no
        // completion.
        state.thrower = '';
        state.view = 'DEFENSE_ACTIVE';
        break;
      }
      default:
        break;
    }
  }

  return state;
}
