import type { MatchEvent } from './events';

/**
 * Prepare a raw MatchEvent log for reduction: filter -> sort -> dedupe.
 *
 * 1. Filter out deleted tombstones.
 * 2. Sort ascending by `effectiveTs`, breaking ties by `id`.
 * 3. Collapse duplicate ids, keeping the first event in that sorted order.
 *
 * The caller scopes the log to a point; this helper makes it deterministic and
 * replay-safe (ADR-0001).
 */
export function prepareEvents(events: readonly MatchEvent[]): MatchEvent[] {
  const sorted = events.filter((event) => !event.deleted).sort((a, b) => {
    if (a.effectiveTs !== b.effectiveTs) {
      return a.effectiveTs - b.effectiveTs;
    }
    // Deterministic tie-break so both language ports agree regardless of
    // input order.
    return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
  });

  const seen = new Set<string>();
  const prepared: MatchEvent[] = [];
  for (const event of sorted) {
    if (seen.has(event.id)) {
      continue;
    }
    seen.add(event.id);
    prepared.push(event);
  }

  return prepared;
}
