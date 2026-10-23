package structure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/alecbakholdin/mybadcoach/server/internal/auth"
	"github.com/alecbakholdin/mybadcoach/server/internal/db"
)

const (
	roleOwner  = "owner"
	roleMember = "member"
)

// base carries the shared dependencies of every structure service.
type base struct {
	sqlDB   *sql.DB
	queries *db.Queries
	now     func() time.Time
}

func newBase(sqlDB *sql.DB, now func() time.Time) base {
	if now == nil {
		now = time.Now
	}
	return base{sqlDB: sqlDB, queries: db.New(sqlDB), now: now}
}

// currentUser returns the session user injected by the session middleware.
func currentUser(ctx context.Context) (auth.User, error) {
	session, ok := auth.SessionFromContext(ctx)
	if !ok {
		return auth.User{}, sessionExpiredError()
	}
	return session.User, nil
}

// teamBySlug resolves a globally unique team slug.
func (b *base) teamBySlug(ctx context.Context, slug string) (db.Team, error) {
	team, err := b.queries.GetTeamBySlug(ctx, slug)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return db.Team{}, notFound("team")
	case err != nil:
		return db.Team{}, internalError()
	}
	return team, nil
}

// membershipFor resolves the caller's membership in a team, refusing
// non-members with permission_denied.
func (b *base) membershipFor(
	ctx context.Context,
	teamID string,
	userID string,
) (db.TeamMembership, error) {
	membership, err := b.queries.GetMembership(ctx, db.GetMembershipParams{
		TeamID: teamID,
		UserID: userID,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return db.TeamMembership{}, permissionDenied()
	case err != nil:
		return db.TeamMembership{}, internalError()
	}
	return membership, nil
}

// authorizeTeam resolves the team and requires the caller to be a member.
// Named results keep the four-value shape readable at call sites.
func (b *base) authorizeTeam(
	ctx context.Context,
	teamSlug string,
) (user auth.User, team db.Team, membership db.TeamMembership, err error) {
	user, err = currentUser(ctx)
	if err != nil {
		return user, team, membership, err
	}
	team, err = b.teamBySlug(ctx, teamSlug)
	if err != nil {
		return user, team, membership, err
	}
	membership, err = b.membershipFor(ctx, team.ID, user.ID)
	return user, team, membership, err
}

// authorizeOwner resolves the team and requires the caller to be its owner.
func (b *base) authorizeOwner(
	ctx context.Context,
	teamSlug string,
) (user auth.User, team db.Team, membership db.TeamMembership, err error) {
	user, team, membership, err = b.authorizeTeam(ctx, teamSlug)
	if err != nil {
		return user, team, membership, err
	}
	if membership.Role != roleOwner {
		return user, team, membership, permissionDenied()
	}
	return user, team, membership, nil
}

// seasonBySlug resolves a season slug within its team.
func (b *base) seasonBySlug(
	ctx context.Context,
	teamID string,
	slug string,
) (db.Season, error) {
	season, err := b.queries.GetSeasonBySlug(ctx, db.GetSeasonBySlugParams{
		TeamID: teamID,
		Slug:   slug,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return db.Season{}, notFound("season")
	case err != nil:
		return db.Season{}, internalError()
	}
	return season, nil
}

// authorizeSeason resolves the team, requires membership, and resolves the
// season within it.
func (b *base) authorizeSeason(
	ctx context.Context,
	teamSlug string,
	seasonSlug string,
) (user auth.User, team db.Team, season db.Season, err error) {
	user, team, _, err = b.authorizeTeam(ctx, teamSlug)
	if err != nil {
		return user, team, season, err
	}
	season, err = b.seasonBySlug(ctx, team.ID, seasonSlug)
	return user, team, season, err
}

// eventBySlug resolves an event slug within its season.
func (b *base) eventBySlug(
	ctx context.Context,
	seasonID string,
	slug string,
) (db.Event, error) {
	event, err := b.queries.GetEventBySlug(ctx, db.GetEventBySlugParams{
		SeasonID: seasonID,
		Slug:     slug,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return db.Event{}, notFound("event")
	case err != nil:
		return db.Event{}, internalError()
	}
	return event, nil
}

// authorizeEvent resolves the team and season, requires membership, and
// resolves the event within the season.
func (b *base) authorizeEvent(
	ctx context.Context,
	teamSlug string,
	seasonSlug string,
	eventSlug string,
) (user auth.User, team db.Team, season db.Season, event db.Event, err error) {
	user, team, season, err = b.authorizeSeason(ctx, teamSlug, seasonSlug)
	if err != nil {
		return user, team, season, event, err
	}
	event, err = b.eventBySlug(ctx, season.ID, eventSlug)
	return user, team, season, event, err
}

// matchBySlug resolves a match slug within its event.
func (b *base) matchBySlug(
	ctx context.Context,
	eventID string,
	slug string,
) (db.Match, error) {
	match, err := b.queries.GetMatchBySlug(ctx, db.GetMatchBySlugParams{
		EventID: eventID,
		Slug:    slug,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return db.Match{}, notFound("match")
	case err != nil:
		return db.Match{}, internalError()
	}
	return match, nil
}

// authorizeMatch resolves the event, requires membership, and resolves the
// match within it.
func (b *base) authorizeMatch(
	ctx context.Context,
	teamSlug string,
	seasonSlug string,
	eventSlug string,
	matchSlug string,
) (db.Match, error) {
	_, _, _, event, err := b.authorizeEvent(ctx, teamSlug, seasonSlug, eventSlug)
	if err != nil {
		return db.Match{}, err
	}
	return b.matchBySlug(ctx, event.ID, matchSlug)
}

// playerBySlug resolves a player slug within its team.
func (b *base) playerBySlug(
	ctx context.Context,
	teamID string,
	slug string,
) (db.Player, error) {
	player, err := b.queries.GetPlayerBySlug(ctx, db.GetPlayerBySlugParams{
		TeamID: teamID,
		Slug:   slug,
	})
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return db.Player{}, notFound("player")
	case err != nil:
		return db.Player{}, internalError()
	}
	return player, nil
}

// authorizePlayer resolves the team, requires membership, and resolves the
// player within it.
func (b *base) authorizePlayer(
	ctx context.Context,
	teamSlug string,
	playerSlug string,
) (user auth.User, team db.Team, player db.Player, err error) {
	user, team, _, err = b.authorizeTeam(ctx, teamSlug)
	if err != nil {
		return user, team, player, err
	}
	player, err = b.playerBySlug(ctx, team.ID, playerSlug)
	return user, team, player, err
}
