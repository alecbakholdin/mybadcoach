# Approve the backend dependency set

Type: task
Status: resolved

## Question

Review and approve the consolidated dependency set for the backend API effort (dependency-vetting
gate, `docs/testing.md`):

- `golang.org/x/crypto` — bcrypt password hashing (Go, runtime)
- `modernc.org/sqlite` — pure-Go SQLite driver (Go, runtime)
- `connectrpc.com/connect` — Go Connect server (Go, runtime)
- `@connectrpc/connect` + `@connectrpc/connect-web` — Connect browser client (npm, client workspace)
- `sqlc` — SQL-first codegen (dev-only tool; SQLite engine is beta — accepted in grilling)
- `buf` — `.proto` lint, breaking-change detection, codegen orchestration (dev-only tool)
- The frontend test-boundary fake chosen by ticket 03 (approved as part of ticket 03's resolution)

Approve or reject each. The resolution records the approved list; nothing lands without it.

## Answer

Approved by Alec on 2026-10-22, all six:

1. `golang.org/x/crypto` — bcrypt; argon2id is available in-package if ticket 04 prefers it
2. `modernc.org/sqlite` — pure-Go SQLite driver
3. `connectrpc.com/connect` — Go Connect server
4. `@connectrpc/connect` + `@connectrpc/connect-web` — Connect browser client
5. `sqlc` — SQL-first codegen (dev-only)
6. `buf` — proto lint/breaking/codegen (dev-only)

Transitives (`google.golang.org/protobuf`, `@bufbuild/protobuf`) are reviewed as lockfile diffs
at install time. The frontend test-boundary fake is approved as part of ticket 03's resolution.
Version pins and installation happen with the implementation tickets (next effort), not here.

7. `@connectrpc/connect-node` — dev-only, the stub Connect server behind frontend-only tests
   (ticket 03's resolution). Approved by Alec on 2026-10-22.
