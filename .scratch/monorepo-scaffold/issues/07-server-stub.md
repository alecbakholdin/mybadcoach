# 07: Go server stub + build

**What to build:** A buildable Go server entry point that imports the reducer and proves it runs, before
any HTTP or DB wiring.

**Blocked by:** 03 — Domain types + event schema + parity harness.

**Status:** resolved

- [x] `cmd/server/main.go` imports the reducer and prints the initial view
- [x] `npm run build:server` (`go build ./cmd/server`) succeeds
- [x] Running the binary prints the initial-view line

## Comments

Resolved by branch `ticket/07-server-stub` (c6e702a), merged as 00e23c2. `build:server` emits `dist/server` (gitignored).
