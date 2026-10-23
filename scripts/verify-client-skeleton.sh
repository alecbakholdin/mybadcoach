#!/usr/bin/env bash
# Verify the SvelteKit client skeleton (ticket 06).
#
# Acceptance is behavioral: the production build succeeds, the dev server
# renders the shared reducer's initial view into its server-side HTML, and
# the shared package tests still pass.
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT" || exit 1

fail=0

echo "== shared tests =="
if npm run test:shared; then
  echo "ok   - npm run test:shared"
else
  echo "FAIL - npm run test:shared"
  fail=1
fi

echo "== production build =="
if npm run build:client; then
  echo "ok   - npm run build:client"
else
  echo "FAIL - npm run build:client"
  fail=1
fi

echo "== dev server SSR =="
log="$(mktemp -t mbc-client-dev)"
set -m
npm run dev:client >"$log" 2>&1 &
dev_pid=$!
set +m

cleanup() {
  kill -- -"$dev_pid" 2>/dev/null || true
  wait "$dev_pid" 2>/dev/null || true
  rm -f "$log"
}
trap cleanup EXIT

url=""
for _ in $(seq 1 120); do
  url="$(grep -oE 'http://localhost:[0-9]+/' "$log" 2>/dev/null | head -1)"
  [ -n "$url" ] && break
  kill -0 "$dev_pid" 2>/dev/null || break
  sleep 0.25
done

if [ -z "$url" ]; then
  echo "FAIL - dev server did not report a local URL"
  cat "$log"
  fail=1
else
  echo "dev server: $url"
  html="$(curl -fsS "$url" 2>/dev/null)"
  if printf '%s' "$html" | grep -qF 'Initial view: LINEUP_SETUP'; then
    echo "ok   - SSR HTML contains 'Initial view: LINEUP_SETUP'"
  else
    echo "FAIL - SSR HTML missing 'Initial view: LINEUP_SETUP'"
    printf '%s\n' "$html" | head -40
    fail=1
  fi
fi

if [ "$fail" -ne 0 ]; then
  echo
  echo "Verification FAILED"
  exit 1
fi

echo
echo "All checks passed."
