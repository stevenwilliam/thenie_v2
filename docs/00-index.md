# Thenie Healthy Catering — documentation index

Everything here was reconstructed by reading the captured mirror at
`site/index.html`. It is **descriptive, not prescriptive**: it records what the
mockup does today, so the mockup can be deployed, understood, and later rebuilt
as a real application without losing any behaviour.

Where a rule was inferred rather than stated on the page, it is marked
**(inferred)**. Where the page is silent, it is listed in
[[09-open-questions]] rather than guessed at.

## Read in this order

| # | Document | What it covers |
|---|----------|----------------|
| 01 | [[01-product-overview]] | What the product is, who it serves, the five product families |
| 02 | [[02-business-rules]] | Normative rules with `BR-x.y` IDs — pricing, cut-offs, day restrictions |
| 03 | [[03-site-structure]] | Tabs, panels, sub-tabs, the eight order cards |
| 04 | [[04-pricing-catalogue]] | Every price in the mockup, in full |
| 05 | [[05-order-flow-and-whatsapp]] | Cart model, validation, the WhatsApp handoff |
| 06 | [[06-design-system]] | Colours, type, components, accessibility findings |
| 07 | [[07-fidelity-and-verification]] | Proof the mirror is byte-exact, and how to re-verify |
| 08 | [[08-technical-inventory]] | File composition, JS surface, what the page does *not* do |
| 09 | [[09-open-questions]] | What the mockup does not answer |
| 13 | [[13-production-deployment-runbook]] | Deploy to the Ubuntu + Nginx server |
| 14 | [[14-whatsapp-fab]] | The floating WhatsApp button — our one addition to the mirror |
|  — | [[PROGRESS]] | Build status |
|  — | `screenshots/` | Rendered evidence — home, menu, order, catering kantor |
|  — | `../tests/README.md` | Pricing-engine tests — 31 passing, run against the real code |

## The single most important fact

This is a **front-end mockup with no backend**. There is no server, no
database, no account system, and no order storage. An order is assembled in
browser memory and handed to WhatsApp as a pre-filled message. Closing the tab
loses everything — there is no `localStorage`, no cookie, no draft recovery.

That is a deliberate property of the mockup, not a defect to fix here. It is
also the single biggest gap between this and a production system; see
[[09-open-questions]].
