package reduce

import (
	"reflect"
	"testing"
)

func eventIDs(events []MatchEvent) []string {
	ids := make([]string, len(events))
	for index, event := range events {
		ids[index] = event.ID
	}
	return ids
}

func TestPrepareEventsSortsByEffectiveTsThenID(t *testing.T) {
	prepared := PrepareEvents([]MatchEvent{
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

	prepared := PrepareEvents([]MatchEvent{tombstone, touchEvent("live", 2000, "player-1")})

	if want := []string{"live"}; !reflect.DeepEqual(eventIDs(prepared), want) {
		t.Fatalf("prepared ids = %#v, want %#v", eventIDs(prepared), want)
	}
}

func TestPrepareEventsDeduplicatesByIDKeepingFirstAfterSort(t *testing.T) {
	prepared := PrepareEvents([]MatchEvent{
		touchEvent("same-id", 2000, "player-late"),
		touchEvent("same-id", 1000, "player-early"),
	})

	if len(prepared) != 1 {
		t.Fatalf("prepared %d events, want 1", len(prepared))
	}
	if prepared[0].EffectiveTs != 1000 {
		t.Fatalf("kept event effectiveTs = %d, want the first after sorting (1000)", prepared[0].EffectiveTs)
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
