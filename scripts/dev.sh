#!/usr/bin/env bash
# Run the Go server and the SvelteKit client dev server together.
#
# Today the Go server is a one-shot stub: it prints the initial view and exits.
# When it grows a listener, this same script keeps it running alongside the client.
set -euo pipefail

cd "$(dirname "$0")/.."

npm run build:server
./dist/server &
SERVER_PID=$!

cleanup() {
  kill "$SERVER_PID" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

npm run dev:client
