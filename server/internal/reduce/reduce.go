// Package reduce ports the shared point reducer to Go.
//
// It must stay behaviorally identical to the TypeScript implementation in
// packages/shared/src/: both are checked against the same language-neutral
// fixtures in the repo-root fixtures directory (ADR-0001).
package reduce

import (
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

// InitialPointState is the state every point starts in: lineup setup, an empty
// lineup, no thrower, and an empty (never nil) stat map.
func InitialPointState(initialRole matchesv1.StartingRole) *matchesv1.PointDerivedState {
	return &matchesv1.PointDerivedState{
		View:        matchesv1.View_VIEW_LINEUP_SETUP,
		InitialRole: initialRole,
		Lineup:      []string{},
		Thrower:     "",
		PlayerStats: map[string]*matchesv1.PlayerStats{},
	}
}

// CreditBaseStat increments one base stat for a player, zero-filling their
// entry on first credit. Mirrors creditBaseStat in
// packages/shared/src/reducer.ts.
func CreditBaseStat(
	playerStats map[string]*matchesv1.PlayerStats,
	playerID string,
	apply func(*matchesv1.PlayerStats),
) {
	stats := playerStats[playerID]
	if stats == nil {
		stats = &matchesv1.PlayerStats{}
		playerStats[playerID] = stats
	}
	apply(stats)
}

// ReducePoint folds a point's MatchEvent log into derived state.
//
// The log is filtered, sorted by effective time, and deduplicated by id before
// any event is applied (ADR-0001).
//
// All nine MatchEvent types are implemented in the MatchEventType declaration
// order, mirroring the TypeScript switch in
// packages/shared/src/reducer.ts. An event whose oneof payload does not match
// its declared type is skipped, matching the TypeScript reducer's
// payload-case guard.
//
//nolint:revive // "reduce.ReducePoint" mirrors the cross-language reducePoint contract (docs/architecture.md); "Point" alone would be ambiguous and diverge from the TS port.
func ReducePoint(events []*matchesv1.MatchEvent, initialRole matchesv1.StartingRole) *matchesv1.PointDerivedState {
	state := InitialPointState(initialRole)

	for _, event := range PrepareEvents(events) {
		switch event.GetType() {
		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH:
			touch := event.GetTouch()
			if touch == nil {
				continue
			}

			// A touch by a different player completes the previous throw. This
			// is the only way completions enter the log: there is no PASS
			// event. A same-player touch (e.g. regaining a tipped disc) is not
			// a pass.
			if state.Thrower != "" && state.Thrower != touch.GetPlayerId() {
				CreditBaseStat(state.PlayerStats, state.Thrower, func(stats *matchesv1.PlayerStats) {
					stats.Completions++
				})
			}

			CreditBaseStat(state.PlayerStats, touch.GetPlayerId(), func(stats *matchesv1.PlayerStats) {
				stats.Touches++
			})
			state.Thrower = touch.GetPlayerId()
			state.View = matchesv1.View_VIEW_OFFENSE_POSSESSION

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_DROP:
			drop := event.GetDrop()
			if drop == nil {
				continue
			}

			// A drop is a receiver fault: credit only the receiver, never the
			// thrower (no completion, no turnover). We no longer have the disc.
			CreditBaseStat(state.PlayerStats, drop.GetPlayerId(), func(stats *matchesv1.PlayerStats) {
				stats.Drops++
			})
			state.Thrower = ""
			state.View = matchesv1.View_VIEW_DEFENSE_ACTIVE

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_TURNOVER:
			turnover := event.GetTurnover()
			if turnover == nil {
				continue
			}

			// The event names the thrower responsible for the throwaway,
			// stall, or other thrower fault. The throw itself is not a
			// completion.
			CreditBaseStat(state.PlayerStats, turnover.GetPlayerId(), func(stats *matchesv1.PlayerStats) {
				stats.Turnovers++
			})
			state.Thrower = ""
			state.View = matchesv1.View_VIEW_DEFENSE_ACTIVE

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_DEFENSIVE_BLOCK:
			block := event.GetDefensiveBlock()
			if block == nil {
				continue
			}

			// Our team gains the disc. A block is not a completed throw, so
			// clear the thrower: the next TOUCH must not credit the blocker a
			// completion. If the blocker caught it, that TOUCH sets them as
			// thrower.
			CreditBaseStat(state.PlayerStats, block.GetPlayerId(), func(stats *matchesv1.PlayerStats) {
				stats.Blocks++
			})
			state.Thrower = ""
			state.View = matchesv1.View_VIEW_OFFENSE_POSSESSION

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_OPPONENT_TURNOVER:
			// Faceless opponent (ADR-0003): no player stats. Our team gains
			// the disc, and clearing the thrower prevents a stale player from
			// being credited a completion on the next touch.
			state.Thrower = ""
			state.View = matchesv1.View_VIEW_OFFENSE_POSSESSION

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE:
			score := event.GetScore()
			if score == nil {
				continue
			}

			// A goal credits only the goal and, when present, the assist. The
			// scorer's touch/completion, if recorded, came from a preceding
			// TOUCH event. The event is terminal for the point.
			CreditBaseStat(state.PlayerStats, score.GetPlayerId(), func(stats *matchesv1.PlayerStats) {
				stats.Goals++
			})

			// Callahan guard: "" means a defensive score (GLOSSARY:
			// Callahan). Never fall back to the current thrower for an assist
			// or completion, and never charge a turnover to anybody.
			if score.GetAssistPlayerId() != "" {
				CreditBaseStat(state.PlayerStats, score.GetAssistPlayerId(), func(stats *matchesv1.PlayerStats) {
					stats.Assists++
				})
			}

			state.Thrower = ""
			state.View = matchesv1.View_VIEW_SCORED

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP:
			lineup := event.GetSetLineup()
			if lineup == nil {
				continue
			}

			state.Lineup = append([]string{}, lineup.GetPlayerIds()...)
			state.View = matchesv1.View_VIEW_PULL_WAIT

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SUB_PLAYER:
			sub := event.GetSubPlayer()
			if sub == nil {
				continue
			}

			// Swap in place so the lineup keeps its size and field order. A
			// substitution credits no base stat. If the thrower leaves the
			// field, possession passes to the incoming player (USAU 7.C.3.a).
			for index, playerID := range state.Lineup {
				if playerID == sub.GetPlayerOutId() {
					state.Lineup[index] = sub.GetPlayerInId()
					break
				}
			}
			if state.Thrower == sub.GetPlayerOutId() {
				state.Thrower = sub.GetPlayerInId()
			}

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_PULL:
			// A pull is not one of the seven base stats. Our player pulled,
			// so this point is being played on defense. A pull is not a
			// completed pass: clear the thrower so the receiving touch
			// credits no completion.
			state.Thrower = ""
			state.View = matchesv1.View_VIEW_DEFENSE_ACTIVE

		case matchesv1.MatchEventType_MATCH_EVENT_TYPE_UNSPECIFIED:
			// Malformed event; skip it.
		}
	}

	return state
}
