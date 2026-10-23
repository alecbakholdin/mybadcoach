package api_test

import (
	"testing"

	"connectrpc.com/connect"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
	teamsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1"
)

// gameplayFixture is a team/season/event/match with a signed-in actor.
type gameplayFixture struct {
	env        *structureEnv
	actor      *structureActor
	teamSlug   string
	seasonSlug string
	eventSlug  string
	matchSlug  string
	matchID    string
}

func newGameplayFixture(t *testing.T) gameplayFixture {
	t.Helper()

	env := newStructureEnv(t)
	actor := env.newActor("coach@example.com")
	team := createTeam(t, actor, "Flick City")
	season := createSeason(t, actor, team.GetSlug(), "Spring")
	event := createEvent(t, actor, team.GetSlug(), season.GetSlug(), "League Night")
	match := createMatch(t, actor, team.GetSlug(), season.GetSlug(), event.GetSlug(), "Rival")
	return gameplayFixture{
		env:        env,
		actor:      actor,
		teamSlug:   team.GetSlug(),
		seasonSlug: season.GetSlug(),
		eventSlug:  event.GetSlug(),
		matchSlug:  match.GetSlug(),
		matchID:    match.GetId(),
	}
}

// appendEvents appends to the fixture's match and fails the test on error.
func (f *gameplayFixture) appendEvents(t *testing.T, events ...*matchesv1.MatchEvent) *matchesv1.AppendMatchEventsResponse {
	t.Helper()

	response, err := f.actor.matches.AppendMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
			TeamSlug:   f.teamSlug,
			SeasonSlug: f.seasonSlug,
			EventSlug:  f.eventSlug,
			MatchSlug:  f.matchSlug,
			Events:     events,
		}),
	)
	if err != nil {
		t.Fatalf("AppendMatchEvents: %v", err)
	}
	return response.Msg
}

func touchEvent(id string, point, ts int64, playerID string) *matchesv1.MatchEvent {
	return &matchesv1.MatchEvent{
		Id:          id,
		PointNumber: point,
		EffectiveTs: ts,
		CreatedAt:   "2026-03-14T09:00:00Z",
		Type:        matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH,
		Payload:     &matchesv1.MatchEvent_Touch{Touch: &matchesv1.TouchPayload{PlayerId: playerID}},
	}
}

func lineupEvent(id string, point, ts int64, playerIDs ...string) *matchesv1.MatchEvent {
	return &matchesv1.MatchEvent{
		Id:          id,
		PointNumber: point,
		EffectiveTs: ts,
		CreatedAt:   "2026-03-14T09:00:00Z",
		Type:        matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP,
		Payload: &matchesv1.MatchEvent_SetLineup{
			SetLineup: &matchesv1.SetLineupPayload{PlayerIds: playerIDs},
		},
	}
}

func scoreEvent(id string, point, ts int64, playerID, assistID string) *matchesv1.MatchEvent {
	return &matchesv1.MatchEvent{
		Id:          id,
		PointNumber: point,
		EffectiveTs: ts,
		CreatedAt:   "2026-03-14T09:00:00Z",
		Type:        matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE,
		Payload: &matchesv1.MatchEvent_Score{
			Score: &matchesv1.ScorePayload{PlayerId: playerID, AssistPlayerId: assistID},
		},
	}
}

func pullEvent(id string, point, ts int64, playerID string) *matchesv1.MatchEvent {
	return &matchesv1.MatchEvent{
		Id:          id,
		PointNumber: point,
		EffectiveTs: ts,
		CreatedAt:   "2026-03-14T09:00:00Z",
		Type:        matchesv1.MatchEventType_MATCH_EVENT_TYPE_PULL,
		Payload:     &matchesv1.MatchEvent_Pull{Pull: &matchesv1.PullPayload{PlayerId: playerID}},
	}
}

func TestAppendMatchEventsIsIdempotentAndStoresShortNames(t *testing.T) {
	fixture := newGameplayFixture(t)

	events := []*matchesv1.MatchEvent{
		lineupEvent("evt-lineup", 1, 1000, "player-a", "player-b"),
		touchEvent("evt-touch", 1, 2000, "player-a"),
		scoreEvent("evt-score", 1, 3000, "player-b", "player-a"),
	}
	first := fixture.appendEvents(t, events...)
	if got, want := first.GetAcceptedIds(), []string{"evt-lineup", "evt-touch", "evt-score"}; !equalStrings(got, want) {
		t.Fatalf("accepted = %v, want %v", got, want)
	}
	if len(first.GetSkippedDuplicateIds()) != 0 {
		t.Fatalf("skipped = %v, want none", first.GetSkippedDuplicateIds())
	}

	// Replaying the same batch is a no-op, not an error.
	second := fixture.appendEvents(t, events...)
	if len(second.GetAcceptedIds()) != 0 {
		t.Fatalf("accepted on replay = %v, want none", second.GetAcceptedIds())
	}
	if got, want := second.GetSkippedDuplicateIds(), []string{"evt-lineup", "evt-touch", "evt-score"}; !equalStrings(got, want) {
		t.Fatalf("skipped on replay = %v, want %v", got, want)
	}
	if count := countRows(t, fixture.env.db, "SELECT COUNT(*) FROM match_events"); count != 3 {
		t.Fatalf("match_events = %d, want 3", count)
	}

	// Storage encoding: short type name plus camelCase protojson payload.
	var eventType, payload string
	if err := fixture.env.db.QueryRow(
		"SELECT type, payload FROM match_events WHERE id = ?",
		"evt-lineup",
	).Scan(&eventType, &payload); err != nil {
		t.Fatalf("read stored event: %v", err)
	}
	if want := "SET_LINEUP"; eventType != want {
		t.Fatalf("stored type = %q, want %q", eventType, want)
	}
	if want := `{"playerIds":["player-a","player-b"]}`; payload != want {
		t.Fatalf("stored payload = %q, want %q", payload, want)
	}
}

func TestAppendMatchEventsRejectsIDReuseWithDifferentContent(t *testing.T) {
	fixture := newGameplayFixture(t)

	fixture.appendEvents(t, touchEvent("evt-touch", 1, 1000, "player-a"))

	_, err := fixture.actor.matches.AppendMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
			Events:     []*matchesv1.MatchEvent{touchEvent("evt-touch", 1, 1000, "player-b")},
		}),
	)
	assertConnectError(t, err, connect.CodeInvalidArgument, "id_conflict")
	assertFieldViolation(t, err, "events[0].id")

	// The original event is untouched.
	list := listEvents(t, &fixture)
	if len(list.GetEvents()) != 1 || list.GetEvents()[0].GetTouch().GetPlayerId() != "player-a" {
		t.Fatalf("events = %v, want the original player-a touch", list.GetEvents())
	}
}

func TestAppendMatchEventsTombstoneLetsACorrectionReplaceAnEvent(t *testing.T) {
	fixture := newGameplayFixture(t)

	fixture.appendEvents(t, touchEvent("evt-touch", 1, 1000, "player-a"))
	// A correction is composed client-side: tombstone the old event, append the
	// corrected one with a fresh id.
	fixture.appendEvents(t, deletedTouchEvent("evt-touch", 1, 1000, "player-a"))
	fixture.appendEvents(t, touchEvent("evt-touch-fixed", 1, 2000, "player-b"))

	list := listEvents(t, &fixture)
	if got, want := list.GetTotal(), int64(2); got != want {
		t.Fatalf("total = %d, want %d (tombstones are listable)", got, want)
	}
	if !list.GetEvents()[0].GetDeleted() {
		t.Fatal("first event should be the listed tombstone")
	}

	state := matchState(t, &fixture)
	if len(state.GetPoints()) != 1 {
		t.Fatalf("points = %d, want 1", len(state.GetPoints()))
	}
	stats := state.GetPoints()[0].GetState().GetPlayerStats()
	if stats["player-a"] != nil {
		t.Fatalf("tombstoned touch still credited to player-a: %v", stats["player-a"])
	}
	if got, want := stats["player-b"].GetTouches(), int64(1); got != want {
		t.Fatalf("player-b touches = %d, want %d", got, want)
	}
}

func TestAppendMatchEventsValidatesTheWholeBatchBeforeWriting(t *testing.T) {
	fixture := newGameplayFixture(t)

	malformed := &matchesv1.MatchEvent{
		Id:          "evt-malformed",
		PointNumber: 1,
		EffectiveTs: 1000,
		Type:        matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH,
		// The declared type is TOUCH but the oneof carries a DROP payload.
		Payload: &matchesv1.MatchEvent_Drop{Drop: &matchesv1.DropPayload{PlayerId: "player-a"}},
	}
	_, err := fixture.actor.matches.AppendMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
			Events: []*matchesv1.MatchEvent{
				touchEvent("evt-ok", 1, 1000, "player-a"),
				malformed,
			},
		}),
	)
	assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
	assertFieldViolation(t, err, "events[1]")

	if count := countRows(t, fixture.env.db, "SELECT COUNT(*) FROM match_events"); count != 0 {
		t.Fatalf("match_events = %d, want 0 (no partial append)", count)
	}

	_, err = fixture.actor.matches.AppendMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
			Events:     []*matchesv1.MatchEvent{{PointNumber: 1, Type: matchesv1.MatchEventType_MATCH_EVENT_TYPE_PULL}},
		}),
	)
	assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
	assertFieldViolation(t, err, "events[0].id")
}

func TestAppendMatchEventsRejectsMoreThanOneBatchCap(t *testing.T) {
	fixture := newGameplayFixture(t)

	events := make([]*matchesv1.MatchEvent, 1001)
	for index := range events {
		events[index] = touchEvent("evt-"+string(rune('a'+index%26)), 1, int64(index), "player-a")
	}
	_, err := fixture.actor.matches.AppendMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
			Events:     events,
		}),
	)
	assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
	assertFieldViolation(t, err, "events")
}

func TestAppendMatchEventsRejectsAnEventForAnotherMatch(t *testing.T) {
	fixture := newGameplayFixture(t)

	foreign := touchEvent("evt-foreign", 1, 1000, "player-a")
	foreign.MatchId = "another-match"
	_, err := fixture.actor.matches.AppendMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
			Events:     []*matchesv1.MatchEvent{foreign},
		}),
	)
	assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
	assertFieldViolation(t, err, "events[0].match_id")
}

func TestListMatchEventsFiltersByPointAndPaginates(t *testing.T) {
	fixture := newGameplayFixture(t)

	fixture.appendEvents(t,
		touchEvent("point-1-a", 1, 1000, "player-a"),
		touchEvent("point-2-a", 2, 2000, "player-b"),
		touchEvent("point-2-b", 2, 3000, "player-c"),
		touchEvent("point-2-c", 2, 4000, "player-d"),
	)

	all := listEvents(t, &fixture)
	if got, want := all.GetTotal(), int64(4); got != want {
		t.Fatalf("total = %d, want %d", got, want)
	}
	if got, want := eventIDs(all.GetEvents()), []string{"point-1-a", "point-2-a", "point-2-b", "point-2-c"}; !equalStrings(got, want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}

	pointTwo, err := fixture.actor.matches.ListMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.ListMatchEventsRequest{
			TeamSlug:    fixture.teamSlug,
			SeasonSlug:  fixture.seasonSlug,
			EventSlug:   fixture.eventSlug,
			MatchSlug:   fixture.matchSlug,
			PointNumber: ptrTo[int64](2),
		}),
	)
	if err != nil {
		t.Fatalf("ListMatchEvents(point 2): %v", err)
	}
	if got, want := pointTwo.Msg.GetTotal(), int64(3); got != want {
		t.Fatalf("point 2 total = %d, want %d", got, want)
	}
	if got, want := eventIDs(pointTwo.Msg.GetEvents()), []string{"point-2-a", "point-2-b", "point-2-c"}; !equalStrings(got, want) {
		t.Fatalf("point 2 ids = %v, want %v", got, want)
	}

	page, err := fixture.actor.matches.ListMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.ListMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
			Offset:     1,
			Limit:      2,
		}),
	)
	if err != nil {
		t.Fatalf("ListMatchEvents(page): %v", err)
	}
	if got, want := eventIDs(page.Msg.GetEvents()), []string{"point-2-a", "point-2-b"}; !equalStrings(got, want) {
		t.Fatalf("page ids = %v, want %v", got, want)
	}
	if got, want := page.Msg.GetTotal(), int64(4); got != want {
		t.Fatalf("paged total = %d, want the unpaginated %d", got, want)
	}
}

func TestGetMatchStateFoldsEachPointInOrder(t *testing.T) {
	fixture := newGameplayFixture(t)

	fixture.appendEvents(t,
		lineupEvent("p1-lineup", 1, 1000, "player-a", "player-b"),
		touchEvent("p1-touch-a", 1, 2000, "player-a"),
		touchEvent("p1-touch-b", 1, 3000, "player-b"),
		scoreEvent("p1-score", 1, 4000, "player-b", "player-a"),
		pullEvent("p2-pull", 2, 5000, "player-c"),
		touchEvent("p2-touch-c", 2, 6000, "player-c"),
	)

	state := matchState(t, &fixture)
	points := state.GetPoints()
	if len(points) != 2 {
		t.Fatalf("points = %d, want 2", len(points))
	}

	first := points[0]
	if got, want := first.GetPointNumber(), int64(1); got != want {
		t.Fatalf("first point number = %d, want %d", got, want)
	}
	if got, want := first.GetState().GetView(), matchesv1.View_VIEW_SCORED; got != want {
		t.Fatalf("first view = %v, want %v", got, want)
	}
	if got, want := first.GetState().GetInitialRole(), matchesv1.StartingRole_STARTING_ROLE_OFFENSE; got != want {
		t.Fatalf("first initialRole = %v, want %v", got, want)
	}
	if got, want := first.GetState().GetLineup(), []string{"player-a", "player-b"}; !equalStrings(got, want) {
		t.Fatalf("first lineup = %v, want %v", got, want)
	}
	if got, want := first.GetState().GetPlayerStats()["player-a"].GetAssists(), int64(1); got != want {
		t.Fatalf("player-a assists = %d, want %d", got, want)
	}
	if got, want := first.GetState().GetPlayerStats()["player-b"].GetGoals(), int64(1); got != want {
		t.Fatalf("player-b goals = %d, want %d", got, want)
	}

	second := points[1]
	if got, want := second.GetPointNumber(), int64(2); got != want {
		t.Fatalf("second point number = %d, want %d", got, want)
	}
	if got, want := second.GetState().GetView(), matchesv1.View_VIEW_OFFENSE_POSSESSION; got != want {
		t.Fatalf("second view = %v, want %v", got, want)
	}
	if got, want := second.GetState().GetThrower(), "player-c"; got != want {
		t.Fatalf("second thrower = %q, want %q", got, want)
	}
}

func TestGetMatchStatsAggregatesEveryPoint(t *testing.T) {
	fixture := newGameplayFixture(t)

	fixture.appendEvents(t,
		lineupEvent("p1-lineup", 1, 1000, "player-a", "player-b"),
		touchEvent("p1-touch-a", 1, 2000, "player-a"),
		touchEvent("p1-touch-b", 1, 3000, "player-b"),
		scoreEvent("p1-score", 1, 4000, "player-b", "player-a"),
		touchEvent("p2-touch-c", 2, 5000, "player-c"),
	)

	response, err := fixture.actor.matches.GetMatchStats(
		t.Context(),
		connect.NewRequest(&matchesv1.GetMatchStatsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
		}),
	)
	if err != nil {
		t.Fatalf("GetMatchStats: %v", err)
	}

	lines := response.Msg.GetPlayerStats()
	if got, want := len(lines), 3; got != want {
		t.Fatalf("player stats = %d, want %d", got, want)
	}
	// Sorted by player id for a deterministic response.
	if got, want := []string{lines[0].GetPlayerId(), lines[1].GetPlayerId(), lines[2].GetPlayerId()},
		[]string{"player-a", "player-b", "player-c"}; !equalStrings(got, want) {
		t.Fatalf("player order = %v, want %v", got, want)
	}

	playerA := lines[0]
	assertStatLine(t, playerA, 1, 1, 0, 0, 0, 0, 1) // touches, completions, drops, turnovers, blocks, goals, assists
	assertDerived(t, playerA, 1, 2, 1)              // receptions, throwAtts, plusMinus

	playerB := lines[1]
	assertStatLine(t, playerB, 1, 0, 0, 0, 0, 1, 0)
	assertDerived(t, playerB, 1, 0, 1)

	playerC := lines[2]
	assertStatLine(t, playerC, 1, 0, 0, 0, 0, 0, 0)
	assertDerived(t, playerC, 1, 1, 0)
}

func TestGetTeamRecordCountsOnlyMatchesWithAScore(t *testing.T) {
	fixture := newGameplayFixture(t)

	// One completed match: two live SCORE events. A second match in the same
	// event has no events at all and must not count as a 0-0 tie.
	fixture.appendEvents(t,
		scoreEvent("score-1", 1, 1000, "player-a", ""),
		scoreEvent("score-2", 2, 2000, "player-b", ""),
	)
	second := createMatch(t, fixture.actor, fixture.teamSlug, fixture.seasonSlug, fixture.eventSlug, "Other Rival")
	if second.GetSlug() == fixture.matchSlug {
		t.Fatal("expected a distinct match slug")
	}

	record := teamRecord(t, fixture.actor, fixture.teamSlug, nil)
	if record.GetWins() != 1 || record.GetLosses() != 0 || record.GetTies() != 0 {
		t.Fatalf("record = %v, want one win", record)
	}

	// Tombstoning both scores removes the match from the completed set.
	fixture.appendEvents(t,
		deletedScoreEvent("score-1", 1, 1000, "player-a", ""),
		deletedScoreEvent("score-2", 2, 2000, "player-b", ""),
	)
	record = teamRecord(t, fixture.actor, fixture.teamSlug, nil)
	if record.GetWins() != 0 || record.GetLosses() != 0 || record.GetTies() != 0 {
		t.Fatalf("record after tombstoning = %v, want zero", record)
	}
}

func TestGetTeamRecordScopesToASeason(t *testing.T) {
	env := newStructureEnv(t)
	actor := env.newActor("coach@example.com")
	team := createTeam(t, actor, "Flick City")

	record := func(seasonSlug *string) *teamsv1.TeamRecord {
		t.Helper()
		response, err := actor.teams.GetTeamRecord(
			t.Context(),
			connect.NewRequest(&teamsv1.GetTeamRecordRequest{
				TeamSlug:   team.GetSlug(),
				SeasonSlug: seasonSlug,
			}),
		)
		if err != nil {
			t.Fatalf("GetTeamRecord: %v", err)
		}
		return response.Msg.GetRecord()
	}

	spring := createSeason(t, actor, team.GetSlug(), "Spring")
	springEvent := createEvent(t, actor, team.GetSlug(), spring.GetSlug(), "Spring League")
	springMatch := createMatch(t, actor, team.GetSlug(), spring.GetSlug(), springEvent.GetSlug(), "Spring Rival")

	fall := createSeason(t, actor, team.GetSlug(), "Fall")
	fallEvent := createEvent(t, actor, team.GetSlug(), fall.GetSlug(), "Fall League")
	fallMatch := createMatch(t, actor, team.GetSlug(), fall.GetSlug(), fallEvent.GetSlug(), "Fall Rival")

	appendTo := func(seasonSlug, eventSlug, matchSlug, eventID string) {
		t.Helper()
		if _, err := actor.matches.AppendMatchEvents(
			t.Context(),
			connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
				TeamSlug:   team.GetSlug(),
				SeasonSlug: seasonSlug,
				EventSlug:  eventSlug,
				MatchSlug:  matchSlug,
				Events:     []*matchesv1.MatchEvent{scoreEvent(eventID, 1, 1000, "player-a", "")},
			}),
		); err != nil {
			t.Fatalf("append to %s: %v", matchSlug, err)
		}
	}
	appendTo(spring.GetSlug(), springEvent.GetSlug(), springMatch.GetSlug(), "spring-score")
	appendTo(fall.GetSlug(), fallEvent.GetSlug(), fallMatch.GetSlug(), "fall-score")

	if got := record(nil).GetWins(); got != 2 {
		t.Fatalf("team wins = %d, want 2", got)
	}
	if got := record(ptrTo(spring.GetSlug())).GetWins(); got != 1 {
		t.Fatalf("spring wins = %d, want 1", got)
	}
	if got := record(ptrTo(fall.GetSlug())).GetWins(); got != 1 {
		t.Fatalf("fall wins = %d, want 1", got)
	}

	_, err := actor.teams.GetTeamRecord(
		t.Context(),
		connect.NewRequest(&teamsv1.GetTeamRecordRequest{
			TeamSlug:   team.GetSlug(),
			SeasonSlug: ptrTo("missing"),
		}),
	)
	assertConnectError(t, err, connect.CodeNotFound, "not_found")
}

func deletedTouchEvent(id string, point, ts int64, playerID string) *matchesv1.MatchEvent {
	event := touchEvent(id, point, ts, playerID)
	event.Deleted = true
	return event
}

func deletedScoreEvent(id string, point, ts int64, playerID, assistID string) *matchesv1.MatchEvent {
	event := scoreEvent(id, point, ts, playerID, assistID)
	event.Deleted = true
	return event
}

func listEvents(t *testing.T, fixture *gameplayFixture) *matchesv1.ListMatchEventsResponse {
	t.Helper()

	response, err := fixture.actor.matches.ListMatchEvents(
		t.Context(),
		connect.NewRequest(&matchesv1.ListMatchEventsRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
		}),
	)
	if err != nil {
		t.Fatalf("ListMatchEvents: %v", err)
	}
	return response.Msg
}

func matchState(t *testing.T, fixture *gameplayFixture) *matchesv1.GetMatchStateResponse {
	t.Helper()

	response, err := fixture.actor.matches.GetMatchState(
		t.Context(),
		connect.NewRequest(&matchesv1.GetMatchStateRequest{
			TeamSlug:   fixture.teamSlug,
			SeasonSlug: fixture.seasonSlug,
			EventSlug:  fixture.eventSlug,
			MatchSlug:  fixture.matchSlug,
		}),
	)
	if err != nil {
		t.Fatalf("GetMatchState: %v", err)
	}
	return response.Msg
}

func teamRecord(t *testing.T, actor *structureActor, teamSlug string, seasonSlug *string) *teamsv1.TeamRecord {
	t.Helper()

	response, err := actor.teams.GetTeamRecord(
		t.Context(),
		connect.NewRequest(&teamsv1.GetTeamRecordRequest{
			TeamSlug:   teamSlug,
			SeasonSlug: seasonSlug,
		}),
	)
	if err != nil {
		t.Fatalf("GetTeamRecord: %v", err)
	}
	return response.Msg.GetRecord()
}

func assertStatLine(
	t *testing.T,
	line *matchesv1.AggregatedPlayerStats,
	touches, completions, drops, turnovers, blocks, goals, assists int64,
) {
	t.Helper()

	base := line.GetBase()
	if base.GetTouches() != touches ||
		base.GetCompletions() != completions ||
		base.GetDrops() != drops ||
		base.GetTurnovers() != turnovers ||
		base.GetBlocks() != blocks ||
		base.GetGoals() != goals ||
		base.GetAssists() != assists {
		t.Fatalf("%s base stats = %v, want touches=%d completions=%d drops=%d turnovers=%d blocks=%d goals=%d assists=%d",
			line.GetPlayerId(), base, touches, completions, drops, turnovers, blocks, goals, assists)
	}
}

func assertDerived(t *testing.T, line *matchesv1.AggregatedPlayerStats, receptions, throwAttempts, plusMinus int64) {
	t.Helper()

	derived := line.GetDerived()
	if derived.GetReceptions() != receptions ||
		derived.GetThrowAttempts() != throwAttempts ||
		derived.GetPlusMinus() != plusMinus {
		t.Fatalf("%s derived stats = %v, want receptions=%d throwAttempts=%d plusMinus=%d",
			line.GetPlayerId(), derived, receptions, throwAttempts, plusMinus)
	}
}

func eventIDs(events []*matchesv1.MatchEvent) []string {
	ids := make([]string, len(events))
	for index, event := range events {
		ids[index] = event.GetId()
	}
	return ids
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}
