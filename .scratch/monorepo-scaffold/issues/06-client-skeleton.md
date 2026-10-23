# 06: Client skeleton (SvelteKit)

**What to build:** A minimal SvelteKit app that imports the shared package as TypeScript source and
renders the reducer's initial state, proving the alias wiring before any real UI is built.

**Blocked by:** 03 — Domain types + event schema + parity harness.

**Status:** resolved

- [x] `apps/client` builds with SvelteKit 5 and `kit.alias` pointing at the shared source
- [x] The home page calls `reducePoint([], 'OFFENSE')` and renders "Initial view: LINEUP_SETUP"
- [x] `npm run dev:client` and `npm run build:client` both succeed

## Comments

Resolved by branch `ticket/06-client-skeleton` (9f95175), merged as 942338e. Criterion restated: "SvelteKit 5" in the ticket means Svelte 5; shipped `@sveltejs/kit` 3.1.0 (current major). The shared-source alias migrated to Vite `resolve.alias` in the review pass because `kit.alias` is deprecated in Kit 3. Intent — client imports shared source and renders the reducer's initial state — met and verified via SSR + browser.
