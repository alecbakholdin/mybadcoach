# 01: toolchain, codegen, and Go lint gates

Status: ready-for-agent

Install the pre-approved dependency set and wire the codegen + lint toolchain.

- Install approved deps (backend-api ticket 01): Go — `connectrpc.com/connect`,
  `modernc.org/sqlite`, `golang.org/x/crypto`; npm — `@connectrpc/connect`,
  `@connectrpc/connect-web`, `@connectrpc/connect-node` (dev); tools — `buf`, `sqlc`. Pin
  versions; lockfile diffs are part of the review (vetting gate).
- Move the proto tree from `.scratch/backend-api/assets/proto/` to repo-root `proto/`; fix
  `buf.gen.yaml` out paths (`server/internal/gen`, `packages/shared/src/gen`).
- Wire `buf lint` (STANDARD) + `buf breaking` into the lint gate.
- Wire golangci-lint for `server/` with the strict linter set from `docs/testing.md`.
- Scripts: `lint` (golangci + buf) and `generate` (buf + sqlc) from the repo root.
- Acceptance: buf lint clean; generated Go + TS compile; golangci-lint clean on the existing
  server code (fix or justify any strict-linter findings).
