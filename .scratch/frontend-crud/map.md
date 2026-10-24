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

## Decisions so far

<!-- one line per closed ticket -->

## Not yet specified

- Membership management (list / change-role / remove members) and member invites by email — not in
  the confirmed CRUD list; revisit once the core CRUD screens land.

## Out of scope

- Live scoring / gameplay screen (the nine event types, live point state) — the next frontend effort.
- Offline-first sync (local event log, write queue, sync) — deferred.
- Live sync (MatchActor bidirectional stream) — deferred.
- Stats review / team-record readout — deferred (secondary surface).
- Deeper design-token spec beyond applying the locked tokens.
