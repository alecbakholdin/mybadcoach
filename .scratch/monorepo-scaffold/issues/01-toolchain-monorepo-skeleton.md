# 01: Toolchain + monorepo skeleton

**What to build:** A working npm-workspaces monorepo with the shared TypeScript base config, git hygiene,
and both toolchains installed, so every later ticket has a green `npm install` and a `go test` to build on.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] `node -v` reports ≥ 20 and `go version` reports ≥ 1.22 in a fresh terminal
- [x] Root `package.json` declares `workspaces: ["packages/*","apps/*"]` and the run scripts
- [x] `tsconfig.base.json` is strict (moduleResolution Bundler, verbatimModuleSyntax, noUncheckedIndexedAccess)
- [x] `.gitignore` covers `node_modules/`, `dist/`, `.svelte-kit/`, `*.tsbuildinfo`, `coverage/`; `.gitattributes` sets `* text=auto eol=lf`
- [x] `npm install` succeeds at the repo root

## Comments

Resolved by branch `ticket/01-toolchain-monorepo-skeleton` (3691e34), merged to integration as da2f60f.
