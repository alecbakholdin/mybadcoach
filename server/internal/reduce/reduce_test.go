package reduce

import (
	"reflect"
	"testing"

	"google.golang.org/protobuf/proto"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

const createdAt = "2026-03-14T09:00:00Z"

func eventBase(id string, effectiveTS int64, eventType matchesv1.MatchEventType) *matchesv1.MatchEvent {
	return &matchesv1.MatchEvent{
		Id:          id,
		MatchId:     "match-1",
		PointNumber: 1,
		EffectiveTs: effectiveTS,
		CreatedAt:   createdAt,
		Deleted:     false,
		Type:        eventType,
	}
}

func touchEvent(id string, effectiveTS int64, playerID string) *matchesv1.MatchEvent {
	event := eventBase(id, effectiveTS, matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH)
	event.Payload = &matchesv1.MatchEvent_Touch{Touch: &matchesv1.TouchPayload{PlayerId: playerID}}
	return event
}

func lineupEvent(id string, effectiveTS int64, playerIDs ...string) *matchesv1.MatchEvent {
	event := eventBase(id, effectiveTS, matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP)
	event.Payload = &matchesv1.MatchEvent_SetLineup{SetLineup: &matchesv1.SetLineupPayload{PlayerIds: playerIDs}}
	return event
}

func TestReducePointStartsEmptyPointInLineupSetup(t *testing.T) {
	state := ReducePoint(nil, matchesv1.StartingRole_STARTING_ROLE_OFFENSE)

	want := &matchesv1.PointDerivedState{
		View:        matchesv1.View_VIEW_LINEUP_SETUP,
		InitialRole: matchesv1.StartingRole_STARTING_ROLE_OFFENSE,
		Lineup:      []string{},
		Thrower:     "",
		PlayerStats: map[string]*matchesv1.PlayerStats{},
	}
	if !proto.Equal(state, want) {
		t.Fatalf("ReducePoint(nil, OFFENSE) = %v, want %v", state, want)
	}
	if state.GetLineup() == nil {
		t.Fatal("lineup must be an empty array, never nil")
	}
	if state.GetPlayerStats() == nil {
		t.Fatal("playerStats must be an empty object, never nil")
	}
}

func TestReducePointRecordsDefensiveRole(t *testing.T) {
	state := ReducePoint(nil, matchesv1.StartingRole_STARTING_ROLE_DEFENSE)

	if want := matchesv1.View_VIEW_LINEUP_SETUP; state.GetView() != want {
		t.Fatalf("view = %q, want %q", state.GetView(), want)
	}
	if want := matchesv1.StartingRole_STARTING_ROLE_DEFENSE; state.GetInitialRole() != want {
		t.Fatalf("initialRole = %q, want %q", state.GetInitialRole(), want)
	}
}

func TestReducePointAppliesSetLineupInEffectiveTimeOrder(t *testing.T) {
	state := ReducePoint([]*matchesv1.MatchEvent{
		lineupEvent("evt-lineup-late", 2000, "player-c", "player-d"),
		lineupEvent("evt-lineup-early", 1000, "player-a", "player-b"),
	}, matchesv1.StartingRole_STARTING_ROLE_OFFENSE)

	if want := []string{"player-c", "player-d"}; !reflect.DeepEqual(state.GetLineup(), want) {
		t.Fatalf("lineup = %#v, want %#v", state.GetLineup(), want)
	}
	if want := matchesv1.View_VIEW_PULL_WAIT; state.GetView() != want {
		t.Fatalf("view = %q, want %q", state.GetView(), want)
	}
}

func TestReducePointCreditsTouch(t *testing.T) {
	state := ReducePoint(
		[]*matchesv1.MatchEvent{touchEvent("evt-touch", 1000, "player-a")},
		matchesv1.StartingRole_STARTING_ROLE_OFFENSE,
	)

	if want := "player-a"; state.GetThrower() != want {
		t.Fatalf("thrower = %q, want %q", state.GetThrower(), want)
	}
	if want := matchesv1.View_VIEW_OFFENSE_POSSESSION; state.GetView() != want {
		t.Fatalf("view = %q, want %q", state.GetView(), want)
	}
	stats := state.GetPlayerStats()["player-a"]
	if got, want := stats.GetTouches(), int64(1); got != want {
		t.Fatalf("player-a touches = %d, want %d", got, want)
	}
}

// A malformed event whose oneof payload does not match its declared type is
// skipped, matching the TypeScript reducer's payload-case guard.
func TestReducePointSkipsPayloadTypeMismatch(t *testing.T) {
	mismatched := eventBase("evt-mismatch", 1000, matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH)
	mismatched.Payload = &matchesv1.MatchEvent_Drop{Drop: &matchesv1.DropPayload{PlayerId: "player-a"}}

	state := ReducePoint(
		[]*matchesv1.MatchEvent{mismatched},
		matchesv1.StartingRole_STARTING_ROLE_OFFENSE,
	)

	if got := state.GetPlayerStats()["player-a"]; got != nil {
		t.Fatalf("player-a stats = %v, want no credit", got)
	}
	if want := matchesv1.View_VIEW_LINEUP_SETUP; state.GetView() != want {
		t.Fatalf("view = %q, want %q", state.GetView(), want)
	}
}
