# Redesign the team + season screens

Status: ready-for-agent

## Problem Statement

The team and season screens (built in the frontend CRUD effort) are functional but visually
generic: a single narrow column of stacked cards, with every action — rename, delete, create
season, create event — always on screen. On mobile this wastes scarce vertical space and buries
the frequent actions among the rare ones. The roster is a plain name list with no per-player
season statistics, so a coach cannot read who is contributing at a glance.

## Solution

Rebuild the team + season screens into a master–detail shell: a persistent left rail listing the
teams with their seasons nested underneath, and a main pane showing the selected team or season.
Move uncommon and destructive actions (rename team, delete team, delete season, create season,
create event) into a Settings tab — with team settings grouped alongside season settings — and
turn the season roster into a sortable per-player statistics table. Add tasteful iconography for
visual separation. On mobile the rail collapses behind a small icon-only toggle and the statistics
table lets its checkbox column scroll away so more stats stay visible.

The design system is unchanged (Night-game tokens). This spec changes rendering and interaction
only: the routes, loaders, RPC calls, owner/member gating, and slug addressing all stay as they
are. The statistics columns render placeholder values until the gameplay effort provides real
match events to aggregate.

## User Stories

1. As a team member, I want a persistent sidebar listing every team I belong to, so that I can
   switch teams without returning to a list page.
2. As a team member, I want each team row to show its name and an Owner/Member role badge, so
   that I know my role on that team.
3. As a team member, I want the active team's seasons nested under it in the sidebar, so that I
   can jump directly between seasons.
4. As a team member, I want the active team and season marked with an amber edge, so that I
   always know where I am.
5. As a team member, I want a compact "+ New team" affordance at the bottom of the sidebar, so
   that I can create a team without leaving my current context.
6. As a team member, I want a compact "+ New season" affordance on the active team, so that I can
   add a season quickly without a large inline form.
7. As a mobile user, I want the sidebar to collapse behind a small icon-only toggle (not a large
   menu button or banner), so that it does not consume my limited screen space.
8. As a team member, I want the team overview to show the team name, my role, and a brief
   summary (season count and member count), so that I can confirm where I am at a glance.
9. As a team member, I want a "Players" action on the team overview, so that I can reach the
   player screen.
10. As a team owner, I want rename-team and delete-team to live in a Settings area rather than
    always on screen, so that the main view stays uncluttered.
11. As a team owner, I want team settings grouped together with season settings, so that I can
    manage the team and its season in one place.
12. As a team member, I want to edit a season's name and dates from the Settings tab, so that I
    can correct a season's details.
13. As a team owner, I want delete-season in the Settings tab, separated into its own danger
    zone, so that destructive actions are clearly set apart from everyday edits.
14. As a team owner, I want a confirmation step before any delete, so that I cannot accidentally
    remove a team or season.
15. As a team member, I want create-season and create-event to be compact "+" affordances rather
    than always-visible forms, so that the screen stays clean.
16. As a team member, I want the season detail organized into tabs (Events, Roster, Settings), so
    that I see only the section I need.
17. As a team member, I want the Roster tab to show per-player season statistics, so that I can
    compare players at a glance.
18. As a team member, I want to sort the roster by any statistics column, so that I can find the
    leaders in goals, assists, or plus-minus.
19. As a team member, I want the player-name column to stay pinned while I scroll through the
    statistics columns, so that I always know whose row I am reading.
20. As a team member, I want to select multiple roster players and remove them in one action, so
    that I can manage the roster quickly.
21. As a team member, I want an autocomplete when adding players to the roster, so that I can
    search and add several existing players at once.
22. As a mobile user, I want the checkbox column to scroll away when I scroll right, so that more
    statistics stay visible.
23. As a team member, I want plus-minus color-coded (positive versus negative), so that I can read
    performance at a glance.
24. As a team member, I want the statistics table to show placeholder values (em-dashes) until
    real gameplay data exists, so that the interface is real but honest about missing data.
25. As a team member who is not an owner, I want destructive controls hidden, so that I do not see
    actions I cannot take.
26. As a team member, I want member-gated edits (season name/dates, create season/event, roster
    changes) still available to me.
27. As a user, I want dark and light themes to work across every redesigned screen.
28. As a user, I want the screens to work at desktop, tablet, and mobile widths without
    horizontal page scrolling.

## Implementation Decisions

- **Master–detail shell.** A new layout component renders a ~264px left rail (teams list + nested
  seasons, compact create-team at the bottom) beside a main content pane. Lists use ledger-style
  hairline rules rather than heavy cards, matching the locked design.
- **Mobile rail.** Below the tablet breakpoint the rail becomes an off-canvas drawer, opened by a
  44px icon-only toggle in the header, with a scrim and Escape-to-close. No large menu button or
  banner.
- **Settings tab.** Uncommon and destructive actions move off the main surface. The season detail
  has tabs `Events | Roster | Settings`; the team overview has `Overview | Settings`. The Settings
  tab holds two grouped subsections: **Team settings** (rename team, delete team) and **Season
  settings** (edit name/dates, delete season). Rename-team and both deletes are owner-gated;
  season name/date edits are member-gated; create-season/create-event stay as compact "+"
  affordances.
- **Roster becomes a statistics table.** The Roster tab renders a sortable table with columns:
  Player (frozen left), Goals, Assists, Blocks, Turnovers, Drops, Touches, Completions, and
  Plus-minus, where `Plus-minus = Goals + Assists + Blocks − Turnovers − Drops`. Column headers
  sort ascending/descending with an arrow indicator; numeric columns are right-aligned with
  tabular figures; Plus-minus is color-coded (success tint positive, error tint negative).
- **Selection + bulk remove.** A checkbox column (with select-all) enables bulk "Remove selected";
  a removal bar appears when rows are selected. On mobile the checkbox column is not pinned, so it
  scrolls away and the player column becomes the single pinned column.
- **Add players.** An autocomplete searches the team's players and supports selecting several at
  once before a single "Add N players" action, replacing the single-select dropdown.
- **Placeholder stats.** Statistics cells render an em-dash until the gameplay effort supplies
  real data. No backend or protobuf change is required; the table reads only the existing roster
  players.
- **Table pattern.** Port the `DataGrid.svelte` pattern from the sibling `main-live-frisbee`
  codebase (sortable columns, frozen first column, selectable rows, sticky header) into this
  repo's Night-game token system. No new dependency is introduced.
- **Iconography.** Small inline stroke SVG icons (`currentColor`) separate list rows, section
  headers, and actions; tasteful, not decorative-everywhere, and never emoji.
- **Unchanged plumbing.** Routes, loaders, `src/lib/api` clients, `session` rune, owner/member
  gating, and slug addressing are unchanged. This is a rendering-and-interaction change only.

## Testing Decisions

- **What makes a good test.** Assert external behavior — what the user sees and does — through
  stable roles and accessible labels, not internal DOM structure beyond that.
- **Seam.** Frontend-only Playwright (`test:integration:frontend`): the real SvelteKit app driven
  against the stub Connect server (`apps/client/test-server/`). No new backend seam.
- **What is tested.** Sidebar navigation and nested season switching; settings-tab rename/delete
  with owner gating (controls hidden for members, destructive RPCs rejected server-side); roster
  tab sorting, selection, and bulk remove; autocomplete multi-add; placeholder stats rendering;
  mobile drawer and the un-pinned checkbox column.
- **Existing specs to update.** `structure.spec.ts` and `player-roster.spec.ts` assert the old
  single-column markup; they must be updated to the new shell and tab structure while preserving
  their behavioral coverage (create → rename → delete, gating, roster add/remove).
- **Visual snapshots.** Add `toHaveScreenshot` for the teams list, team detail, and season detail
  (Roster tab) at a fixed viewport and theme, per the snapshot policy — human-validated, never
  auto-updated.
- **Prior art.** The existing create → rename → delete and gating flows in
  `e2e/tests/frontend/structure.spec.ts` and `e2e/tests/frontend/player-roster.spec.ts` are the
  pattern to follow.

## Out of Scope

- Live scoring / gameplay screen and its nine event types (the next frontend effort).
- Real per-player statistics aggregation — deferred until match events exist; this spec ships
  placeholder values.
- Membership management and member invites.
- Redesign of the events, matches, players, and auth screens (separate follow-on specs).
- Any backend or protobuf change.
- Offline-first sync.

## Further Notes

- The winning visual source is the throwaway mockup at
  `.scratch/frontend-crud/prototype-teams-seasons/round-3/design-a.html` (with the mobile
  checkbox-scroll fix). Fold it into real components and keep the prototype directory out of
  `main` (commit to a throwaway branch or delete once folded in).
- Design tokens are unchanged and come from `docs/design-system.html` / `apps/client/src/app.css`.
- `DataGrid.svelte` is not in this repository; port its pattern, do not import it (different
  project and styling stack).
- Standing prototyping preferences are recorded in `docs/agents/prototyping.md`.
