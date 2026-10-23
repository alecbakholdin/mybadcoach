package gameplay

import (
	"testing"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

func pointState(stats map[string]*matchesv1.PlayerStats) *matchesv1.PointDerivedState {
	return &matchesv1.PointDerivedState{PlayerStats: stats}
}

func TestAggregateMatchStatsReturnsEmptySliceForNoPoints(t *testing.T) {
	lines := AggregateMatchStats(nil)

	if lines == nil {
		t.Fatal("lines must be an empty slice, never nil")
	}
	if len(lines) != 0 {
		t.Fatalf("got %d lines, want 0", len(lines))
	}
}

func TestAggregateMatchStatsSumsBaseStatsAcrossPoints(t *testing.T) {
	lines := AggregateMatchStats([]*matchesv1.PointDerivedState{
		pointState(map[string]*matchesv1.PlayerStats{
			"player-a": {Touches: 2, Completions: 1, Turnovers: 1, Goals: 1},
		}),
		pointState(map[string]*matchesv1.PlayerStats{
			"player-a": {Touches: 1, Drops: 1, Blocks: 2, Assists: 1},
		}),
	})

	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1", len(lines))
	}
	line := lines[0]
	if got, want := line.GetBase().GetTouches(), int64(3); got != want {
		t.Fatalf("touches = %d, want %d", got, want)
	}
	if got, want := line.GetBase().GetCompletions(), int64(1); got != want {
		t.Fatalf("completions = %d, want %d", got, want)
	}
	if got, want := line.GetBase().GetDrops(), int64(1); got != want {
		t.Fatalf("drops = %d, want %d", got, want)
	}
	if got, want := line.GetBase().GetTurnovers(), int64(1); got != want {
		t.Fatalf("turnovers = %d, want %d", got, want)
	}
	if got, want := line.GetBase().GetBlocks(), int64(2); got != want {
		t.Fatalf("blocks = %d, want %d", got, want)
	}
	if got, want := line.GetBase().GetGoals(), int64(1); got != want {
		t.Fatalf("goals = %d, want %d", got, want)
	}
	if got, want := line.GetBase().GetAssists(), int64(1); got != want {
		t.Fatalf("assists = %d, want %d", got, want)
	}
}

func TestAggregateMatchStatsZeroFillsAbsentPoints(t *testing.T) {
	lines := AggregateMatchStats([]*matchesv1.PointDerivedState{
		pointState(map[string]*matchesv1.PlayerStats{
			"player-a": {Touches: 2},
			"player-b": {Goals: 1},
		}),
		pointState(map[string]*matchesv1.PlayerStats{
			"player-a": {Touches: 1},
		}),
	})

	playerB := lineFor(t, lines, "player-b")
	if got, want := playerB.GetBase().GetTouches(), int64(0); got != want {
		t.Fatalf("player-b touches = %d, want %d", got, want)
	}
	if got, want := playerB.GetBase().GetGoals(), int64(1); got != want {
		t.Fatalf("player-b goals = %d, want %d", got, want)
	}
}

func TestAggregateMatchStatsDerivesReceptionsAndThrowAttempts(t *testing.T) {
	lines := AggregateMatchStats([]*matchesv1.PointDerivedState{
		pointState(map[string]*matchesv1.PlayerStats{
			"player-a": {Touches: 3, Completions: 2, Goals: 1},
			// A callahan: goal with no logged TOUCH.
			"player-callahan": {Goals: 1, Blocks: 1},
		}),
	})

	playerA := lineFor(t, lines, "player-a")
	if got, want := playerA.GetDerived().GetReceptions(), int64(3); got != want {
		t.Fatalf("player-a receptions = %d, want %d", got, want)
	}
	if got, want := playerA.GetDerived().GetThrowAttempts(), int64(4); got != want {
		t.Fatalf("player-a throwAttempts = %d, want %d", got, want)
	}
	if got, want := playerA.GetDerived().GetPlusMinus(), int64(1); got != want {
		t.Fatalf("player-a plusMinus = %d, want %d", got, want)
	}

	callahan := lineFor(t, lines, "player-callahan")
	if got, want := callahan.GetDerived().GetReceptions(), int64(0); got != want {
		t.Fatalf("callahan receptions = %d, want %d", got, want)
	}
	if got, want := callahan.GetDerived().GetThrowAttempts(), int64(-1); got != want {
		t.Fatalf("callahan throwAttempts = %d, want %d", got, want)
	}
	if got, want := callahan.GetDerived().GetPlusMinus(), int64(2); got != want {
		t.Fatalf("callahan plusMinus = %d, want %d", got, want)
	}
}

func TestAggregateMatchStatsSortsByPlayerID(t *testing.T) {
	lines := AggregateMatchStats([]*matchesv1.PointDerivedState{
		pointState(map[string]*matchesv1.PlayerStats{
			"player-z": {Touches: 1},
			"player-a": {Touches: 1},
		}),
	})

	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
	if lines[0].GetPlayerId() != "player-a" || lines[1].GetPlayerId() != "player-z" {
		t.Fatalf("order = %q, %q; want player-a, player-z",
			lines[0].GetPlayerId(), lines[1].GetPlayerId())
	}
}

func lineFor(t *testing.T, lines []*matchesv1.AggregatedPlayerStats, playerID string) *matchesv1.AggregatedPlayerStats {
	t.Helper()
	for _, line := range lines {
		if line.GetPlayerId() == playerID {
			return line
		}
	}
	t.Fatalf("no stat line for %q", playerID)
	return nil
}
