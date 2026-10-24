# Settle the client data layer and session/auth state

Type: grilling
Status: open

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
