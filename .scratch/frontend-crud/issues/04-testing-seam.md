# Stand up the minimal frontend testing seam

Type: task
Status: open

## Question

Stand up the minimal frontend testing seam so screens are not built untested:

- The stub Connect server (`@connectrpc/connect-node` + an in-memory fake store, per
  `.scratch/backend-api/issues/03`) for frontend-only tests.
- The strict ESLint config from `docs/testing.md` for `apps/client`.
- One Playwright smoke e2e proving the client renders against the stub server.

Keep it minimal — deeper per-screen coverage lands with the screens themselves.
