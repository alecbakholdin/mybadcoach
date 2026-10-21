# 06: Client skeleton (SvelteKit)

**What to build:** A minimal SvelteKit app that imports the shared package as TypeScript source and
renders the reducer's initial state, proving the alias wiring before any real UI is built.

**Blocked by:** 03 — Domain types + event schema + parity harness.

**Status:** ready-for-agent

- [ ] `apps/client` builds with SvelteKit 5 and `kit.alias` pointing at the shared source
- [ ] The home page calls `reducePoint([], 'OFFENSE')` and renders "Initial view: LINEUP_SETUP"
- [ ] `npm run dev:client` and `npm run build:client` both succeed
