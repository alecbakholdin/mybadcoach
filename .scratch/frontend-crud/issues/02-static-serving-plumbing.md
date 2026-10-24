# Wire adapter-static + go:embed serving + Vite dev proxy

Type: task
Status: resolved

## Question

Stand up the deployment shape settled in grilling:

- Switch `apps/client` to `@sveltejs/adapter-static` with `ssr=false` and SPA fallback
  (client-side routing; deep links resolve to `index.html`).
- Have the Go server `go:embed` the built client (`apps/client/build`) and serve it on one origin
  alongside Connect-RPC, with every non-RPC path falling back to `index.html` and the Connect mount
  paths still reaching the services.
- Add a Vite dev proxy so `npm run dev:client` forwards Connect-RPC calls to the Go server (no Go
  rebuild needed during development).
- Verify `npm run build:client && npm run build:server` yields a single Go binary that serves the SPA.

Note: the approved dependency `@sveltejs/adapter-static` is a build-time tool, not a runtime dep —
list it in `docs/dependencies.md` if it is not already recorded.

## Answer

Resolved 2026-10-24. Deployed shape wired and verified:

- **Client**: `@sveltejs/adapter-static@^4.0.0` replaces adapter-auto (recorded in
  `docs/dependencies.md`, approved); `vite.config.ts` uses `adapter({ fallback: 'index.html' })`; a
  new root `+layout.ts` sets `export const ssr = false`. Dev proxy: `/mybadcoach` → `http://localhost:8080`.
- **Go**: new `server/web` package embeds `//go:embed all:static`; `web.Handler()` serves real files
  (`_app` assets) as-is and falls back to `index.html` for `/`, `/index.html`, and any unmatched path.
  Mounted as `mux.Handle("/", web.Handler())` after the Connect services, so RPC routes still win.
- **Build**: `scripts/copy-client-build.sh` syncs `apps/client/build` → `server/web/static/`, preserving
  the tracked `.gitkeep`; `build:server` runs it before `go build`; `npm run build` (`build:client` +
  `build:server`) yields the single binary `dist/server`. A client-less checkout still compiles and
  serves 404 (via the placeholder) rather than failing the embed.
- **Verified**: smoke test — `/`, `/login`, `/teams/abc/deep` → 200 `text/html`; `/_app/*` → asset;
  Connect `Me` → 401 (route wins, not index.html). `server/web/embed_test.go` covers the handler and
  the no-shell 404 case. `go test ./...`, `go vet`, `gofmt` clean.

Follow-on (deferred): docker-compose still runs a separate `client` `vite preview` service; folding
the SPA into the Go server's single origin and pointing e2e at `server:8080` is left for the
testing-seam/e2e work.
