#!/usr/bin/env bash
# Install the pinned codegen/lint tools into .tools/bin (gitignored).
#
# Usage: scripts/install-tools.sh [buf|golangci-lint|sqlc ...]
# With no arguments, installs all three. CI installs only the tool a job needs.
#
# This is the reproducible install path for CI and the dev container. The
# scripts in this directory prepend .tools/bin to PATH, so an install here
# takes precedence over anything already on the machine.
#
# Tool versions are pinned to the versions the repo's output was generated and
# linted with.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p "$ROOT/.tools/bin"

install_buf() {
  echo "installing buf into .tools/bin"
  GOBIN="$ROOT/.tools/bin" go install "github.com/bufbuild/buf/cmd/buf@v1.75.0"
  "$ROOT/.tools/bin/buf" --version
}

install_golangci_lint() {
  echo "installing golangci-lint into .tools/bin"
  GOBIN="$ROOT/.tools/bin" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0"
  "$ROOT/.tools/bin/golangci-lint" version
}

install_sqlc() {
  echo "installing sqlc into .tools/bin"
  GOBIN="$ROOT/.tools/bin" go install "github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1"
  "$ROOT/.tools/bin/sqlc" version
}

if [ "$#" -eq 0 ]; then
  set -- buf golangci-lint sqlc
fi

for tool in "$@"; do
  case "$tool" in
    buf)           install_buf ;;
    golangci-lint) install_golangci_lint ;;
    sqlc)          install_sqlc ;;
    *) echo "unknown tool: $tool (want buf, golangci-lint, or sqlc)" >&2; exit 1 ;;
  esac
done

# Point git at the repo's versioned hooks (.githooks/), so the pre-commit
# codegen-sync hook is active on this checkout.
git config core.hooksPath .githooks

echo "done"
