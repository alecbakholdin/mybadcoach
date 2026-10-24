# Dependencies

Every direct dependency is listed here for approval review. This file is the
single place to audit the set: npm packages, Go modules, Docker base images,
and CI actions. See the dependency policy in `docs/testing.md` and `CLAUDE.md`.

Status: approved. The Go/npm backend set below predates ticket 06 and was
approved with the backend-api map. The container, e2e, and CI categories added
by ticket 06 (Playwright, Docker base images, and CI actions) were approved by
Alec on 2026-10-23.

## npm packages

| Package | Version | Scope | Purpose | New |
| --- | --- | --- | --- | --- |
| `@playwright/test` | `1.64.1` | `e2e` (independent package) | Browser + HTTP driver for the full-stack smoke. | **approved 2026-10-23** |
| `eslint` | `10.13.0` | root (shared by all workspaces) | Linter engine for the strict TypeScript/Svelte gate. | **approved 2026-10-24** |
| `@eslint/js` | `10.0.1` | root | ESLint's core recommended rule set. | **approved 2026-10-24** |
| `typescript-eslint` | `8.72.0` | root | Type-aware parser/plugin; `strict-type-checked` preset via project service. | **approved 2026-10-24** |
| `eslint-plugin-svelte` | `3.23.1` | root | Svelte parser integration + `recommended` rules (Svelte 5 runes mode). | **approved 2026-10-24** |
| `svelte-eslint-parser` | `1.10.0` | root | Direct Svelte parser dependency required by the typed config. | **approved 2026-10-24** |
| `eslint-plugin-playwright` | `2.12.1` | root | Playwright `recommended` rules for `e2e/**` specs. | **approved 2026-10-24** |
| `eslint-plugin-import-x` | `4.17.1` | root | Import sorting/grouping (builtins, external, internal aliases, relative). | **approved 2026-10-24** |
| `globals` | `17.13.0` | root | Browser/Node global sets for the flat config. | **approved 2026-10-24** |
| `@sveltejs/adapter-static` | `^4.0.0` | `apps/client` (devDependency) | SvelteKit build adapter: emits the client as a static SPA the Go server `go:embed`s and serves. | **approved 2026-10-24** |
| `oxfmt` | `^0.74.0` | root (devDependency) | Prettier-compatible formatter (`.oxfmtrc.json`: tabs, single quotes) for TS/Svelte/CSS; replaces Prettier. | **approved 2026-10-24** |

The ESLint set is the "standard set" Alec approved on 2026-10-24. `import-x` was chosen
over the unmaintained `eslint-plugin-import`; module resolution stays with TypeScript/Vite
(`import-x/no-unresolved` is off), so no extra resolver package is needed. The root
`typescript` pin (`6.0.3`) is not a new dependency: it is the already-approved TypeScript,
pinned at the root because `typescript-eslint@8.72.0` supports `<6.1`, while
`packages/shared` keeps its own TypeScript 7 for `tsc --noEmit`. `@connectrpc/connect-node`
for the stub server was already approved with the backend set.

Alternatives considered for the e2e driver: Puppeteer (smaller but no test
runner or API-request fixtures), Cypress (heavier, no first-class API-only
requests), or a plain Go/shell script against the API (loses the real-browser
assertion that the SvelteKit client serves). Playwright is the standard the
testing policy already names for the three integration seams.

The e2e package is deliberately *not* part of the root npm workspaces: it has
its own `e2e/package-lock.json`, so the root lockfile and the client build are
untouched by Playwright and its transitives.

## Docker base images

All images are pinned by tag; none use `latest`.

| Image | Version tag | Used by | Purpose | New |
| --- | --- | --- | --- | --- |
| `golang` | `1.27.3-alpine` | server build stage, devcontainer | Compiles `./cmd/server` (also matches `server/go.mod`'s `toolchain go1.27.3`). | **approved 2026-10-23** |
| `alpine` | `3.21` | server runtime stage | Minimal runtime for the static Go binary; supplies `nc` for the healthcheck. | **approved 2026-10-23** |
| `node` | `22.22-alpine` | client build/runtime stage, devcontainer | Builds and previews the SvelteKit client; matches the root `engines.node` floor. | **approved 2026-10-23** |
| `mcr.microsoft.com/playwright` | `v1.64.1-noble` | e2e runner | Official Playwright image; bundles the Chromium build matching `@playwright/test@1.64.1` so CI installs no browsers. | **approved 2026-10-23** |

Alternatives considered: `gcr.io/distroless/static` for the server runtime
(smaller, but no shell/`nc`, so the compose healthcheck and the local
`npm`/`go` workflow would need another mechanism); a Debian-based Node image
(larger, no benefit); `npx playwright install --with-deps chromium` in a plain
Node image (adds a network-dependent browser install and an unpinned browser
revision at build time).

The devcontainer reuses `golang:1.27.3-alpine` and `node:22.22-alpine` by
copying the Go toolchain from the former into the latter, so it introduces no
third base image. It adds only Alpine OS packages: `bash`, `git`, `docker-cli`,
and `docker-cli-compose`.

## CI actions

| Action | Version | Purpose | New |
| --- | --- | --- | --- |
| `actions/checkout` | `v4` | Clone the repo. | **approved 2026-10-23** |
| `actions/setup-go` | `v5` | Install Go 1.27.3 with module cache. | **approved 2026-10-23** |
| `actions/setup-node` | `v4` | Install Node 22.22 with npm cache. | **approved 2026-10-23** |

These are the standard first-party GitHub actions; no third-party action is
used.

## Backend set (approved with the backend-api map)

Go modules (`server/go.mod`): `connectrpc.com/connect`, `golang.org/x/crypto`,
`google.golang.org/protobuf`, `modernc.org/sqlite` plus their transitive
dependencies.

npm packages (`package-lock.json`): SvelteKit/Svelte/Vite/TypeScript/Vitest for
the client and shared package. The generated-code runtime packages land in this
branch, in ticket 01, rather than pre-existing: `@bufbuild/protobuf` 2.16.0 in
`packages/shared`, and `@connectrpc/connect` 2.2.0, `@connectrpc/connect-web`
2.2.0, and `@connectrpc/connect-node` 2.2.0 (a client devDependency for the
stub-server seam) in `apps/client`. They were approved with the backend-api map
before implementation began.

Build-time tools (from `scripts/install-tools.sh` and the dev image): `buf`
1.75.0, `golangci-lint` 2.15.0, `sqlc` 1.31.1.
