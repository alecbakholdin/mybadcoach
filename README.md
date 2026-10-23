# mybadcoach
Ultimate Frisbee stat tracker that streams live results to anyone in the world

Gameplay is event-sourced: every stat tap is an atomic `MatchEvent`, and a pure reducer shared by the
TypeScript client and the Go server turns the log into live statistics.

## Prerequisites

- **Node.js 22.17 or newer** — required by the root `engines.node` field (`>=22.17`). It runs the
  shared TypeScript package, the SvelteKit client, and the workspace scripts.
- **Go 1.27 or newer** — required by `server/go.mod`. It runs the Go server and its half of the
  parity suite.
- **buf 1.75.0** — proto lint, breaking-change detection, and codegen. `scripts/install-tools.sh`
  installs this pinned version into `.tools/bin/` (gitignored), which the lint and generate
  scripts put first on `PATH`.
- **sqlc 1.31.1** — SQL codegen (`npm run generate`); supplied by the dev image.
- **golangci-lint 2.15.0** — strict Go lint (`npm run lint`); `scripts/install-tools.sh` installs
  this pinned version into `.tools/bin/` alongside buf.

## Getting started

```sh
git clone <repository-url> mybadcoach
cd mybadcoach
npm install
bash scripts/install-tools.sh
npm test
```

`npm install` installs the npm workspaces (`packages/shared`, `apps/client`). The Go module in
`server/` pins its dependencies in `go.mod`/`go.sum`, fetched on first build. `npm test` is the
gate: it runs the shared TypeScript suite and the Go suite, and both execute the same 24 parity
fixtures.

## Protos and generated code

`proto/` is the API schema of record (Connect RPC, ADR-0008). `npm run generate` runs
`buf generate` over it, writing Go to `server/internal/gen/` and TypeScript to
`packages/shared/src/gen/`. Both trees are committed and must not be hand-edited; edit the
`.proto` and regenerate.

## Commands

| Command                 | Description                                                                                   |
| ----------------------- | --------------------------------------------------------------------------------------------- |
| `npm test`              | Run both test suites: `test:shared` then `test:server`.                                       |
| `npm run test:shared`   | Typecheck and run the shared TypeScript suite (Vitest), including all 24 parity fixtures.      |
| `npm run test:server`   | Run the Go server suite (`go test ./...`), including the same 24 parity fixtures.              |
| `npm run lint`          | Strict gate: `golangci-lint` on `server/`, `buf lint` on `proto/`, and `buf breaking`.        |
| `npm run generate`      | Regenerate Go/TypeScript from `proto/`, then run sqlc when migrations/queries exist.          |
| `npm run dev`           | Run the Go server and the SvelteKit client dev server together (server is a one-shot stub for now). |
| `npm run build:server`  | Build the Go server binary to `dist/server`.                                                   |
| `npm run dev:client`    | Start the SvelteKit client dev server.                                                         |
| `npm run build:client`  | Build the SvelteKit client for production.                                                     |
| `npm run verify:rules`  | Verify the checked-in USAU rules Markdown conversion (`scripts/verify-usau-rules-md.sh`).      |
| `npm run verify:client` | Verify the client skeleton: shared tests, production build, and dev-server SSR.               |
| `docker compose run --rm e2e` | Full-stack Playwright smoke against the real server, client, and SQLite in the compose network. |

## Containers and the full-stack seam

`docker-compose.yml` orchestrates the Go server, the production SvelteKit client, and the
Playwright runner on one network. The e2e smoke drives the Connect API directly (register →
login → team/season/event/match → player/roster → live events → folded state/stats) and loads
the client to prove it serves. The same command runs locally, in the devcontainer, and in CI:

```sh
docker compose build
docker compose run --rm e2e
docker compose down --volumes   # fresh SQLite for the next run
```

Every base image, npm package, and CI action introduced by this stack is listed for review in
`docs/dependencies.md`.

## Documentation

- `GLOSSARY.md` — domain vocabulary; read this first.
- `docs/adr/` — the settled architecture decisions (event sourcing, stats, rosters, slugs, seasons).
- `docs/architecture.md` — the architecture write-up: parity contract, canonical state, fixtures,
  planned SQLite schema, and spec-gap defaults.
- `docs/reference/usau-rules-2026-27.md` — grep-able 2026–27 USAU rules; the official PDF sits
  alongside it.
- `fixtures/README.md` — the parity fixture format, equality rules, and spec-gap defaults (normative).
