package reduce

import (
	"reflect"
	"testing"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

func eventIDs(events []*matchesv1.MatchEvent) []string {
	ids := make([]string, len(events))
	for index, event := range events {
		ids[index] = event.GetId()
	}
	return ids
}

func TestPrepareEventsSortsByEffectiveTsThenID(t *testing.T) {
	prepared := PrepareEvents([]*matchesv1.MatchEvent{
		touchEvent("b", 1000, "player-1"),
		touchEvent("early", 500, "player-1"),
		touchEvent("a", 1000, "player-1"),
	})

	if want := []string{"early", "a", "b"}; !reflect.DeepEqual(eventIDs(prepared), want) {
		t.Fatalf("prepared ids = %#v, want %#v", eventIDs(prepared), want)
	}
}

func TestPrepareEventsFiltersDeletedEvents(t *testing.T) {
	tombstone := touchEvent("tombstone", 1000, "player-1")
	tombstone.Deleted = true

	prepared := PrepareEvents([]*matchesv1.MatchEvent{tombstone, touchEvent("live", 2000, "player-1")})

	if want := []string{"live"}; !reflect.DeepEqual(eventIDs(prepared), want) {
		t.Fatalf("prepared ids = %#v, want %#v", eventIDs(prepared), want)
	}
}

func TestPrepareEventsDeduplicatesByIDKeepingFirstAfterSort(t *testing.T) {
	prepared := PrepareEvents([]*matchesv1.MatchEvent{
		touchEvent("same-id", 2000, "player-late"),
		touchEvent("same-id", 1000, "player-early"),
	})

	if len(prepared) != 1 {
		t.Fatalf("prepared %d events, want 1", len(prepared))
	}
	if got := prepared[0].GetEffectiveTs(); got != 1000 {
		t.Fatalf("kept event effectiveTs = %d, want the first after sorting (1000)", got)
	}
}

func TestPrepareEventsReturnsNonNilEmptySlice(t *testing.T) {
	prepared := PrepareEvents(nil)

	if prepared == nil {
		t.Fatal("prepared events must be an empty slice, never nil")
	}
	if len(prepared) != 0 {
		t.Fatalf("prepared %d events, want 0", len(prepared))
	}
}
