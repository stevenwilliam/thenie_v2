#!/usr/bin/env bash
# Verify that site/index.html is still a byte-exact copy of the live page.
# Usage: /home/dev/projects/thenie_v2/scripts/verify-mirror.sh
set -euo pipefail

SOURCE_URL="https://thenie-catering-order.netlify.app/"
MIRROR="/home/dev/projects/thenie_v2/site/index.html"
EXPECTED="9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49"

echo "== 1. mirror on disk =="
actual="$(sha256sum "$MIRROR" | awk '{print $1}')"
echo "expected: $EXPECTED"
echo "actual:   $actual"
if [ "$actual" != "$EXPECTED" ]; then
  echo "FAIL: the local mirror has been modified. Restore it from git." >&2
  exit 1
fi
echo "OK: mirror matches the recorded hash."

echo
echo "== 2. live page =="
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
if ! curl -sSL --fail "$SOURCE_URL" -o "$tmp"; then
  echo "WARN: could not reach $SOURCE_URL — skipping upstream comparison." >&2
  exit 0
fi
live="$(sha256sum "$tmp" | awk '{print $1}')"
echo "live:     $live"
if [ "$live" = "$EXPECTED" ]; then
  echo "OK: upstream is unchanged since capture."
else
  echo
  echo "NOTE: upstream has CHANGED since 2026-08-22."
  echo "Do NOT hand-edit the mirror. Re-capture it and record the new hash:"
  echo "  wget -O $MIRROR $SOURCE_URL"
  echo "  # then update EXPECTED here and in docs/07-fidelity-and-verification.md"
  exit 2
fi
