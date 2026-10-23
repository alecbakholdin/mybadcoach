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

## Comments

Implemented and merged to integration as `60c8b06` (branch `ticket/01-toolchain-codegen-lint`,
commits `bf258c7` + `60c8b06`). All gates pass (buf lint/build/breaking, codegen idempotent,
`go build`/`go test`/`go vet`, TS typecheck, `npm test` incl. 24-fixture parity) **except**
`npm run lint`: the installed golangci-lint v1.64.8 is built with Go 1.25.3 and aborts on the
Go 1.27.3 module; the v2.x upgrade is pending Alec's dependency approval (see below).

Contract notes: `buf.gen.yaml` now pins remote plugin versions (latest `connectrpc/go` would
emit the unapproved connect-go v2); `players.proto` dropped the invalid `optional repeated` on
`names` (proto3 has no repeated presence) — empty/absent `names` means "leave names unchanged".
