package gameplay

import (
	"testing"

	"google.golang.org/protobuf/proto"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

func TestMatchEventFromJSONBuildsTypedPayload(t *testing.T) {
	event, err := MatchEventFromJSON([]byte(`{
		"id": "evt-score",
		"matchId": "match-1",
		"pointNumber": 2,
		"effectiveTs": 4000,
		"createdAt": "2026-03-14T09:00:04Z",
		"deleted": false,
		"type": "SCORE",
		"payload": {"playerId": "player-2", "assistPlayerId": "player-1"}
	}`))
	if err != nil {
		t.Fatalf("MatchEventFromJSON: %v", err)
	}

	if want := matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE; event.GetType() != want {
		t.Fatalf("type = %v, want %v", event.GetType(), want)
	}
	score := event.GetScore()
	if score == nil {
		t.Fatal("payload is not a ScorePayload")
	}
	if got, want := score.GetPlayerId(), "player-2"; got != want {
		t.Fatalf("playerId = %q, want %q", got, want)
	}
	if got, want := score.GetAssistPlayerId(), "player-1"; got != want {
		t.Fatalf("assistPlayerId = %q, want %q", got, want)
	}
}

func TestEventStorageColumnsUsesShortTypeAndCamelCasePayload(t *testing.T) {
	event := &matchesv1.MatchEvent{
		Type: matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP,
		Payload: &matchesv1.MatchEvent_SetLineup{
			SetLineup: &matchesv1.SetLineupPayload{PlayerIds: []string{"player-a", "player-b"}},
		},
	}

	eventType, payload, err := EventStorageColumns(event)
	if err != nil {
		t.Fatalf("EventStorageColumns: %v", err)
	}
	if want := "SET_LINEUP"; eventType != want {
		t.Fatalf("type = %q, want %q", eventType, want)
	}
	if want := `{"playerIds":["player-a","player-b"]}`; payload != want {
		t.Fatalf("payload = %q, want %q", payload, want)
	}
}

func TestEventStorageColumnsRejectsPayloadTypeMismatch(t *testing.T) {
	event := &matchesv1.MatchEvent{
		Type:    matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH,
		Payload: &matchesv1.MatchEvent_Drop{Drop: &matchesv1.DropPayload{PlayerId: "player-a"}},
	}

	if _, _, err := EventStorageColumns(event); err == nil {
		t.Fatal("expected an error for a mismatched payload, got nil")
	}
}

func TestEventStorageColumnsIsStableAcrossEncodes(t *testing.T) {
	event := &matchesv1.MatchEvent{
		Type: matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE,
		Payload: &matchesv1.MatchEvent_Score{
			Score: &matchesv1.ScorePayload{PlayerId: "player-b", AssistPlayerId: "player-a"},
		},
	}

	_, want, err := EventStorageColumns(event)
	if err != nil {
		t.Fatalf("EventStorageColumns: %v", err)
	}
	// protojson randomizes insignificant whitespace by design; the storage
	// encoding must be canonical anyway, because it is also the idempotency
	// comparison key.
	for attempt := 0; attempt < 50; attempt++ {
		_, payload, encodeErr := EventStorageColumns(event)
		if encodeErr != nil {
			t.Fatalf("EventStorageColumns attempt %d: %v", attempt, encodeErr)
		}
		if payload != want {
			t.Fatalf("payload differs across encodes: %q != %q", payload, want)
		}
	}
}

func TestSetPayloadFromStorageRoundTrips(t *testing.T) {
	original := &matchesv1.MatchEvent{
		Id:          "evt-sub",
		MatchId:     "match-1",
		PointNumber: 3,
		EffectiveTs: 9000,
		CreatedAt:   "2026-03-14T09:00:09Z",
		Type:        matchesv1.MatchEventType_MATCH_EVENT_TYPE_SUB_PLAYER,
		Payload: &matchesv1.MatchEvent_SubPlayer{
			SubPlayer: &matchesv1.SubPlayerPayload{PlayerOutId: "player-a", PlayerInId: "player-b"},
		},
	}

	eventType, payload, err := EventStorageColumns(original)
	if err != nil {
		t.Fatalf("EventStorageColumns: %v", err)
	}

	restored := &matchesv1.MatchEvent{
		Id:          original.GetId(),
		MatchId:     original.GetMatchId(),
		PointNumber: original.GetPointNumber(),
		EffectiveTs: original.GetEffectiveTs(),
		CreatedAt:   original.GetCreatedAt(),
	}
	if err := SetPayloadFromStorage(restored, eventType, payload); err != nil {
		t.Fatalf("SetPayloadFromStorage: %v", err)
	}

	if !proto.Equal(restored, original) {
		t.Fatalf("round trip mismatch\n got: %v\nwant: %v", restored, original)
	}
}

func TestPointStateToJSONUsesShortNamesAndZeroFilledStats(t *testing.T) {
	state := &matchesv1.PointDerivedState{
		View:        matchesv1.View_VIEW_SCORED,
		InitialRole: matchesv1.StartingRole_STARTING_ROLE_DEFENSE,
		Lineup:      []string{"player-a"},
		Thrower:     "",
		PlayerStats: map[string]*matchesv1.PlayerStats{
			"player-a": {Touches: 1, Goals: 1},
		},
	}

	actual, err := PointStateToJSON(state)
	if err != nil {
		t.Fatalf("PointStateToJSON: %v", err)
	}

	if want := "SCORED"; actual.View != want {
		t.Fatalf("view = %q, want %q", actual.View, want)
	}
	if want := "DEFENSE"; actual.InitialRole != want {
		t.Fatalf("initialRole = %q, want %q", actual.InitialRole, want)
	}
	if want := (FixturePlayerStats{Touches: 1, Goals: 1}); actual.PlayerStats["player-a"] != want {
		t.Fatalf("player-a stats = %#v, want %#v", actual.PlayerStats["player-a"], want)
	}
}
