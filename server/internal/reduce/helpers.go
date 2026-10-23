package reduce

import "sort"

// PrepareEvents makes a raw MatchEvent log deterministic and replay-safe:
// filter out deleted tombstones, sort by effectiveTs then id, and collapse
// duplicate ids keeping the first event in sorted order (ADR-0001).
//
// The caller scopes the log to a point.
func PrepareEvents(events []MatchEvent) []MatchEvent {
	live := make([]MatchEvent, 0, len(events))
	for _, event := range events {
		if !event.Deleted {
			live = append(live, event)
		}
	}

	sort.SliceStable(live, func(i, j int) bool {
		if live[i].EffectiveTs != live[j].EffectiveTs {
			return live[i].EffectiveTs < live[j].EffectiveTs
		}
		return live[i].ID < live[j].ID
	})

	seen := make(map[string]struct{}, len(live))
	prepared := make([]MatchEvent, 0, len(live))
	for _, event := range live {
		if _, duplicate := seen[event.ID]; duplicate {
			continue
		}
		seen[event.ID] = struct{}{}
		prepared = append(prepared, event)
	}

	return prepared
}
