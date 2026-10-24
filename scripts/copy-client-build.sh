#!/usr/bin/env bash
# Copy the built SvelteKit client into the Go server's embed directory so a
# `go build` bundles the SPA. Leaves the tracked .gitkeep placeholder in place,
# so a client-less checkout still compiles and serves nothing (404) until the
# client is built.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
src="$root/apps/client/build"
dst="$root/server/web/static"

if [ ! -d "$src" ]; then
  echo "warning: $src not found — Go will embed the placeholder (run 'npm run build:client' first)" >&2
  exit 0
fi

mkdir -p "$dst"
# Remove prior build output but keep the tracked .gitkeep placeholder.
find "$dst" -mindepth 1 -maxdepth 1 ! -name '.gitkeep' -exec rm -rf {} +
cp -r "$src"/. "$dst"/
