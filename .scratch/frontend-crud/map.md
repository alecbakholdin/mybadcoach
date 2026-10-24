# Frontend CRUD wayfinding map

## Destination

A deployable CRUD frontend: a SvelteKit SPA (`adapter-static`, `ssr=false`) served by the Go server
on one origin via `go:embed`, implementing the structure-CRUD + auth screens — register / login /
logout, and create/read/update/delete for **team, season, event, match, player**, plus season &
event rosters — over the existing Connect-RPC services, styled with the locked Night-game tokens
(self-hosted Sora). A minimal testing seam (stub Connect server + strict ESLint + a couple of
Playwright CRUD e2e) so nothing ships untested.

## Notes

- **Domain**: Team → Season → Event → Match, two roster levels (season + event), players with
  names/aliases. Consult `GLOSSARY.md`, `docs/architecture.md`, and `docs/adr/`. Grilling tickets are
  HITL — worked live with Alec, never answered for him.
- **Design**: locked "Night game" direction in `docs/design-system.html` (Sora, dark-first, cool
  neutrals, one warm amber accent, flat, 10px radius). The rejected "Chalk on the Field" draft is a
  negative reference — do not revive. Self-host Sora (woff2), no runtime CDN; static assets get
  long-lived cache headers (hashed Vite filenames, one origin, one connection). **`View` is already a
  glossary term (point phase) — UI pages are *screens* or *routes*, never "views".**
- **Deployment**: `@sveltejs/adapter-static` + `ssr=false`; Go serves the embedded static build +
  Connect-RPC on one origin; Vite dev proxy for development. Later: Cloudflare in front — then the
  CSRF middleware must trust `X-Forwarded-Proto` / `X-Forwarded-Host` (see the backend handoff).
- **Transport**: Connect-RPC (`@connectrpc/connect` + `connect-web`, already approved). Cookie
  session; CSRF custom header on mutations.
- **Testing**: `docs/testing.md` (strict ESLint, mock-minimized). Frontend-only tests run against the
  stub Connect server seam (`@connectrpc/connect-node` + in-memory fake store, per
  `.scratch/backend-api/issues/03`). Minimal slice now.
- **Work style**: delegate small AFK subtasks to haiku-class subagents (Alec's cost preference).
- **Conventions**: prefer Svelte 5 runes over context/stores wherever possible (Alec). A public
  marketing/hero home page at `/` (login + register buttons now; demo content hydrates later).

## Decisions so far

- [Settle the client data layer and session/auth state](issues/01-client-data-layer.md): one same-origin
  Connect transport + per-service clients in `src/lib/api/`; CSRF via an unconditional `X-Requested-With`
  interceptor; session as a Svelte 5 rune module hydrated by `Me()`; root `+layout.ts` gate with public
  routes `/`, `/login`, `/register`; shared `unauthenticated` → clear + redirect handler.
- [Wire adapter-static + go:embed serving + Vite dev proxy](issues/02-static-serving-plumbing.md):
  `@sveltejs/adapter-static` (`fallback: 'index.html'`, `ssr=false`), `server/web` embeds and serves
  the SPA with client-side-route fallback, dev proxy forwards `/mybadcoach` → `:8080`, and
  `npm run build` produces the single binary.
- [Apply the Night-game design tokens](issues/03-design-tokens-applied.md): `app.css` ports the locked
  token system verbatim; Sora self-hosted as subset woff2 (`@font-face`, variable 600–800, no runtime
  CDN); `+layout.svelte` + `src/lib/theme.svelte.ts` (rune) wire `data-theme` light/dark switching with
  a no-flash pre-paint script; `server/web` adds immutable cache headers for `_app/immutable/` and
  `fonts/`.

## Not yet specified

- Membership management (list / change-role / remove members) and member invites by email — not in
  the confirmed CRUD list; revisit once the core CRUD screens land.
- Container-stack consolidation — docker-compose still runs a separate `client` `vite preview` service;
  fold the SPA into the Go server's single origin and point the e2e at `server:8080` (with the
  testing-seam work).

## Out of scope

- Live scoring / gameplay screen (the nine event types, live point state) — the next frontend effort.
- Offline-first sync (local event log, write queue, sync) — deferred.
- Live sync (MatchActor bidirectional stream) — deferred.
- Stats review / team-record readout — deferred (secondary surface).
- Deeper design-token spec beyond applying the locked tokens.
