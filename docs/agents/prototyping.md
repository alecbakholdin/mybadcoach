# Prototyping

Standing preferences and locked decisions for prototype / design-exploration sessions. The parent
agent reads this before every round; update it whenever Alec locks a new pattern or rejects a
direction.

## Workflow (how we prototype)

- **Fan out in parallel** to several low-powered (haiku-class) subagents — one design per agent —
  then assemble a switcher for Alec to flip through. Do not draft pages sequentially in one agent.
- **Speed + token-minimization are the top priority.** Prototypes are vibe-based: each agent writes a
  decent static mockup and stops. No browser/Playwright verification, no focus-trap/inert/Escape
  rigor, no exhaustive copy cross-checks — judge by eye and iterate.
- **3 structurally distinct layouts per round** unless a layout is locked (below). "Distinct" means
  different layout, information hierarchy, and primary affordance — not colour/copy tweaks. More
  than ~4 is noise.
- **Locking:** when Alec says "I like the layout of design A," every subsequent round reuses design
  A's layout and varies only what he asked to vary. Record the lock under "Locked decisions."
- **Design system is constant.** Agents all use the same tokens (below); only layout/pattern varies.
- **Deliverable:** each agent writes ONE standalone HTML page (`<!doctype html>` … `</html>`, inline
  CSS, system/self-hosted fonts, no build step). The parent writes an `index.html` switcher that
  loads each page in an `<iframe>`.
- **Switcher must always expose:** dark/light theme toggle, desktop / tablet / mobile viewport
  toggle, and prev/next (or tabs) to flip between designs. Keyboard arrow keys to flip are nice to
  have.
- Prototypes are throwaway — keep them out of `main`. Commit to a throwaway branch and leave a
  pointer on the issue (see `mattpocock-skills:prototype`).

## Design system (constant — do not restyle)

- **"Night game" direction**: Sora (600 + 800), dark-first, cool blue-black neutrals, one warm amber
  accent, flat surfaces, 10px radius. Full tokens: `docs/design-system.html` (reference) and
  `apps/client/src/app.css` (ported, self-hosted woff2).
- Copy the `:root` / `:root[data-theme='light']` tokens and the `data-theme` attribute convention so
  the switcher's dark/light toggle actually works on every page.
- Full player names, never initials. Per-player stat rows, not a few global buttons.
- Neutral base colour — no green or green-tinted background.
- `View` is a glossary term (point phase); call UI surfaces "screens" or "routes", never "views".

## Locked decisions

- **Live scoring screen** (resolved 2026-10-23): Night game + Warm & round's rounded score card, in
  Hybrid A's roster-first layout. See `.scratch/design-system/issues/01-candidate-directions.md`.
- **Teams + seasons — sidebar layout locked** (2026-10-24): design A's left rail (teams + nested
  seasons, inline create-team, ledger-style rules) is the direction. Mobile rail collapses to a
  drawer, but the toggle must be a small icon button — not a large/invasive menu. Iterate within this
  shell; don't re-propose a non-sidebar layout for this surface.
- **Teams + seasons — settings-tab mechanism locked** (2026-10-24): uncommon actions live in a
  Settings tab (Overview | Settings; Events | Roster | Settings). Team settings (rename/delete team)
  are grouped WITH season settings in that tab.
- **Roster = per-player season stats** (2026-10-24): the Roster tab becomes a sortable stats table.
  No new table dependency — port `main-live-frisbee`'s `DataGrid.svelte` pattern (sortable columns,
  frozen first column, selectable rows) into the Night-game token system.

## Rejected (negative references — do not revive)

- **"Chalk on the Field"** — green base, literal "frisbee → disc buttons" metaphor.

## Consistent preferences (persist across sessions)

1. **Tasteful iconography** — enough icons to visually separate elements (list rows, section headers,
   actions); small stroke icons in `currentColor`, never emoji, never decorative-everywhere.
2. **Uncommon + destructive actions stay tucked away.** Rename team/season, delete team/season, create
   season, create event live behind a settings page / collapsible section / settings tab / modal — not
   always-visible inline, especially on mobile.
3. **Roster add is multi-select + autocomplete** — search existing players and add several at once.
4. **Prefer left-sidebar (master–detail) layouts** for teams + seasons.

_Append more as sessions proceed._
