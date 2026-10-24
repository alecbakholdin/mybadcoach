# Wire adapter-static + go:embed serving + Vite dev proxy

Type: task
Status: open

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
