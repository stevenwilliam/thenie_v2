# 14 — Overlays

The mirror is never edited ([[07-fidelity-and-verification]]). Anything we need
to change about the deployed page goes in `site/overlays/`, and
`scripts/build-site.sh` stitches it in.

---

## How it works

```
site/index.html          the untouched capture, sha256-pinned
        +
site/overlays/*.html     every file, sorted by name
        ↓
scripts/build-site.sh    inserts them immediately before </body>
        ↓
dist/index.html          what actually gets deployed   (git-ignored)
```

The script:

1. Hashes `site/index.html` and **refuses to run** unless it matches
   `MIRROR_SHA`.
2. Refuses to run unless the mirror contains **exactly one** `</body>`.
3. Refuses to run if `site/overlays/` is empty.
4. Writes everything before the `</body>` line, then each overlay in sorted
   order, then the rest.
5. Prints the input sizes and the output hash.

Because overlays land at the very end of `<body>`, their CSS wins on equal
specificity and their JS sees a fully parsed document.

```bash
./scripts/build-site.sh              # → dist/index.html
./scripts/build-site.sh /tmp/out.html
```

`dist/` is git-ignored. **Rebuild after every pull** — this is the single most
common deployment mistake, and the runbook calls it out.

---

## Current overlays

### `fab-clearance.html` — floating-button clearance

**Added:** 2026-08-27 · **Replaces:** `whatsapp-fab.html`

#### Why the previous overlay was retired

The 2026-08-22 capture had no WhatsApp button, so this repository added one.

The 2026-08-27 capture **ships its own**:

```html
<a href="#" class="fab-wa wa-link" data-msg="home" aria-label="Chat WhatsApp">
```

styled `position:fixed; bottom:22px; right:22px; z-index:200`, 60px, `#25D366`,
wired to the same `62818100523` number with a canned message. Keeping our
overlay would have put **two** WhatsApp buttons on the page. It was deleted.

#### What this overlay does instead

The capture pins three things to the bottom of the viewport and only reconciles
two of them:

| Element | Position | Height |
|---------|----------|--------|
| `.cart-bar` | `bottom:0`, full width, `z-index:150` | ~102px |
| `.float-nav` | `bottom:20px`, centred — **lifted to 96px on `#order`** | 60px |
| `.fab-wa` | `bottom:22px`, right, `z-index:200` | 60px |

The page already lifts the nav pill clear of the cart bar, using
`body:has(#order.page.active)`. **It never does the same for the WhatsApp
button.** So:

- **On the Order page**, `.fab-wa` occupies 22–82px, entirely inside the cart
  bar's 0–102px band. It covers the right edge of "Lihat Order & Checkout" —
  which is full-width on a phone — and the running total beside it.
- **Below about 360px**, the nav pill (~186px wide, centred) and the button
  (60px at `right:22px`) overlap horizontally on **every** page.

Both were confirmed by rendering the bare mirror at 360×760 in headless Chrome.
Screenshot evidence: the WhatsApp button sits on top of both the "Rp 0" total
and the checkout button.

The overlay lifts the button over whatever is actually beneath it:

| Viewport | Page | `.fab-wa` bottom |
|----------|------|-----------------|
| ≥ 561px | any except Order | 22px *(capture default, untouched)* |
| ≥ 561px | Order | 110px — clears the cart bar |
| ≤ 560px | any except Order | 92px — stacks above the nav pill |
| ≤ 560px | Order | 168px — above the lifted nav pill |

All four use `calc(… + env(safe-area-inset-bottom))` so the button stays clear
of the home indicator on notched phones.

It also suppresses the button's hover `transform` under
`prefers-reduced-motion`, which the capture's global reduced-motion rule misses
— that rule zeroes durations, not transforms (see [[06-design-system]]).

**Nothing else changes.** Same size, colour, icon, link, `aria-label` and
right-edge anchor.

#### It depends on `:has()`

`body:has(#order.page.active)` is the same selector the capture itself uses for
the nav pill, so the overlay adds no new requirement. On a browser without
`:has()` (Firefox before 121) both the pill and the button fall back to the
capture's unfixed behaviour — see Q-26 in [[09-open-questions]].

#### Removing it

```bash
rm site/overlays/fab-clearance.html
./scripts/build-site.sh
```

The build will then stop, because it refuses to run with no overlays. That is
deliberate: an empty `overlays/` almost always means a file was deleted by
accident. If you genuinely want the bare capture deployed, publish
`site/index.html` directly and skip the build.

### `hydrate.html` — content hydration

**Added:** 2026-08-27

Fetches `/api/v1/site-config` from the backend engine and rewrites the weekly
menus and contact links in place, so a menu change is a database write instead
of a re-capture. Full detail in [[15-backend-engine]].

Three rules govern it, and they are the reason it is safe to ship inside a
frozen page:

1. **Never break the page.** No API, bad JSON, an unexpected shape, a thrown
   exception — every path leaves the captured content exactly as it was.
   Verified by stopping the service and reloading: the page renders fully, all
   299 calendar cells present, one informational console line.
2. **Only touch what JavaScript does not read.** The order app captures
   `data-rates` into a closure at parse time, and overlays are injected *after*
   that script has run. Writing those attributes would change what is displayed
   without changing what is calculated. So rates are **checked and reported**,
   never written.
3. **Same shape, same classes**, so the page's own CSS keeps working and a
   before/after diff is readable.

It caches to `localStorage`, sends `If-None-Match` (a 304 costs 200 bytes
instead of 25 KB), and gives up after 6 seconds so a slow API never holds the
page. `site.hydration_enabled=false` in `sys_parameters` turns it off without a
deploy.

#### Where it looks for the API

In order: `window.THENIE_API_BASE`, then `<meta name="thenie-api">`, then
same-origin `/api/v1`. `build-site.sh` injects the first one from the
`THENIE_API_BASE` environment variable when it is set:

```bash
THENIE_API_BASE=https://api.thenie.id/api/v1 ./scripts/build-site.sh
```

Leaving it unset is correct — and simpler — when Nginx proxies `/api/` to the
service on the same host, because then no CORS configuration is needed either.

---

## Verifying an overlay reached production

The capture now contains `class="fab-wa"` itself, so grepping for it no longer
proves you published `dist/` rather than the raw mirror. Grep for the overlay's
own marker instead:

```bash
grep -c 'Floating-button clearance' /var/www/thenie/index.html   # 1
grep -c 'class="fab-wa"'            /var/www/thenie/index.html   # 1
```

The first is the one that matters. If it prints `0`, `site/index.html` was
published instead of `dist/index.html`.

---

## Adding an overlay

1. Create `site/overlays/<name>.html`. Plain HTML — a `<style>` block, a
   `<script>` block, markup, or any mix.
2. Open it with a comment saying **what it changes and why**, and that it is not
   part of the capture. Every overlay in this repository does.
3. Files are concatenated in **sorted filename order**. Prefix with a number if
   order matters.
4. `./scripts/build-site.sh`, then check the result in a browser.
5. Document it in this file.

### What belongs in an overlay

- Fixing a layout collision the capture ships with — like this one.
- Adding something the capture lacks and the business needs.
- Anything that must be reversible by deleting one file.

### What does not

- Changing prices, menus or business rules. Those live upstream; changing them
  here makes the mirror a lie and silently invalidates
  [[02-business-rules]], [[04-pricing-catalogue]] and the tests.
- Anything large enough to be a redesign. That is the v2 rebuild — Q-16 in
  [[09-open-questions]].

---

## History

| Date | Overlay | Change |
|------|---------|--------|
| 2026-08-22 | `whatsapp-fab.html` | Added a floating WhatsApp button; the capture had none |
| 2026-08-27 | `whatsapp-fab.html` | **Removed** — the new capture ships its own |
| 2026-08-27 | `fab-clearance.html` | **Added** — keeps the capture's own button clear of the cart bar and the nav pill |
| 2026-08-27 | `hydrate.html` | **Added** — fetches content from the backend engine and rewrites the weekly menus in place |

Related: [[07-fidelity-and-verification]] · [[13-production-deployment-runbook]] · [[03-site-structure]]
