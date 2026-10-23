package reduce

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/alecbakholdin/mybadcoach/server/internal/gameplay"
	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

var requiredFixtures = []string{
	"01-empty-point.json",
	"02-out-of-order-sort.json",
	"03-duplicate-id-dedupe.json",
}

// expectedFixtureCount is the full parity suite: every MatchEvent type plus
// the documented edge cases. Both ports run this exact count.
const expectedFixtureCount = 24

type fixtureFile struct {
	Name        string            `json:"name"`
	InitialRole string            `json:"initialRole"`
	Events      []json.RawMessage `json:"events"`
	Expected    json.RawMessage   `json:"expected"`
}

// fixturesDir resolves the repo-root fixtures directory. `go test` runs with
// the working directory set to this package, three levels below the repo root.
func fixturesDir(t *testing.T) string {
	t.Helper()

	dir := filepath.Join("..", "..", "..", "fixtures")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("shared fixtures directory %q not found: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("shared fixtures path %q is not a directory", dir)
	}
	return dir
}

func loadFixtures(t *testing.T) map[string]fixtureFile {
	t.Helper()

	dir := fixturesDir(t)
	fsys := os.DirFS(dir)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("read fixtures directory: %v", err)
	}

	fixtures := make(map[string]fixtureFile)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, entry.Name())
		if err != nil {
			t.Fatalf("read fixture %s: %v", entry.Name(), err)
		}
		var fixture fixtureFile
		if err := json.Unmarshal(raw, &fixture); err != nil {
			t.Fatalf("decode fixture %s: %v", entry.Name(), err)
		}
		fixtures[entry.Name()] = fixture
	}
	return fixtures
}

func TestParityFixtures(t *testing.T) {
	fixtures := loadFixtures(t)

	if len(fixtures) != expectedFixtureCount {
		t.Errorf("found %d fixtures, want %d", len(fixtures), expectedFixtureCount)
	}

	for _, name := range requiredFixtures {
		if _, ok := fixtures[name]; !ok {
			t.Errorf("required fixture %s is missing", name)
		}
	}

	names := make([]string, 0, len(fixtures))
	for name := range fixtures {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fixture := fixtures[name]
		t.Run(name, func(t *testing.T) {
			role, err := gameplay.StartingRoleFromShortName(fixture.InitialRole)
			if err != nil {
				t.Fatalf("fixture %s: %v", name, err)
			}

			events := make([]*matchesv1.MatchEvent, 0, len(fixture.Events))
			for index, raw := range fixture.Events {
				event, decodeErr := gameplay.MatchEventFromJSON(raw)
				if decodeErr != nil {
					t.Fatalf("fixture %s: decode event %d: %v", name, index, decodeErr)
				}
				events = append(events, event)
			}

			actual := ReducePoint(events, role)
			if actual.GetLineup() == nil {
				t.Fatal("lineup must never be nil")
			}
			if actual.GetPlayerStats() == nil {
				t.Fatal("playerStats must never be nil")
			}

			fixtureState, err := gameplay.PointStateToJSON(actual)
			if err != nil {
				t.Fatalf("convert actual state: %v", err)
			}
			actualJSON, err := json.Marshal(fixtureState)
			if err != nil {
				t.Fatalf("marshal actual state: %v", err)
			}
			actualCanonical := canonicalJSON(t, actualJSON)
			expectedCanonical := canonicalJSON(t, fixture.Expected)

			if actualCanonical != expectedCanonical {
				t.Fatalf("fixture %s (%s)\nactual:   %s\nexpected: %s",
					name, fixture.Name, actualCanonical, expectedCanonical)
			}
		})
	}
}

// canonicalJSON renders JSON with object keys sorted at every level and no
// whitespace. Numbers keep their source lexeme. See fixtures/README.md.
func canonicalJSON(t *testing.T, raw []byte) string {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode JSON %q: %v", string(raw), err)
	}

	var builder strings.Builder
	writeCanonical(t, &builder, value)
	return builder.String()
}

func writeCanonical(t *testing.T, builder *strings.Builder, value any) {
	t.Helper()

	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		_ = builder.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				_ = builder.WriteByte(',')
			}
			_, _ = builder.WriteString(strconv.Quote(key))
			_ = builder.WriteByte(':')
			writeCanonical(t, builder, typed[key])
		}
		_ = builder.WriteByte('}')
	case []any:
		_ = builder.WriteByte('[')
		for index, entry := range typed {
			if index > 0 {
				_ = builder.WriteByte(',')
			}
			writeCanonical(t, builder, entry)
		}
		_ = builder.WriteByte(']')
	case json.Number:
		_, _ = builder.WriteString(typed.String())
	case string:
		_, _ = builder.WriteString(strconv.Quote(typed))
	case bool:
		_, _ = builder.WriteString(strconv.FormatBool(typed))
	case nil:
		_, _ = builder.WriteString("null")
	default:
		t.Fatalf("unsupported JSON value %T", value)
	}
}
