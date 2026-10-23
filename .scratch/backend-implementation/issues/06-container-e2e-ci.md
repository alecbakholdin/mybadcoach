# 06: container, full-stack e2e, and CI gates

Status: ready-for-agent

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
