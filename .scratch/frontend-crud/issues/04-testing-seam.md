# Stand up the minimal frontend testing seam

Type: task
Status: resolved

## Question

Stand up the minimal frontend testing seam so screens are not built untested:

- The stub Connect server (`@connectrpc/connect-node` + an in-memory fake store, per
  `.scratch/backend-api/issues/03`) for frontend-only tests.
- The strict ESLint config from `docs/testing.md` for `apps/client`.
- One Playwright smoke e2e proving the client renders against the stub server.

Keep it minimal — deeper per-screen coverage lands with the screens themselves.

## Answer

Implemented.

**Strict ESLint gate**

- Root flat config `eslint.config.js`, shared by every workspace. `scripts/lint.sh` gained
  an `eslint` target (`eslint . --max-warnings 0`); `npm run lint` runs Go + proto +
  ESLint. New `lint-eslint` CI job (Node 22.22, `npm ci`, `bash scripts/lint.sh eslint`).
- Rules: typescript-eslint `strictTypeChecked` via the project service
  (`extraFileExtensions: ['.svelte']`), eslint-plugin-svelte `recommended` with Svelte 5
  runes mode and the TypeScript parser for `.svelte.ts`, eslint-plugin-playwright
  `recommended` for `e2e/**`, and `import-x` ordering (builtin → external → internal
  aliases → relative).
- Policy: every rule is error or off — the plugins' `warn` rules are promoted and
  `--print-config` shows zero warning-severity rules. `no-explicit-any`, unused
  vars/imports/params (`_` opt-out) and `no-console` are errors (console is allowed only
  in the stub-server startup process); escape hatches must name the rule and reason
  (`ban-ts-comment` `allow-with-description`, `reportUnusedDisableDirectives: 'error'`).
- Only genuine fixes in existing code: typed `Snippet` children in `+layout.svelte`,
  `toggle` as an arrow property in `theme.svelte.ts`, import ordering, and braces in
  `parity.test.ts`. No `eslint-disable` directives were added.

**Dependencies** — recorded in `docs/dependencies.md`, all **approved 2026-10-24**:
`eslint@10.13.0`, `@eslint/js@10.0.1`, `typescript-eslint@8.72.0`,
`eslint-plugin-svelte@3.23.1`, `svelte-eslint-parser@1.10.0`,
`eslint-plugin-playwright@2.12.1`, `eslint-plugin-import-x@4.17.1`, `globals@17.13.0`.
The root `typescript@6.0.3` pin is the already-approved TypeScript, pinned at the root
only to satisfy `typescript-eslint`'s `<6.1` peer (the shared package keeps its own
version). No dependency beyond the approved set was added; the stub uses the
already-approved `@connectrpc/connect-node`.

**Stub Connect server** — `apps/client/test-server/` (`main.ts`, `auth.ts`, `store.ts`),
started with `node --experimental-strip-types apps/client/test-server/main.ts` (env
`MBC_STUB_HOST`/`MBC_STUB_PORT`, defaults `127.0.0.1:4174`) — native type stripping,
no `tsx`/`ts-node`. `connectNodeAdapter` registers the generated `GenService`
descriptors over an in-memory `FakeStore`; it is a real HTTP/Connect boundary with real
codecs, cookies, and credentialed CORS. `GET /__test/health` and `POST /__test/reset`
serve readiness and per-test isolation. `AuthService` is implemented faithfully
(register/login/logout/me; `mbc_session` HttpOnly, SameSite=Lax, Path=/, 30-day;
`X-Requested-With` + same-origin Origin CSRF; `email_taken` → `already_exists`,
`invalid_credentials`/`session_expired` → `unauthenticated`, validation →
`invalid_argument` with field violations). Teams, seasons, events, matches, and players
are registered as `{}`, so the router answers `unimplemented`; `main.ts` documents the
extension point tickets 06–08 use.
- To make generated code run under Node's native strip-only mode, `proto/buf.gen.yaml`
  now emits `import_extension=.ts` (Node ESM needs explicit extensions) and
  `erasable_syntax=true` (no TS `enum`); generated `_pb.ts` files were regenerated and
  `tsconfig.base.json` gained `allowImportingTsExtensions`.

**Client → stub reachability (for ticket 05)** — same-origin, matching ticket 01's
`baseUrl: '/'`; the client is not changed. Vite's `server.proxy` maps `/mybadcoach` to
`http://localhost:8080` in dev (real Go server), and `preview.proxy` maps `/mybadcoach`
to `http://127.0.0.1:4174` (this stub) in the preview used by the seam. Ticket 05 keeps
`baseUrl: '/'`; in frontend integration tests the Vite proxy routes it to the stub. The
stub still implements credentialed CORS so the seam behaves like the real deployment if
the client is ever served from a different origin.

**Playwright frontend seam** — `e2e/playwright.frontend.config.ts`,
`e2e/tests/frontend/smoke.spec.ts`, and an `e2e/tsconfig.json`; root script
`npm run test:integration:frontend`. The existing `playwright.config.ts` and
`full-stack.spec.ts` are untouched. `webServer` starts the stub and serves the
production build with `vite preview`. The smoke proves the app shell renders, `Me`
without a session returns HTTP 401 `unauthenticated`, and Register mints a real
`mbc_session` cookie that `Me` honors.

**Verification (2026-10-24)** — `npm run lint` passes (golangci-lint 0 issues, ESLint
clean, buf lint/breaking); `npm run test` passes (49 shared tests + all Go packages);
`npm run test:integration:frontend` passes 3/3; `npm run build:client` succeeds.

Known follow-up, out of scope here: CI's e2e job still runs only the full-stack
Docker seam, so `test:integration:frontend` is not yet wired into CI. Its Playwright
browser revision must be installed locally (`npx playwright install
chromium-headless-shell`); the existing Playwright Docker image already provides it.

