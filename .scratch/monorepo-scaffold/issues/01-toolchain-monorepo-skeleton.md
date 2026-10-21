# 01: Toolchain + monorepo skeleton

**What to build:** A working npm-workspaces monorepo with the shared TypeScript base config, git hygiene,
and both toolchains installed, so every later ticket has a green `npm install` and a `go test` to build on.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

- [ ] `node -v` reports ≥ 20 and `go version` reports ≥ 1.22 in a fresh terminal
- [ ] Root `package.json` declares `workspaces: ["packages/*","apps/*"]` and the run scripts
- [ ] `tsconfig.base.json` is strict (moduleResolution Bundler, verbatimModuleSyntax, noUncheckedIndexedAccess)
- [ ] `.gitignore` covers `node_modules/`, `dist/`, `.svelte-kit/`, `*.tsbuildinfo`, `coverage/`; `.gitattributes` sets `* text=auto eol=lf`
- [ ] `npm install` succeeds at the repo root
