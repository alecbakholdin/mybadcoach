package structure

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	playersv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1"
	seasonsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1/seasonsv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/slugify"
)

// SeasonService implements the SeasonService Connect handler: seasons and
// their rosters.
type SeasonService struct {
	base
}

var _ seasonsv1connect.SeasonServiceHandler = (*SeasonService)(nil)

// NewSeasonService builds a SeasonService over the given database.
func NewSeasonService(sqlDB *sql.DB, now func() time.Time) *SeasonService {
	return &SeasonService{base: newBase(sqlDB, now)}
}

// CreateSeason creates a season under a team; the slug is unique within the
// team.
func (s *SeasonService) CreateSeason(
	ctx context.Context,
	req *connect.Request[seasonsv1.CreateSeasonRequest],
) (*connect.Response[seasonsv1.CreateSeasonResponse], error) {
	_, team, _, err := s.authorizeTeam(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	name, validationErr := requiredText(req.Msg.GetName(), "name")
	if validationErr != nil {
		return nil, validationErr
	}
	id, err := newID()
	if err != nil {
		return nil, internalError()
	}
	slug, err := uniqueSlug(
		slugify.Slug(name),
		id,
		func(candidate string) (bool, error) {
			count, countErr := s.queries.SeasonSlugExists(ctx, db.SeasonSlugExistsParams{
				TeamID: team.ID,
				Slug:   candidate,
			})
			return count > 0, countErr
		},
	)
	if err != nil {
		return nil, internalError()
	}
	season, err := s.queries.CreateSeason(ctx, db.CreateSeasonParams{
		ID:        id,
		TeamID:    team.ID,
		Name:      name,
		Slug:      slug,
		StartDate: nullStringForNullable(req.Msg.GetStartDate()),
		EndDate:   nullStringForNullable(req.Msg.GetEndDate()),
		CreatedAt: s.timestamp(),
	})
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&seasonsv1.CreateSeasonResponse{
		Season: seasonMessage(&season),
	}), nil
}

// GetSeason returns a season to any team member.
func (s *SeasonService) GetSeason(
	ctx context.Context,
	req *connect.Request[seasonsv1.GetSeasonRequest],
) (*connect.Response[seasonsv1.GetSeasonResponse], error) {
	_, _, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&seasonsv1.GetSeasonResponse{
		Season: seasonMessage(&season),
	}), nil
}

// UpdateSeason applies the present optional fields; empty dates clear their
// columns. The slug never changes.
func (s *SeasonService) UpdateSeason(
	ctx context.Context,
	req *connect.Request[seasonsv1.UpdateSeasonRequest],
) (*connect.Response[seasonsv1.UpdateSeasonResponse], error) {
	_, _, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
	if err != nil {
		return nil, err
	}
	if raw, present := stringPointer(req.Msg.Name); present {
		name, validationErr := requiredText(raw, "name")
		if validationErr != nil {
			return nil, validationErr
		}
		if err := s.queries.UpdateSeasonName(ctx, db.UpdateSeasonNameParams{
			Name: name,
			ID:   season.ID,
		}); err != nil {
			return nil, internalError()
		}
		season.Name = name
	}
	if raw, present := stringPointer(req.Msg.StartDate); present {
		startDate := nullStringForNullable(raw)
		if err := s.queries.UpdateSeasonStartDate(ctx, db.UpdateSeasonStartDateParams{
			StartDate: startDate,
			ID:        season.ID,
		}); err != nil {
			return nil, internalError()
		}
		season.StartDate = startDate
	}
	if raw, present := stringPointer(req.Msg.EndDate); present {
		endDate := nullStringForNullable(raw)
		if err := s.queries.UpdateSeasonEndDate(ctx, db.UpdateSeasonEndDateParams{
			EndDate: endDate,
			ID:      season.ID,
		}); err != nil {
			return nil, internalError()
		}
		season.EndDate = endDate
	}
	return connect.NewResponse(&seasonsv1.UpdateSeasonResponse{
		Season: seasonMessage(&season),
	}), nil
}

// ListSeasons returns a page of the team's seasons plus the unpaginated total.
func (s *SeasonService) ListSeasons(
	ctx context.Context,
	req *connect.Request[seasonsv1.ListSeasonsRequest],
) (*connect.Response[seasonsv1.ListSeasonsResponse], error) {
	_, team, _, err := s.authorizeTeam(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	offset, limit := pageBounds(req.Msg.GetOffset(), req.Msg.GetLimit())
	seasons, err := s.queries.ListSeasons(ctx, db.ListSeasonsParams{
		TeamID: team.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, internalError()
	}
	total, err := s.queries.CountSeasons(ctx, team.ID)
	if err != nil {
		return nil, internalError()
	}
	messages := make([]*seasonsv1.Season, 0, len(seasons))
	for i := range seasons {
		messages = append(messages, seasonMessage(&seasons[i]))
	}
	return connect.NewResponse(&seasonsv1.ListSeasonsResponse{
		Seasons: messages,
		Total:   total,
	}), nil
}

// DeleteSeason hard-deletes a season and everything that cascades from it. It
// is owner-only.
func (s *SeasonService) DeleteSeason(
	ctx context.Context,
	req *connect.Request[seasonsv1.DeleteSeasonRequest],
) (*connect.Response[seasonsv1.DeleteSeasonResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	season, err := s.seasonBySlug(ctx, team.ID, req.Msg.GetSeasonSlug())
	if err != nil {
		return nil, err
	}
	if err := s.queries.DeleteSeason(ctx, season.ID); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&seasonsv1.DeleteSeasonResponse{}), nil
}

// AddSeasonRosterEntry registers a team player on a season roster. Adding an
// already-registered player is a no-op.
func (s *SeasonService) AddSeasonRosterEntry(
	ctx context.Context,
	req *connect.Request[seasonsv1.AddSeasonRosterEntryRequest],
) (*connect.Response[seasonsv1.AddSeasonRosterEntryResponse], error) {
	_, team, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
	if err != nil {
		return nil, err
	}
	player, err := s.playerBySlug(ctx, team.ID, req.Msg.GetPlayerSlug())
	if err != nil {
		return nil, err
	}
	entryID, err := newID()
	if err != nil {
		return nil, internalError()
	}
	if addErr := s.queries.AddSeasonRosterEntry(ctx, db.AddSeasonRosterEntryParams{
		ID:        entryID,
		SeasonID:  season.ID,
		PlayerID:  player.ID,
		CreatedAt: s.timestamp(),
	}); addErr != nil {
		return nil, internalError()
	}
	message, err := playerMessage(&player)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&seasonsv1.AddSeasonRosterEntryResponse{
		Player: message,
	}), nil
}

// RemoveSeasonRosterEntry removes a player from a season roster. Removing an
// absent entry is a no-op.
func (s *SeasonService) RemoveSeasonRosterEntry(
	ctx context.Context,
	req *connect.Request[seasonsv1.RemoveSeasonRosterEntryRequest],
) (*connect.Response[seasonsv1.RemoveSeasonRosterEntryResponse], error) {
	_, team, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
	if err != nil {
		return nil, err
	}
	player, err := s.playerBySlug(ctx, team.ID, req.Msg.GetPlayerSlug())
	if err != nil {
		return nil, err
	}
	if err := s.queries.RemoveSeasonRosterEntry(ctx, db.RemoveSeasonRosterEntryParams{
		SeasonID: season.ID,
		PlayerID: player.ID,
	}); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&seasonsv1.RemoveSeasonRosterEntryResponse{}), nil
}

// ListSeasonRoster returns a page of the season's registered players plus the
// unpaginated total.
func (s *SeasonService) ListSeasonRoster(
	ctx context.Context,
	req *connect.Request[seasonsv1.ListSeasonRosterRequest],
) (*connect.Response[seasonsv1.ListSeasonRosterResponse], error) {
	_, _, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
	if err != nil {
		return nil, err
	}
	offset, limit := pageBounds(req.Msg.GetOffset(), req.Msg.GetLimit())
	players, err := s.queries.ListSeasonRoster(ctx, db.ListSeasonRosterParams{
		SeasonID: season.ID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, internalError()
	}
	total, err := s.queries.CountSeasonRoster(ctx, season.ID)
	if err != nil {
		return nil, internalError()
	}
	messages, err := playerMessages(players)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&seasonsv1.ListSeasonRosterResponse{
		Players: messages,
		Total:   total,
	}), nil
}

// CopySeasonRoster copies every source-roster player not already registered on
// the target roster and returns the players it added.
func (s *SeasonService) CopySeasonRoster(
	ctx context.Context,
	req *connect.Request[seasonsv1.CopySeasonRosterRequest],
) (*connect.Response[seasonsv1.CopySeasonRosterResponse], error) {
	_, team, _, err := s.authorizeTeam(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	from, err := s.seasonBySlug(ctx, team.ID, req.Msg.GetFromSeasonSlug())
	if err != nil {
		return nil, err
	}
	to, err := s.seasonBySlug(ctx, team.ID, req.Msg.GetToSeasonSlug())
	if err != nil {
		return nil, err
	}
	players, err := s.queries.ListSeasonRosterPlayersNotInTarget(
		ctx,
		db.ListSeasonRosterPlayersNotInTargetParams{
			SeasonID:   from.ID,
			SeasonID_2: to.ID,
		},
	)
	if err != nil {
		return nil, internalError()
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, internalError()
	}
	defer func() { _ = tx.Rollback() }()
	queries := s.queries.WithTx(tx)
	for i := range players {
		entryID, idErr := newID()
		if idErr != nil {
			return nil, internalError()
		}
		if addErr := queries.AddSeasonRosterEntry(ctx, db.AddSeasonRosterEntryParams{
			ID:        entryID,
			SeasonID:  to.ID,
			PlayerID:  players[i].ID,
			CreatedAt: s.timestamp(),
		}); addErr != nil {
			return nil, internalError()
		}
	}
	if commitErr := tx.Commit(); commitErr != nil {
		return nil, internalError()
	}

	messages, err := playerMessages(players)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&seasonsv1.CopySeasonRosterResponse{
		Copied: messages,
	}), nil
}

// playerMessages maps a slice of stored players to proto messages.
func playerMessages(players []db.Player) ([]*playersv1.Player, error) {
	messages := make([]*playersv1.Player, 0, len(players))
	for i := range players {
		message, err := playerMessage(&players[i])
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}
