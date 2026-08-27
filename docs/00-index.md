# Thenie Healthy Catering — documentation index

Everything here was reconstructed by reading the captured mirror at
`site/index.html`. It is **descriptive, not prescriptive**: it records what the
site does today, so it can be deployed, understood, and later rebuilt as a real
application without losing any behaviour.

Where a rule was inferred rather than stated on the page, it is marked
**(inferred)**. Where the page is silent, it is listed in
[[09-open-questions]] rather than guessed at.

_Capture date: **2026-08-27**. SHA-256 `b66ed302…` — see
[[07-fidelity-and-verification]]._

## Read in this order

| # | Document | What it covers |
|---|----------|----------------|
| 01 | [[01-product-overview]] | What the product is, who it serves, the five product families |
| 02 | [[02-business-rules]] | Normative rules with `BR-x.y` IDs — pricing, cut-offs, day restrictions |
| 03 | [[03-site-structure]] | The six client-side pages, and the order app's three tab levels |
| 04 | [[04-pricing-catalogue]] | Every price on the site, in full |
| 05 | [[05-order-flow-and-whatsapp]] | Cart model, validation, the WhatsApp handoff |
| 06 | [[06-design-system]] | Colours, type, components, accessibility findings |
| 07 | [[07-fidelity-and-verification]] | Proof the mirror is byte-exact, and how to re-verify |
| 08 | [[08-technical-inventory]] | File composition, JS surface, what the page does *not* do |
| 09 | [[09-open-questions]] | What the site does not answer |
| 13 | [[13-production-deployment-runbook]] | Deploy to the Ubuntu + Nginx server |
| 15 | [[15-backend-engine]] | The Go + PostgreSQL service that makes the content editable |
| 16 | [[16-server-side-pricing]] | The order calculator on the backend, and configurable price rules |
| 14 | [[14-overlays]] | The overlay mechanism — our only additions to the mirror |
|  — | [[PROGRESS]] | Build status |
|  — | `screenshots/` | Rendered evidence — all six pages, desktop and mobile |
|  — | `../tests/README.md` | Pricing-engine tests — 31 passing, run against the real code |

## What this site is

Two things, welded together and shipped as **one HTML file**:

1. **A marketing site** — five content pages (Beranda, Tentang Kami, Menu &
   Layanan, Harga, Kontak) covering the story, the service lines, the full
   price list, and the contact routes.
2. **An order application** — a sixth page (Pesan Online) that is the *entire*
   previous version of this project, carried over almost verbatim: eight order
   cards, calendar-driven date selection, a five-tier pricing engine, add-ons,
   a cart, and a WhatsApp checkout.

They share one stylesheet, one router, and one file. Navigation between them is
client-side, on the URL fragment (`#home`, `#order`, …).

## The backend

Since 2026-08-27 there is one — `server/`, a Go + PostgreSQL service that holds
the content and serves it to the page at runtime, so publishing next week's menu
is an API call rather than an HTML edit. It does **not** change anything below:
the mirror is still frozen, the page still works with the service switched off,
and the order flow is still WhatsApp. See [[15-backend-engine]].

## The single most important fact about the page itself

This is a **front end with no backend of its own**. There is no server, no database, no
account system, and no order storage. An order is assembled in browser memory
and handed to WhatsApp as a pre-filled message. Closing the tab loses
everything — there is no `localStorage`, no cookie, no draft recovery.

That is a deliberate property of the site, not a defect to fix here. It is also
the single biggest gap between this and a production system; see
[[09-open-questions]].

## What changed on 2026-08-27

The previous capture (2026-08-22) was the order form **alone** — one page, no
marketing, 4.6 MB. This capture wraps a complete marketing site around it.

| | 2026-08-22 | 2026-08-27 |
|---|---|---|
| Pages | 1 | 6 (client-side router) |
| Size | 4,615,031 B | 6,983,019 B |
| Embedded images | 13 | 44 (32 unique) |
| Typeface | system fonts | **Baloo 2, from Google Fonts** — the only external request |
| Palette | maroon `#7a1f2b` on cream | coral `#E1614A` + olive on white |
| Pricing engine | — | **byte-identical**, verified by diff |
| Our overlay | added a WhatsApp button | keeps the page's *own* button clear of the cart bar |

The order app's `analyze()` function — the whole pricing engine — is
**character-for-character identical** between the two captures. Every `BR-3.x`
rule in [[02-business-rules]] carried over unchanged, and the test suite passes
against the new mirror without a single assertion being edited. The only
differences in the order app's JavaScript are two CSS class renames
(`.card` → `.order-card`, `main` → `.order-app-main`) made to avoid colliding
with the new marketing markup.
