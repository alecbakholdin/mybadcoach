package structure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1/matchesv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/slugify"
)

// MatchService implements the MatchService Connect handler's structure half.
// The gameplay RPCs belong to ticket 05 and return CodeUnimplemented.
type MatchService struct {
	base
}

var _ matchesv1connect.MatchServiceHandler = (*MatchService)(nil)

// NewMatchService builds a MatchService over the given database.
func NewMatchService(sqlDB *sql.DB, now func() time.Time) *MatchService {
	return &MatchService{base: newBase(sqlDB, now)}
}

// CreateMatch creates a match under an event; the slug derives from the
// opponent name and is unique within the event.
func (s *MatchService) CreateMatch(
	ctx context.Context,
	req *connect.Request[matchesv1.CreateMatchRequest],
) (*connect.Response[matchesv1.CreateMatchResponse], error) {
	_, team, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
	)
	if err != nil {
		return nil, err
	}
	opponentName, validationErr := requiredText(req.Msg.GetOpponentName(), "opponent_name")
	if validationErr != nil {
		return nil, validationErr
	}
	pointCap, pointCapPresent := int64Pointer(req.Msg.PointCap)
	if pointCapPresent && pointCap < 0 {
		return nil, invalidArgument(fieldViolation("point_cap", "must not be negative"))
	}
	id, err := newID()
	if err != nil {
		return nil, internalError()
	}
	slug, err := uniqueSlug(
		slugify.Slug(opponentName),
		id,
		func(candidate string) (bool, error) {
			count, countErr := s.queries.MatchSlugExists(ctx, db.MatchSlugExistsParams{
				EventID: event.ID,
				Slug:    candidate,
			})
			return count > 0, countErr
		},
	)
	if err != nil {
		return nil, internalError()
	}
	var scheduledAt sql.NullString
	var capValue sql.NullInt64
	if raw, present := stringPointer(req.Msg.ScheduledAt); present {
		scheduledAt = nullStringForNullable(raw)
	}
	if pointCapPresent {
		capValue = nullInt64For(pointCap)
	}
	match, err := s.queries.CreateMatch(ctx, db.CreateMatchParams{
		ID:           id,
		EventID:      event.ID,
		OurTeamID:    team.ID,
		OpponentName: opponentName,
		Slug:         slug,
		ScheduledAt:  scheduledAt,
		PointCap:     capValue,
		CreatedAt:    s.timestamp(),
	})
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&matchesv1.CreateMatchResponse{
		Match: matchMessage(&match),
	}), nil
}

// GetMatch returns a match to any team member.
func (s *MatchService) GetMatch(
	ctx context.Context,
	req *connect.Request[matchesv1.GetMatchRequest],
) (*connect.Response[matchesv1.GetMatchResponse], error) {
	match, err := s.authorizeMatch(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
		req.Msg.GetMatchSlug(),
	)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&matchesv1.GetMatchResponse{
		Match: matchMessage(&match),
	}), nil
}

// UpdateMatch applies the present optional fields; an empty scheduled_at
// clears its column. The slug never changes.
func (s *MatchService) UpdateMatch(
	ctx context.Context,
	req *connect.Request[matchesv1.UpdateMatchRequest],
) (*connect.Response[matchesv1.UpdateMatchResponse], error) {
	match, err := s.authorizeMatch(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
		req.Msg.GetMatchSlug(),
	)
	if err != nil {
		return nil, err
	}
	if raw, present := stringPointer(req.Msg.OpponentName); present {
		opponentName, validationErr := requiredText(raw, "opponent_name")
		if validationErr != nil {
			return nil, validationErr
		}
		if err := s.queries.UpdateMatchOpponentName(
			ctx,
			db.UpdateMatchOpponentNameParams{
				OpponentName: opponentName,
				ID:           match.ID,
			},
		); err != nil {
			return nil, internalError()
		}
		match.OpponentName = opponentName
	}
	if raw, present := stringPointer(req.Msg.ScheduledAt); present {
		scheduledAt := nullStringForNullable(raw)
		if err := s.queries.UpdateMatchScheduledAt(
			ctx,
			db.UpdateMatchScheduledAtParams{
				ScheduledAt: scheduledAt,
				ID:          match.ID,
			},
		); err != nil {
			return nil, internalError()
		}
		match.ScheduledAt = scheduledAt
	}
	if pointCap, present := int64Pointer(req.Msg.PointCap); present {
		if pointCap < 0 {
			return nil, invalidArgument(fieldViolation("point_cap", "must not be negative"))
		}
		capValue := nullInt64For(pointCap)
		if err := s.queries.UpdateMatchPointCap(
			ctx,
			db.UpdateMatchPointCapParams{
				PointCap: capValue,
				ID:       match.ID,
			},
		); err != nil {
			return nil, internalError()
		}
		match.PointCap = capValue
	}
	return connect.NewResponse(&matchesv1.UpdateMatchResponse{
		Match: matchMessage(&match),
	}), nil
}

// DeleteMatch hard-deletes a match and its gameplay events. Any team member
// may delete a match.
func (s *MatchService) DeleteMatch(
	ctx context.Context,
	req *connect.Request[matchesv1.DeleteMatchRequest],
) (*connect.Response[matchesv1.DeleteMatchResponse], error) {
	match, err := s.authorizeMatch(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
		req.Msg.GetMatchSlug(),
	)
	if err != nil {
		return nil, err
	}
	if err := s.queries.DeleteMatch(ctx, match.ID); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&matchesv1.DeleteMatchResponse{}), nil
}

// ListMatches returns a page of the event's matches plus the unpaginated
// total.
func (s *MatchService) ListMatches(
	ctx context.Context,
	req *connect.Request[matchesv1.ListMatchesRequest],
) (*connect.Response[matchesv1.ListMatchesResponse], error) {
	_, _, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
	)
	if err != nil {
		return nil, err
	}
	offset, limit := pageBounds(req.Msg.GetOffset(), req.Msg.GetLimit())
	matches, err := s.queries.ListMatches(ctx, db.ListMatchesParams{
		EventID: event.ID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, internalError()
	}
	total, err := s.queries.CountMatches(ctx, event.ID)
	if err != nil {
		return nil, internalError()
	}
	messages := make([]*matchesv1.Match, 0, len(matches))
	for i := range matches {
		messages = append(messages, matchMessage(&matches[i]))
	}
	return connect.NewResponse(&matchesv1.ListMatchesResponse{
		Matches: messages,
		Total:   total,
	}), nil
}

// AppendMatchEvents is implemented by ticket 05.
func (s *MatchService) AppendMatchEvents(
	_ context.Context,
	_ *connect.Request[matchesv1.AppendMatchEventsRequest],
) (*connect.Response[matchesv1.AppendMatchEventsResponse], error) {
	return nil, unimplemented("AppendMatchEvents")
}

// ListMatchEvents is implemented by ticket 05.
func (s *MatchService) ListMatchEvents(
	_ context.Context,
	_ *connect.Request[matchesv1.ListMatchEventsRequest],
) (*connect.Response[matchesv1.ListMatchEventsResponse], error) {
	return nil, unimplemented("ListMatchEvents")
}

// GetMatchState is implemented by ticket 05.
func (s *MatchService) GetMatchState(
	_ context.Context,
	_ *connect.Request[matchesv1.GetMatchStateRequest],
) (*connect.Response[matchesv1.GetMatchStateResponse], error) {
	return nil, unimplemented("GetMatchState")
}

// GetMatchStats is implemented by ticket 05.
func (s *MatchService) GetMatchStats(
	_ context.Context,
	_ *connect.Request[matchesv1.GetMatchStatsRequest],
) (*connect.Response[matchesv1.GetMatchStatsResponse], error) {
	return nil, unimplemented("GetMatchStats")
}

// unimplemented builds the CodeUnimplemented error returned by ticket 05's
// gameplay RPCs.
func unimplemented(method string) *connect.Error {
	return connect.NewError(
		connect.CodeUnimplemented,
		errors.New(method+" is implemented by ticket 05"),
	)
}
