package structure

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	"github.com/alecbakholdin/mybadcoach/server/internal/gameplay"
	authv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1"
	eventsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1"
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
	playersv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1"
	seasonsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1"
	teamsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1"
)

const defaultPageSize = 50

// timestamp returns the current time as a UTC RFC3339Nano string, the
// storage convention for created_at columns.
func (b *base) timestamp() string {
	return b.now().UTC().Format(time.RFC3339Nano)
}

// pageBounds normalizes offset/limit: a negative offset becomes zero, an
// unset limit (zero) becomes the default page size, and a negative limit
// clamps to one, so a page always requests at least one row.
func pageBounds(offset, limit int32) (normalizedOffset, normalizedLimit int64) {
	if offset < 0 {
		offset = 0
	}
	switch {
	case limit == 0:
		limit = defaultPageSize
	case limit < 0:
		limit = 1
	}
	return int64(offset), int64(limit)
}

// requiredText trims raw and rejects an empty value with an
// invalid_argument FieldViolation.
func requiredText(raw, field string) (string, *connect.Error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalidArgument(fieldViolation(field, "must not be empty"))
	}
	return trimmed, nil
}

// normalizeNames trims a player name list and rejects empty entries. It is
// used for creates and for non-empty wholesale updates.
func normalizeNames(names []string) ([]string, *connect.Error) {
	if len(names) == 0 {
		return nil, invalidArgument(fieldViolation("names", "must contain at least one name"))
	}
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return nil, invalidArgument(fieldViolation("names", "must not contain empty names"))
		}
		normalized = append(normalized, trimmed)
	}
	return normalized, nil
}

// stringPointer reports whether an optional proto string field is present.
func stringPointer(value *string) (string, bool) {
	if value == nil {
		return "", false
	}
	return *value, true
}

// int64Pointer reports whether an optional proto int64 field is present.
func int64Pointer(value *int64) (int64, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}

// nullStringForNullable stores a present optional string: an empty string
// clears the column to NULL.
func nullStringForNullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

// nullStringValue renders a nullable text column as a proto string.
func nullStringValue(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

// nullInt64For stores a present optional int64.
func nullInt64For(value int64) sql.NullInt64 {
	return sql.NullInt64{Int64: value, Valid: true}
}

// nullInt64Value renders a nullable integer column as a proto int64.
func nullInt64Value(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}

// marshalNames encodes a player name list for its JSON column.
func marshalNames(names []string) (string, error) {
	raw, err := json.Marshal(names)
	if err != nil {
		return "", fmt.Errorf("encode player names: %w", err)
	}
	return string(raw), nil
}

// teamMessage maps a stored team to its proto message.
func teamMessage(team db.Team) *teamsv1.Team {
	return &teamsv1.Team{Id: team.ID, Slug: team.Slug, Name: team.Name}
}

// membershipMessage maps a membership-with-user row to its proto message.
func membershipMessage(row *db.GetMembershipWithUserRow) *teamsv1.Membership {
	return &teamsv1.Membership{
		User: &authv1.User{Id: row.UserID, Email: row.Email, Name: row.Name},
		Role: row.Role,
	}
}

// membershipForUser maps a known user and role to a proto membership.
func membershipForUser(user db.User, role string) *teamsv1.Membership {
	return &teamsv1.Membership{
		User: &authv1.User{Id: user.ID, Email: user.Email, Name: user.Name},
		Role: role,
	}
}

// seasonMessage maps a stored season to its proto message.
func seasonMessage(season *db.Season) *seasonsv1.Season {
	return &seasonsv1.Season{
		Id:        season.ID,
		Slug:      season.Slug,
		Name:      season.Name,
		StartDate: nullStringValue(season.StartDate),
		EndDate:   nullStringValue(season.EndDate),
	}
}

// eventMessage maps a stored event to its proto message.
func eventMessage(event *db.Event) *eventsv1.Event {
	return &eventsv1.Event{
		Id:        event.ID,
		Slug:      event.Slug,
		Name:      event.Name,
		StartDate: nullStringValue(event.StartDate),
		EndDate:   nullStringValue(event.EndDate),
		Location:  nullStringValue(event.Location),
	}
}

// matchMessage maps a stored match to its proto message.
func matchMessage(match *db.Match) *matchesv1.Match {
	return &matchesv1.Match{
		Id:           match.ID,
		Slug:         match.Slug,
		OpponentName: match.OpponentName,
		ScheduledAt:  nullStringValue(match.ScheduledAt),
		PointCap:     nullInt64Value(match.PointCap),
	}
}

// matchEventMessage maps a stored gameplay event to its proto message,
// decoding the payload according to the row's short type name.
func matchEventMessage(row *db.MatchEvent) (*matchesv1.MatchEvent, error) {
	event := &matchesv1.MatchEvent{
		Id:          row.ID,
		MatchId:     row.MatchID,
		PointNumber: row.PointNumber,
		EffectiveTs: row.EffectiveTs,
		CreatedAt:   row.CreatedAt,
		Deleted:     row.Deleted != 0,
	}
	if err := gameplay.SetPayloadFromStorage(event, row.Type, row.Payload); err != nil {
		return nil, fmt.Errorf("decode stored match event %s: %w", row.ID, err)
	}
	return event, nil
}

// playerMessage maps a stored player to its proto message, decoding its JSON
// name list.
func playerMessage(player *db.Player) (*playersv1.Player, error) {
	var names []string
	if err := json.Unmarshal([]byte(player.Names), &names); err != nil {
		return nil, fmt.Errorf("decode player names: %w", err)
	}
	return &playersv1.Player{Id: player.ID, Slug: player.Slug, Names: names}, nil
}
