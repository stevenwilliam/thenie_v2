#!/usr/bin/env bash
# Verify that site/index.html is still a byte-exact copy of the source page.
# Usage: /home/dev/projects/thenie_v2/scripts/verify-mirror.sh
set -euo pipefail

SOURCE_URL="https://thenie-catering-order.netlify.app/"
MIRROR="/home/dev/projects/thenie_v2/site/index.html"
EXPECTED="b66ed30212d3cb3ffe00c1385ea9a23996d8611cb3bed40a288fed99b6ed9689"

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

# Netlify injects a five-line hosting banner into <head> on the way out (an HTML
# comment plus two <meta> tags). It is added by the CDN, not by the page's
# author, so it is NOT part of the site and must come off before comparing —
# otherwise the mirror can never match, no matter how faithful it is. Verified
# 2026-08-27: after stripping these five lines, the live page and the capture in
# site/index.html are identical.
#
# The same filter is run over BOTH sides. That is not redundant: the capture has
# no trailing newline and grep appends one, so filtering only the live copy would
# leave a phantom one-byte difference. Running it over both normalises that away.
strip_netlify() {
  grep -v -e 'This site is hosted on Netlify' \
          -e 'like this one for free: https://netlify.new/' \
          -e 'Netlify hosting facts for this site' \
          -e '<meta name="hosting-provider" content="Netlify">' \
          -e '<meta name="netlify-deploy"' \
          "$1"
}

raw="$(sha256sum "$tmp" | awk '{print $1}')"
live="$(strip_netlify "$tmp"    | sha256sum | awk '{print $1}')"
mine="$(strip_netlify "$MIRROR" | sha256sum | awk '{print $1}')"
echo "live (raw):                $raw"
echo "live (banner stripped):    $live"
echo "mirror (same filter):      $mine"
if [ "$live" = "$mine" ]; then
  echo "OK: upstream is unchanged since capture."
else
  echo
  echo "NOTE: upstream has CHANGED since 2026-08-27."
  echo "Do NOT hand-edit the mirror. Re-capture it and record the new hash:"
  echo "  curl -sSL $SOURCE_URL -o /tmp/capture.html"
  echo "  # strip the five Netlify banner lines (see strip_netlify above), then:"
  echo "  #   cp the result over $MIRROR"
  echo "  #   update EXPECTED here, MIRROR_SHA in scripts/build-site.sh,"
  echo "  #   and the hash in docs/07-fidelity-and-verification.md"
  exit 2
fi
