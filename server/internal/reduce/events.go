// Package reduce ports the shared point reducer to Go.
//
// It must stay behaviorally identical to the TypeScript implementation in
// packages/shared/src/: both are checked against the same language-neutral
// fixtures in the repo-root fixtures directory (ADR-0001).
package reduce

import "encoding/json"

// EventType is one of the nine atomic MatchEvent types. The constants are
// declared in the same order as MATCH_EVENT_TYPES in packages/shared/src/events.ts.
type EventType string

// Match event type constants.
const (
	EventTouch            EventType = "TOUCH"
	EventDrop             EventType = "DROP"
	EventTurnover         EventType = "TURNOVER"
	EventDefensiveBlock   EventType = "DEFENSIVE_BLOCK"
	EventOpponentTurnover EventType = "OPPONENT_TURNOVER"
	EventScore            EventType = "SCORE"
	EventSetLineup        EventType = "SET_LINEUP"
	EventSubPlayer        EventType = "SUB_PLAYER"
	EventPull             EventType = "PULL"
)

// TouchPayload credits a player catching or possessing the disc.
type TouchPayload struct {
	PlayerID string `json:"playerId"`
}

// DropPayload credits the receiver who dropped a catchable throw.
type DropPayload struct {
	PlayerID string `json:"playerId"`
}

// TurnoverPayload credits the thrower who committed a turnover.
type TurnoverPayload struct {
	PlayerID string `json:"playerId"`
}

// DefensiveBlockPayload credits our player who blocked a throw.
type DefensiveBlockPayload struct {
	PlayerID string `json:"playerId"`
}

// OpponentTurnoverPayload carries no fields: the opponent is faceless
// (ADR-0003), so there is no opponent player to attribute.
type OpponentTurnoverPayload struct{}

// ScorePayload credits a goal and, when non-empty, the assisting player.
type ScorePayload struct {
	PlayerID       string `json:"playerId"`
	AssistPlayerID string `json:"assistPlayerId"`
}

// SetLineupPayload lists the up-to-seven players on the field.
type SetLineupPayload struct {
	PlayerIDs []string `json:"playerIds"`
}

// SubPlayerPayload swaps one player out for another.
type SubPlayerPayload struct {
	PlayerOutID string `json:"playerOutId"`
	PlayerInID  string `json:"playerInId"`
}

// PullPayload records the player who pulled.
type PullPayload struct {
	PlayerID string `json:"playerId"`
}

// MatchEvent is an atomic, time-ordered, idempotent-by-id gameplay event.
// Payload is decoded according to Type.
type MatchEvent struct {
	// Pointer-bearing fields come first so the garbage collector scans the
	// smallest possible prefix (fieldalignment); JSON order is irrelevant
	// because canonical comparison sorts object keys.
	ID          string          `json:"id"`
	MatchID     string          `json:"matchId"`
	CreatedAt   string          `json:"createdAt"`
	Type        EventType       `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	PointNumber int             `json:"pointNumber"`
	EffectiveTS int64           `json:"effectiveTs"`
	Deleted     bool            `json:"deleted"`
}
