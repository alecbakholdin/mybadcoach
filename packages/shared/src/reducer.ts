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
 * This is the tracer-bullet reducer: it implements only the behavior pinned by
 * the three parity fixtures. Ticket 04 fills in the remaining event types.
 */
export function reducePoint(
  events: readonly MatchEvent[],
  initialRole: StartingRole,
): PointDerivedState {
  const state = initialPointState(initialRole);

  for (const event of prepareEvents(events)) {
    switch (event.type) {
      case 'TOUCH': {
        const stats = state.playerStats[event.payload.playerId] ?? emptyPlayerStats();
        stats.touches += 1;
        state.playerStats[event.payload.playerId] = stats;
        state.thrower = event.payload.playerId;
        state.view = 'OFFENSE_POSSESSION';
        break;
      }
      case 'SET_LINEUP': {
        state.lineup = [...event.payload.playerIds];
        state.view = 'PULL_WAIT';
        break;
      }
      // Ticket 04 adds the remaining seven MatchEvent types here, in
      // MATCH_EVENT_TYPES order, mirroring the Go switch.
      default:
        break;
    }
  }

  return state;
}
