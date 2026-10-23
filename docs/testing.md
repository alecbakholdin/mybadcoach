# Testing and Linting Standards

This document is the repository's testing and linting philosophy. Config files hold rule
details; this document holds policy and is normative where they disagree. Read it alongside
`docs/architecture.md` (contracts) and `fixtures/README.md` (the parity suite).

## Principles

1. **Real code over mocks.** Test doubles appear only at external boundaries (the network, the
   clock, randomness), never between internal collaborators. A test that needs a mock of an
   internal type is a design smell, not a testing technique.
2. **Strict linting, zero warnings.** Lint errors block the build; warnings are treated as
   errors. Suppressions must carry a justification.
3. **Three integration seams.** The frontend-only, backend-only, and full-stack layers defined
   below are first-class; a feature is not done until its seam tests exist.
4. **UI drift is gated on human validation.** A visual snapshot change is never accepted
   automatically; the human reviews the diff and affirms it was intentional.

## Linting

### TypeScript and Svelte — ESLint (strict)

ESLint flat config (`eslint.config.js` at the repo root, shared by all workspaces), built on:

- `typescript-eslint` at `strict-type-checked` — the strictest maintained preset. Project
  references supply type information to every rule that needs it.
- `eslint-plugin-svelte` at its `recommended` preset, with Svelte 5's runes rules enabled.
- `eslint-plugin-playwright` for e2e specs, at `recommended`.

Policy on top of the presets:

- `no-explicit-any` is an error. Escape hatches (`@ts-expect-error`,
  `// eslint-disable-next-line`) must name the rule and state the reason inline.
- Unused code (variables, imports, parameters) is an error.
- `console` calls are an error outside server startup/CLI code, where they must be explicit.
- Imports are sorted and grouped (builtins, external, internal aliases, relative) — enforced by
  `eslint-plugin-import-x` or the equivalent maintained plugin at adoption time.
- All three workspaces lint under one command; `npm run lint` exits non-zero on any finding in
  any workspace.

The exact rule list lives in the config, not here. Changing the strictness ceiling is a
deliberate decision recorded in a PR, not a quiet config edit.

### Go — golangci-lint (strict)

`golint` itself is deprecated and archived, so the enforced Go linter is **golangci-lint**
running `revive` (golint's successor) under a strict configuration, plus the maintained
analyzers. Enabled linters, with `issues.exclude-use-default: false` and no
`exclude-rules` carve-outs except for `_test.go` files where justified:

- `errcheck` — unchecked errors are errors.
- `govet` with all analyzers — the standard correctness suite.
- `staticcheck` with all checks (`SA*`, `S*`, `ST*`, `QF*`) — the strongest static analysis.
- `ineffassign`, `unused` — dead assignments and dead code are errors.
- `revive` under a strict ruleset — idiomatic Go (the golint successor).
- `gocritic` at a high bar — style and performance critiques.
- `gosec` — security checks; findings are errors, not advisories.
- `exhaustive` — switch statements over enums must be exhaustive (guards the event-type and
  view-enum switch tables the reducer relies on).
- `nilerr`, `misspell`, `goimports`-equivalent formatting.

`gofmt`/`gofumpt` formatting is enforced in the same gate: unformatted code fails lint.

### Lint gates

- `npm run lint` (both languages) must pass before merge; CI runs it on every push.
- No warnings-as-noise: every configured rule is either enforced at error level or not enabled.

## Testing seams

### Unit — the base

- **Shared (TypeScript):** Vitest, colocated with the module (`packages/shared/src/*.test.ts`).
  Pure functions only: the reducer, aggregation, slug logic.
- **Server (Go):** stdlib `testing`, table-driven where behavior varies over inputs. The
  existing `internal/reduce` and `internal/slugify` suites are the pattern.
- **Client (Svelte):** Vitest for extracted logic (stores, derived state). Component rendering
  behavior is covered at the e2e seam; do not build a parallel component-testing harness unless
  a seam proves too slow for a case.

### Parity — the cross-language contract

The hand-authored fixture suite in `fixtures/` executed unchanged by Vitest and `go test`
(`docs/architecture.md`) is part of the testing standard: any behavior change to the reducer
lands with a fixture that pins it in both languages, and the fixed suite count is asserted by
both runners.

### Integration seams (the three layers)

Each layer is a named command and runs identically in a remote container and in CI.

1. **Frontend-only** (`test:integration:frontend`). Playwright drives the real SvelteKit app.
   The *only* test double is a fake API at the network boundary (Mock Service Worker or a stub
   HTTP server): the client's real fetching, stores, rendering, and navigation run against
   realistic responses. There are no module mocks and no mocked components.

2. **Backend-only** (`test:integration:backend`, i.e. `go test` with the integration tag or
   build tag split). Real HTTP handlers exercised through `net/http/httptest` against a real
   SQLite database (a per-test temp file, never an in-memory stand-in and never a mocked
   driver). No interfaces are mocked; handlers are wired exactly as in production.

3. **Full-stack** (`test:e2e`). Playwright drives the real client in a browser against the real
   Go server over real HTTP, with real SQLite persistence. The whole stack is orchestrated by
   `docker compose` so the exact same command runs locally, in a remote container
   (devcontainer/Codespaces), and in CI. This is the test that proves the two ports of the
   event-sourced contract interoperate end to end.

## Mock minimization policy

- **No mocking libraries by default.** No `gomock`, no testify mocks, no `vi.mock` of modules.
  Add one only if a concrete case cannot be covered by a boundary fake, and justify it in the
  PR.
- **Fakes over mocks.** When a double is unavoidable, write a small real implementation (a fake
  API with recorded state) rather than an expectation-recording mock.
- **Never add an interface to production code solely to make it mockable.** Interfaces exist
  for polymorphism, not for test seams; the seams are the boundaries above.
- **Boundary doubles only:** network (frontend-only layer), the clock (inject a clock value
  rather than freezing globals), and randomness (seeded generators). Time-dependent logic takes
  a `now` parameter or a clock.
- **Real dependencies by default.** Real SQLite, real handlers, real HTTP. A test that
  substitutes a dependency that is not at one of the three boundaries is asking to be rewritten
  at the integration seam.

## Playwright e2e policy

- **Every major action has an e2e test.** As features land, the major actions list is
  maintained in the spec and mirrored by tests — starting with team/season/event/match
  creation and each live-scoring tap (TOUCH, DROP, TURNOVER, DEFENSIVE_BLOCK, OPPONENT_TURNOVER,
  SCORE, SET_LINEUP, SUB_PLAYER, PULL). An action without an e2e test is not done.
- **Every major screen has a visual snapshot.** Once a screen exists (not yet — the UI is still
  a skeleton), it gets a `toHaveScreenshot` assertion at a fixed viewport and theme, run inside
  the deterministic container so renders are stable. Snapshots are committed to the repo.
- **UI drift is surfaced for validation.** Snapshot changes fail CI and attach the before/after
  diff as an artifact. CI never updates snapshots. The human reviews the diff and either
  accepts it as intentional (committing the updated snapshot — the explicit act of approval) or
  fixes the code. Locally, `--update-snapshots` is that same explicit act and must be reviewed
  in the PR that carries it. A PR that changes a snapshot without saying why in its description
  is rejected.
- **No flaky-snapshot tuning.** If a snapshot flakes, fix the determinism (fonts, viewport,
  animation, or data) rather than raising tolerance.

## Remote container

- A `.devcontainer/` (devcontainer.json + Dockerfile/compose) is part of this standard: every
  command in this document runs unchanged in a remote container (VS Code Remote, Codespaces)
  and in CI from the same compose definition.
- Bare-metal remains first-class for day-to-day iteration (`npm run test`, `go test ./...` are
  the fast paths); the container is the authority for e2e and snapshots.

## Dependency vetting

Every dependency is vetted by the human before it lands. This covers npm packages, Go
modules, Docker base images, and CI actions alike — including build-time tooling (linters,
test runners, Playwright browsers).

- A change that introduces a dependency names it explicitly — what it is, what it's for, and
  the alternatives considered — and is not merged until the human approves. Nothing is
  installed "as part of the work" without that approval.
- Transitives are covered by the same gate: lockfile changes (`package-lock.json`, `go.sum`)
  are part of the review, and a change that sneaks in an unapproved direct dependency is
  rejected.
- Approval is per direct dependency, once. Routine version bumps of an already-approved
  dependency do not re-trigger the gate; adding a new direct dependency always does.
- Implementation tickets that will need dependencies (the linting and e2e tickets, for
  example) list their proposed dependency set up front, so approval happens once before any
  code lands.

## Gates

Merge is blocked on:

1. Dependency vetting — no new dependency without explicit human approval (above).
2. `npm run lint` — ESLint and golangci-lint, both strict, zero warnings.
3. `npm run test` — unit suites in both languages plus the parity suite.
4. The three integration seams, including snapshots — with any snapshot change explicitly
   validated by a human.
