#!/usr/bin/env bash
# Regenerate Go and TypeScript from the proto tree, then run sqlc.
#
# Run from the repo root via `npm run generate`. sqlc stays a no-op until
# ticket 02 adds migrations/queries under server/internal/db/.
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
first_sql="$(find server/internal/db -type f -name '*.sql' -print -quit 2>/dev/null || true)"
if [ -f server/sqlc.yaml ] && [ -n "$first_sql" ]; then
  if ! command -v sqlc >/dev/null 2>&1; then
    echo "missing required tool: sqlc" >&2
    exit 1
  fi
  (cd server && sqlc generate)
else
  echo "skipped: no migrations/queries yet (ticket 02 adds them)"
fi
