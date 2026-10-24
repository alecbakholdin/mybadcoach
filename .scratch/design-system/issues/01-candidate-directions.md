# Candidate directions for the live scoring screen

Type: prototype
Status: resolved

## Question

Produce 2–4 distinct design directions for the live scoring screen as static HTML mockups,
and let Alec react to narrow them down.

- Neutral base color — no green or green-tinted background.
- Light and dark theme shown for every direction.
- Static HTML files in this worktree (Artifacts unavailable this session).
- Every direction renders the same real content so they compare cleanly: the live point
  (score, point number, phase, thrower, lineup) and the stat taps (Touch, Drop, Turn, Block,
  Score).
- Directions differ in type personality and shape language, not just accent-color swaps, and
  avoid the known AI-design clichés the `frontend-design` skill lists.

## Answer

Resolved 2026-10-23. Chosen direction: Night game's look (Sora, dark-first, cool blue-black
neutrals, warm amber #F5B74A / #B47A1F, flat, 10px radius) combined with Warm & round's rounded
score card, in Hybrid A's roster-first layout. Full player names; per-player Touch/Drop/Turn rows;
the thrower marked by an amber left edge only (no "has the disc" text); half/soft/hard cap as a
muted second strip inside the score card. Assets: `docs/directions/07-hybrid-a.html` and the token
reference `docs/design-system.html`.
