#!/usr/bin/env bash
# Regenerate Go and TypeScript from the proto tree, then run sqlc.
#
# Run from the repo root via `npm run generate`. sqlc stays a no-op until
# ticket 04 adds queries under server/internal/db/queries/.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [ -d "$ROOT/.tools/bin" ]; then
  PATH="$ROOT/.tools/bin:$PATH"
  export PATH
fi

if ! command -v buf >/dev/null 2>&1; then
  echo "missing required tool: buf (run scripts/install-tools.sh)" >&2
  exit 1
fi

echo "== buf generate (proto) =="
# Run inside proto/: buf looks for buf.gen.yaml in the working directory, and
# the plugin out paths are relative to that file.
(cd "$ROOT/proto" && buf generate)

echo "== sqlc (server) =="
# sqlc needs both the schema (migrations) and at least one query file. Until
# ticket 04 adds queries under server/internal/db/queries/, there is nothing to
# generate, so skip rather than fail on the missing queries directory.
queries_dir="server/internal/db/queries"
if [ -f server/sqlc.yaml ] && [ -d "$queries_dir" ] && [ -n "$(find "$queries_dir" -type f -name '*.sql' -print -quit 2>/dev/null)" ]; then
  if ! command -v sqlc >/dev/null 2>&1; then
    echo "missing required tool: sqlc" >&2
    exit 1
  fi
  (cd server && sqlc generate)
else
  echo "skipped: no sqlc queries yet (ticket 04 adds them)"
fi
