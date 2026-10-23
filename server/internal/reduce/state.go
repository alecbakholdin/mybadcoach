package reduce

// StartingRole is whether we start a point on offense or defense.
type StartingRole string

// Starting-role constants.
const (
	Offense StartingRole = "OFFENSE"
	Defense StartingRole = "DEFENSE"
)

// View is the current phase of a point.
type View string

// View constants, in point lifecycle order.
const (
	ViewLineupSetup       View = "LINEUP_SETUP"
	ViewPullWait          View = "PULL_WAIT"
	ViewDefenseActive     View = "DEFENSE_ACTIVE"
	ViewOffensePossession View = "OFFENSE_POSSESSION"
	ViewScored            View = "SCORED"
)

// PlayerStats holds the seven base stats credited by the reducer (ADR-0002).
// All fields are required integers and must be zero-filled, never null.
type PlayerStats struct {
	Touches     int `json:"touches"`
	Completions int `json:"completions"`
	Drops       int `json:"drops"`
	Turnovers   int `json:"turnovers"`
	Blocks      int `json:"blocks"`
	Goals       int `json:"goals"`
	Assists     int `json:"assists"`
}

// PointDerivedState is complete derived state for one point. It is zero-filled
// and never null: Lineup is always an array, Thrower is "" when nobody has the
// disc, and PlayerStats is always a map. This matches the TypeScript
// PointDerivedState under canonical JSON.
type PointDerivedState struct {
	// Lineup is last so the garbage collector scans the smallest possible
	// prefix (fieldalignment): its data pointer is the final live pointer and
	// its length/capacity are not scanned. Field order does not affect JSON
	// output — canonical comparison sorts keys and every field carries a tag.
	View        View                   `json:"view"`
	InitialRole StartingRole           `json:"initialRole"`
	Thrower     string                 `json:"thrower"`
	PlayerStats map[string]PlayerStats `json:"playerStats"`
	Lineup      []string               `json:"lineup"`
}

// InitialPointState is the state every point starts in.
func InitialPointState(initialRole StartingRole) PointDerivedState {
	return PointDerivedState{
		View:        ViewLineupSetup,
		InitialRole: initialRole,
		Lineup:      []string{},
		Thrower:     "",
		PlayerStats: map[string]PlayerStats{},
	}
}
