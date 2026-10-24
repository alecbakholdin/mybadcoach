# Settle the client data layer and session/auth state

Type: grilling
Status: resolved

## Question

Settle the shape of the client's Connect-RPC data layer and its session/auth state, because every
CRUD screen hangs off it.

- How are the six service clients instantiated — one `createConnectTransport` with a same-origin
  `baseUrl`, one typed client per service (`auth`, `teams`, `seasons`, `events`, `matches`, `players`)?
- Where does the current-user session live (a Svelte 5 rune store?), how is `AuthService.me()`
  hydrated on app load, and how do protected routes redirect to `/login` on an absent session or 401?
- How does connect-web attach the CSRF custom header on mutations? Confirm the exact header name and
  mechanics against `server/internal/auth/middleware.go` (cookie session + CSRF) and
  `.scratch/backend-api/issues/04` so the client and server agree.

## Answer

Resolved 2026-10-24 (grilling, with Alec). Settled client shape:

- **Request layer** (`src/lib/api/`): one `createConnectTransport({ baseUrl: '/' })`, one generated
  client per service (`auth`, `teams`, `seasons`, `events`, `matches`, `players`), and a connect-web
  interceptor that adds `X-Requested-With: XMLHttpRequest` to every request (unconditional; harmless
  on non-mutating calls). Server facts confirmed: mutating RPCs require a non-empty `X-Requested-With`
  + same-origin `Origin`; every structure RPC requires a session; same-origin fetch sends `mbc_session`.
- **Session state**: Svelte 5 rune module `src/lib/session.svelte.ts` holding
  `currentUser: User | null | undefined` (undefined = loading, null = unauthenticated) + a `ready`
  flag, hydrated once on startup by `AuthService.me()`.
- **Routing**: root `+layout.ts` (`export const ssr = false`) awaits `Me()` before first render and
  redirects unauthenticated users. Public routes: `/` (marketing/hero home — login + register buttons
  now, demo content hydrates later), `/login`, `/register`; everything else protected. Post-login →
  team list.
- **Session expiry**: one shared error handler maps `unauthenticated` → clear session + redirect to
  `/login` with a "session expired" notice.

Standing preference recorded: prefer Svelte 5 runes over context/stores wherever possible (Alec).
