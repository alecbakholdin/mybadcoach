# mybadcoach
Ultimate Frisbee stat tracker that streams live results to anyone in the world

Gameplay is event-sourced: every stat tap is an atomic `MatchEvent`, and a pure reducer shared by the
TypeScript client and the Go server turns the log into live statistics.

## Prerequisites

- **Node.js 22.17 or newer** — required by the root `engines.node` field (`>=22.17`). It runs the
  shared TypeScript package, the SvelteKit client, and the workspace scripts.
- **Go 1.27 or newer** — required by `server/go.mod`. It runs the Go server and its half of the
  parity suite.

## Getting started

```sh
git clone <repository-url> mybadcoach
cd mybadcoach
npm install
npm test
```

`npm install` installs the npm workspaces (`packages/shared`, `apps/client`). The Go module in
`server/` has no third-party dependencies, so no separate fetch step is needed. `npm test` is the
gate: it runs the shared TypeScript suite and the Go suite, and both execute the same 24 parity
fixtures.

## Commands

| Command                 | Description                                                                                   |
| ----------------------- | --------------------------------------------------------------------------------------------- |
| `npm test`              | Run both test suites: `test:shared` then `test:server`.                                       |
| `npm run test:shared`   | Typecheck and run the shared TypeScript suite (Vitest), including all 24 parity fixtures.      |
| `npm run test:server`   | Run the Go server suite (`go test ./...`), including the same 24 parity fixtures.              |
| `npm run dev`           | Run the Go server and the SvelteKit client dev server together (server is a one-shot stub for now). |
| `npm run build:server`  | Build the Go server binary to `dist/server`.                                                   |
| `npm run dev:client`    | Start the SvelteKit client dev server.                                                         |
| `npm run build:client`  | Build the SvelteKit client for production.                                                     |
| `npm run verify:rules`  | Verify the checked-in USAU rules Markdown conversion (`scripts/verify-usau-rules-md.sh`).      |
| `npm run verify:client` | Verify the client skeleton: shared tests, production build, and dev-server SSR.               |

## Documentation

- `GLOSSARY.md` — domain vocabulary; read this first.
- `docs/adr/` — the settled architecture decisions (event sourcing, stats, rosters, slugs, seasons).
- `docs/architecture.md` — the architecture write-up: parity contract, canonical state, fixtures,
  planned SQLite schema, and spec-gap defaults.
- `docs/reference/usau-rules-2026-27.md` — grep-able 2026–27 USAU rules; the official PDF sits
  alongside it.
- `fixtures/README.md` — the parity fixture format, equality rules, and spec-gap defaults (normative).
