# PROGRESS

✅ done & verified by running · 🟡 partial · ⬜ not started

A ✅ here means it was **actually executed and observed**, not merely written.

_Last updated: 2026-08-27 — the site was re-captured and everything below was
re-verified against the new mirror._

---

## 1. Repository

| Item | Status | Evidence |
|------|--------|----------|
| Folder `/home/dev/projects/thenie_v2` | ✅ | exists |
| `git init`, branch `main` | ✅ | ran |
| Remote → `git@github.com:stevenwilliam/thenie_v2.git` | ✅ | `git remote -v` |
| Credentials reused from `healthy_catering` | ✅ | same SSH key |
| `user.name` / `user.email` set locally | ✅ | `stevenwilliam` / `itdept.sfg@gmail.com` |

## 2. Site capture — 2026-08-27

| Item | Status | Evidence |
|------|--------|----------|
| New capture stored as the mirror | ✅ | 6,983,019 B, SHA-256 `b66ed302…` |
| Confirmed identical to the live page | ✅ | Netlify injects a 5-line hosting banner; stripping it from both sides gives identical digests (`962a450a…`) |
| Netlify-banner strip built into the verifier | ✅ | `strip_netlify()` in `scripts/verify-mirror.sh` |
| Trailing-newline trap found and handled | ✅ | `grep` appends one; the filter is applied to both sides so it cancels |
| `scripts/verify-mirror.sh` | ✅ | ran; passed local **and** upstream |
| Exactly one `</body>` | ✅ | build asserts it |
| HTML parses cleanly | ✅ | 0 errors, 0 unclosed tags |
| Both `<script>` blocks parse | ✅ | `node --check` — 5,722 and 76,188 chars |
| Both `<style>` blocks brace-balanced | ✅ | 418/418, 8/8 |
| **Rendered and visually inspected** | ✅ | Chrome 151 headless — all six pages, desktop + mobile |
| JavaScript verified executing | ✅ | 299 calendar cells, 7 tier cells, Kantor price line, checkout gate |
| Screenshots regenerated | ✅ | 8 files in `docs/screenshots/` |

## 3. Diff against the previous capture

| Item | Status | Finding |
|------|--------|---------|
| `analyze()` extracted from both captures and diffed | ✅ | **byte-identical** — the pricing engine did not change |
| Whole order-app script diffed | ✅ | 28 diff lines, all CSS-class renames (`.card`→`.order-card`, `main`→`.order-app-main`) |
| Order-app markup diffed | ✅ | prices, menus, add-ons, terms all unchanged |
| Marketing site | ✅ | entirely new — 5 pages, new palette, new typeface |
| Payload | ✅ | 4.6 MB → 6.7 MB; 13 images → 44 (32 unique) |
| New external dependency | ✅ | Google Fonts (Baloo 2) — the previous capture had none |

## 4. Overlays

| Item | Status | Evidence |
|------|--------|----------|
| Old `whatsapp-fab.html` retired | ✅ | the new capture ships its own `.fab-wa`; keeping ours would double the button |
| Collision confirmed by rendering the bare mirror | ✅ | at 360×760 the button covers the running total **and** the checkout button |
| `fab-clearance.html` written | ✅ | lifts `.fab-wa` clear of the cart bar and the nav pill, four breakpoint/page cases |
| Fix confirmed by rendering `dist/` | ✅ | same viewport, no overlap |
| Reduced-motion gap closed for `.fab-wa` | ✅ | the capture's global rule zeroes durations, not transforms |

## 5. Build and scripts

| Item | Status | Evidence |
|------|--------|----------|
| `MIRROR_SHA` re-pinned in `build-site.sh` | ✅ | `b66ed302…` |
| `EXPECTED` re-pinned in `verify-mirror.sh` | ✅ | same |
| `verify-mirror.sh` rewritten for the Netlify banner | ✅ | ran clean against live |
| `build-site.sh` refuses a tampered mirror | ✅ | hash gate unchanged, re-tested |
| `dist/index.html` built | ✅ | 6,985,620 B, SHA-256 `dbd63510…` |
| Output proven to be mirror + one insertion | ✅ | prefix and suffix byte-compared against the mirror |
| gzip measured | ✅ | 5,129,765 B at level 6 — 27% saved |

## 6. Tests

| Item | Status | Evidence |
|------|--------|----------|
| Harness updated for the two-script page | ✅ | now finds the block that defines `analyze()` |
| `readRates()` regex tightened | ✅ | `data-sub`/`data-rates` matched adjacently, so rate-less cards cannot borrow the next card's table |
| Suite run against the new mirror | ✅ | **31 tests, 31 passing, 0 assertions changed** |
| Rate tables read from the new mirror | ✅ | all four cards, five keys each, `monthly ≤ weekly ≤ daily` |

## 7. Documentation

Every document was rewritten from the new mirror.

| Doc | Status | Note |
|-----|--------|------|
| `00-index.md` | ✅ | rewritten |
| `01-product-overview.md` | ✅ | rewritten — five product families |
| `02-business-rules.md` | ✅ | rewritten — 15 groups, `BR-x.y` IDs, all re-verified |
| `03-site-structure.md` | ✅ | rewritten — six pages, three tab levels, eight cards |
| `04-pricing-catalogue.md` | ✅ | rewritten — every price, plus a calculator-vs-published consistency check |
| `05-order-flow-and-whatsapp.md` | ✅ | rewritten — message format re-read from the code |
| `06-design-system.md` | ✅ | rewritten — new palette, contrast **calculated** via `scripts/contrast.py`, 16 a11y findings |
| `07-fidelity-and-verification.md` | ✅ | rewritten — includes the Netlify-banner proof |
| `08-technical-inventory.md` | ✅ | rewritten — 11 code-quality observations |
| `09-open-questions.md` | ✅ | rewritten — 27 questions, 8 new, each with a default |
| `13-production-deployment-runbook.md` | ✅ | **preserved and surgically updated** — see below |
| `14-overlays.md` | ✅ | replaces `14-whatsapp-fab.md` |
| `README.md` | ✅ | rewritten |
| `tests/README.md` | ✅ | rewritten |

### The runbook was deliberately not regenerated

All the hard-won server work in `13-production-deployment-runbook.md` — the
Nginx server blocks, the `add_header`-replacement trap, the ACME-challenge
ordering, the ERR_TOO_MANY_REDIRECTS diagnosis, the whole Cloudflare appendix
with Origin Certificates and real-IP restoration — was **kept as it is**.
Only these changed:

| Change | Reason |
|--------|--------|
| Mirror hash, in 2 places | new capture |
| `4.6 MB` → `6.7 MB`, in 6 places | new payload |
| gzip estimate → measured 4.9 MB | the new file compresses far better |
| the `14-whatsapp-fab` wiki-link → `14-overlays`, 3 places | overlay renamed |
| Deploy check `class="wa-fab"` → `Floating-button clearance`, 4 places | the capture now ships its own button, so the old grep no longer proves `dist/` was published |
| "the page ships no `<h1>`" | no longer true — it has six |
| "no client-side routes" | no longer true — they are fragments, so nginx is unaffected, but the comment was wrong |
| **Added:** a "nothing here needs redoing" banner | so an already-configured server is not rebuilt from zero |
| **Added:** the Google Fonts dependency section | new, and it matters if a CSP is ever added |

## 8. Verified end-to-end

| Claim | How it was checked |
|-------|--------------------|
| The mirror is the live page | fetched live, stripped the CDN banner from both sides, digests match |
| The pricing engine is unchanged | extracted from both captures, `diff` returns nothing |
| The documented rules still hold | 31 tests pass against the new mirror unmodified |
| The build output is mirror + overlay only | prefix/suffix byte-compared |
| The page works | rendered in Chrome 151, six pages, JS execution confirmed in the DOM |
| The overlay fixes something real | before/after screenshots at 360×760 |

## 9. Not done

| Item | Status | Why |
|------|--------|-----|
| Deployed to `thenie.id` | ⬜ | ready to deploy; the runbook is current |
| Accessibility fixes | ⬜ | cannot touch the mirror — Q-25 in [[09-open-questions]] |
| SEO baseline | ⬜ | same reason — Q-13 |
| Image deduplication (1.06 MB, 16%) | ⬜ | same reason — Q-15 |
| Self-hosted font | ⬜ | same reason — Q-24 |
| Backend, persistence, validation | ⬜ | the v2 rebuild — Q-1 to Q-4, Q-16 |

Related: [[00-index]] · [[07-fidelity-and-verification]] · [[09-open-questions]]
