#!/usr/bin/env bash
# Strict lint gate: Go (golangci-lint) plus protobuf (buf lint + buf breaking).
#
# Usage: scripts/lint.sh [go|buf]   (no argument runs both)
#
# Run from the repo root via `npm run lint`. CI splits the two into separate
# jobs by passing `go` or `buf`. ESLint is a separate, deferred effort and is
# intentionally not wired here.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [ -d "$ROOT/.tools/bin" ]; then
  PATH="$ROOT/.tools/bin:$PATH"
  export PATH
fi

TARGET="${1:-all}"
case "$TARGET" in
  go|buf|all) ;;
  *) echo "usage: $0 [go|buf]" >&2; exit 2 ;;
esac

require() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required tool: $1" >&2
    echo "run scripts/install-tools.sh (buf, golangci-lint, and sqlc)" >&2
    exit 1
  fi
}

if [ "$TARGET" = "all" ] || [ "$TARGET" = "go" ]; then
  require golangci-lint
  echo "== golangci-lint (server) =="
  (cd server && golangci-lint run ./...)
fi

if [ "$TARGET" = "all" ] || [ "$TARGET" = "buf" ]; then
  require buf
  echo "== buf lint (proto) =="
  buf lint proto

  echo "== buf breaking (proto) =="
  # The first commit that introduces proto/ has no baseline to compare against.
  # CI runs this check once proto/ is on the default branch; BUF_BREAKING_BASELINE
  # overrides the auto-detected ref (a git ref, or a directory for a local
  # baseline).
  baseline="${BUF_BREAKING_BASELINE:-}"
  if [ -z "$baseline" ]; then
    for candidate in origin/main main; do
      if git rev-parse --verify --quiet "$candidate" >/dev/null \
        && git cat-file -e "$candidate:proto/buf.yaml" 2>/dev/null; then
        baseline="$candidate"
        break
      fi
    done
  fi

  if [ -z "$baseline" ]; then
    echo "skipped: no baseline branch contains proto/buf.yaml yet"
    echo "         (first run on this branch; CI enforces it once proto/ lands on main)"
  elif [ -d "$baseline" ]; then
    buf breaking proto --against "$baseline"
  else
    # The baseline module lives at proto/ inside the git ref.
    buf breaking proto --against ".git#ref=$baseline,subdir=proto"
  fi
fi
