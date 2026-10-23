package reduce

import (
	"encoding/json"
	"reflect"
	"testing"
)

func payload(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}

func touchEvent(id string, effectiveTs int64, playerID string) MatchEvent {
	return MatchEvent{
		ID:          id,
		MatchID:     "match-1",
		PointNumber: 1,
		EffectiveTs: effectiveTs,
		CreatedAt:   "2026-03-14T09:00:00Z",
		Deleted:     false,
		Type:        EventTouch,
		Payload:     payload(TouchPayload{PlayerID: playerID}),
	}
}

func lineupEvent(id string, effectiveTs int64, playerIDs ...string) MatchEvent {
	return MatchEvent{
		ID:          id,
		MatchID:     "match-1",
		PointNumber: 1,
		EffectiveTs: effectiveTs,
		CreatedAt:   "2026-03-14T09:00:00Z",
		Deleted:     false,
		Type:        EventSetLineup,
		Payload:     payload(SetLineupPayload{PlayerIDs: playerIDs}),
	}
}

func TestReducePointStartsEmptyPointInLineupSetup(t *testing.T) {
	state := ReducePoint(nil, Offense)

	want := PointDerivedState{
		View:        ViewLineupSetup,
		InitialRole: Offense,
		Lineup:      []string{},
		Thrower:     "",
		PlayerStats: map[string]PlayerStats{},
	}
	if !reflect.DeepEqual(state, want) {
		t.Fatalf("ReducePoint(nil, OFFENSE) = %#v, want %#v", state, want)
	}
	if state.Lineup == nil {
		t.Fatal("lineup must be an empty array, never nil")
	}
	if state.PlayerStats == nil {
		t.Fatal("playerStats must be an empty object, never nil")
	}
}

func TestReducePointRecordsDefensiveRole(t *testing.T) {
	state := ReducePoint(nil, Defense)

	if state.View != ViewLineupSetup {
		t.Fatalf("view = %q, want %q", state.View, ViewLineupSetup)
	}
	if state.InitialRole != Defense {
		t.Fatalf("initialRole = %q, want %q", state.InitialRole, Defense)
	}
}

func TestReducePointAppliesSetLineupInEffectiveTimeOrder(t *testing.T) {
	state := ReducePoint([]MatchEvent{
		lineupEvent("evt-lineup-late", 2000, "player-c", "player-d"),
		lineupEvent("evt-lineup-early", 1000, "player-a", "player-b"),
	}, Offense)

	if want := []string{"player-c", "player-d"}; !reflect.DeepEqual(state.Lineup, want) {
		t.Fatalf("lineup = %#v, want %#v", state.Lineup, want)
	}
	if state.View != ViewPullWait {
		t.Fatalf("view = %q, want %q", state.View, ViewPullWait)
	}
}

func TestReducePointCreditsTouch(t *testing.T) {
	state := ReducePoint([]MatchEvent{touchEvent("evt-touch", 1000, "player-a")}, Offense)

	if state.Thrower != "player-a" {
		t.Fatalf("thrower = %q, want %q", state.Thrower, "player-a")
	}
	if state.View != ViewOffensePossession {
		t.Fatalf("view = %q, want %q", state.View, ViewOffensePossession)
	}
	want := PlayerStats{Touches: 1}
	if stats := state.PlayerStats["player-a"]; stats != want {
		t.Fatalf("player-a stats = %#v, want %#v", stats, want)
	}
}
