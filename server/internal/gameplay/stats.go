package gameplay

import (
	"sort"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

// AggregateMatchStats folds a match's reduced points into per-player stat
// totals, sorted by player id for a deterministic response.
//
// It is the Go port of aggregateMatchStats in
// packages/shared/src/aggregateStats.ts: the seven base stats are summed and
// the three derived stats are recomputed from those totals (ADR-0002,
// ADR-0004). The returned slice is empty (never nil) when no player was
// credited.
func AggregateMatchStats(points []*matchesv1.PointDerivedState) []*matchesv1.AggregatedPlayerStats {
	// Start every player from a complete zero-filled line: a player credited
	// in one point but absent from another contributes zeros for that point.
	totals := make(map[string]*matchesv1.PlayerStats)
	for _, point := range points {
		for playerID, stats := range point.GetPlayerStats() {
			total, ok := totals[playerID]
			if !ok {
				total = &matchesv1.PlayerStats{}
				totals[playerID] = total
			}
			accumulateBaseStats(total, stats)
		}
	}

	playerIDs := make([]string, 0, len(totals))
	for playerID := range totals {
		playerIDs = append(playerIDs, playerID)
	}
	sort.Strings(playerIDs)

	lines := make([]*matchesv1.AggregatedPlayerStats, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		base := totals[playerID]
		lines = append(lines, &matchesv1.AggregatedPlayerStats{
			PlayerId: playerID,
			Base:     base,
			Derived:  deriveStats(base),
		})
	}
	return lines
}

// accumulateBaseStats adds one player's seven base stats into a running total.
func accumulateBaseStats(target, source *matchesv1.PlayerStats) {
	target.Touches += source.GetTouches()
	target.Completions += source.GetCompletions()
	target.Drops += source.GetDrops()
	target.Turnovers += source.GetTurnovers()
	target.Blocks += source.GetBlocks()
	target.Goals += source.GetGoals()
	target.Assists += source.GetAssists()
}

// deriveStats computes the three derived stats from base totals (ADR-0002).
func deriveStats(base *matchesv1.PlayerStats) *matchesv1.DerivedStats {
	return &matchesv1.DerivedStats{
		// receptions = touches. GLOSSARY defines Touch as "a player catching or
		// possessing the disc" — the action a reception names. A callahan is
		// logged as DEFENSIVE_BLOCK then unassisted SCORE with no TOUCH, so it
		// is goals=1, touches=0 and reports 0 receptions.
		Receptions: base.GetTouches(),
		// Formula pinned by GLOSSARY: touches + completions − goals − turnovers.
		// Kept literal (never clamped), so a callahan's algebraically negative
		// result is preserved rather than invented into a zero.
		ThrowAttempts: base.GetTouches() + base.GetCompletions() - base.GetGoals() - base.GetTurnovers(),
		// Formula pinned by GLOSSARY: goals + assists + blocks − turnovers − drops.
		PlusMinus: base.GetGoals() + base.GetAssists() + base.GetBlocks() - base.GetTurnovers() - base.GetDrops(),
	}
}
