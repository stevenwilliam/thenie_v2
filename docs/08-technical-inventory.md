# 08 — Technical inventory

## File composition

| Part | Size | Share |
|------|-----:|------:|
| Base64 image data (13 URIs) | 4,429,184 B | **96.0%** |
| Inline JavaScript (1 block) | 76,145 B | 1.7% |
| Inline CSS (1 block) | 22,040 B | 0.5% |
| HTML markup and text | ~87,662 B | 1.9% |
| **Total** | **4,615,031 B** | 100% |

Images: **12 JPEG + 1 PNG**. The actual code — markup, style and behaviour — is
about **186 KB**; everything else is photography.

## Stack

**None.** No framework, no bundler, no transpiler, no package manager, no build
step. Hand-written HTML, CSS and vanilla ES6 in one file.

- No React, Vue, jQuery or any library
- No CSS framework
- No web fonts
- No polyfills
- No source map, no minification — the JS ships with its explanatory comments intact

Those comments are unusually good and were the primary source for
[[02-business-rules]]. They explain *why* rules exist, including deliberate
revisions to the pricing tiers.

## JavaScript surface

35 named functions, all inside IIFEs — no globals beyond the module boundary.

| Area | Functions |
|------|-----------|
| **Dates** | `toDateObj`, `toKey`, `formatTgl`, `formatDateListLabel`, `mondayKeyOf`, `addDate`, `removeDate`, `getEligibleDates`, `fillSkippingDays`, `quickFill`, `quickFillWeek`, `computeKantorDates` |
| **Calendar UI** | `renderCal`, `renderCalendar`, `renderChips` |
| **Pricing** | `analyze`, `tierFor`, `tierIdxFor`, `renderTierTable`, `recalc`, `paxTableDayTotal`, `cartTotal` |
| **Add-ons** | `allowedDaySet`, `renderAddonDayPickers`, `updateAddonAvailability` |
| **Delivery** | `getTodayCutoff`, `applyDeliveryCutoff`, `getDeliveryTime` |
| **Cart** | `updateCartBar`, `renderCartList`, `flashAdded`, `getRecipient`, `getPrefText`, `notifyChange` |
| **Checkout** | `validate`, `render` |

`analyze()` is the heart of the system — it takes the selected dates and returns
`{n, consecutive, span, tier}`, implementing BR-3.1 – BR-3.12.

## Data in the DOM

Configuration is stored in `data-*` attributes and parsed at runtime, which
keeps pricing declarative and out of the code:

| Attribute | Count | Carries |
|-----------|------:|---------|
| `data-price` | 54 | Add-on prices |
| `data-restrict-days` | 31 | Add-on weekday restrictions |
| `data-plans-select` | 3 | JSON package + tier tables (bento, nasi kuning, acara) |
| `data-rates` | 4 | JSON five-rate tables (the four Daily Order cards) |
| `data-goto-subtab` | 15 | Home CTA deep links |
| `data-sub` | 7 | Card / product-family key |
| `data-days`, `data-skip` | 16, 8 | Calendar quick-fill helpers |
| `data-jenis`, `data-periode`, `data-nasi` | 2 each | Catering Kantor selectors |

## What the page does *not* do

| Absent | Consequence |
|--------|-------------|
| **No backend** | No server of any kind. Pure static file. |
| **No `localStorage` / `sessionStorage`** | Confirmed by search — zero hits. The cart dies with the tab (BR-1.7). |
| **No cookies** | Nothing is set. |
| **No routing** | Tabs toggle CSS classes. The URL never changes, no history entry, back button leaves the site. |
| **No analytics or tracking** | No GA, no pixel, no beacon. |
| **No external requests at runtime** | Works fully offline once loaded. |
| **No service worker / PWA** | Not installable, no offline cache. |
| **No SEO baseline** | Only `<title>` and `viewport`. No meta description, no Open Graph, no JSON-LD, no canonical, no `robots.txt`, no `sitemap.xml`. Link previews will be bare. |
| **No `<h1>`** | The header title is styled text, not a heading element. |
| **No form `action`** | Nothing posts anywhere; submission is the WhatsApp deep link (BR-12.1). |
| **No input format validation** | No phone pattern, no length caps, no sanitisation. |
| **No error handling** | No try/catch around parsing or message construction. |

## Privacy

Genuinely strong, almost by accident: recipient names, phone numbers and
addresses are typed into the page, held in memory, and handed to WhatsApp. They
are **never transmitted anywhere else**, never stored, and never observed by a
third party — there is no third party. Closing the tab erases everything.

The counterpart is that the business has no record either until the customer
presses send in WhatsApp (BR-12.2).

## Performance

| Aspect | Assessment |
|--------|------------|
| **Payload** | 4.6 MB in one uncacheable-as-parts document — the main weakness. Every visit re-downloads all 13 photos, because they cannot be cached separately from the HTML. |
| **Requests** | 1. As few as physically possible. |
| **Render** | No render-blocking external CSS/JS; no font swap; no layout shift from late assets. |
| **Runtime** | Trivial — plain DOM work on a small tree. |
| **Compression** | Netlify serves it gzip/brotli-encoded. JPEG data URIs compress poorly (already compressed), so the wire cost stays high. |

The single highest-value optimisation for a rebuild — extracting the 13 images
to separate cacheable files with `Cache-Control: immutable` — is precisely what
the "exact mirror" requirement forbids here. It belongs to the rebuild, not to
this repo. The deployment runbook mitigates what it can at the server level; see
[[13-production-deployment-runbook]].

Related: [[07-fidelity-and-verification]] · [[09-open-questions]]
