package structure

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	eventsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1/eventsv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/slugify"
)

// EventService implements the EventService Connect handler: events and their
// rosters.
type EventService struct {
	base
}

var _ eventsv1connect.EventServiceHandler = (*EventService)(nil)

// NewEventService builds an EventService over the given database.
func NewEventService(sqlDB *sql.DB, now func() time.Time) *EventService {
	return &EventService{base: newBase(sqlDB, now)}
}

// CreateEvent creates an event under a season; the slug is unique within the
// season.
func (s *EventService) CreateEvent(
	ctx context.Context,
	req *connect.Request[eventsv1.CreateEventRequest],
) (*connect.Response[eventsv1.CreateEventResponse], error) {
	_, _, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
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
			count, countErr := s.queries.EventSlugExists(ctx, db.EventSlugExistsParams{
				SeasonID: season.ID,
				Slug:     candidate,
			})
			return count > 0, countErr
		},
	)
	if err != nil {
		return nil, internalError()
	}
	event, err := s.queries.CreateEvent(ctx, db.CreateEventParams{
		ID:        id,
		SeasonID:  season.ID,
		Name:      name,
		Slug:      slug,
		StartDate: nullStringForNullable(req.Msg.GetStartDate()),
		EndDate:   nullStringForNullable(req.Msg.GetEndDate()),
		Location:  nullStringForNullable(req.Msg.GetLocation()),
		CreatedAt: s.timestamp(),
	})
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&eventsv1.CreateEventResponse{
		Event: eventMessage(&event),
	}), nil
}

// GetEvent returns an event to any team member.
func (s *EventService) GetEvent(
	ctx context.Context,
	req *connect.Request[eventsv1.GetEventRequest],
) (*connect.Response[eventsv1.GetEventResponse], error) {
	_, _, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
	)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&eventsv1.GetEventResponse{
		Event: eventMessage(&event),
	}), nil
}

// UpdateEvent applies the present optional fields; empty nullable strings
// clear their columns. The slug never changes.
func (s *EventService) UpdateEvent(
	ctx context.Context,
	req *connect.Request[eventsv1.UpdateEventRequest],
) (*connect.Response[eventsv1.UpdateEventResponse], error) {
	_, _, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
	)
	if err != nil {
		return nil, err
	}
	if raw, present := stringPointer(req.Msg.Name); present {
		name, validationErr := requiredText(raw, "name")
		if validationErr != nil {
			return nil, validationErr
		}
		if err := s.queries.UpdateEventName(ctx, db.UpdateEventNameParams{
			Name: name,
			ID:   event.ID,
		}); err != nil {
			return nil, internalError()
		}
		event.Name = name
	}
	if raw, present := stringPointer(req.Msg.StartDate); present {
		startDate := nullStringForNullable(raw)
		if err := s.queries.UpdateEventStartDate(ctx, db.UpdateEventStartDateParams{
			StartDate: startDate,
			ID:        event.ID,
		}); err != nil {
			return nil, internalError()
		}
		event.StartDate = startDate
	}
	if raw, present := stringPointer(req.Msg.EndDate); present {
		endDate := nullStringForNullable(raw)
		if err := s.queries.UpdateEventEndDate(ctx, db.UpdateEventEndDateParams{
			EndDate: endDate,
			ID:      event.ID,
		}); err != nil {
			return nil, internalError()
		}
		event.EndDate = endDate
	}
	if raw, present := stringPointer(req.Msg.Location); present {
		location := nullStringForNullable(raw)
		if err := s.queries.UpdateEventLocation(ctx, db.UpdateEventLocationParams{
			Location: location,
			ID:       event.ID,
		}); err != nil {
			return nil, internalError()
		}
		event.Location = location
	}
	return connect.NewResponse(&eventsv1.UpdateEventResponse{
		Event: eventMessage(&event),
	}), nil
}

// ListEvents returns a page of the season's events plus the unpaginated total.
func (s *EventService) ListEvents(
	ctx context.Context,
	req *connect.Request[eventsv1.ListEventsRequest],
) (*connect.Response[eventsv1.ListEventsResponse], error) {
	_, _, season, err := s.authorizeSeason(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
	)
	if err != nil {
		return nil, err
	}
	offset, limit := pageBounds(req.Msg.GetOffset(), req.Msg.GetLimit())
	events, err := s.queries.ListEvents(ctx, db.ListEventsParams{
		SeasonID: season.ID,
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return nil, internalError()
	}
	total, err := s.queries.CountEvents(ctx, season.ID)
	if err != nil {
		return nil, internalError()
	}
	messages := make([]*eventsv1.Event, 0, len(events))
	for i := range events {
		messages = append(messages, eventMessage(&events[i]))
	}
	return connect.NewResponse(&eventsv1.ListEventsResponse{
		Events: messages,
		Total:  total,
	}), nil
}

// DeleteEvent hard-deletes an event and everything that cascades from it. Any
// team member may delete an event.
func (s *EventService) DeleteEvent(
	ctx context.Context,
	req *connect.Request[eventsv1.DeleteEventRequest],
) (*connect.Response[eventsv1.DeleteEventResponse], error) {
	_, _, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
	)
	if err != nil {
		return nil, err
	}
	if err := s.queries.DeleteEvent(ctx, event.ID); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&eventsv1.DeleteEventResponse{}), nil
}

// AddEventRosterEntry registers a team player on an event roster. Adding an
// already-registered player is a no-op.
func (s *EventService) AddEventRosterEntry(
	ctx context.Context,
	req *connect.Request[eventsv1.AddEventRosterEntryRequest],
) (*connect.Response[eventsv1.AddEventRosterEntryResponse], error) {
	_, team, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
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
	if addErr := s.queries.AddEventRosterEntry(ctx, db.AddEventRosterEntryParams{
		ID:        entryID,
		EventID:   event.ID,
		PlayerID:  player.ID,
		CreatedAt: s.timestamp(),
	}); addErr != nil {
		return nil, internalError()
	}
	message, err := playerMessage(&player)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&eventsv1.AddEventRosterEntryResponse{
		Player: message,
	}), nil
}

// RemoveEventRosterEntry removes a player from an event roster. Removing an
// absent entry is a no-op.
func (s *EventService) RemoveEventRosterEntry(
	ctx context.Context,
	req *connect.Request[eventsv1.RemoveEventRosterEntryRequest],
) (*connect.Response[eventsv1.RemoveEventRosterEntryResponse], error) {
	_, team, _, event, err := s.authorizeEvent(
		ctx,
		req.Msg.GetTeamSlug(),
		req.Msg.GetSeasonSlug(),
		req.Msg.GetEventSlug(),
	)
	if err != nil {
		return nil, err
	}
	player, err := s.playerBySlug(ctx, team.ID, req.Msg.GetPlayerSlug())
	if err != nil {
		return nil, err
	}
	if err := s.queries.RemoveEventRosterEntry(ctx, db.RemoveEventRosterEntryParams{
		EventID:  event.ID,
		PlayerID: player.ID,
	}); err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&eventsv1.RemoveEventRosterEntryResponse{}), nil
}

// ListEventRoster returns a page of the event's registered players plus the
// unpaginated total.
func (s *EventService) ListEventRoster(
	ctx context.Context,
	req *connect.Request[eventsv1.ListEventRosterRequest],
) (*connect.Response[eventsv1.ListEventRosterResponse], error) {
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
	players, err := s.queries.ListEventRoster(ctx, db.ListEventRosterParams{
		EventID: event.ID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, internalError()
	}
	total, err := s.queries.CountEventRoster(ctx, event.ID)
	if err != nil {
		return nil, internalError()
	}
	messages, err := playerMessages(players)
	if err != nil {
		return nil, internalError()
	}
	return connect.NewResponse(&eventsv1.ListEventRosterResponse{
		Players: messages,
		Total:   total,
	}), nil
}
