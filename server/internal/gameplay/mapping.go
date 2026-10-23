// Package gameplay holds the gameplay-side plumbing shared by the reducer's
// callers: the wire<->fixture<->storage mapping and the match-scoped stat and
// team-record aggregations.
//
// The mapping layer is deliberately the only place that knows the fixture
// spelling of enums (short names such as TOUCH) and the storage encoding of
// match_events rows. The reducer itself consumes generated types directly.
package gameplay

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

// matchEventTypeShortNames maps generated enum values to the short names used
// by fixtures and by the match_events.type column.
var matchEventTypeShortNames = map[matchesv1.MatchEventType]string{
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH:             "TOUCH",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_DROP:              "DROP",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_TURNOVER:          "TURNOVER",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_DEFENSIVE_BLOCK:   "DEFENSIVE_BLOCK",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_OPPONENT_TURNOVER: "OPPONENT_TURNOVER",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE:             "SCORE",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP:        "SET_LINEUP",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_SUB_PLAYER:        "SUB_PLAYER",
	matchesv1.MatchEventType_MATCH_EVENT_TYPE_PULL:              "PULL",
}

// matchEventTypesByShortName is the inverse of matchEventTypeShortNames.
var matchEventTypesByShortName = func() map[string]matchesv1.MatchEventType {
	types := make(map[string]matchesv1.MatchEventType, len(matchEventTypeShortNames))
	for eventType, shortName := range matchEventTypeShortNames {
		types[shortName] = eventType
	}
	return types
}()

// viewShortNames maps generated view values to their fixture spelling.
var viewShortNames = map[matchesv1.View]string{
	matchesv1.View_VIEW_LINEUP_SETUP:       "LINEUP_SETUP",
	matchesv1.View_VIEW_PULL_WAIT:          "PULL_WAIT",
	matchesv1.View_VIEW_DEFENSE_ACTIVE:     "DEFENSE_ACTIVE",
	matchesv1.View_VIEW_OFFENSE_POSSESSION: "OFFENSE_POSSESSION",
	matchesv1.View_VIEW_SCORED:             "SCORED",
}

// startingRoleShortNames maps generated starting roles to their fixture spelling.
var startingRoleShortNames = map[matchesv1.StartingRole]string{
	matchesv1.StartingRole_STARTING_ROLE_OFFENSE: "OFFENSE",
	matchesv1.StartingRole_STARTING_ROLE_DEFENSE: "DEFENSE",
}

// MatchEventTypeShortName returns the fixture/storage short name for a type.
func MatchEventTypeShortName(eventType matchesv1.MatchEventType) (string, error) {
	shortName, ok := matchEventTypeShortNames[eventType]
	if !ok {
		return "", fmt.Errorf("unsupported match event type %s", eventType)
	}
	return shortName, nil
}

// MatchEventTypeFromShortName resolves a fixture/storage short name.
func MatchEventTypeFromShortName(shortName string) (matchesv1.MatchEventType, error) {
	eventType, ok := matchEventTypesByShortName[shortName]
	if !ok {
		return matchesv1.MatchEventType_MATCH_EVENT_TYPE_UNSPECIFIED, fmt.Errorf("unknown match event type %q", shortName)
	}
	return eventType, nil
}

// ViewShortName returns the fixture short name for a view.
func ViewShortName(view matchesv1.View) (string, error) {
	shortName, ok := viewShortNames[view]
	if !ok {
		return "", fmt.Errorf("unsupported view %s", view)
	}
	return shortName, nil
}

// StartingRoleShortName returns the fixture short name for a starting role.
func StartingRoleShortName(role matchesv1.StartingRole) (string, error) {
	shortName, ok := startingRoleShortNames[role]
	if !ok {
		return "", fmt.Errorf("unsupported starting role %s", role)
	}
	return shortName, nil
}

// StartingRoleFromShortName resolves a fixture spelling of a starting role.
func StartingRoleFromShortName(shortName string) (matchesv1.StartingRole, error) {
	for role, name := range startingRoleShortNames {
		if name == shortName {
			return role, nil
		}
	}
	return matchesv1.StartingRole_STARTING_ROLE_UNSPECIFIED, fmt.Errorf("unknown starting role %q", shortName)
}

// fixtureEvent is the fixture's envelope shape: a short type name and a bare
// payload object rather than the generated oneof.
type fixtureEvent struct {
	ID          string          `json:"id"`
	MatchID     string          `json:"matchId"`
	CreatedAt   string          `json:"createdAt"`
	Type        string          `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	PointNumber int64           `json:"pointNumber"`
	EffectiveTS int64           `json:"effectiveTs"`
	Deleted     bool            `json:"deleted"`
}

// MatchEventFromJSON decodes a fixture-shaped event. It is used by the parity
// harness; the fixture payload's camelCase keys match protojson.
func MatchEventFromJSON(raw []byte) (*matchesv1.MatchEvent, error) {
	var decoded fixtureEvent
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, fmt.Errorf("decode fixture event: %w", err)
	}

	eventType, err := MatchEventTypeFromShortName(decoded.Type)
	if err != nil {
		return nil, err
	}

	event := &matchesv1.MatchEvent{
		Id:          decoded.ID,
		MatchId:     decoded.MatchID,
		PointNumber: decoded.PointNumber,
		EffectiveTs: decoded.EffectiveTS,
		CreatedAt:   decoded.CreatedAt,
		Deleted:     decoded.Deleted,
		Type:        eventType,
	}

	payload, err := setPayloadForType(event, eventType)
	if err != nil {
		return nil, err
	}
	if len(decoded.Payload) > 0 {
		if err := protojson.Unmarshal(decoded.Payload, payload); err != nil {
			return nil, fmt.Errorf("decode %s payload: %w", decoded.Type, err)
		}
	}
	return event, nil
}

// SetPayloadFromStorage builds an event from a match_events row. The type
// column holds the short name and payload holds protojson-encoded payload.
func SetPayloadFromStorage(event *matchesv1.MatchEvent, eventType, payload string) error {
	parsedType, err := MatchEventTypeFromShortName(eventType)
	if err != nil {
		return err
	}
	event.Type = parsedType

	message, err := setPayloadForType(event, parsedType)
	if err != nil {
		return err
	}
	if payload != "" {
		if err := protojson.Unmarshal([]byte(payload), message); err != nil {
			return fmt.Errorf("decode stored %s payload: %w", eventType, err)
		}
	}
	return nil
}

// EventStorageColumns returns the match_events type/payload column values for
// an event, validating that the payload matches its declared type.
func EventStorageColumns(event *matchesv1.MatchEvent) (eventType, payload string, err error) {
	eventType, err = MatchEventTypeShortName(event.GetType())
	if err != nil {
		return "", "", err
	}

	message, err := eventPayload(event)
	if err != nil {
		return "", "", err
	}

	encoded, err := protojson.Marshal(message)
	if err != nil {
		return "", "", fmt.Errorf("encode %s payload: %w", eventType, err)
	}
	return eventType, string(encoded), nil
}

// eventPayload returns the oneof member when it matches the event's declared
// type. A mismatch is a malformed event.
func eventPayload(event *matchesv1.MatchEvent) (proto.Message, error) {
	// The getters return a typed nil when the oneof holds a different member,
	// which would satisfy a plain nil check on a proto.Message interface; the
	// wrappers are inspected directly instead.
	var payload proto.Message
	switch event.GetType() {
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_Touch); ok && wrapper.Touch != nil {
			payload = wrapper.Touch
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_DROP:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_Drop); ok && wrapper.Drop != nil {
			payload = wrapper.Drop
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_TURNOVER:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_Turnover); ok && wrapper.Turnover != nil {
			payload = wrapper.Turnover
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_DEFENSIVE_BLOCK:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_DefensiveBlock); ok && wrapper.DefensiveBlock != nil {
			payload = wrapper.DefensiveBlock
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_OPPONENT_TURNOVER:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_OpponentTurnover); ok && wrapper.OpponentTurnover != nil {
			payload = wrapper.OpponentTurnover
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_Score); ok && wrapper.Score != nil {
			payload = wrapper.Score
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_SetLineup); ok && wrapper.SetLineup != nil {
			payload = wrapper.SetLineup
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SUB_PLAYER:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_SubPlayer); ok && wrapper.SubPlayer != nil {
			payload = wrapper.SubPlayer
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_PULL:
		if wrapper, ok := event.GetPayload().(*matchesv1.MatchEvent_Pull); ok && wrapper.Pull != nil {
			payload = wrapper.Pull
		}
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_UNSPECIFIED:
		return nil, fmt.Errorf("match event type is unspecified")
	}
	if payload == nil {
		return nil, fmt.Errorf("payload missing for match event type %s", event.GetType())
	}
	return payload, nil
}

// setPayloadForType allocates the generated payload message, attaches its oneof
// wrapper to the event, and returns the message for decoding.
func setPayloadForType(event *matchesv1.MatchEvent, eventType matchesv1.MatchEventType) (proto.Message, error) {
	switch eventType {
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_TOUCH:
		payload := &matchesv1.TouchPayload{}
		event.Payload = &matchesv1.MatchEvent_Touch{Touch: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_DROP:
		payload := &matchesv1.DropPayload{}
		event.Payload = &matchesv1.MatchEvent_Drop{Drop: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_TURNOVER:
		payload := &matchesv1.TurnoverPayload{}
		event.Payload = &matchesv1.MatchEvent_Turnover{Turnover: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_DEFENSIVE_BLOCK:
		payload := &matchesv1.DefensiveBlockPayload{}
		event.Payload = &matchesv1.MatchEvent_DefensiveBlock{DefensiveBlock: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_OPPONENT_TURNOVER:
		payload := &matchesv1.OpponentTurnoverPayload{}
		event.Payload = &matchesv1.MatchEvent_OpponentTurnover{OpponentTurnover: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SCORE:
		payload := &matchesv1.ScorePayload{}
		event.Payload = &matchesv1.MatchEvent_Score{Score: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SET_LINEUP:
		payload := &matchesv1.SetLineupPayload{}
		event.Payload = &matchesv1.MatchEvent_SetLineup{SetLineup: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_SUB_PLAYER:
		payload := &matchesv1.SubPlayerPayload{}
		event.Payload = &matchesv1.MatchEvent_SubPlayer{SubPlayer: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_PULL:
		payload := &matchesv1.PullPayload{}
		event.Payload = &matchesv1.MatchEvent_Pull{Pull: payload}
		return payload, nil
	case matchesv1.MatchEventType_MATCH_EVENT_TYPE_UNSPECIFIED:
		return nil, fmt.Errorf("match event type is unspecified")
	}
	return nil, fmt.Errorf("unsupported match event type %s", eventType)
}

// FixturePlayerStats is the fixture's plain-number stat shape.
type FixturePlayerStats struct {
	Touches     int64 `json:"touches"`
	Completions int64 `json:"completions"`
	Drops       int64 `json:"drops"`
	Turnovers   int64 `json:"turnovers"`
	Blocks      int64 `json:"blocks"`
	Goals       int64 `json:"goals"`
	Assists     int64 `json:"assists"`
}

// FixturePointState is the fixture's derived-state shape: short enum names and
// plain numbers.
type FixturePointState struct {
	PlayerStats map[string]FixturePlayerStats `json:"playerStats"`
	View        string                        `json:"view"`
	InitialRole string                        `json:"initialRole"`
	Thrower     string                        `json:"thrower"`
	Lineup      []string                      `json:"lineup"`
}

// PointStateToJSON converts generated derived state to the fixture shape used
// by the parity contract and by the shared JSON byte-equality rule.
func PointStateToJSON(state *matchesv1.PointDerivedState) (FixturePointState, error) {
	view, err := ViewShortName(state.GetView())
	if err != nil {
		return FixturePointState{}, err
	}
	role, err := StartingRoleShortName(state.GetInitialRole())
	if err != nil {
		return FixturePointState{}, err
	}

	lineup := state.GetLineup()
	if lineup == nil {
		lineup = []string{}
	}
	playerStats := make(map[string]FixturePlayerStats, len(state.GetPlayerStats()))
	for playerID, stats := range state.GetPlayerStats() {
		playerStats[playerID] = FixturePlayerStats{
			Touches:     stats.GetTouches(),
			Completions: stats.GetCompletions(),
			Drops:       stats.GetDrops(),
			Turnovers:   stats.GetTurnovers(),
			Blocks:      stats.GetBlocks(),
			Goals:       stats.GetGoals(),
			Assists:     stats.GetAssists(),
		}
	}

	return FixturePointState{
		View:        view,
		InitialRole: role,
		Lineup:      lineup,
		Thrower:     state.GetThrower(),
		PlayerStats: playerStats,
	}, nil
}
