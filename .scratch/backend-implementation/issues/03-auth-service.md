# 03: auth service

Status: ready-for-agent

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
