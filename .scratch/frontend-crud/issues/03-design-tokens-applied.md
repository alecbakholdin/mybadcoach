# Apply the Night-game design tokens to the client

Type: task
Status: open

## Question

Apply the locked Night-game token system from `docs/design-system.html` to the actual client:

- Replace the placeholder `apps/client/src/app.css` with the tokens — Sora, dark-first cool neutrals,
  one warm amber accent (`#F5B74A` / `#B47A1F`), flat surfaces, `10px` radius, semantic hues.
- Self-host Sora's woff2 locally (no runtime CDN fetch — offline-first), wired through `+layout.svelte`.
- Provide light/dark switching via `data-theme`, and give static assets long-lived cache headers.

The rejected "Chalk on the Field" `app.css` is a negative reference — do not revive its green palette.
