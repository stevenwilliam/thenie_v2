# thenie_v2

A byte-exact mirror of the **Thenie Healthy Catering** website, the
documentation reconstructed from it, and everything needed to deploy it.

```
site/index.html        the capture — NEVER edited, sha256-pinned
site/overlays/         our changes, stitched in at build time
scripts/build-site.sh  mirror + overlays → dist/index.html
scripts/verify-mirror.sh  proves the mirror still matches upstream
scripts/contrast.py    WCAG contrast for the palette — the source of docs/06's table
server/                the Go + PostgreSQL engine that makes the content editable
dist/index.html        what gets deployed (git-ignored — rebuild after pull)
docs/                  the reconstructed specification
tests/                 pricing-engine tests, run against the real shipped code
```

## The one rule

> **`site/index.html` is never edited.**

It is a byte-exact copy of the live page. Everything the documentation, the
tests and the deployment depend on is derived from it by reading. Editing it
turns all of that into a description of a file that exists nowhere else.

The rule is enforced, not merely stated: `build-site.sh` hashes the mirror
before doing anything and refuses to run on a mismatch.

Changes go in `site/overlays/` — see [docs/14-overlays.md](docs/14-overlays.md).

## Quick start

```bash
# check the mirror is intact, and still matches upstream
./scripts/verify-mirror.sh

# build the deployable page
./scripts/build-site.sh          # → dist/index.html

# run the pricing tests against the real shipped code
node --test tests/

# re-check the palette's contrast ratios
python3 scripts/contrast.py

# look at it
python3 -m http.server 8080 --directory dist
```

## The backend engine

`server/` is a Go + PostgreSQL service holding the content the page used to
hard-code. Publishing next week's menu is one API call instead of an HTML edit:

```bash
cd server && go build -o bin/thenied ./cmd/thenied && cd ..
./server/bin/thenied migrate up
./server/bin/thenied seed        # extracts the content FROM site/index.html
./server/bin/thenied serve
```

There is an admin UI at `/admin/` with accounts, roles and an audit trail —
create the first account with `thenied user create --email you@thenie.id
--name "You" --roles owner`. See
[docs/17-admin-ui-and-rbac.md](docs/17-admin-ui-and-rbac.md).

The page still works with the whole service switched off — see
[docs/15-backend-engine.md](docs/15-backend-engine.md).

## What the site is

One HTML file, 6.7 MB, no backend. Two things welded together:

- **A marketing site** — Beranda, Tentang Kami, Menu & Layanan, Harga, Kontak.
- **An order application** — Pesan Online: eight order cards, calendar-driven
  date selection, a six-tier pricing engine, add-ons, a cart, and a WhatsApp
  checkout.

Navigation between them is client-side on the URL fragment. Nothing is stored
anywhere — an order is assembled in browser memory and handed to WhatsApp as a
pre-filled message.

## Current state

| | |
|---|---|
| Capture | **2026-08-27** |
| Size | 6,983,019 B |
| SHA-256 | `b66ed30212d3cb3ffe00c1385ea9a23996d8611cb3bed40a288fed99b6ed9689` |
| Source | `https://thenie-catering-order.netlify.app/` |
| Verified identical to live | ✅ (after stripping Netlify's injected banner) |
| Overlays | 1 — `fab-clearance.html` |
| Tests | 31 passing |
| Deploy target | `thenie.id` on the existing Ubuntu + Nginx server |

## Documentation

Start at [docs/00-index.md](docs/00-index.md).

| | |
|---|---|
| [01](docs/01-product-overview.md) | What the product is |
| [02](docs/02-business-rules.md) | Every rule, with `BR-x.y` IDs |
| [03](docs/03-site-structure.md) | Pages, tabs, cards |
| [04](docs/04-pricing-catalogue.md) | Every price |
| [05](docs/05-order-flow-and-whatsapp.md) | Cart and the WhatsApp handoff |
| [06](docs/06-design-system.md) | Tokens, components, measured contrast |
| [07](docs/07-fidelity-and-verification.md) | Proof the mirror is exact |
| [08](docs/08-technical-inventory.md) | What it is made of |
| [09](docs/09-open-questions.md) | What it does not answer |
| [13](docs/13-production-deployment-runbook.md) | **Deploy it** |
| [14](docs/14-overlays.md) | The overlay mechanism |
| [15](docs/15-backend-engine.md) | **The backend engine** — menus as data, not markup |
| [16](docs/16-server-side-pricing.md) | **Server-side pricing** — the calculator in Go, rules as config |
| [17](docs/17-admin-ui-and-rbac.md) | **Admin UI & RBAC** — accounts, roles, audit trail |

## Deploying

Already set up? The whole job is three commands on the server:

```bash
cd /opt/thenie_v2 && git pull
./scripts/build-site.sh
sudo cp dist/index.html /var/www/thenie/index.html
```

**Do not skip the build.** `dist/` is git-ignored, so `git pull` never brings
it. Skipping the build silently publishes a stale file.

First time, or changing anything about the server, DNS or TLS: read
[docs/13-production-deployment-runbook.md](docs/13-production-deployment-runbook.md).
Nothing in the 2026-08-27 re-capture requires reconfiguring any of it.
