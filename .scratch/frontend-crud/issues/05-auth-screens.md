# Build the auth screens and protected routing

Type: task
Status: resolved
Blocked by: 01, 02, 03, 04

## Question

Build the auth screens on top of the settled data layer (ticket 01):

- A public marketing/hero home page at `/` with login + register buttons (demo content hydrates later).
- Register, login, and logout screens using the `auth` service.
- Me-gated protected routing: hydrate `AuthService.me()` on load, redirect to `/login` when
  unauthenticated, attach the CSRF header on mutations, and redirect to the team list after login.

Follow the auth contract in `.scratch/backend-api/issues/04` (unary Register/Login/Logout/Me, cookie
session, CSRF custom header).

## Answer

Implemented the auth screens, session state, protected routing, and the settled client data layer.

**Data layer — `apps/client/src/lib/api/`**

- `transport.ts` — one `createConnectTransport({ baseUrl: '/' })`; the first interceptor adds
  `X-Requested-With: XMLHttpRequest` to every request (unconditional); a custom fetch sets
  `credentials: 'include'`; a second interceptor routes non-form-outcome `unauthenticated` errors
  to one registered shared handler. Login/register failures (`invalid_credentials`) and the
  startup `Me` hydration are excluded so they stay form/route outcomes.
- `clients.ts` — `auth`, `teams`, `seasons`, `events`, `matches`, `players`, each via
  `createClient` over the generated `GenService` descriptors. Only `auth` is exercised here.
- `errors.ts` — `errorDetailOf` / `machineCodeOf` / `fieldErrorsOf` / `isInvalidCredentials` over
  the generated `ErrorDetail` detail.
- `index.ts` — the public `#lib/api` surface.

**Session — `apps/client/src/lib/session.svelte.ts`**

Svelte 5 rune module (no store/context): `currentUser: User | null | undefined`, `ready`,
memoized `loadSession()` (one `Me` on startup), `setUser`, `clear`, and best-effort `logout`.
It registers the shared unauthenticated handler: clear the session and `goto('/login?notice=session-expired')`
unless already on `/login`.

**Routing and screens**

- `src/routes/+layout.ts` keeps `export const ssr = false`; its `load` awaits the session once and
  redirects unauthenticated users to `/login` on every route except the public set
  (`/`, `/login`, `/register`).
- `src/routes/+layout.svelte` gains a minimal header: brand, current-user name + Log out when
  authenticated, and the existing theme toggle (unchanged).
- `src/routes/+page.svelte` — public hero with Log in / Register.
- `src/routes/login/+page.svelte` — email + password, `invalid_credentials` surfaced inline,
  redirects to `/teams`.
- `src/routes/register/+page.svelte` — name + email + password, `email_taken` and
  `invalid_argument` field violations surfaced, redirects to `/teams`.
- `src/routes/teams/+page.svelte` — minimal protected placeholder showing the signed-in user;
  logout lives in the header. Real CRUD is tickets 06-08.

**Vite proxy fix (important for later tickets)** — the dev and preview proxies were written as
string shorthands, which Vite expands with `changeOrigin: true`; that rewrote `Host` to the
upstream target while the browser kept its own `Origin`, so the server's same-origin CSRF check
rejected every mutating RPC. Both proxies are now objects with `changeOrigin: false`, preserving
the browser's Host so Origin and Host agree.

**Import strategy** — extensionless `$shared/gen/mybadcoach/<service>/v1/<service>_pb` imports,
resolved by the existing `$shared` Vite alias and tsconfig path. No `.ts` suffix was needed in app
code (generated files keep their own `.ts` specifiers).

**Tests** — `e2e/tests/frontend/auth.spec.ts` (6 specs: protected-route redirect, register →
`/teams`, logout → `/`, login, invalid credentials, duplicate email), all against the stub with
`/__test/reset` in `beforeEach`; `smoke.spec.ts` updated for the new hero. No Vitest unit tests
were added: the remaining logic is thin Connect error mapping covered at the e2e seam.

**Verification (2026-10-24)** — `npm run lint` passes (golangci-lint 0 issues, ESLint clean,
buf lint/breaking clean); `npm run test` passes (shared + all Go packages); 
`npm run test:integration:frontend` passes 9/9; `npm run build:client` succeeds.
