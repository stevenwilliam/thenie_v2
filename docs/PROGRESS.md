# PROGRESS

✅ done & verified by running · 🟡 partial · ⬜ not started

A ✅ here means it was **actually executed and observed**, not merely written.

_Last updated: 2026-08-22_

## 1. Repository

| Item | Status | Evidence |
|------|--------|----------|
| Folder `/home/dev/projects/thenie_v2` | ✅ | Created |
| `git init`, branch `main` | ✅ | Ran |
| Remote → `git@github.com:stevenwilliam/thenie_v2.git` | ✅ | `git remote -v` |
| Credentials reused from `healthy_catering` | ✅ | Same SSH key; `ssh -T git@github.com` → `Hi stevenwilliam!` |
| `user.name` / `user.email` set locally | ✅ | `stevenwilliam` / `itdept.sfg@gmail.com` |
| Remote repository exists and was empty | ✅ | `git ls-remote` returned no refs |
| First commit pushed | ✅ | See git log |

## 2. Site capture

| Item | Status | Evidence |
|------|--------|----------|
| Full crawl of the source URL | ✅ | `wget --mirror --page-requisites` → exactly 1 file |
| Byte-exact mirror stored | ✅ | 4,615,031 B, SHA-256 `9d4cfefb…` |
| Two independent fetches agree | ✅ | `curl` and `wget` produced identical digests |
| Nothing added, changed or removed | ✅ | Hash equals the live page's |
| Extra paths probed | ✅ | `robots.txt`, `sitemap.xml`, `favicon.ico`, `manifest.json`, `_redirects` → all 404 upstream |
| `scripts/verify-mirror.sh` | ✅ | Ran; passed local **and** upstream |
| Serves correctly over HTTP | ✅ | Local `http.server` → 200, `Content-Length: 4615031`, hash matched |
| **Rendered and visually inspected** | ✅ | Headless Chromium, 390px viewport — 4 screenshots in `docs/screenshots/` |
| Catering Kantor prices confirmed from the render | ✅ | `order_kantor.png` matches the documented table |

## 3. Documentation

| Doc | Status |
|-----|--------|
| `00-index.md` | ✅ |
| `01-product-overview.md` | ✅ |
| `02-business-rules.md` — 12 groups, `BR-x.y` IDs | ✅ |
| `03-site-structure.md` | ✅ |
| `04-pricing-catalogue.md` — every price | ✅ |
| `05-order-flow-and-whatsapp.md` | ✅ |
| `06-design-system.md` — contrast **calculated** | ✅ |
| `07-fidelity-and-verification.md` | ✅ |
| `08-technical-inventory.md` | ✅ |
| `09-open-questions.md` — 19 questions, each with a default | ✅ |
| `13-production-deployment-runbook.md` | ✅ |
| `README.md` | ✅ |

All content was read out of the mirror. Rules that were inferred are marked
**(inferred)**; genuine gaps went to `09-open-questions.md` rather than being
guessed at.

## 4. Tests

| Item | Status | Evidence |
|------|--------|----------|
| `analyze()` extracted verbatim from the mirror | ✅ | `tests/extract-analyze.js` |
| Pricing-tier test suite (BR-3.1–3.12) | ✅ | `tests/pricing.test.js` — see run output in `tests/README.md` |

## 5. Deployment

| Item | Status | Note |
|------|--------|------|
| Runbook written | ✅ | Parts 1–13 + Appendices A–C, absolute paths, `vi` |
| Matches the SCHOOL_CATERING server model | ✅ | Same Ubuntu + Nginx front door, added as one more subdomain |
| **Deployed to the server** | ⬜ | **Not run — needs your server.** See below. |
| Nginx config syntax-checked on the server | ⬜ | Needs `nginx -t` there |
| TLS issued | ⬜ | Needs DNS first |

## 6. Not verified — honest list

These are written but **not executed**, because this environment has no access
to the target server or a browser:

1. **Nothing is deployed.** The runbook has never been run. Nginx is not
   installed here, so `nginx -t` has not validated the config — it is written to
   the same pattern as your working SCHOOL_CATERING config, but it is unproven.
2. **Rendered headlessly, not on a real device.** Four screenshots were taken
   and inspected, and they corrected the docs in five places. But headless
   Chromium is not an iPhone — the page uses a system font stack, so type will
   differ. Worth one look on a real phone.
3. **The WhatsApp handoff was never exercised.** Doing so would send a real
   message to a real business number. Nothing was added to a cart either.
4. **BR-3.3 is confirmed in code but not with the business.** The 15–19
   consecutive-day dead zone (full price) is now proven by executing the real
   `analyze()` against it — see `tests/`. Whether the business *intends* it is
   still open (Q-7).

## 7. Next steps

- Decide the subdomain — Q-14 suggests `thenie.sunshinefood.co.id`
- Run the runbook
- Answer the 19 questions in [[09-open-questions]]
- Decide Q-16: is `thenie_v2` a rebuild, with `site/` frozen as the v1 reference?
