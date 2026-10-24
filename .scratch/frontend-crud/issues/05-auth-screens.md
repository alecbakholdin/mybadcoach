# Build the auth screens and protected routing

Type: task
Status: open
Blocked by: 01, 02, 03, 04

## Question

Build the auth screens on top of the settled data layer (ticket 01):

- A public marketing/hero home page at `/` with login + register buttons (demo content hydrates later).
- Register, login, and logout screens using the `auth` service.
- Me-gated protected routing: hydrate `AuthService.me()` on load, redirect to `/login` when
  unauthenticated, attach the CSRF header on mutations, and redirect to the team list after login.

Follow the auth contract in `.scratch/backend-api/issues/04` (unary Register/Login/Logout/Me, cookie
session, CSRF custom header).
