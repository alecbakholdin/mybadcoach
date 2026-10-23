# Auth service RPC design

Type: grilling
Status: resolved
Blocked by: 02

## Question

Walk the AuthService RPCs with Alec: register, login, logout, me. Settle per-RPC request/response
messages, the session cookie flow over Connect from the SvelteKit client (httpOnly, SameSite=Lax,
custom-header CSRF posture), Connect error mapping with stable machine-code details, bcrypt cost,
30-day absolute expiry, and how the dev seed admin fits. Output: settled auth messages + flows.

## Answer

Resolved 2026-10-22. All eight round-1 recommendations accepted by Alec:

- `AuthService` (`mybadcoach.auth.v1`): `Register`, `Login`, `Logout`, `Me`, all unary.
- Session cookie: `mbc_session`, HttpOnly, SameSite=Lax, Path=/, Secure in prod, Max-Age 30d.
  CSRF = `X-Requested-With` header on mutating RPCs + Origin check; no CSRF tokens.
- `RegisterRequest { email, password, name }` → `RegisterResponse { user }` + cookie; password
  8+ chars, max 72 bytes (bcrypt truncation).
- `LoginRequest { email, password }` → `LoginResponse { user }` + cookie; identical
  `unauthenticated` / `invalid_credentials` error for a bad email or a bad password; per-IP rate
  limiting deferred to backlog.
- `Logout`: deletes the current session, clears the cookie; revoke-all-sessions out of scope.
- `Me` → `{ user }` only. Roles live on `team_memberships` (Alec's confirmation), so membership
  and role data come from the structure endpoints.
- Errors: Connect codes + machine-code details (`email_taken` → already_exists,
  `invalid_credentials` → unauthenticated, `session_expired` → unauthenticated, field validation
  → invalid_argument with field paths). Code table documented in ticket 07's proto contract.
- bcrypt cost 12; 30-day absolute expiry, no sliding renewal, no remember-me.
- `User` message: `{ id, email, name }`.
- Dev-seed admin flows through the same RPCs — no special path.
