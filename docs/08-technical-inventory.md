# 08 — Technical inventory

What the file is made of, what its JavaScript does, and — as importantly — what
it does not do.

---

## 1. Composition

| | |
|---|---|
| Files | **1** |
| Size | 6,983,019 bytes (6.7 MiB) |
| Lines | 4,887 |
| Markup, CSS and JS | ~257 KB (3.7%) |
| Embedded base64 images | ~6.73 MB (96.3%) |
| Build tooling | none — hand-authored HTML |
| Framework | none |
| Dependencies | none, except the web font |

### Embedded images

| | |
|---|---|
| `data:` images | **44** |
| Unique payloads | **32** — 6 images are embedded more than once |
| JPEG | 31 |
| PNG | 13 |
| Decoded total | ~4.81 MB |
| Largest single image | ~505 KB (a menu poster) |

The most-repeated payload is the Thenie logo, embedded **7 times** — the hero,
five page heroes, and the footer — followed by a menu poster embedded 3 times
(Healthy and Bulking share their menus) and two more embedded twice each.
Deduplicating all of them would remove **1,110,700 bytes of base64** (~1.06 MB
of the file, ~0.79 MB decoded) — about **16% of the whole page**, for no visual
change at all. It requires editing the mirror, which is forbidden here, so it is
rebuild work.

### Compression

| | |
|---|---|
| Uncompressed (`dist/index.html`) | 6,985,620 B |
| gzip level 6 | **5,129,765 B** — 27% saved |

Better than it looks: the previous capture only compressed by about 4%, because
it was almost entirely base64 JPEG. This one carries far more real text, and
text compresses.

---

## 2. External dependencies

**Exactly one**, and it is new in this capture:

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Baloo+2:wght@400;500;600;700;800&display=swap" rel="stylesheet">
```

| Property | Value |
|----------|-------|
| What | Baloo 2, five weights |
| From | Google Fonts (`fonts.googleapis.com` → `fonts.gstatic.com`) |
| When | On every page load, from the visitor's browser |
| If blocked | Page still renders, in the fallback stack; `display=swap` means no invisible text |
| Privacy | The visitor's IP and User-Agent reach Google on every load |

There is **no** analytics, **no** tag manager, **no** tracking pixel, **no**
third-party script, **no** CDN library, and **no** iframe. Outbound links go
only to `wa.me`, `instagram.com`, and `mailto:`.

The previous capture had *no* external requests at all.

---

## 3. JavaScript surface

Two inline `<script>` blocks, both plain ES2015+, both wrapped in IIFEs. No
modules, no build step, no `async`/`await`, no `fetch`.

### Block 1 — the site shell (5,722 chars)

| Concern | What it does |
|---------|--------------|
| WhatsApp links | Four canned messages; rewrites every `.wa-link` `href` on load |
| Router | `hashchange` → `showPage()`; toggles `.active`, rewrites `document.title`, sets `aria-current` |
| Mobile nav | Toggles `#navlinks`, maintains `aria-expanded` |
| Counters | `IntersectionObserver` at `threshold:0.4`, cubic ease-out over 1,100ms, runs once |
| Pricing tabs | Measures the active button and moves `#tabHighlight`; re-syncs on resize, page entry, `load`, and `document.fonts.ready` |
| Inquiry control | Kontak page's Korporat/Personal/Event segmented control |
| Footer year | `new Date().getFullYear()` |

### Block 2 — the order application (76,188 chars)

| Concern | What it does |
|---------|--------------|
| Tabs | Three levels, plus programmatic cross-tab jumps |
| `.multi-date-picker` | Shared multi-date calendar for Nasi Bento / Kuning / Acara; exposes `getDates()` and `_resetMulti()` |
| `.single-date-picker` | Single-date calendar for Catering Kantor; exposes `_resetCal()` |
| `analyze()` | **The pricing engine.** Classifies a date set into one of six tiers — `BR-3.x` |
| `.order-card[data-rates]` | The four Daily Order cards: dates, quick-fills, range fill, pax, add-ons with per-day pickers, preferences, recipient, subtotal |
| `.plan-grid[data-plans-select]` | The three tier-based cards |
| Kantor IIFE | Its own `RATES` table, pax bands, and weekday-walking date logic |
| Cart | Push, remove by index, running total, summary rendering |
| Checkout | Validation, `sameRecipient` detection, message assembly, `window.open` |
| Reset | `confirm()`, then clears the cart and every card |

### Browser APIs used

`document.querySelector(All)` · `addEventListener` · `classList` ·
`dataset` · `JSON.parse` · `Date` · `Number.toLocaleString('id-ID')` ·
`IntersectionObserver` · `matchMedia` · `requestAnimationFrame` ·
`document.fonts.ready` · `encodeURIComponent` · `window.open` · `confirm`

### CSS features that set the floor

| Feature | Minimum |
|---------|---------|
| `:has()` | Chrome 105 · Safari 15.4 · Firefox 121 |
| `text-wrap: balance` | Chrome 114 · Safari 17.5 · Firefox 121 |
| `aspect-ratio` | widely supported |
| `clamp()` | widely supported |
| `env(safe-area-inset-*)` | iOS 11+ |

`:has()` is load-bearing: `body:has(#order.page.active)` is what lifts the nav
pill clear of the cart bar. On a browser without it, the pill overlaps the
checkout button. **Firefox 121 (Dec 2023) is the effective floor for the site
to lay out correctly.**

---

## 4. What the page does NOT do

| Absent | Consequence |
|--------|-------------|
| Any backend | No orders are stored anywhere |
| Any persistence | Reload destroys the cart (BR-1.4) |
| Any form `POST` | There is no `<form>` submission at all |
| Any input validation beyond presence | BR-12.5 |
| Any error handling | No `try`/`catch` anywhere; a single throw stops the app silently |
| Any loading state | Nothing is asynchronous except the font |
| Service worker / offline | None |
| `robots.txt` / `sitemap.xml` / `favicon.ico` / `manifest.json` | All 404 upstream |
| Meta description | None |
| Open Graph / Twitter cards | None — the site shares as a bare link with no title card |
| Canonical URL | None |
| JSON-LD structured data | None — no `LocalBusiness`, no `Restaurant`, no menu markup |
| Print stylesheet | None |

The page **does** now have six real `<h1>`s and per-page `<title>`s, which the
previous capture did not. That is a genuine SEO improvement, but it is not a
baseline on its own — see Q-13 in [[09-open-questions]].

---

## 5. Code-quality observations

Recorded for a rebuild. **None is fixable here** without breaking the
exact-mirror rule ([[07-fidelity-and-verification]]).

| # | Observation |
|---|-------------|
| **TQ-1** | **User input is interpolated into `innerHTML` unescaped.** `renderCartList()` builds the summary sheet with a template literal containing `item.recipient.name`, `.address`, `.phone` and `item.detail` (which carries the free-text "Catatan lain"). Typing `<img src=x onerror=alert(1)>` into a name field executes it. It is **self-XSS only** — there is no URL parameter, no shared state and no other user to attack — so the practical risk here is negligible. In a rebuild where an admin views submitted orders, the same pattern becomes a real stored-XSS vector. Escape on output. |
| **TQ-2** | The delivery-slot cut-off is re-checked at add-to-cart time on the tier-based and Kantor cards, but **not** on the four Daily Order cards. A Daily Order card left open across 12:00 can add an item with a slot that has closed. |
| **TQ-3** | All dates and cut-offs use the **customer's device clock and timezone**. A wrong clock, or a customer abroad, gets the wrong cut-off. |
| **TQ-4** | `TODAY` is computed **once** at script load. A tab left open overnight still believes it is yesterday. |
| **TQ-5** | The remove-from-cart handler captures the array index at render time. It re-renders after every removal, so it is correct — but it is fragile by construction. |
| **TQ-6** | Roughly 400 lines of near-identical markup are repeated across the four Daily Order cards (add-on lists, recipient fields, delivery radios). Any change must be made four times consistently. |
| **TQ-7** | Weekly menus are hard-coded markup, not data (BR-15.1). Publishing next week's menu means editing HTML. |
| **TQ-8** | `.grain::after` is declared twice; the first declaration is dead. |
| **TQ-9** | A `:root` comment reads *"single committed dark theme"* on what is a light theme, and `.date-picker-row` is present in every Daily Order card with `style="display:none"` — both are leftovers from earlier revisions. |
| **TQ-10** | The page's own copy says same-day ordering is closed; the code allows it until 12:00 (BR-7.3). Copy and behaviour disagree. |
| **TQ-11** | No `try`/`catch` anywhere. `JSON.parse` on a malformed `data-rates` would take down the entire order app with nothing shown to the customer. |

---

## 6. Performance

| Metric | Value |
|--------|-------|
| Requests | **2** — the document, then the font stylesheet (plus font files) |
| Transferred | ~4.9 MB gzipped |
| Blocking resources | None. CSS and JS are inline; the font uses `display=swap` |
| Repeat visits | `must-revalidate` means the whole 4.9 MB is re-fetched whenever the file changes |
| Images | All inline, so **none can be cached separately or lazy-loaded** |

The single-file design trades repeat-visit efficiency for zero-dependency
deployment. On a slow mobile connection the first paint waits on the whole
document. Extracting the images to cacheable files is the single largest
available win, and it is rebuild work — see Q-15 in [[09-open-questions]].

Related: [[06-design-system]] · [[07-fidelity-and-verification]] · [[09-open-questions]]
