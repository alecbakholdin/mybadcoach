package api_test

import (
	"context"
	"database/sql"
	"testing"

	"connectrpc.com/connect"

	eventsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1"
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
	playersv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1"
	seasonsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1"
	teamsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1"
)

func TestSeasonCRUDScopedSlugsAndPagination(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	team := createTeam(t, owner, "Season Team")

	first := createSeason(t, owner, team.GetSlug(), "Fall 2026")
	if first.GetSlug() != "fall-2026" {
		t.Fatalf("slug = %q, want fall-2026", first.GetSlug())
	}
	second := createSeason(t, owner, team.GetSlug(), "Fall 2026")
	if second.GetSlug() != "fall-2026-2" {
		t.Fatalf("collision slug = %q, want fall-2026-2", second.GetSlug())
	}

	anotherTeam := createTeam(t, owner, "Other Team")
	otherScoped := createSeason(t, owner, anotherTeam.GetSlug(), "Fall 2026")
	if otherScoped.GetSlug() != "fall-2026" {
		t.Fatalf("slug in another team = %q, want fall-2026", otherScoped.GetSlug())
	}

	updated, err := owner.seasons.UpdateSeason(ctx, connect.NewRequest(&seasonsv1.UpdateSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: first.GetSlug(),
		Name:       ptrTo("Autumn 2026"),
		StartDate:  ptrTo("2026-09-01"),
		EndDate:    ptrTo("2026-11-30"),
	}))
	if err != nil {
		t.Fatalf("UpdateSeason: %v", err)
	}
	season := updated.Msg.GetSeason()
	if season.GetName() != "Autumn 2026" {
		t.Fatalf("name = %q", season.GetName())
	}
	if season.GetStartDate() != "2026-09-01" || season.GetEndDate() != "2026-11-30" {
		t.Fatalf("dates = %q..%q", season.GetStartDate(), season.GetEndDate())
	}
	if season.GetSlug() != first.GetSlug() {
		t.Fatalf("slug changed on rename: %q", season.GetSlug())
	}

	_, err = owner.seasons.UpdateSeason(ctx, connect.NewRequest(&seasonsv1.UpdateSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: first.GetSlug(),
		Name:       ptrTo("Autumn"),
	}))
	if err != nil {
		t.Fatalf("partial UpdateSeason: %v", err)
	}
	got, err := owner.seasons.GetSeason(ctx, connect.NewRequest(&seasonsv1.GetSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: first.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("GetSeason: %v", err)
	}
	if got.Msg.GetSeason().GetStartDate() != "2026-09-01" {
		t.Fatalf("partial update cleared start_date: %q", got.Msg.GetSeason().GetStartDate())
	}

	cleared, err := owner.seasons.UpdateSeason(ctx, connect.NewRequest(&seasonsv1.UpdateSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: first.GetSlug(),
		StartDate:  ptrTo(""),
	}))
	if err != nil {
		t.Fatalf("clear start_date: %v", err)
	}
	if cleared.Msg.GetSeason().GetStartDate() != "" {
		t.Fatalf("start_date = %q, want empty", cleared.Msg.GetSeason().GetStartDate())
	}
	if cleared.Msg.GetSeason().GetEndDate() != "2026-11-30" {
		t.Fatalf("end_date = %q, want unchanged", cleared.Msg.GetSeason().GetEndDate())
	}

	third := createSeason(t, owner, team.GetSlug(), "Spring 2027")
	_ = third

	page, err := owner.seasons.ListSeasons(ctx, connect.NewRequest(&seasonsv1.ListSeasonsRequest{
		TeamSlug: team.GetSlug(),
		Limit:    2,
	}))
	if err != nil {
		t.Fatalf("ListSeasons: %v", err)
	}
	if page.Msg.GetTotal() != 3 {
		t.Fatalf("total = %d, want 3", page.Msg.GetTotal())
	}
	if len(page.Msg.GetSeasons()) != 2 {
		t.Fatalf("page size = %d, want 2", len(page.Msg.GetSeasons()))
	}

	rest, err := owner.seasons.ListSeasons(ctx, connect.NewRequest(&seasonsv1.ListSeasonsRequest{
		TeamSlug: team.GetSlug(),
		Offset:   2,
		Limit:    2,
	}))
	if err != nil {
		t.Fatalf("ListSeasons offset: %v", err)
	}
	if len(rest.Msg.GetSeasons()) != 1 || rest.Msg.GetTotal() != 3 {
		t.Fatalf("rest = %d items, total %d; want 1, 3", len(rest.Msg.GetSeasons()), rest.Msg.GetTotal())
	}

	defaulted, err := owner.seasons.ListSeasons(ctx, connect.NewRequest(&seasonsv1.ListSeasonsRequest{
		TeamSlug: team.GetSlug(),
		Limit:    0,
	}))
	if err != nil {
		t.Fatalf("ListSeasons default limit: %v", err)
	}
	if len(defaulted.Msg.GetSeasons()) != 3 {
		t.Fatalf("default page size = %d, want 3", len(defaulted.Msg.GetSeasons()))
	}

	clamped, err := owner.seasons.ListSeasons(ctx, connect.NewRequest(&seasonsv1.ListSeasonsRequest{
		TeamSlug: team.GetSlug(),
		Limit:    -5,
	}))
	if err != nil {
		t.Fatalf("ListSeasons negative limit: %v", err)
	}
	if len(clamped.Msg.GetSeasons()) != 1 {
		t.Fatalf("negative limit page size = %d, want clamped to 1", len(clamped.Msg.GetSeasons()))
	}

	if _, err := owner.seasons.CreateSeason(ctx, connect.NewRequest(&seasonsv1.CreateSeasonRequest{
		TeamSlug: team.GetSlug(),
		Name:     "   ",
	})); err == nil {
		t.Fatal("CreateSeason with a blank name succeeded")
	} else {
		assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
		assertFieldViolation(t, err, "name")
	}
}

func TestSeasonDeleteGatingAndCascade(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	member := env.newActor("member@example.com")
	team := createTeam(t, owner, "Cascade Team")
	if _, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
		TeamSlug: team.GetSlug(),
		Email:    "member@example.com",
	})); err != nil {
		t.Fatalf("AddMembership: %v", err)
	}
	season := createSeason(t, owner, team.GetSlug(), "To Delete")
	other := createSeason(t, owner, team.GetSlug(), "To Keep")
	event := createEvent(t, owner, team.GetSlug(), season.GetSlug(), "Sectionals")
	match := createMatch(t, owner, team.GetSlug(), season.GetSlug(), event.GetSlug(), "Rival")
	player := createPlayer(t, owner, team.GetSlug(), "Keeper")
	if _, err := owner.seasons.AddSeasonRosterEntry(ctx, connect.NewRequest(&seasonsv1.AddSeasonRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		PlayerSlug: player.GetSlug(),
	})); err != nil {
		t.Fatalf("AddSeasonRosterEntry: %v", err)
	}
	insertMatchEvent(t, env.db, "match-event-1", match.GetId())

	_, err := member.seasons.DeleteSeason(ctx, connect.NewRequest(&seasonsv1.DeleteSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
	}))
	if err == nil {
		t.Fatal("member DeleteSeason succeeded")
	}
	assertConnectError(t, err, connect.CodePermissionDenied, "permission_denied")

	if _, err := owner.seasons.DeleteSeason(ctx, connect.NewRequest(&seasonsv1.DeleteSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
	})); err != nil {
		t.Fatalf("DeleteSeason: %v", err)
	}

	if _, err := owner.seasons.GetSeason(ctx, connect.NewRequest(&seasonsv1.GetSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
	})); err == nil {
		t.Fatal("GetSeason after delete succeeded")
	} else {
		assertConnectError(t, err, connect.CodeNotFound, "not_found")
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM events"); count != 0 {
		t.Fatalf("events = %d, want 0 after season cascade", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM matches"); count != 0 {
		t.Fatalf("matches = %d, want 0 after season cascade", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM match_events"); count != 0 {
		t.Fatalf("match_events = %d, want 0 after season cascade", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM season_roster_entries"); count != 0 {
		t.Fatalf("season_roster_entries = %d, want 0", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM players"); count != 1 {
		t.Fatalf("players = %d, want the player to survive", count)
	}
	if _, err := owner.seasons.GetSeason(ctx, connect.NewRequest(&seasonsv1.GetSeasonRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: other.GetSlug(),
	})); err != nil {
		t.Fatalf("sibling season was deleted: %v", err)
	}
}

func TestCopySeasonRosterSkipsExistingPlayers(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	team := createTeam(t, owner, "Roster Team")
	from := createSeason(t, owner, team.GetSlug(), "From")
	to := createSeason(t, owner, team.GetSlug(), "To")

	shared := createPlayer(t, owner, team.GetSlug(), "Shared")
	onlyFrom := createPlayer(t, owner, team.GetSlug(), "Only From")
	onlyTo := createPlayer(t, owner, team.GetSlug(), "Only To")

	for _, player := range []*playersv1.Player{shared, onlyFrom} {
		if _, err := owner.seasons.AddSeasonRosterEntry(ctx, connect.NewRequest(&seasonsv1.AddSeasonRosterEntryRequest{
			TeamSlug:   team.GetSlug(),
			SeasonSlug: from.GetSlug(),
			PlayerSlug: player.GetSlug(),
		})); err != nil {
			t.Fatalf("add source roster: %v", err)
		}
	}
	if _, err := owner.seasons.AddSeasonRosterEntry(ctx, connect.NewRequest(&seasonsv1.AddSeasonRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: to.GetSlug(),
		PlayerSlug: shared.GetSlug(),
	})); err != nil {
		t.Fatalf("add target roster: %v", err)
	}
	if _, err := owner.seasons.AddSeasonRosterEntry(ctx, connect.NewRequest(&seasonsv1.AddSeasonRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: to.GetSlug(),
		PlayerSlug: onlyTo.GetSlug(),
	})); err != nil {
		t.Fatalf("add target-only roster: %v", err)
	}

	copied, err := owner.seasons.CopySeasonRoster(ctx, connect.NewRequest(&seasonsv1.CopySeasonRosterRequest{
		TeamSlug:       team.GetSlug(),
		FromSeasonSlug: from.GetSlug(),
		ToSeasonSlug:   to.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("CopySeasonRoster: %v", err)
	}
	if len(copied.Msg.GetCopied()) != 1 {
		t.Fatalf("copied = %d, want 1", len(copied.Msg.GetCopied()))
	}
	if copied.Msg.GetCopied()[0].GetSlug() != onlyFrom.GetSlug() {
		t.Fatalf("copied player = %q, want %q", copied.Msg.GetCopied()[0].GetSlug(), onlyFrom.GetSlug())
	}

	roster, err := owner.seasons.ListSeasonRoster(ctx, connect.NewRequest(&seasonsv1.ListSeasonRosterRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: to.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("ListSeasonRoster: %v", err)
	}
	if roster.Msg.GetTotal() != 3 {
		t.Fatalf("target roster total = %d, want 3", roster.Msg.GetTotal())
	}

	again, err := owner.seasons.CopySeasonRoster(ctx, connect.NewRequest(&seasonsv1.CopySeasonRosterRequest{
		TeamSlug:       team.GetSlug(),
		FromSeasonSlug: from.GetSlug(),
		ToSeasonSlug:   to.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("second CopySeasonRoster: %v", err)
	}
	if len(again.Msg.GetCopied()) != 0 {
		t.Fatalf("second copy copied %d players, want 0", len(again.Msg.GetCopied()))
	}
}

func TestEventCRUDScopedSlugsAndRoster(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	member := env.newActor("member@example.com")
	team := createTeam(t, owner, "Event Team")
	season := createSeason(t, owner, team.GetSlug(), "Season")
	otherSeason := createSeason(t, owner, team.GetSlug(), "Other Season")
	if _, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
		TeamSlug: team.GetSlug(),
		Email:    "member@example.com",
	})); err != nil {
		t.Fatalf("AddMembership: %v", err)
	}

	first := createEvent(t, owner, team.GetSlug(), season.GetSlug(), "Sectionals")
	if first.GetSlug() != "sectionals" {
		t.Fatalf("slug = %q", first.GetSlug())
	}
	second := createEvent(t, owner, team.GetSlug(), season.GetSlug(), "Sectionals")
	if second.GetSlug() != "sectionals-2" {
		t.Fatalf("collision slug = %q", second.GetSlug())
	}
	otherScoped := createEvent(t, owner, team.GetSlug(), otherSeason.GetSlug(), "Sectionals")
	if otherScoped.GetSlug() != "sectionals" {
		t.Fatalf("slug in another season = %q", otherScoped.GetSlug())
	}

	updated, err := owner.events.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  first.GetSlug(),
		Name:       ptrTo("Regionals"),
		StartDate:  ptrTo("2026-06-01"),
		Location:   ptrTo("Field 3"),
	}))
	if err != nil {
		t.Fatalf("UpdateEvent: %v", err)
	}
	if updated.Msg.GetEvent().GetName() != "Regionals" || updated.Msg.GetEvent().GetLocation() != "Field 3" {
		t.Fatalf("updated event = %+v", updated.Msg.GetEvent())
	}
	if updated.Msg.GetEvent().GetSlug() != first.GetSlug() {
		t.Fatalf("slug changed on rename: %q", updated.Msg.GetEvent().GetSlug())
	}

	cleared, err := owner.events.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  first.GetSlug(),
		Location:   ptrTo(""),
	}))
	if err != nil {
		t.Fatalf("clear location: %v", err)
	}
	if cleared.Msg.GetEvent().GetLocation() != "" {
		t.Fatalf("location = %q, want empty", cleared.Msg.GetEvent().GetLocation())
	}
	if cleared.Msg.GetEvent().GetStartDate() != "2026-06-01" {
		t.Fatalf("start_date = %q, want unchanged", cleared.Msg.GetEvent().GetStartDate())
	}

	events, err := owner.events.ListEvents(ctx, connect.NewRequest(&eventsv1.ListEventsRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		Limit:      1,
	}))
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if events.Msg.GetTotal() != 2 || len(events.Msg.GetEvents()) != 1 {
		t.Fatalf("events = %d/%d, want 1/2", len(events.Msg.GetEvents()), events.Msg.GetTotal())
	}

	one := createPlayer(t, owner, team.GetSlug(), "One")
	two := createPlayer(t, owner, team.GetSlug(), "Two")
	for _, player := range []*playersv1.Player{one, two} {
		_, err = member.events.AddEventRosterEntry(ctx, connect.NewRequest(&eventsv1.AddEventRosterEntryRequest{
			TeamSlug:   team.GetSlug(),
			SeasonSlug: season.GetSlug(),
			EventSlug:  first.GetSlug(),
			PlayerSlug: player.GetSlug(),
		}))
		if err != nil {
			t.Fatalf("member AddEventRosterEntry: %v", err)
		}
	}
	roster, err := owner.events.ListEventRoster(ctx, connect.NewRequest(&eventsv1.ListEventRosterRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  first.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("ListEventRoster: %v", err)
	}
	if roster.Msg.GetTotal() != 2 {
		t.Fatalf("roster total = %d, want 2", roster.Msg.GetTotal())
	}
	_, err = member.events.RemoveEventRosterEntry(ctx, connect.NewRequest(&eventsv1.RemoveEventRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  first.GetSlug(),
		PlayerSlug: one.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("RemoveEventRosterEntry: %v", err)
	}
	roster, err = owner.events.ListEventRoster(ctx, connect.NewRequest(&eventsv1.ListEventRosterRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  first.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("ListEventRoster after remove: %v", err)
	}
	if roster.Msg.GetTotal() != 1 {
		t.Fatalf("roster total after remove = %d, want 1", roster.Msg.GetTotal())
	}

	match := createMatch(t, owner, team.GetSlug(), season.GetSlug(), first.GetSlug(), "Rival")
	insertMatchEvent(t, env.db, "event-delete-event", match.GetId())
	if _, err := member.events.DeleteEvent(ctx, connect.NewRequest(&eventsv1.DeleteEventRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  first.GetSlug(),
	})); err != nil {
		t.Fatalf("member DeleteEvent: %v", err)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM matches"); count != 0 {
		t.Fatalf("matches = %d, want 0 after event cascade", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM event_roster_entries"); count != 0 {
		t.Fatalf("event_roster_entries = %d, want 0 after event cascade", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM match_events"); count != 0 {
		t.Fatalf("match_events = %d, want 0 after event cascade", count)
	}
}

func TestPlayerCRUDNamesAndCascade(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	member := env.newActor("member@example.com")
	team := createTeam(t, owner, "Player Team")
	if _, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
		TeamSlug: team.GetSlug(),
		Email:    "member@example.com",
	})); err != nil {
		t.Fatalf("AddMembership: %v", err)
	}
	season := createSeason(t, owner, team.GetSlug(), "Season")
	event := createEvent(t, owner, team.GetSlug(), season.GetSlug(), "Event")
	match := createMatch(t, owner, team.GetSlug(), season.GetSlug(), event.GetSlug(), "Rival")

	player := createPlayer(t, owner, team.GetSlug(), "Alice", "Al")
	if player.GetSlug() != "alice" {
		t.Fatalf("slug = %q, want alice", player.GetSlug())
	}
	if len(player.GetNames()) != 2 || player.GetNames()[0] != "Alice" {
		t.Fatalf("names = %v", player.GetNames())
	}

	unchanged, err := member.players.UpdatePlayer(ctx, connect.NewRequest(&playersv1.UpdatePlayerRequest{
		TeamSlug:   team.GetSlug(),
		PlayerSlug: player.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("UpdatePlayer with empty names: %v", err)
	}
	if len(unchanged.Msg.GetPlayer().GetNames()) != 2 {
		t.Fatalf("empty update changed names: %v", unchanged.Msg.GetPlayer().GetNames())
	}

	updated, err := member.players.UpdatePlayer(ctx, connect.NewRequest(&playersv1.UpdatePlayerRequest{
		TeamSlug:   team.GetSlug(),
		PlayerSlug: player.GetSlug(),
		Names:      []string{"Alice Smith", "Alice"},
	}))
	if err != nil {
		t.Fatalf("UpdatePlayer: %v", err)
	}
	if got := updated.Msg.GetPlayer().GetNames(); len(got) != 2 || got[0] != "Alice Smith" {
		t.Fatalf("names = %v", got)
	}
	if updated.Msg.GetPlayer().GetSlug() != player.GetSlug() {
		t.Fatalf("slug changed on names update: %q", updated.Msg.GetPlayer().GetSlug())
	}

	_, err = member.players.UpdatePlayer(ctx, connect.NewRequest(&playersv1.UpdatePlayerRequest{
		TeamSlug:   team.GetSlug(),
		PlayerSlug: player.GetSlug(),
		Names:      []string{"  "},
	}))
	if err == nil {
		t.Fatal("UpdatePlayer with a blank name succeeded")
	} else {
		assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
		assertFieldViolation(t, err, "names")
	}

	createPlayer(t, owner, team.GetSlug(), "Second")
	createPlayer(t, owner, team.GetSlug(), "Third")
	page, err := owner.players.ListPlayers(ctx, connect.NewRequest(&playersv1.ListPlayersRequest{
		TeamSlug: team.GetSlug(),
		Limit:    2,
	}))
	if err != nil {
		t.Fatalf("ListPlayers: %v", err)
	}
	if page.Msg.GetTotal() != 3 || len(page.Msg.GetPlayers()) != 2 {
		t.Fatalf("players = %d/%d, want 2/3", len(page.Msg.GetPlayers()), page.Msg.GetTotal())
	}

	_, err = owner.seasons.AddSeasonRosterEntry(ctx, connect.NewRequest(&seasonsv1.AddSeasonRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		PlayerSlug: player.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("AddSeasonRosterEntry: %v", err)
	}
	_, err = owner.events.AddEventRosterEntry(ctx, connect.NewRequest(&eventsv1.AddEventRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  event.GetSlug(),
		PlayerSlug: player.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("AddEventRosterEntry: %v", err)
	}
	insertMatchEvent(t, env.db, "player-delete-event", match.GetId())

	_, err = member.players.DeletePlayer(ctx, connect.NewRequest(&playersv1.DeletePlayerRequest{
		TeamSlug:   team.GetSlug(),
		PlayerSlug: player.GetSlug(),
	}))
	if err == nil {
		t.Fatal("member DeletePlayer succeeded")
	}
	assertConnectError(t, err, connect.CodePermissionDenied, "permission_denied")

	if _, err := owner.players.DeletePlayer(ctx, connect.NewRequest(&playersv1.DeletePlayerRequest{
		TeamSlug:   team.GetSlug(),
		PlayerSlug: player.GetSlug(),
	})); err != nil {
		t.Fatalf("owner DeletePlayer: %v", err)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM season_roster_entries"); count != 0 {
		t.Fatalf("season_roster_entries = %d, want 0", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM event_roster_entries"); count != 0 {
		t.Fatalf("event_roster_entries = %d, want 0", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM match_events"); count != 1 {
		t.Fatalf("match_events = %d, want the gameplay log untouched", count)
	}
	if _, err := owner.players.GetPlayer(ctx, connect.NewRequest(&playersv1.GetPlayerRequest{
		TeamSlug:   team.GetSlug(),
		PlayerSlug: player.GetSlug(),
	})); err == nil {
		t.Fatal("GetPlayer after delete succeeded")
	} else {
		assertConnectError(t, err, connect.CodeNotFound, "not_found")
	}

	if _, err := owner.players.CreatePlayer(ctx, connect.NewRequest(&playersv1.CreatePlayerRequest{
		TeamSlug: team.GetSlug(),
	})); err == nil {
		t.Fatal("CreatePlayer with no names succeeded")
	} else {
		assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
		assertFieldViolation(t, err, "names")
	}
}

func TestMatchCRUDAndGameplayStubs(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	member := env.newActor("member@example.com")
	team := createTeam(t, owner, "Match Team")
	if _, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
		TeamSlug: team.GetSlug(),
		Email:    "member@example.com",
	})); err != nil {
		t.Fatalf("AddMembership: %v", err)
	}
	season := createSeason(t, owner, team.GetSlug(), "Season")
	event := createEvent(t, owner, team.GetSlug(), season.GetSlug(), "Event")

	created, err := owner.matches.CreateMatch(ctx, connect.NewRequest(&matchesv1.CreateMatchRequest{
		TeamSlug:     team.GetSlug(),
		SeasonSlug:   season.GetSlug(),
		EventSlug:    event.GetSlug(),
		OpponentName: "Fury",
		ScheduledAt:  ptrTo("2026-05-01T10:00:00Z"),
		PointCap:     ptrTo[int64](15),
	}))
	if err != nil {
		t.Fatalf("CreateMatch: %v", err)
	}
	match := created.Msg.GetMatch()
	if match.GetSlug() != "fury" {
		t.Fatalf("slug = %q, want fury", match.GetSlug())
	}
	if match.GetScheduledAt() != "2026-05-01T10:00:00Z" || match.GetPointCap() != 15 {
		t.Fatalf("match = %+v", match)
	}

	second := createMatch(t, owner, team.GetSlug(), season.GetSlug(), event.GetSlug(), "Fury")
	if second.GetSlug() != "fury-2" {
		t.Fatalf("collision slug = %q, want fury-2", second.GetSlug())
	}

	got, err := member.matches.GetMatch(ctx, connect.NewRequest(&matchesv1.GetMatchRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  event.GetSlug(),
		MatchSlug:  match.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("GetMatch: %v", err)
	}
	if got.Msg.GetMatch().GetId() != match.GetId() {
		t.Fatalf("GetMatch id = %q", got.Msg.GetMatch().GetId())
	}

	updated, err := member.matches.UpdateMatch(ctx, connect.NewRequest(&matchesv1.UpdateMatchRequest{
		TeamSlug:     team.GetSlug(),
		SeasonSlug:   season.GetSlug(),
		EventSlug:    event.GetSlug(),
		MatchSlug:    match.GetSlug(),
		OpponentName: ptrTo("Fury Ultimate"),
		ScheduledAt:  ptrTo(""),
		PointCap:     ptrTo[int64](0),
	}))
	if err != nil {
		t.Fatalf("UpdateMatch: %v", err)
	}
	if updated.Msg.GetMatch().GetOpponentName() != "Fury Ultimate" {
		t.Fatalf("opponent = %q", updated.Msg.GetMatch().GetOpponentName())
	}
	if updated.Msg.GetMatch().GetScheduledAt() != "" {
		t.Fatalf("scheduled_at = %q, want cleared", updated.Msg.GetMatch().GetScheduledAt())
	}
	if updated.Msg.GetMatch().GetPointCap() != 0 {
		t.Fatalf("point_cap = %d, want 0", updated.Msg.GetMatch().GetPointCap())
	}
	if updated.Msg.GetMatch().GetSlug() != match.GetSlug() {
		t.Fatalf("slug changed on update: %q", updated.Msg.GetMatch().GetSlug())
	}

	_, err = member.matches.UpdateMatch(ctx, connect.NewRequest(&matchesv1.UpdateMatchRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  event.GetSlug(),
		MatchSlug:  match.GetSlug(),
		PointCap:   ptrTo[int64](-1),
	}))
	if err == nil {
		t.Fatal("negative point_cap succeeded")
	} else {
		assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
		assertFieldViolation(t, err, "point_cap")
	}

	matches, err := owner.matches.ListMatches(ctx, connect.NewRequest(&matchesv1.ListMatchesRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  event.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("ListMatches: %v", err)
	}
	if matches.Msg.GetTotal() != 2 || len(matches.Msg.GetMatches()) != 2 {
		t.Fatalf("matches = %d/%d, want 2/2", len(matches.Msg.GetMatches()), matches.Msg.GetTotal())
	}

	t.Run("gameplay endpoints accept an empty log", func(t *testing.T) {
		eventKey := func() (string, string, string, string) {
			return team.GetSlug(), season.GetSlug(), event.GetSlug(), match.GetSlug()
		}
		teamSlug, seasonSlug, eventSlug, matchSlug := eventKey()
		appendResponse, err := member.matches.AppendMatchEvents(
			ctx,
			connect.NewRequest(&matchesv1.AppendMatchEventsRequest{
				TeamSlug:   teamSlug,
				SeasonSlug: seasonSlug,
				EventSlug:  eventSlug,
				MatchSlug:  matchSlug,
			}),
		)
		if err != nil {
			t.Fatalf("AppendMatchEvents: %v", err)
		}
		if len(appendResponse.Msg.GetAcceptedIds()) != 0 || len(appendResponse.Msg.GetSkippedDuplicateIds()) != 0 {
			t.Fatalf("append response = %v, want no ids", appendResponse.Msg)
		}
		listResponse, err := member.matches.ListMatchEvents(
			ctx,
			connect.NewRequest(&matchesv1.ListMatchEventsRequest{
				TeamSlug:   teamSlug,
				SeasonSlug: seasonSlug,
				EventSlug:  eventSlug,
				MatchSlug:  matchSlug,
			}),
		)
		if err != nil {
			t.Fatalf("ListMatchEvents: %v", err)
		}
		if listResponse.Msg.GetTotal() != 0 || len(listResponse.Msg.GetEvents()) != 0 {
			t.Fatalf("list response = %v, want an empty log", listResponse.Msg)
		}
		stateResponse, err := member.matches.GetMatchState(
			ctx,
			connect.NewRequest(&matchesv1.GetMatchStateRequest{
				TeamSlug:   teamSlug,
				SeasonSlug: seasonSlug,
				EventSlug:  eventSlug,
				MatchSlug:  matchSlug,
			}),
		)
		if err != nil {
			t.Fatalf("GetMatchState: %v", err)
		}
		if len(stateResponse.Msg.GetPoints()) != 0 {
			t.Fatalf("state = %v, want no points", stateResponse.Msg)
		}
		statsResponse, err := member.matches.GetMatchStats(
			ctx,
			connect.NewRequest(&matchesv1.GetMatchStatsRequest{
				TeamSlug:   teamSlug,
				SeasonSlug: seasonSlug,
				EventSlug:  eventSlug,
				MatchSlug:  matchSlug,
			}),
		)
		if err != nil {
			t.Fatalf("GetMatchStats: %v", err)
		}
		if len(statsResponse.Msg.GetPlayerStats()) != 0 {
			t.Fatalf("stats = %v, want no players", statsResponse.Msg)
		}
	})

	insertMatchEvent(t, env.db, "match-delete-event", match.GetId())
	if _, err := member.matches.DeleteMatch(ctx, connect.NewRequest(&matchesv1.DeleteMatchRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  event.GetSlug(),
		MatchSlug:  match.GetSlug(),
	})); err != nil {
		t.Fatalf("member DeleteMatch: %v", err)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM matches"); count != 1 {
		t.Fatalf("matches = %d, want 1 remaining", count)
	}
	if count := countRows(t, env.db, "SELECT COUNT(*) FROM match_events"); count != 0 {
		t.Fatalf("match_events = %d, want 0 after match cascade", count)
	}
}

func TestDeleteTeamCascadesEverything(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	team := createTeam(t, owner, "Doomed Team")
	season := createSeason(t, owner, team.GetSlug(), "Season")
	event := createEvent(t, owner, team.GetSlug(), season.GetSlug(), "Event")
	match := createMatch(t, owner, team.GetSlug(), season.GetSlug(), event.GetSlug(), "Rival")
	player := createPlayer(t, owner, team.GetSlug(), "Player")
	if _, err := owner.seasons.AddSeasonRosterEntry(ctx, connect.NewRequest(&seasonsv1.AddSeasonRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		PlayerSlug: player.GetSlug(),
	})); err != nil {
		t.Fatalf("AddSeasonRosterEntry: %v", err)
	}
	if _, err := owner.events.AddEventRosterEntry(ctx, connect.NewRequest(&eventsv1.AddEventRosterEntryRequest{
		TeamSlug:   team.GetSlug(),
		SeasonSlug: season.GetSlug(),
		EventSlug:  event.GetSlug(),
		PlayerSlug: player.GetSlug(),
	})); err != nil {
		t.Fatalf("AddEventRosterEntry: %v", err)
	}
	insertMatchEvent(t, env.db, "team-delete-event", match.GetId())

	if _, err := owner.teams.DeleteTeam(ctx, connect.NewRequest(&teamsv1.DeleteTeamRequest{
		TeamSlug: team.GetSlug(),
	})); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}

	checks := []struct {
		table string
		query string
	}{
		{table: "teams", query: "SELECT COUNT(*) FROM teams"},
		{table: "team_memberships", query: "SELECT COUNT(*) FROM team_memberships"},
		{table: "seasons", query: "SELECT COUNT(*) FROM seasons"},
		{table: "events", query: "SELECT COUNT(*) FROM events"},
		{table: "matches", query: "SELECT COUNT(*) FROM matches"},
		{table: "players", query: "SELECT COUNT(*) FROM players"},
		{table: "season_roster_entries", query: "SELECT COUNT(*) FROM season_roster_entries"},
		{table: "event_roster_entries", query: "SELECT COUNT(*) FROM event_roster_entries"},
		{table: "match_events", query: "SELECT COUNT(*) FROM match_events"},
	}
	for _, check := range checks {
		if count := countRows(t, env.db, check.query); count != 0 {
			t.Fatalf("%s = %d, want 0 after team cascade", check.table, count)
		}
	}
}

// insertMatchEvent writes a raw gameplay row. Ticket 05 owns the append RPC;
// the structure tests only need a row that references a match.
func insertMatchEvent(t *testing.T, sqlDB *sql.DB, id, matchID string) {
	t.Helper()
	_, err := sqlDB.Exec(
		`INSERT INTO match_events
		 (id, match_id, point_number, effective_ts, type, payload, deleted, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id,
		matchID,
		1,
		0,
		"TOUCH",
		"{}",
		0,
		"2026-01-01T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert match event: %v", err)
	}
}
