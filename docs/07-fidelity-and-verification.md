# 07 — Fidelity and verification

The brief for this repo was explicit: reproduce the site **exactly as is** —
add nothing, change nothing, remove nothing. This document records how that was
achieved and how to re-prove it at any time.

## What was captured

| Property | Value |
|----------|-------|
| Source | `https://thenie-catering-order.netlify.app/` |
| Captured | 2026-08-22 |
| Local path | `site/index.html` |
| Size | **4,615,031 bytes** |
| SHA-256 | `9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49` |
| Files | **1** — the entire site |
| External requests at runtime | **0** |

## Why one file was enough

A `wget --mirror --page-requisites` crawl returned **exactly one file**. The
page is fully self-contained:

- CSS — one inline `<style>` block, 22,040 chars
- JS — one inline `<script>` block, 76,145 chars
- Images — **13 base64 data URIs** (12 JPEG, 1 PNG), 4,429,184 bytes, **96.0%
  of the file**
- Fonts — none requested; system stack only
- Third-party — none. No analytics, no CDN, no tracker, no external API

So a byte-exact copy of `index.html` is a byte-exact copy of the entire site.
Nothing can be missing, because there is nothing else to fetch.

These paths returned **404** and do not exist upstream: `robots.txt`,
`sitemap.xml`, `favicon.ico`, `manifest.json`, `_redirects`.

## How fidelity was preserved

| Decision | Reason |
|----------|--------|
| `--convert-links=off` | `wget` rewrites URLs by default. Disabled — that would have altered bytes. |
| No prettifying or minifying | The file is stored exactly as served. |
| No re-encoding | Written as received; no charset or line-ending conversion. |
| Data URIs untouched | All 13 images left inline. Extracting them to files would be "better practice" and would break the requirement. |
| Nothing added | No favicon, no `robots.txt`, no meta tags, no analytics — the original has none, so neither does the mirror. |
| Documentation kept outside | Every observation lives in `docs/`. Not one comment was added to `site/index.html`. |

## Verification — three independent confirmations

1. **Two separate fetches** (`curl`, then `wget`) produced identical SHA-256 digests.
2. **`scripts/verify-mirror.sh`** re-hashed the stored file and re-fetched
   upstream — both matched.
3. **Byte count** matches the `content-length` header returned by Netlify.

Run it yourself:

```bash
/home/dev/projects/thenie_v2/scripts/verify-mirror.sh
```

Exit codes: `0` all good · `1` the local mirror was modified · `2` upstream has
changed since capture.

Last run: **2026-08-22 — passed**, both local and upstream.

## If upstream changes

**Do not hand-edit the mirror to match.** Re-capture it:

```bash
wget -O /home/dev/projects/thenie_v2/site/index.html \
     https://thenie-catering-order.netlify.app/
sha256sum /home/dev/projects/thenie_v2/site/index.html
```

Then update the hash in **three** places — `scripts/verify-mirror.sh`, this
document, and `README.md` — commit the new capture on its own, and re-read the
docs for drift.

## Visual verification

The mirror was also **rendered and looked at**, not just hashed. Headless
Chromium (Playwright's cached build) served the file over a local HTTP server at
a 390px-wide mobile viewport:

| Screenshot | Shows |
|------------|-------|
| `screenshots/home.png` | Header, Halal certificate block, Cara Order, sticky cart bar |
| `screenshots/menu.png` | Menu tab, weekly poster artwork for Minggu ke-35 |
| `screenshots/order.png` | Order tab, Daily Order, the on-page tier explanation |
| `screenshots/order_kantor.png` | Catering Kantor pricing matrix, live subtotal |

**What rendering caught that reading the markup did not:** the tagline, the
BPJPH Halal certificate, the Sunday restriction on Healthy/Bulking (BR-2.6), the
free-fruit-Friday inclusion on Catering Kantor (BR-5.7), and the logo wordmark
"Food & Coffee Restaurant" — which exists only inside the base64 artwork and is
therefore invisible to any text search of the source.

`order_kantor.png` independently confirms the Catering Kantor rate table in
[[04-pricing-catalogue]] — Rp 24.000 / 23.000 / 22.000 / 21.000 / 20.000 across
the five tiers — rendered from the page rather than transcribed from the JSON.

**The mirror itself was not modified to do this.** Tab switching needs a click,
so throwaway copies carrying a small auto-click script were rendered from a
scratch directory. Those copies were discarded; `site/index.html` never changed,
as the hash check above still proves.

## Known limits of this capture

- **One point in time.** If the site is edited upstream, this is a snapshot of
  2026-08-22, not a live copy.
- **No server behaviour captured.** Netlify's headers (HSTS, caching, edge
  behaviour) are not reproduced by serving the file elsewhere; the deployment
  runbook sets its own — see [[13-production-deployment-runbook]].
- **Rendered headlessly, not on a real device.** Screenshots came from headless
  Chromium at a 390px viewport. Fonts are a system stack (see
  [[06-design-system]]), so a real iPhone or Android will differ slightly in
  type rendering. Open it once on a real phone.
- **Interactive flows were not exercised.** Nothing was added to a cart and no
  WhatsApp message was generated — that would send a real message to a real
  business number.

Related: [[08-technical-inventory]] · [[13-production-deployment-runbook]]
