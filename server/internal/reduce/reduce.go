package reduce

import "encoding/json"

// ReducePoint folds a point's MatchEvent log into derived state.
//
// The log is filtered, sorted by effective time, and deduplicated by id before
// any event is applied (ADR-0001).
//
// This is the tracer-bullet reducer: it implements only the behavior pinned by
// the three parity fixtures. Ticket 04 fills in the remaining event types.
func ReducePoint(events []MatchEvent, initialRole StartingRole) PointDerivedState {
	state := InitialPointState(initialRole)

	for _, event := range PrepareEvents(events) {
		switch event.Type {
		case EventTouch:
			var touch TouchPayload
			_ = json.Unmarshal(event.Payload, &touch)

			stats := state.PlayerStats[touch.PlayerID]
			stats.Touches++
			state.PlayerStats[touch.PlayerID] = stats
			state.Thrower = touch.PlayerID
			state.View = ViewOffensePossession

		case EventSetLineup:
			var lineup SetLineupPayload
			_ = json.Unmarshal(event.Payload, &lineup)

			state.Lineup = append([]string{}, lineup.PlayerIDs...)
			state.View = ViewPullWait

			// Ticket 04 adds the remaining seven MatchEvent types here, in
			// EventType declaration order, mirroring the TypeScript switch.
		}
	}

	return state
}
