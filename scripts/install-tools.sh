#!/usr/bin/env bash
# Install the pinned codegen/lint tools into .tools/bin (gitignored).
#
# This is the reproducible install path for CI and the dev container. The
# scripts in this directory prepend .tools/bin to PATH, so an install here
# takes precedence over anything already on the machine.
#
# Tool versions are pinned to the versions the repo's output was generated and
# linted with. sqlc and golangci-lint are listed in README.md's toolchain
# section: sqlc has no install step here because it is a SQL parser with a cgo
# dependency, and golangci-lint is supplied by the dev image.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BUF_VERSION="1.75.0"

mkdir -p "$ROOT/.tools/bin"

echo "installing buf v${BUF_VERSION} into .tools/bin"
GOBIN="$ROOT/.tools/bin" go install "github.com/bufbuild/buf/cmd/buf@v${BUF_VERSION}"

echo "done:"
"$ROOT/.tools/bin/buf" --version
