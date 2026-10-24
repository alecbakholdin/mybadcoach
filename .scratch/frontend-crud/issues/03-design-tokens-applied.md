# Apply the Night-game design tokens to the client

Type: task
Status: resolved

## Question

Apply the locked Night-game token system from `docs/design-system.html` to the actual client:

- Replace the placeholder `apps/client/src/app.css` with the tokens — Sora, dark-first cool neutrals,
  one warm amber accent (`#F5B74A` / `#B47A1F`), flat surfaces, `10px` radius, semantic hues.
- Self-host Sora's woff2 locally (no runtime CDN fetch — offline-first), wired through `+layout.svelte`.
- Provide light/dark switching via `data-theme`, and give static assets long-lived cache headers.

The rejected "Chalk on the Field" `app.css` is a negative reference — do not revive its green palette.

## Answer

Implemented the Night-game token system on the client and long-lived cache headers in the Go SPA
handler.

- **Tokens** — `apps/client/src/app.css` ports the locked system verbatim from
  `docs/design-system.html`: the dark `:root` and `:root[data-theme='light']` blocks, `--radius:
  10px`, body typography (Sora 600, tabular-nums, antialiasing, `--backdrop`/`--ink`), the
  `* { box-sizing }` reset, `.theme-toggle`, and `.btn` / `.btn-primary` / `.btn-secondary` /
  `.btn-tertiary` with their focus-visible outlines. No green/Chalk palette.
- **Self-hosted Sora** — `apps/client/static/fonts/sora-latin.woff2` and
  `sora-latin-ext.woff2`, declared with `@font-face` in `app.css`. Google serves one variable
  woff2 per unicode subset for both requested weights (600 and 800), so each file is declared
  `font-weight: 600 800` rather than duplicating identical blobs. No Google Fonts link or
  `fonts.googleapis.com` / `fonts.gstatic.com` reference remains in client source or the build.
- **Theme + layout** — `apps/client/src/routes/+layout.svelte` imports `app.css` and renders a
  working toggle backed by a Svelte 5 rune module (`apps/client/src/lib/theme.svelte.ts`, no store
  or context). `app.html` defaults `<html data-theme="dark">` and runs a tiny pre-paint script
  that restores a persisted choice from `localStorage`, so a light-theme user sees no dark flash.
  SvelteKit 3 removed `$lib`, so the `#lib` / `#lib/*` import map was added to
  `apps/client/package.json`.
- **Cache headers** — `server/web/embed.go` sets
  `Cache-Control: public, max-age=31536000, immutable` for `_app/immutable/` and `fonts/`; the
  SPA shell/fallback stays `text/html` with no cache header. Font names are not hashed, so the
  code comment records the contract: a font change must ship under a new filename.

### Verification

- `npm run build:client` — passed; build output contains `index.html`, `_app/immutable/...`, and
  both font files, with the tokens and `@font-face` in the emitted CSS.
- `cd server && go test ./web/... && go vet ./web/...` — passed; `gofmt -l` clean on
  `server/web/`.
- `npm run lint` — passed (golangci-lint v2.15.0 installed into the gitignored `.tools/bin`,
  matching `scripts/install-tools.sh`; the preinstalled v1.64.8 was built with Go 1.25 and will
  not load the repo's `go 1.27` target). buf lint and buf breaking clean.
- `npm run verify:client` is stale after the `ssr=false` switch (it greps SSR HTML for
  `Initial view: LINEUP_SETUP`), so it was intentionally not used as an acceptance gate.
