package structure

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	"github.com/alecbakholdin/mybadcoach/server/internal/gameplay"
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1/matchesv1connect"
	idgen "github.com/alecbakholdin/mybadcoach/server/internal/id"
	"github.com/alecbakholdin/mybadcoach/server/internal/reduce"
	"github.com/alecbakholdin/mybadcoach/server/internal/slugify"
)

// MatchService implements the MatchService Connect handler: match structure
// plus the gameplay event log, match state/stats, and event append/read RPCs.
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
	id, err := idgen.New()
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

// maxEventBatch is the largest AppendMatchEvents batch accepted in one call.
const maxEventBatch = 1000

// preparedEvent is one validated event, converted to its storage columns.
type preparedEvent struct {
	id          string
	eventType   string
	payload     string
	createdAt   string
	pointNumber int64
	effectiveTS int64
	deleted     int64
}

// AppendMatchEvents upserts a batch of gameplay events by id. Identical
// replays are skipped instead of erroring, an id re-used with different
// content is rejected with id_conflict, and the deleted tombstone flag is
// last-writer-wins. The whole batch is validated before any write and applied
// in one transaction, so a malformed event cannot leave a partial append.
func (s *MatchService) AppendMatchEvents(
	ctx context.Context,
	req *connect.Request[matchesv1.AppendMatchEventsRequest],
) (*connect.Response[matchesv1.AppendMatchEventsResponse], error) {
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

	events := req.Msg.GetEvents()
	if len(events) > maxEventBatch {
		return nil, invalidArgument(fieldViolation(
			"events",
			fmt.Sprintf("must contain at most %d events", maxEventBatch),
		))
	}

	prepared := make([]preparedEvent, 0, len(events))
	for index, event := range events {
		eventID := event.GetId()
		if eventID == "" {
			return nil, invalidArgument(fieldViolation(
				fmt.Sprintf("events[%d].id", index),
				"must not be empty",
			))
		}
		// The event's match id is optional (the URL identifies the match), but
		// when supplied it must agree with the addressed match.
		if event.GetMatchId() != "" && event.GetMatchId() != match.ID {
			return nil, invalidArgument(fieldViolation(
				fmt.Sprintf("events[%d].match_id", index),
				"must match the addressed match",
			))
		}
		eventType, payload, storageErr := gameplay.EventStorageColumns(event)
		if storageErr != nil {
			return nil, invalidArgument(fieldViolation(
				fmt.Sprintf("events[%d]", index),
				storageErr.Error(),
			))
		}
		createdAt := event.GetCreatedAt()
		if createdAt == "" {
			createdAt = s.timestamp()
		}
		prepared = append(prepared, preparedEvent{
			id:          eventID,
			pointNumber: event.GetPointNumber(),
			effectiveTS: event.GetEffectiveTs(),
			eventType:   eventType,
			payload:     payload,
			deleted:     boolToInt64(event.GetDeleted()),
			createdAt:   createdAt,
		})
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, internalError()
	}
	defer func() { _ = tx.Rollback() }()

	queries := s.queries.WithTx(tx)
	accepted := make([]string, 0, len(prepared))
	skipped := make([]string, 0)
	for index := range prepared {
		event := &prepared[index]
		existing, getErr := queries.GetMatchEventByID(ctx, event.id)
		switch {
		case errors.Is(getErr, sql.ErrNoRows):
			if createErr := queries.CreateMatchEvent(ctx, db.CreateMatchEventParams{
				ID:          event.id,
				MatchID:     match.ID,
				PointNumber: event.pointNumber,
				EffectiveTs: event.effectiveTS,
				Type:        event.eventType,
				Payload:     event.payload,
				Deleted:     event.deleted,
				CreatedAt:   event.createdAt,
			}); createErr != nil {
				return nil, internalError()
			}
			accepted = append(accepted, event.id)
		case getErr != nil:
			return nil, internalError()
		case !sameEventContent(&existing, match.ID, event):
			return nil, idConflictError(fmt.Sprintf("events[%d].id", index))
		case existing.Deleted != event.deleted:
			// Same content, changed tombstone: last writer wins.
			if updateErr := queries.UpdateMatchEventDeleted(ctx, db.UpdateMatchEventDeletedParams{
				Deleted: event.deleted,
				ID:      event.id,
			}); updateErr != nil {
				return nil, internalError()
			}
			accepted = append(accepted, event.id)
		default:
			skipped = append(skipped, event.id)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, internalError()
	}

	return connect.NewResponse(&matchesv1.AppendMatchEventsResponse{
		AcceptedIds:         accepted,
		SkippedDuplicateIds: skipped,
	}), nil
}

// sameEventContent reports whether a stored event already holds the submitted
// event's content. Created_at is metadata rather than content, so a replay that
// regenerates it is still the same event.
func sameEventContent(existing *db.MatchEvent, matchID string, event *preparedEvent) bool {
	return existing.MatchID == matchID &&
		existing.PointNumber == event.pointNumber &&
		existing.EffectiveTs == event.effectiveTS &&
		existing.Type == event.eventType &&
		existing.Payload == event.payload
}

// ListMatchEvents returns a page of the match's event log, newest last, with an
// optional point filter. Tombstones are included: they are part of the
// offline-sync contract, not deletion semantics.
func (s *MatchService) ListMatchEvents(
	ctx context.Context,
	req *connect.Request[matchesv1.ListMatchEventsRequest],
) (*connect.Response[matchesv1.ListMatchEventsResponse], error) {
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
	offset, limit := pageBounds(req.Msg.GetOffset(), req.Msg.GetLimit())

	var rows []db.MatchEvent
	var total int64
	if pointNumber, present := int64Pointer(req.Msg.PointNumber); present {
		rows, err = s.queries.ListMatchEventsByPoint(ctx, db.ListMatchEventsByPointParams{
			MatchID:     match.ID,
			PointNumber: pointNumber,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return nil, internalError()
		}
		total, err = s.queries.CountMatchEventsByPoint(ctx, db.CountMatchEventsByPointParams{
			MatchID:     match.ID,
			PointNumber: pointNumber,
		})
		if err != nil {
			return nil, internalError()
		}
	} else {
		rows, err = s.queries.ListMatchEvents(ctx, db.ListMatchEventsParams{
			MatchID: match.ID,
			Limit:   limit,
			Offset:  offset,
		})
		if err != nil {
			return nil, internalError()
		}
		total, err = s.queries.CountMatchEvents(ctx, match.ID)
		if err != nil {
			return nil, internalError()
		}
	}
	messages := make([]*matchesv1.MatchEvent, 0, len(rows))
	for i := range rows {
		message, messageErr := matchEventMessage(&rows[i])
		if messageErr != nil {
			return nil, internalError()
		}
		messages = append(messages, message)
	}
	return connect.NewResponse(&matchesv1.ListMatchEventsResponse{
		Events: messages,
		Total:  total,
	}), nil
}

// GetMatchState folds every point of the match and returns them in ascending
// point order. Points are folded from STARTING_ROLE_OFFENSE: the event log does
// not record whether a point began on offense or defense, and reads need a
// deterministic default.
func (s *MatchService) GetMatchState(
	ctx context.Context,
	req *connect.Request[matchesv1.GetMatchStateRequest],
) (*connect.Response[matchesv1.GetMatchStateResponse], error) {
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
	points, err := s.reducedPoints(ctx, match.ID)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&matchesv1.GetMatchStateResponse{
		Points: points,
	}), nil
}

// GetMatchStats aggregates the match's reduced points into per-player totals
// (the Go port of aggregateMatchStats).
func (s *MatchService) GetMatchStats(
	ctx context.Context,
	req *connect.Request[matchesv1.GetMatchStatsRequest],
) (*connect.Response[matchesv1.GetMatchStatsResponse], error) {
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
	points, err := s.reducedPoints(ctx, match.ID)
	if err != nil {
		return nil, err
	}
	states := make([]*matchesv1.PointDerivedState, 0, len(points))
	for _, point := range points {
		states = append(states, point.GetState())
	}
	return connect.NewResponse(&matchesv1.GetMatchStatsResponse{
		PlayerStats: gameplay.AggregateMatchStats(states),
	}), nil
}

// reducedPoints loads the whole event log for a match and folds each point.
func (s *MatchService) reducedPoints(
	ctx context.Context,
	matchID string,
) ([]*matchesv1.MatchPointState, error) {
	rows, err := s.queries.ListAllMatchEvents(ctx, matchID)
	if err != nil {
		return nil, internalError()
	}

	byPoint := make(map[int64][]*matchesv1.MatchEvent)
	for i := range rows {
		event, messageErr := matchEventMessage(&rows[i])
		if messageErr != nil {
			return nil, internalError()
		}
		pointNumber := event.GetPointNumber()
		byPoint[pointNumber] = append(byPoint[pointNumber], event)
	}

	pointNumbers := make([]int64, 0, len(byPoint))
	for pointNumber := range byPoint {
		pointNumbers = append(pointNumbers, pointNumber)
	}
	sort.Slice(pointNumbers, func(i, j int) bool { return pointNumbers[i] < pointNumbers[j] })

	points := make([]*matchesv1.MatchPointState, 0, len(pointNumbers))
	for _, pointNumber := range pointNumbers {
		points = append(points, &matchesv1.MatchPointState{
			PointNumber: pointNumber,
			State:       reduce.ReducePoint(byPoint[pointNumber], matchesv1.StartingRole_STARTING_ROLE_OFFENSE),
		})
	}
	return points, nil
}

// boolToInt64 stores a bool in the SQLite integer column convention.
func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
