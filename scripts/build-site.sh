#!/usr/bin/env bash
# Build the deployable page: the untouched mirror + every overlay in
# site/overlays/, injected immediately before </body>.
#
# site/index.html stays byte-for-byte identical to the capture (see README and
# scripts/verify-mirror.sh); everything we add lives in site/overlays/ and is
# stitched in here. Output: dist/index.html (git-ignored — rebuild after pull).
#
# Usage: scripts/build-site.sh [output-file]
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIRROR="$ROOT/site/index.html"
OVERLAY_DIR="$ROOT/site/overlays"
OUT="${1:-$ROOT/dist/index.html}"
MIRROR_SHA="b66ed30212d3cb3ffe00c1385ea9a23996d8611cb3bed40a288fed99b6ed9689"

[ -f "$MIRROR" ] || { echo "FAIL: missing $MIRROR" >&2; exit 1; }

# Refuse to build from a tampered mirror — a silently edited capture is exactly
# what the byte-for-byte rule exists to catch.
actual="$(sha256sum "$MIRROR" | awk '{print $1}')"
if [ "$actual" != "$MIRROR_SHA" ]; then
  echo "FAIL: site/index.html no longer matches the recorded capture hash." >&2
  echo "  expected: $MIRROR_SHA" >&2
  echo "  actual:   $actual" >&2
  echo "Restore it with: git -C $ROOT checkout -- site/index.html" >&2
  exit 1
fi

# The mirror has exactly one </body>; bail loudly rather than guess if that ever
# stops being true.
n="$(grep -c '</body>' "$MIRROR")"
[ "$n" = "1" ] || { echo "FAIL: expected 1 '</body>' in the mirror, found $n" >&2; exit 1; }

overlays=()
if [ -d "$OVERLAY_DIR" ]; then
  while IFS= read -r f; do overlays+=("$f"); done < <(find "$OVERLAY_DIR" -maxdepth 1 -name '*.html' | sort)
fi
[ "${#overlays[@]}" -gt 0 ] || { echo "FAIL: no overlays found in $OVERLAY_DIR" >&2; exit 1; }

mkdir -p "$(dirname "$OUT")"
tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

# The hydration overlay needs to know where the config API lives. Same-origin
# ("/api/v1") is the default and is correct when Nginx proxies /api to the
# service; set THENIE_API_BASE when the API is on another host.
#
# This is injected as its own tiny <script> BEFORE the overlays, so the value is
# already set by the time the overlay runs, and so the overlay itself stays a
# static file with nothing substituted into it.
api_script=""
if [ -n "${THENIE_API_BASE:-}" ]; then
  case "$THENIE_API_BASE" in
    *[\"\'\<\>]*)
      echo "FAIL: THENIE_API_BASE must not contain quotes or angle brackets" >&2
      exit 1 ;;
  esac
  api_script="<script>window.THENIE_API_BASE=\"${THENIE_API_BASE}\";</script>"
fi

# Everything up to (not including) the </body> line, then the overlays, then the rest.
line="$(grep -n '</body>' "$MIRROR" | cut -d: -f1)"
head -n "$((line - 1))" "$MIRROR"  > "$tmp"
if [ -n "$api_script" ]; then
  printf '\n%s\n' "$api_script" >> "$tmp"
fi
for f in "${overlays[@]}"; do
  printf '\n' >> "$tmp"
  cat "$f"     >> "$tmp"
done
printf '\n' >> "$tmp"
tail -n "+$line" "$MIRROR" >> "$tmp"

mv "$tmp" "$OUT"
trap - EXIT
chmod 644 "$OUT"

echo "built: $OUT"
echo "  mirror:   $(wc -c < "$MIRROR") bytes (unchanged, sha256 verified)"
for f in "${overlays[@]}"; do echo "  overlay:  ${f#$ROOT/}"; done
echo "  api base: ${THENIE_API_BASE:-(same origin, /api/v1)}"
echo "  output:   $(wc -c < "$OUT") bytes  sha256 $(sha256sum "$OUT" | awk '{print $1}')"
