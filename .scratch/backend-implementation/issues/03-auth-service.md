# 03: auth service

Status: resolved

Blocked by: 02

Implement `AuthService` per `proto/mybadcoach/auth/v1/auth.proto` (backend-api ticket 04).

- Register: bcrypt cost 12, password 8+ chars and ≤72 bytes, `email_taken` on duplicates.
- Login: identical `invalid_credentials` error for a bad email or bad password.
- Logout: delete the current session, clear the cookie. Me: `{ user }` only.
- Sessions: `crypto/rand` 256-bit tokens, SHA-256 hashed at rest, 30-day absolute expiry, stored
  in the `sessions` table. Cookie `mbc_session`: HttpOnly, SameSite=Lax, Path=/, Secure in prod.
- CSRF middleware: require `X-Requested-With` on mutating RPCs + validate `Origin`.
- Attach `ErrorDetail` (errors.proto) via `connect.WithErrorDetails` with the documented machine
  codes.
- Tests: backend integration — httptest + real temp-file SQLite; happy paths, bad-credential
  symmetry, expiry, cookie/CSRF posture.

## Comments

Resolved on branch `ticket/03-auth-service`. Package layout for tickets 04/05:

- `server/internal/auth` — `Service` (the `AuthServiceHandler`), session token/store
  (`session.go`), cookie construction (`cookie.go`), context helpers (`context.go`), Connect
  error helpers (`errors.go`), and reusable `Middleware` (`middleware.go`).
- `server/internal/api` — `NewHandler(api.Config)` is the "DB + config → http.Handler" assembly.
  Tickets 04/05 add their service to the mux in `handler.go` and extend the gates via
  `Config.AdditionalMutatingProcedures` / `Config.AdditionalAuthenticatedProcedures`.
- `server/cmd/server/main.go` — env: `MBC_DB_DSN` (default `mbc.db`), `MBC_ADDR` (default
  `:8080`), `MBC_ENV=production` sets Secure cookies, `MBC_DEV_SEED=1` runs the seed.

CSRF: mutating RPCs are an explicit procedure-name set (defaults: Register, Login, Logout) matched
against `r.URL.Path`; extend with `AdditionalMutatingProcedures`. Mutating requests need a
non-empty `X-Requested-With` and, when `Origin` is present, a same-origin `Origin`. Rejections are
`permission_denied` + `ErrorDetail{code: "csrf_rejected"}`.

Code table implemented: `email_taken` → already_exists, `invalid_credentials` → unauthenticated,
`session_expired` → unauthenticated, `invalid_argument` → invalid_argument (+ field violations),
`csrf_rejected` → permission_denied.
