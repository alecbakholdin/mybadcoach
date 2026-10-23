package reduce

import (
	"sort"

	matchesv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1"
)

// PrepareEvents makes a raw MatchEvent log deterministic and replay-safe:
// filter out deleted tombstones, sort by effectiveTs then id, and collapse
// duplicate ids keeping the first event in sorted order (ADR-0001).
//
// The caller scopes the log to a point.
func PrepareEvents(events []*matchesv1.MatchEvent) []*matchesv1.MatchEvent {
	live := make([]*matchesv1.MatchEvent, 0, len(events))
	for _, event := range events {
		if !event.GetDeleted() {
			live = append(live, event)
		}
	}

	sort.SliceStable(live, func(i, j int) bool {
		if live[i].GetEffectiveTs() != live[j].GetEffectiveTs() {
			return live[i].GetEffectiveTs() < live[j].GetEffectiveTs()
		}
		return live[i].GetId() < live[j].GetId()
	})

	seen := make(map[string]struct{}, len(live))
	prepared := make([]*matchesv1.MatchEvent, 0, len(live))
	for _, event := range live {
		if _, duplicate := seen[event.GetId()]; duplicate {
			continue
		}
		seen[event.GetId()] = struct{}{}
		prepared = append(prepared, event)
	}

	return prepared
}
