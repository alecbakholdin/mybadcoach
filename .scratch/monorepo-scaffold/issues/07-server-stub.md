# 07: Go server stub + build

**What to build:** A buildable Go server entry point that imports the reducer and proves it runs, before
any HTTP or DB wiring.

**Blocked by:** 03 — Domain types + event schema + parity harness.

**Status:** ready-for-agent

- [ ] `cmd/server/main.go` imports the reducer and prints the initial view
- [ ] `npm run build:server` (`go build ./cmd/server`) succeeds
- [ ] Running the binary prints the initial-view line
