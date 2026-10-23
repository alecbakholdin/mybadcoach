#!/usr/bin/env bash
# Regenerate Go and TypeScript from the proto tree, then run sqlc.
#
# Run from the repo root via `npm run generate`. sqlc reads the migrations under
# server/internal/db/migrations/ and the queries under
# server/internal/db/queries/; both are committed.
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
# sqlc needs both the schema (migrations) and at least one query file. Both are
# committed; the guard keeps the script usable while queries are being added.
queries_dir="server/internal/db/queries"
if [ -f server/sqlc.yaml ] && [ -d "$queries_dir" ] && [ -n "$(find "$queries_dir" -type f -name '*.sql' -print -quit 2>/dev/null)" ]; then
  if ! command -v sqlc >/dev/null 2>&1; then
    echo "missing required tool: sqlc (run scripts/install-tools.sh)" >&2
    exit 1
  fi
  (cd server && sqlc generate)
else
  echo "skipped: no sqlc queries found under $queries_dir"
fi
