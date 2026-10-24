# Design system wayfinding map

## Destination

A design direction for the live scoring screen, chosen by Alec from 2–4 candidate
directions (static HTML mockups, neutral base, light + dark each) and narrowed down until
he likes it. The endpoint is the approved mockup; the token system is extracted from it
afterward, not invented first.

## Notes

- Product: mobile-web-first, offline-first Ultimate live stat tracker. The core surface is
  the live scoring screen — the live point (score, point number, phase, thrower, lineup)
  and the rapid stat taps (Touch, Drop, Turn, Block, Score). Consult `GLOSSARY.md` for
  vocabulary; the screen's phases come from `reducePoint`'s `view` (`LINEUP_SETUP`,
  `PULL_WAIT`, `DEFENSE_ACTIVE`, `OFFENSE_POSSESSION`, `SCORED`).
- Constraints from Alec (grilling round 1): neutral base color — no green or green-tinted
  background; light and dark theme both required; 2–4 distinct directions to pick from;
  delivered as static HTML files (Artifacts unavailable this session — local files only).
- Rejected draft #1 ("Chalk on the Field", `docs/design-system.html` + `apps/client/src/app.css`):
  green base, literal "frisbee → disc buttons" metaphor. Treat as a negative reference; do
  not revive. The `app.css` it left behind is superseded by whatever direction wins.
- Prototype tickets are HITL — worked live with Alec, never answered for him. Consult the
  `frontend-design` skill for the anti-cliché guidance and `prototype` for the artifact style.

- Reference — Alec's past implementation (4 screenshots, OCR'd 2026-10-23): score header with
  Breaks / Break-opps / Turns beside the raw score; a lineup screen with O/D line groups and
  Rested/Played columns; per-player stat entry (each player row = full name + action buttons,
  offense actions split from defense actions). Explicit preference: show the player's FULL name,
  never initials. The live-scoring mockup must use the per-player model, not five global buttons.

## Decisions so far

- [Candidate directions for the live scoring screen](issues/01-candidate-directions.md): Night
  game look (Sora, dark-first, amber, flat) + Warm & round's score card + Hybrid A roster layout;
  full names, per-player Touch/Drop/Turn, caps as a muted score-card strip. Token ref:
  `docs/design-system.html`.

## Not yet specified

- The scoreboard reads dense (score + stats + caps in one card) — Alec flagged "too much text in
  one spot" as a followup; likely move the caps to the top bar or tighten the hierarchy.

## Out of scope

- The token system spec and `app.css` rewrite — the next effort, after the direction is locked.
- Secondary surfaces (stats review, roster/lineup management, match/season/event setup).
- Working Svelte components for the live scoring screen — this effort stops at static mockups.
