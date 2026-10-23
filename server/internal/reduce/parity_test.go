package reduce

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var requiredFixtures = []string{
	"01-empty-point.json",
	"02-out-of-order-sort.json",
	"03-duplicate-id-dedupe.json",
}

type fixtureFile struct {
	Name        string          `json:"name"`
	InitialRole StartingRole    `json:"initialRole"`
	Events      []MatchEvent    `json:"events"`
	Expected    json.RawMessage `json:"expected"`
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
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read fixtures directory: %v", err)
	}

	fixtures := make(map[string]fixtureFile)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
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
			actual := ReducePoint(fixture.Events, fixture.InitialRole)

			if actual.Lineup == nil {
				t.Fatal("lineup must never be nil")
			}
			if actual.PlayerStats == nil {
				t.Fatal("playerStats must never be nil")
			}

			actualJSON, err := json.Marshal(actual)
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

	decoder := json.NewDecoder(strings.NewReader(string(raw)))
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

		builder.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(strconv.Quote(key))
			builder.WriteByte(':')
			writeCanonical(t, builder, typed[key])
		}
		builder.WriteByte('}')
	case []any:
		builder.WriteByte('[')
		for index, entry := range typed {
			if index > 0 {
				builder.WriteByte(',')
			}
			writeCanonical(t, builder, entry)
		}
		builder.WriteByte(']')
	case json.Number:
		builder.WriteString(typed.String())
	case string:
		builder.WriteString(strconv.Quote(typed))
	case bool:
		builder.WriteString(strconv.FormatBool(typed))
	case nil:
		builder.WriteString("null")
	default:
		t.Fatalf("unsupported JSON value %T", value)
	}
}
