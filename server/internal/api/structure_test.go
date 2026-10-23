package api_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/api"
	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	authv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1/authv1connect"
	eventsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1/eventsv1connect"
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1/matchesv1connect"
	playersv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1/playersv1connect"
	seasonsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1/seasonsv1connect"
	teamsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1/teamsv1connect"
)

const testLoginPassword = "correct-horse"

// structureEnv wires the production handler against a real temp-file SQLite
// database and a real HTTP server, per docs/testing.md's backend integration
// seam.
type structureEnv struct {
	t      *testing.T
	db     *sql.DB
	server *httptest.Server
}

func newStructureEnv(t *testing.T) *structureEnv {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	handler, err := api.NewHandler(api.Config{DB: sqlDB})
	if err != nil {
		t.Fatalf("assemble handler: %v", err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("close test database: %v", closeErr)
		}
	})
	return &structureEnv{t: t, db: sqlDB, server: server}
}

// structureActor is an authenticated browser-like client with its own cookie
// jar, mirroring a distinct signed-in user.
type structureActor struct {
	teams   teamsv1connect.TeamServiceClient
	seasons seasonsv1connect.SeasonServiceClient
	events  eventsv1connect.EventServiceClient
	players playersv1connect.PlayerServiceClient
	matches matchesv1connect.MatchServiceClient
	userID  string
}

// newActor registers a fresh account and returns clients sharing its cookie
// jar.
func (e *structureEnv) newActor(email string) *structureActor {
	e.t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatalf("create cookie jar: %v", err)
	}
	httpClient := &http.Client{Jar: jar}

	authClient := authv1connect.NewAuthServiceClient(
		httpClient,
		e.server.URL,
		connect.WithInterceptors(browserHeaders("")),
	)
	registered, err := authClient.Register(
		context.Background(),
		connect.NewRequest(&authv1.RegisterRequest{
			Email:    email,
			Password: testLoginPassword,
			Name:     strings.SplitN(email, "@", 2)[0],
		}),
	)
	if err != nil {
		e.t.Fatalf("register %q: %v", email, err)
	}

	return &structureActor{
		userID: registered.Msg.GetUser().GetId(),
		teams: teamsv1connect.NewTeamServiceClient(
			httpClient,
			e.server.URL,
			connect.WithInterceptors(browserHeaders("")),
		),
		seasons: seasonsv1connect.NewSeasonServiceClient(
			httpClient,
			e.server.URL,
			connect.WithInterceptors(browserHeaders("")),
		),
		events: eventsv1connect.NewEventServiceClient(
			httpClient,
			e.server.URL,
			connect.WithInterceptors(browserHeaders("")),
		),
		players: playersv1connect.NewPlayerServiceClient(
			httpClient,
			e.server.URL,
			connect.WithInterceptors(browserHeaders("")),
		),
		matches: matchesv1connect.NewMatchServiceClient(
			httpClient,
			e.server.URL,
			connect.WithInterceptors(browserHeaders("")),
		),
	}
}

// anonymousTeamClient sends the CSRF header but carries no session cookie.
func (e *structureEnv) anonymousTeamClient() teamsv1connect.TeamServiceClient {
	e.t.Helper()
	return teamsv1connect.NewTeamServiceClient(
		&http.Client{},
		e.server.URL,
		connect.WithInterceptors(browserHeaders("")),
	)
}

func ptrTo[T any](value T) *T {
	return &value
}

func createTeam(t *testing.T, actor *structureActor, name string) *teamsv1.Team {
	t.Helper()
	response, err := actor.teams.CreateTeam(
		context.Background(),
		connect.NewRequest(&teamsv1.CreateTeamRequest{Name: name}),
	)
	if err != nil {
		t.Fatalf("CreateTeam(%q): %v", name, err)
	}
	return response.Msg.GetTeam()
}

func createSeason(
	t *testing.T,
	actor *structureActor,
	teamSlug string,
	name string,
) *seasonsv1.Season {
	t.Helper()
	response, err := actor.seasons.CreateSeason(
		context.Background(),
		connect.NewRequest(&seasonsv1.CreateSeasonRequest{
			TeamSlug: teamSlug,
			Name:     name,
		}),
	)
	if err != nil {
		t.Fatalf("CreateSeason(%q): %v", name, err)
	}
	return response.Msg.GetSeason()
}

func createEvent(
	t *testing.T,
	actor *structureActor,
	teamSlug string,
	seasonSlug string,
	name string,
) *eventsv1.Event {
	t.Helper()
	response, err := actor.events.CreateEvent(
		context.Background(),
		connect.NewRequest(&eventsv1.CreateEventRequest{
			TeamSlug:   teamSlug,
			SeasonSlug: seasonSlug,
			Name:       name,
		}),
	)
	if err != nil {
		t.Fatalf("CreateEvent(%q): %v", name, err)
	}
	return response.Msg.GetEvent()
}

func createPlayer(
	t *testing.T,
	actor *structureActor,
	teamSlug string,
	names ...string,
) *playersv1.Player {
	t.Helper()
	response, err := actor.players.CreatePlayer(
		context.Background(),
		connect.NewRequest(&playersv1.CreatePlayerRequest{
			TeamSlug: teamSlug,
			Names:    names,
		}),
	)
	if err != nil {
		t.Fatalf("CreatePlayer(%v): %v", names, err)
	}
	return response.Msg.GetPlayer()
}

func createMatch(
	t *testing.T,
	actor *structureActor,
	teamSlug string,
	seasonSlug string,
	eventSlug string,
	opponentName string,
) *matchesv1.Match {
	t.Helper()
	response, err := actor.matches.CreateMatch(
		context.Background(),
		connect.NewRequest(&matchesv1.CreateMatchRequest{
			TeamSlug:     teamSlug,
			SeasonSlug:   seasonSlug,
			EventSlug:    eventSlug,
			OpponentName: opponentName,
		}),
	)
	if err != nil {
		t.Fatalf("CreateMatch(%q): %v", opponentName, err)
	}
	return response.Msg.GetMatch()
}

// assertCode requires err to be a Connect error with the given code, without
// requiring a machine-readable detail (used for CodeUnimplemented).
func assertCode(t *testing.T, err error, code connect.Code) {
	t.Helper()
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("error = %v, want a *connect.Error", err)
	}
	if connectErr.Code() != code {
		t.Fatalf("connect code = %v, want %v", connectErr.Code(), code)
	}
}

func countRows(t *testing.T, sqlDB *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var count int64
	if err := sqlDB.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows (%s): %v", query, err)
	}
	return count
}

func TestTeamCRUDAndSlugStability(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")

	first := createTeam(t, owner, "Alpha Squad")
	if first.GetSlug() != "alpha-squad" {
		t.Fatalf("slug = %q, want %q", first.GetSlug(), "alpha-squad")
	}
	if first.GetName() != "Alpha Squad" {
		t.Fatalf("name = %q, want %q", first.GetName(), "Alpha Squad")
	}

	second := createTeam(t, owner, "Alpha Squad")
	if second.GetSlug() != "alpha-squad-2" {
		t.Fatalf("collision slug = %q, want %q", second.GetSlug(), "alpha-squad-2")
	}

	got, err := owner.teams.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{
		TeamSlug: first.GetSlug(),
	}))
	if err != nil {
		t.Fatalf("GetTeam: %v", err)
	}
	if got.Msg.GetTeam().GetId() != first.GetId() {
		t.Fatalf("GetTeam id = %q, want %q", got.Msg.GetTeam().GetId(), first.GetId())
	}

	renamed, err := owner.teams.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{
		TeamSlug: first.GetSlug(),
		Name:     ptrTo("Renamed Squad"),
	}))
	if err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	if renamed.Msg.GetTeam().GetName() != "Renamed Squad" {
		t.Fatalf("renamed name = %q", renamed.Msg.GetTeam().GetName())
	}
	if renamed.Msg.GetTeam().GetSlug() != first.GetSlug() {
		t.Fatalf("slug changed on rename: %q -> %q", first.GetSlug(), renamed.Msg.GetTeam().GetSlug())
	}

	listed, err := owner.teams.ListMyTeams(ctx, connect.NewRequest(&teamsv1.ListMyTeamsRequest{}))
	if err != nil {
		t.Fatalf("ListMyTeams: %v", err)
	}
	if len(listed.Msg.GetMemberships()) != 2 {
		t.Fatalf("memberships = %d, want 2", len(listed.Msg.GetMemberships()))
	}
	for _, membership := range listed.Msg.GetMemberships() {
		if membership.GetRole() != "owner" {
			t.Fatalf("role = %q, want owner", membership.GetRole())
		}
		if membership.GetTeam().GetId() == "" {
			t.Fatal("membership has no team id")
		}
	}

	if _, err := owner.teams.DeleteTeam(ctx, connect.NewRequest(&teamsv1.DeleteTeamRequest{
		TeamSlug: first.GetSlug(),
	})); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	if _, err := owner.teams.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{
		TeamSlug: first.GetSlug(),
	})); err == nil {
		t.Fatal("GetTeam after delete succeeded")
	} else {
		assertConnectError(t, err, connect.CodeNotFound, "not_found")
	}
}

func TestTeamSlugFallbackForNonLatinName(t *testing.T) {
	env := newStructureEnv(t)
	owner := env.newActor("owner@example.com")

	first := createTeam(t, owner, "日本チーム")
	second := createTeam(t, owner, "日本チーム")

	if first.GetSlug() == "" || second.GetSlug() == "" {
		t.Fatalf("fallback slugs = %q, %q; want non-empty", first.GetSlug(), second.GetSlug())
	}
	if first.GetSlug() == second.GetSlug() {
		t.Fatalf("fallback slugs collided: %q", first.GetSlug())
	}
	for _, slug := range []string{first.GetSlug(), second.GetSlug()} {
		if len(slug) != 32 {
			t.Fatalf("fallback slug = %q, want a 32-character id-derived slug", slug)
		}
	}
}

func TestTeamGating(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	member := env.newActor("member@example.com")
	outsider := env.newActor("outsider@example.com")

	team := createTeam(t, owner, "Gated Team")
	if _, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
		TeamSlug: team.GetSlug(),
		Email:    "member@example.com",
	})); err != nil {
		t.Fatalf("AddMembership: %v", err)
	}

	t.Run("member may read and mutate structure", func(t *testing.T) {
		if _, err := member.teams.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{
			TeamSlug: team.GetSlug(),
		})); err != nil {
			t.Fatalf("member GetTeam: %v", err)
		}
		if _, err := member.seasons.CreateSeason(ctx, connect.NewRequest(&seasonsv1.CreateSeasonRequest{
			TeamSlug: team.GetSlug(),
			Name:     "Member Season",
		})); err != nil {
			t.Fatalf("member CreateSeason: %v", err)
		}
	})

	t.Run("member cannot use owner-only operations", func(t *testing.T) {
		cases := map[string]func() error{
			"UpdateTeam": func() error {
				_, err := member.teams.UpdateTeam(ctx, connect.NewRequest(&teamsv1.UpdateTeamRequest{
					TeamSlug: team.GetSlug(),
					Name:     ptrTo("Hijacked"),
				}))
				return err
			},
			"DeleteTeam": func() error {
				_, err := member.teams.DeleteTeam(ctx, connect.NewRequest(&teamsv1.DeleteTeamRequest{
					TeamSlug: team.GetSlug(),
				}))
				return err
			},
			"AddMembership": func() error {
				_, err := member.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
					TeamSlug: team.GetSlug(),
					Email:    "outsider@example.com",
				}))
				return err
			},
			"DeleteSeason": func() error {
				_, err := member.seasons.DeleteSeason(ctx, connect.NewRequest(&seasonsv1.DeleteSeasonRequest{
					TeamSlug:   team.GetSlug(),
					SeasonSlug: "does-not-matter",
				}))
				return err
			},
			"DeletePlayer": func() error {
				_, err := member.players.DeletePlayer(ctx, connect.NewRequest(&playersv1.DeletePlayerRequest{
					TeamSlug:   team.GetSlug(),
					PlayerSlug: "does-not-matter",
				}))
				return err
			},
		}
		for name, call := range cases {
			t.Run(name, func(t *testing.T) {
				err := call()
				if err == nil {
					t.Fatal("call succeeded, want permission_denied")
				}
				assertConnectError(t, err, connect.CodePermissionDenied, "permission_denied")
			})
		}
	})

	t.Run("non-member is denied", func(t *testing.T) {
		_, err := outsider.teams.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{
			TeamSlug: team.GetSlug(),
		}))
		if err == nil {
			t.Fatal("non-member GetTeam succeeded")
		}
		assertConnectError(t, err, connect.CodePermissionDenied, "permission_denied")
	})

	t.Run("anonymous request is rejected by the session gate", func(t *testing.T) {
		_, err := env.anonymousTeamClient().GetTeam(
			ctx,
			connect.NewRequest(&teamsv1.GetTeamRequest{TeamSlug: team.GetSlug()}),
		)
		if err == nil {
			t.Fatal("anonymous GetTeam succeeded")
		}
		assertConnectError(t, err, connect.CodeUnauthenticated, "session_expired")
	})

	t.Run("unknown team is not_found", func(t *testing.T) {
		_, err := outsider.teams.GetTeam(ctx, connect.NewRequest(&teamsv1.GetTeamRequest{
			TeamSlug: "no-such-team",
		}))
		if err == nil {
			t.Fatal("GetTeam of an unknown slug succeeded")
		}
		assertConnectError(t, err, connect.CodeNotFound, "not_found")
	})
}

func TestMembershipManagementAndLastOwnerGuard(t *testing.T) {
	env := newStructureEnv(t)
	ctx := context.Background()
	owner := env.newActor("owner@example.com")
	member := env.newActor("member@example.com")
	team := createTeam(t, owner, "Membership Team")

	t.Run("unknown email is user_not_found", func(t *testing.T) {
		_, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
			TeamSlug: team.GetSlug(),
			Email:    "nobody@example.com",
		}))
		if err == nil {
			t.Fatal("AddMembership with an unknown email succeeded")
		}
		assertConnectError(t, err, connect.CodeNotFound, "user_not_found")
	})

	if _, err := owner.teams.AddMembership(ctx, connect.NewRequest(&teamsv1.AddMembershipRequest{
		TeamSlug: team.GetSlug(),
		Email:    "member@example.com",
	})); err != nil {
		t.Fatalf("AddMembership: %v", err)
	}

	teams, err := member.teams.ListMyTeams(ctx, connect.NewRequest(&teamsv1.ListMyTeamsRequest{}))
	if err != nil {
		t.Fatalf("member ListMyTeams: %v", err)
	}
	if len(teams.Msg.GetMemberships()) != 1 {
		t.Fatalf("member memberships = %d, want 1", len(teams.Msg.GetMemberships()))
	}
	if teams.Msg.GetMemberships()[0].GetRole() != "member" {
		t.Fatalf("role = %q, want member", teams.Msg.GetMemberships()[0].GetRole())
	}

	t.Run("last owner cannot be demoted", func(t *testing.T) {
		_, err := owner.teams.UpdateMembership(ctx, connect.NewRequest(&teamsv1.UpdateMembershipRequest{
			TeamSlug: team.GetSlug(),
			UserId:   owner.userID,
			Role:     "member",
		}))
		if err == nil {
			t.Fatal("demoting the last owner succeeded")
		}
		assertConnectError(t, err, connect.CodePermissionDenied, "last_owner")
	})

	t.Run("last owner cannot be removed", func(t *testing.T) {
		_, err := owner.teams.RemoveMembership(ctx, connect.NewRequest(&teamsv1.RemoveMembershipRequest{
			TeamSlug: team.GetSlug(),
			UserId:   owner.userID,
		}))
		if err == nil {
			t.Fatal("removing the last owner succeeded")
		}
		assertConnectError(t, err, connect.CodePermissionDenied, "last_owner")
	})

	t.Run("invalid role is rejected", func(t *testing.T) {
		_, err := owner.teams.UpdateMembership(ctx, connect.NewRequest(&teamsv1.UpdateMembershipRequest{
			TeamSlug: team.GetSlug(),
			UserId:   member.userID,
			Role:     "superuser",
		}))
		if err == nil {
			t.Fatal("invalid role succeeded")
		}
		assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
		assertFieldViolation(t, err, "role")
	})

	t.Run("unknown membership is not_found", func(t *testing.T) {
		_, err := owner.teams.UpdateMembership(ctx, connect.NewRequest(&teamsv1.UpdateMembershipRequest{
			TeamSlug: team.GetSlug(),
			UserId:   "not-a-member",
			Role:     "member",
		}))
		if err == nil {
			t.Fatal("update of an unknown membership succeeded")
		}
		assertConnectError(t, err, connect.CodeNotFound, "not_found")
	})

	if _, err := owner.teams.UpdateMembership(ctx, connect.NewRequest(&teamsv1.UpdateMembershipRequest{
		TeamSlug: team.GetSlug(),
		UserId:   member.userID,
		Role:     "owner",
	})); err != nil {
		t.Fatalf("promote member to owner: %v", err)
	}

	if _, err := owner.teams.UpdateMembership(ctx, connect.NewRequest(&teamsv1.UpdateMembershipRequest{
		TeamSlug: team.GetSlug(),
		UserId:   owner.userID,
		Role:     "member",
	})); err != nil {
		t.Fatalf("demote owner while another owner exists: %v", err)
	}

	if _, err := member.teams.RemoveMembership(ctx, connect.NewRequest(&teamsv1.RemoveMembershipRequest{
		TeamSlug: team.GetSlug(),
		UserId:   owner.userID,
	})); err != nil {
		t.Fatalf("remove demoted owner: %v", err)
	}

	t.Run("GetTeamRecord belongs to ticket 05", func(t *testing.T) {
		_, err := owner.teams.GetTeamRecord(ctx, connect.NewRequest(&teamsv1.GetTeamRecordRequest{
			TeamSlug: team.GetSlug(),
		}))
		assertCode(t, err, connect.CodeUnimplemented)
	})
}
