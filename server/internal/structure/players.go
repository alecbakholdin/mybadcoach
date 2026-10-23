package structure

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	playersv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1/playersv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/slugify"
)

// PlayerService implements the PlayerService Connect handler.
type PlayerService struct {
	base
}

var _ playersv1connect.PlayerServiceHandler = (*PlayerService)(nil)

// NewPlayerService builds a PlayerService over the given database.
func NewPlayerService(sqlDB *sql.DB, now func() time.Time) *PlayerService {
	return &PlayerService{base: newBase(sqlDB, now)}
}

// CreatePlayer creates a team player; the slug derives from the display name
// and is unique within the team.
func (s *PlayerService) CreatePlayer(
	ctx context.Context,
	req *connect.Request[playersv1.CreatePlayerRequest],
) (*connect.Response[playersv1.CreatePlayerResponse], error) {
	_, team, _, err := s.authorizeTeam(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	names, validationErr := normalizeNames(req.Msg.GetNames())
	if validationErr != nil {
		return nil, validationErr
	}
	encoded, err := marshalNames(names)
	if err != nil {
		return nil, internalError()
	}
	id, err := newID()
	if err != nil {
		return nil, internalError()
	}
	slug, err := uniqueSlug(
		slugify.Slug(names[0]),
		id,
		func(candidate string) (bool, error) {
			count, countErr := s.queries.PlayerSlugExists(ctx, db.PlayerSlugExistsParams{
				TeamID: team.ID,
				Slug:   candidate,
			})
			return count > 0, countErr
		},
	)
	if err != nil {
		return nil, internalError()
	}
	player, err := s.queries.CreatePlayer(ctx, db.CreatePlayerParams{
		ID:        id,
		TeamID:    team.ID,
		Names:     encoded,
		Slug:      slug,
		CreatedAt: s.timestamp(),
	})
	if err != nil {
		return nil, internalError()
	}
	message, err := playerMessage(&player)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&playersv1.CreatePlayerResponse{
		Player: message,
	}), nil
}

// GetPlayer returns a player to any team member.
func (s *PlayerService) GetPlayer(
	ctx context.Context,
	req *connect.Request[playersv1.GetPlayerRequest],
) (*connect.Response[playersv1.GetPlayerResponse], error) {
	_, _, player, err := s.authorizePlayer(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetPlayerSlug(),
	)
	if err != nil {
		return nil, err
	}
	message, err := playerMessage(&player)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&playersv1.GetPlayerResponse{
		Player: message,
	}), nil
}

// UpdatePlayer replaces the whole name list when it is non-empty; an empty
// list leaves the names unchanged. The slug never changes.
func (s *PlayerService) UpdatePlayer(
	ctx context.Context,
	req *connect.Request[playersv1.UpdatePlayerRequest],
) (*connect.Response[playersv1.UpdatePlayerResponse], error) {
	_, _, player, err := s.authorizePlayer(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetPlayerSlug(),
	)
	if err != nil {
		return nil, err
	}
	if len(req.Msg.GetNames()) > 0 {
		names, validationErr := normalizeNames(req.Msg.GetNames())
		if validationErr != nil {
			return nil, validationErr
		}
		encoded, marshalErr := marshalNames(names)
		if marshalErr != nil {
			return nil, internalError()
		}
		if updateErr := s.queries.UpdatePlayerNames(ctx, db.UpdatePlayerNamesParams{
			Names: encoded,
			ID:    player.ID,
		}); updateErr != nil {
			return nil, internalError()
		}
		player.Names = encoded
	}
	message, err := playerMessage(&player)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&playersv1.UpdatePlayerResponse{
		Player: message,
	}), nil
}

// DeletePlayer hard-deletes a player and its roster entries. It is
// owner-only and leaves the match-event log untouched.
func (s *PlayerService) DeletePlayer(
	ctx context.Context,
	req *connect.Request[playersv1.DeletePlayerRequest],
) (*connect.Response[playersv1.DeletePlayerResponse], error) {
	_, team, _, err := s.authorizeOwner(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	player, err := s.playerBySlug(ctx, team.ID, req.Msg.GetPlayerSlug())
	if err != nil {
		return nil, err
	}
	if err := s.queries.DeletePlayer(ctx, player.ID); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&playersv1.DeletePlayerResponse{}), nil
}

// ListPlayers returns a page of the team's players plus the unpaginated total.
func (s *PlayerService) ListPlayers(
	ctx context.Context,
	req *connect.Request[playersv1.ListPlayersRequest],
) (*connect.Response[playersv1.ListPlayersResponse], error) {
	_, team, _, err := s.authorizeTeam(ctx, req.Msg.GetTeamSlug())
	if err != nil {
		return nil, err
	}
	offset, limit := pageBounds(req.Msg.GetOffset(), req.Msg.GetLimit())
	players, err := s.queries.ListPlayers(ctx, db.ListPlayersParams{
		TeamID: team.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, internalError()
	}
	total, err := s.queries.CountPlayers(ctx, team.ID)
	if err != nil {
		return nil, internalError()
	}
	messages, err := playerMessages(players)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&playersv1.ListPlayersResponse{
		Players: messages,
		Total:   total,
	}), nil
}
