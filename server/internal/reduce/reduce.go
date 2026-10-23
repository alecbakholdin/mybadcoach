package reduce

import "encoding/json"

// ReducePoint folds a point's MatchEvent log into derived state.
//
// The log is filtered, sorted by effective time, and deduplicated by id before
// any event is applied (ADR-0001).
//
// All nine MatchEvent types are implemented in the EventType declaration
// order, mirroring the TypeScript switch in
// packages/shared/src/reducer.ts.
func ReducePoint(events []MatchEvent, initialRole StartingRole) PointDerivedState {
	state := InitialPointState(initialRole)

	for _, event := range PrepareEvents(events) {
		switch event.Type {
		case EventTouch:
			var touch TouchPayload
			_ = json.Unmarshal(event.Payload, &touch)

			// A catch by a different player completes the previous throw. This
			// is the only way completions enter the log: there is no PASS
			// event. A same-player touch (e.g. regaining a tipped disc) is not
			// a pass.
			if state.Thrower != "" && state.Thrower != touch.PlayerID {
				throwerStats := state.PlayerStats[state.Thrower]
				throwerStats.Completions++
				state.PlayerStats[state.Thrower] = throwerStats
			}

			stats := state.PlayerStats[touch.PlayerID]
			stats.Touches++
			state.PlayerStats[touch.PlayerID] = stats
			state.Thrower = touch.PlayerID
			state.View = ViewOffensePossession

		case EventDrop:
			var drop DropPayload
			_ = json.Unmarshal(event.Payload, &drop)

			// A drop is a receiver fault: credit only the receiver, never the
			// thrower (no completion, no turnover). We no longer have the disc.
			stats := state.PlayerStats[drop.PlayerID]
			stats.Drops++
			state.PlayerStats[drop.PlayerID] = stats
			state.Thrower = ""
			state.View = ViewDefenseActive

		case EventTurnover:
			var turnover TurnoverPayload
			_ = json.Unmarshal(event.Payload, &turnover)

			// The event names the thrower responsible for the throwaway,
			// stall, or other thrower fault. The throw itself is not a
			// completion.
			stats := state.PlayerStats[turnover.PlayerID]
			stats.Turnovers++
			state.PlayerStats[turnover.PlayerID] = stats
			state.Thrower = ""
			state.View = ViewDefenseActive

		case EventDefensiveBlock:
			var block DefensiveBlockPayload
			_ = json.Unmarshal(event.Payload, &block)

			// Our team gains the disc. A block is not a completed throw, so
			// clear the thrower: the next TOUCH must not credit the blocker a
			// completion. If the blocker caught it, that TOUCH sets them as
			// thrower.
			stats := state.PlayerStats[block.PlayerID]
			stats.Blocks++
			state.PlayerStats[block.PlayerID] = stats
			state.Thrower = ""
			state.View = ViewOffensePossession

		case EventOpponentTurnover:
			// Faceless opponent (ADR-0003): no player stats. Our team gains
			// the disc, and clearing the thrower prevents a stale player from
			// being credited a completion on the next touch.
			state.Thrower = ""
			state.View = ViewOffensePossession

		case EventScore:
			var score ScorePayload
			_ = json.Unmarshal(event.Payload, &score)

			// A goal credits only the goal and, when present, the assist. The
			// scoring catch's touch/completion, if recorded, came from a
			// preceding TOUCH event. The event is terminal for the point.
			scorerStats := state.PlayerStats[score.PlayerID]
			scorerStats.Goals++
			state.PlayerStats[score.PlayerID] = scorerStats

			// Callahan guard: "" means a defensive score (GLOSSARY:
			// Callahan). Never fall back to the current thrower for an assist
			// or completion, and never charge a turnover to anybody.
			if score.AssistPlayerID != "" {
				assistStats := state.PlayerStats[score.AssistPlayerID]
				assistStats.Assists++
				state.PlayerStats[score.AssistPlayerID] = assistStats
			}

			state.Thrower = ""
			state.View = ViewScored

		case EventSetLineup:
			var lineup SetLineupPayload
			_ = json.Unmarshal(event.Payload, &lineup)

			state.Lineup = append([]string{}, lineup.PlayerIDs...)
			state.View = ViewPullWait

		case EventSubPlayer:
			var sub SubPlayerPayload
			_ = json.Unmarshal(event.Payload, &sub)

			// Swap in place so the lineup keeps its size and field order. A
			// substitution credits no base stat. If the thrower leaves the
			// field, possession passes to the incoming player (USAU 7.C.3.a).
			for index, playerID := range state.Lineup {
				if playerID == sub.PlayerOutID {
					state.Lineup[index] = sub.PlayerInID
					break
				}
			}
			if state.Thrower == sub.PlayerOutID {
				state.Thrower = sub.PlayerInID
			}

		case EventPull:
			// A pull is not one of the seven base stats. Our player pulled,
			// so this point is being played on defense. A pull is not a
			// completed pass: clear the thrower so the receiving touch
			// credits no completion.
			state.Thrower = ""
			state.View = ViewDefenseActive
		}
	}

	return state
}
