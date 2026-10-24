# Build event + match CRUD screens

Type: task
Status: resolved
Blocked by: 06

## Question

Build the event and match CRUD screens:

- Events: create / list / edit / delete within a season via the `events` service.
- Matches: create / list / edit / delete within an event (`opponent_name`, `scheduled_at`,
  `point_cap`) via the `matches` service.

CRUD only — no live-scoring screen; gameplay is out of scope for this map.


## Answer

Resolved 2026-10-24.

**Routes (nested-slug addressing; ids never appear in requests):**

- `/teams/[team_slug]/seasons/[season_slug]` — season detail gained an Events section: `ListEvents`
  (offset/limit) with links to each event, plus a `CreateEvent` form (name, start/end date,
  location).
- `/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]` — event detail: `UpdateEvent`
  (name/dates/location), member-gated confirm-then-`DeleteEvent` (returns to the season),
  `ListMatches`, and `CreateMatch` (`opponent_name`, `scheduled_at`, `point_cap`).
- `/teams/[team_slug]/seasons/[season_slug]/events/[event_slug]/matches/[match_slug]` — match
  detail: `UpdateMatch` (`opponent_name`/`scheduled_at`/`point_cap`) and member-gated
  confirm-then-`DeleteMatch` (returns to the event).
- Event slugs are generated from the event name and unique within the season; match slugs from
  `opponent_name` and unique within the event. Both are stable across renames (ADR-0006).

**Stub server:** `apps/client/test-server/store.ts` gained in-memory events (scoped by
team+season) and matches (scoped by team+season+event) with parent-scoped `slugify` + numeric
suffix collisions, and hard deletes that cascade (season → events → matches, event → matches,
team → everything). `structure.ts` now also exports `createEventImplementation` and
`createMatchImplementation`; `main.ts` registers both services over the same cookie-session
identity and membership check as teams/seasons, and the six Event/Match CRUD mutations joined the
CSRF-header set. `/__test/reset` clears the new data. Roster RPCs on EventService and the
gameplay RPCs on MatchService are still omitted, so they answer `unimplemented` (ticket 08).

**Gating:** every Event/Match read and mutation requires team membership; `DeleteEvent` and
`DeleteMatch` are member-gated (any member), not owner-only, matching the structure contract. The
screens only render when the caller's `ListMyTeams` role is non-empty, so a non-member sees the
not-found state; the delete controls stay visible to every member, and the stub enforces the same
rule server-side.

**Tests:** new `e2e/tests/frontend/event-match.spec.ts` covers create season → create event →
appears in list → edit event → create match → edit match → delete match → delete event, plus a
seeded owner/member case proving a non-owner member can create, edit, and delete events and
matches. No Vitest unit tests were added: slugs stay server-side and the client gained no pure
logic.

**Verification:** `npm run lint` (Go, buf, ESLint), `npm run test` (49 shared + Go suites),
`npm run test:integration:frontend` (17 specs), and `npm run build:client` all pass.

**For ticket 08:** the event screen is a single `EventDetailData` load (`getEvent` +
`listMatches`, with `getSeason`/`ListMyTeams` for the breadcrumb and role). Add
`ListEventRoster` to that same `Promise.all` and render a roster panel alongside Matches, reusing
the season screen's empty-roster pattern; no other route restructuring is needed.
