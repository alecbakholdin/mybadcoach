#!/usr/bin/env bash
# Install the pinned codegen/lint tools into .tools/bin (gitignored).
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

BUF_VERSION="1.75.0"
GOLANGCI_LINT_VERSION="2.15.0"
SQLC_VERSION="1.31.1"

mkdir -p "$ROOT/.tools/bin"

echo "installing buf v${BUF_VERSION} into .tools/bin"
GOBIN="$ROOT/.tools/bin" go install "github.com/bufbuild/buf/cmd/buf@v${BUF_VERSION}"

echo "installing golangci-lint v${GOLANGCI_LINT_VERSION} into .tools/bin"
GOBIN="$ROOT/.tools/bin" go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v${GOLANGCI_LINT_VERSION}"

echo "installing sqlc v${SQLC_VERSION} into .tools/bin"
GOBIN="$ROOT/.tools/bin" go install "github.com/sqlc-dev/sqlc/cmd/sqlc@v${SQLC_VERSION}"

# Point git at the repo's versioned hooks (.githooks/), so the pre-commit
# codegen-sync hook is active on this checkout.
git config core.hooksPath .githooks

echo "done:"
"$ROOT/.tools/bin/buf" --version
"$ROOT/.tools/bin/golangci-lint" version
"$ROOT/.tools/bin/sqlc" version
