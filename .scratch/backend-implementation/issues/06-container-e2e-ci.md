# 06: container, full-stack e2e, and CI gates

Status: resolved

Blocked by: 05

Make the stack run identically in a remote container and CI, with a full-stack proof.

- Dockerfile for the Go server + docker compose orchestration (client + server + e2e runner) and
  the `.devcontainer/` setup, per the remote-container section of `docs/testing.md`.
- Full-stack Playwright smoke: register → login → create team/season/event/match → create player
  + roster → append MatchEvents (live taps) → read folded state and stats — against the real
  stack (real browser, real client, real server, real SQLite) in the container.
- CI workflow (GitHub Actions): lint (golangci + buf) → unit + parity → backend integration →
  full-stack e2e, run from the same compose definition. Snapshot policy comes later, when screens
  exist (client effort).
- The frontend-only stub Connect server seam is deferred to the client effort.

## Comments

Resolved on branch `ticket/06-container-e2e-ci` (a496a5a, b52952b), merged to integration.

Delivered: `server/Dockerfile` + `apps/client/Dockerfile` + `docker-compose.yml` (server/client/e2e
with healthchecks), `.devcontainer/`, `.github/workflows/ci.yml` (lint → test → integration →
e2e), and a full-stack Playwright smoke (`e2e/`) that ran green (`docker compose run --rm e2e` →
1 passed). Because the full client UI is deferred, the smoke drives the Connect API directly and
only loads the client to prove it serves.

New dependencies (NOT in the pre-approved backend set — need Alec's approval, all recorded in
`docs/dependencies.md`): base images `golang:1.27.3-alpine`, `alpine:3.21`, `node:22.22-alpine`,
`mcr.microsoft.com/playwright:v1.64.1-noble`; npm `@playwright/test@1.64.1`; CI actions
`actions/checkout@v4`, `actions/setup-go@v5`, `actions/setup-node@v4`.

Wire note: the Connect JSON codec requires **full** enum names on the wire
(`MATCH_EVENT_TYPE_TOUCH`, `VIEW_SCORED`), unlike the parity fixtures' short names; the e2e sends
full names and int64 fields as JSON strings.
