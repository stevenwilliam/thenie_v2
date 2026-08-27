# 07 — Fidelity and verification

The rule this repository is built on:

> **`site/index.html` is a byte-exact copy of the source page and is never
> edited.** Everything we add lives in `site/overlays/` and is stitched in at
> build time.

This document records how that was established, and how to re-check it.

---

## The capture

| | |
|---|---|
| **Captured** | 2026-08-27 |
| **Source** | `https://thenie-catering-order.netlify.app/` |
| **Size** | **6,983,019 bytes** |
| **SHA-256** | `b66ed30212d3cb3ffe00c1385ea9a23996d8611cb3bed40a288fed99b6ed9689` |
| **Files** | Exactly one. No CSS, JS, image or font file is fetched from the origin. |
| **Trailing newline** | None. The file ends at `</html>`. |

### Previous capture, for the record

| | |
|---|---|
| Captured | 2026-08-22 |
| Size | 4,615,031 bytes |
| SHA-256 | `9d4cfefba381b6a8c3adbc822281e701c7b8cca98d1e7d40b5ac1ccafbb0df49` |

It remains in the git history. `git log --oneline -- site/index.html` finds it.

---

## The one legitimate difference from the live page

The live URL is served by Netlify, which **injects a five-line hosting banner
into `<head>` on the way out**:

```html
<!-- This site is hosted on Netlify. Anyone can build and deploy a site
     like this one for free: https://netlify.new/?utm_campaign=…
     Netlify hosting facts for this site: static/SSR served via Netlify Edge. -->
<meta name="hosting-provider" content="Netlify">
<meta name="netlify-deploy" content="https://netlify.new/?utm_campaign=…">
```

That is added by the CDN, not by the page's author. It is not part of the site,
and it must be stripped before any comparison — otherwise the mirror can never
match, however faithful it is.

**Verified 2026-08-27:** with those five lines removed, the live page and
`site/index.html` are **identical**. The mirror is the clean upstream artefact.

```
live page, raw                    6,983,555 B   sha e8fefaf6…
live page, banner stripped        6,983,019 B   sha 962a450a…
site/index.html, same filter      6,983,019 B   sha 962a450a…   ← match
site/index.html, unfiltered       6,983,019 B   sha b66ed302…   ← the pinned hash
```

> The filter is applied to **both** sides, not just the live copy. `grep` appends
> a trailing newline to a file that lacks one, so filtering only the live page
> leaves a phantom one-byte difference. This was found the hard way; the script
> now normalises both.

---

## What was checked

| Check | Result |
|-------|--------|
| Byte count matches the pinned figure | ✅ 6,983,019 |
| SHA-256 matches the pinned digest | ✅ `b66ed302…` |
| Live page matches, after stripping the Netlify banner | ✅ identical |
| Exactly one `</body>` in the file | ✅ (the build refuses otherwise) |
| HTML parses with no unclosed or stray tags | ✅ 0 errors, 0 unclosed |
| Both inline `<script>` blocks parse | ✅ `node --check`, 5,722 and 76,188 chars |
| Both inline `<style>` blocks are brace-balanced | ✅ 418/418 and 8/8 |
| Renders in a real browser | ✅ Chrome 151 headless, all six pages |
| JavaScript executes end to end | ✅ router, counters, calendars, tier tables, cart |
| Nothing added, changed or removed | ✅ no favicon, no `robots.txt`, no meta tags, no analytics — the original has none, so neither does the mirror |

### Evidence the JS actually runs

Rendered `#order` at 1280×900 and inspected the resulting DOM:

| Signal | Observed |
|--------|----------|
| Footer year written by JS | `2026` |
| Order page routed active | yes |
| Calendar cells rendered | 299 `.mc-day` buttons |
| Tier tables computed | 7 `.active-tier` cells |
| Kantor price line | `Rp 24.000/pax/hari × 5 pax × 5 hari (Mingguan)` |
| Add-on day pickers wired | 34 `.addon-days` containers |
| Checkout correctly gated | `#wa-submit-btn` still `disabled` with an empty cart |

Screenshots for all six pages, desktop and mobile, are in `screenshots/`.

---

## Proof the pricing engine did not change

The 2026-08-27 capture is a different *site*, but the same *engine*. This was
not assumed — it was checked by extracting `analyze()` from both captures by
brace-matching and diffing them:

```
$ diff analyze.OLD.js analyze.NEW.js
$ echo $?
0
```

**Character-for-character identical.** The whole order-app script differs by
only 28 diff lines, and every one of them is a rename made to avoid colliding
with the new marketing markup:

| Old | New |
|-----|-----|
| `main > .panel` | `.order-app-main > .panel` |
| `.card[data-rates]` | `.order-card[data-rates]` |
| `closest('.card')` | `closest('.order-card')` |
| `querySelectorAll('.card')` | `querySelectorAll('.order-card')` |

No behavioural change. Consequently every `BR-3.x` rule in [[02-business-rules]]
carried over intact, and the test suite passes against the new mirror **without
a single assertion being edited**.

---

## How to re-verify

```bash
/home/dev/projects/thenie_v2/scripts/verify-mirror.sh
```

| Exit | Meaning |
|------|---------|
| `0` | The mirror matches its hash, and upstream is unchanged (or unreachable — that case warns and passes). |
| `1` | **The local mirror has been modified.** Restore it: `git checkout -- site/index.html`. |
| `2` | **Upstream has changed.** Do not hand-edit anything — re-capture, and update all three pinned hashes. |

### Re-capturing

If upstream changes and you want to adopt it:

1. `curl -sSL https://thenie-catering-order.netlify.app/ -o /tmp/capture.html`
2. Strip the five Netlify banner lines (`strip_netlify()` in
   `scripts/verify-mirror.sh` shows exactly which).
3. Replace `site/index.html`.
4. Update the hash in **three** places:
   - `EXPECTED` in `scripts/verify-mirror.sh`
   - `MIRROR_SHA` in `scripts/build-site.sh`
   - the table at the top of this document
5. Re-run `node --test tests/` — the tests read the mirror at run time, so they
   will immediately tell you whether the pricing logic moved.
6. Re-run `./scripts/build-site.sh` and re-check the screenshots.

---

## Enforcement

The rule is not a convention — it is enforced by the build:

```bash
$ ./scripts/build-site.sh
FAIL: site/index.html no longer matches the recorded capture hash.
  expected: b66ed302…
  actual:   1a2b3c4d…
Restore it with: git -C /home/dev/projects/thenie_v2 checkout -- site/index.html
```

`build-site.sh` hashes the mirror before it does anything else and refuses to
run on a mismatch. A silently edited capture is exactly what the rule exists to
catch, so the build stops rather than shipping one.

It also refuses to guess: if the mirror ever contains anything other than
exactly one `</body>`, it exits rather than pick an insertion point.

---

## Why this matters

The mirror is the **specification**. Every rule in [[02-business-rules]], every
figure in [[04-pricing-catalogue]], and the tests in `../tests/` are all derived
from it by reading. If someone edits the mirror to "fix" something, all of that
silently becomes a description of a file that no longer exists anywhere else,
and there is no longer any way to tell what the business actually shipped.

Fixes go in `site/overlays/` — see [[14-overlays]].

Related: [[08-technical-inventory]] · [[14-overlays]] · [[13-production-deployment-runbook]]
