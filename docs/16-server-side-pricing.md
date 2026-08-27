# 16 — Server-side pricing

The order calculator, ported from the browser to Go, with its thresholds lifted
out of the code into configuration.

Two separate things, and they succeed to different degrees:

| Goal | Status |
|------|--------|
| The formula runs on the backend | ✅ ported, and **proven equivalent** on 758 cases |
| The backend's number is what gets charged | ✅ in `authoritative` mode, ⚠️ opt-in |
| Price rules configurable without a deploy | ✅ |

---

## 1. The port

`server/internal/domain/pricing` is the captured page's calculator in Go:

| Front end | Ported to | Covers |
|-----------|-----------|--------|
| `analyze()` | `pricing.Analyze` | the six-tier classifier, BR-3.1 – BR-3.13 |
| `render()` money block | `pricing.QuoteSubscription` | totals, the 1–5 pax group table (BR-5.x), add-ons (BR-6.x) |
| `recalc()` in the tier IIFE | `pricing.QuoteTierProduct` | Nasi Bento, Nasi Kuning, Paket Acara (BR-14.x) |
| the Kantor IIFE | `pricing.QuoteKantor` | pax bands, the weekday walk (BR-13.x) |

It is a **port, not a reimplementation**. Where the JavaScript does something
surprising, the Go does the same surprising thing and says why:

- **The 15–19 day cliff.** A consecutive run past the Flexi ceiling but short of
  Bulanan pays the *full* daily rate — more per day than a 5-day order. Q-7 in
  [[09-open-questions]] asks whether that is intended; until it is answered, both
  engines do it.
- **The band fallback.** `bandFor` mirrors the front end's `tiers.find(...)`
  *including* its fallback to the last band when nothing matches. Diverging here
  would mean the server quoting one price for an out-of-range quantity and the
  page showing another.
- **Clamping, not rejecting.** A quantity below a product's minimum is clamped
  up, exactly as the page does. Rejecting would let the page accept an order the
  server refuses.
- **`Math.round` is half-up.** JavaScript rounds `.5` away from zero.
  `roundHalfUp` reproduces that with integer arithmetic, so the two engines
  cannot drift by a rupiah on exactly the inputs nobody checks by hand.

No float touches a price anywhere. Money is `int64` whole rupiah throughout.

### Proven, not asserted

`scripts/gen-tier-cases.js` runs the **real `analyze()`**, extracted verbatim
from `site/index.html` by `tests/extract-analyze.js`, over 758 date shapes and
records what it decided. `TestMatchesShippedEngine` asserts the Go port agrees
on every one — tier, rate, day count, consecutive flag and span.

```
agreed with the shipped engine on 758/758 cases
coverage: map[daily:85 flexi:85 flexi-monthly:57 flexi-weekly:300 monthly:147 weekly:84]
```

The fixture uses a seeded PRNG, so regenerating it produces an identical file
unless upstream actually changed. That matters: after a re-capture, a diff on
`tier_cases.json` is the fastest possible answer to "did they change the pricing
rules?"

The shapes are chosen to walk every boundary in the ladder: consecutive runs of
1–32 days from all seven weekdays (crossing 5, 14, 15, 19 and 20), Mon–Fri and
Mon–Sat routines of 1–26 days from six different start weeks, both worked
examples from the source's own BR-3.12 comment, and 260 scattered sets in
windows of 7–62 days.

### And verified through the real UI

A headless browser drove the actual page — clicking quick-fill buttons, typing
pax counts, ticking add-ons, flipping the rice toggle — and compared the page's
own displayed total against a server quote built from the same DOM state:

```
healthy 5d/4pax/addons        page=1000000  server=1000000  MATCH
healthy 20d mon-fri           page=3760000  server=3760000  MATCH
bulking 6d/12pax/meat         page=4200000  server=4200000  MATCH
reguler 5d/5pax dengan nasi   page=590000   server=590000   MATCH
reguler 5d/5pax tanpa nasi    page=555000   server=555000   MATCH
reguler 5d/8pax (linear)      page=888000   server=888000   MATCH
nasibento 60box/2dates        page=2880000  server=2880000  MATCH
kantor 15pax bulanan          page=6600000  server=6600000  MATCH
```

That covers the add-on weekday restrictions, the Flexi meat cap path, the pax
table in all three of its modes, tier products and Kantor's weekday walk.

---

## 2. Configurable rules

The thresholds used to be literals inside `analyze()`: `n >= 20`, `n <= 14`,
`span <= 45`. They are now a row.

| Column | Default | Governs |
|--------|--------:|---------|
| `weekly_min_days` | 5 | consecutive days that qualify for Mingguan (BR-3.4) |
| `monthly_min_days` | 20 | consecutive days that qualify for Bulanan (BR-3.1) |
| `consecutive_flexi_weekly_max_days` | 14 | the Flexi Mingguan ceiling; past it and short of Bulanan the order pays full price (BR-3.2/3.3) |
| `flexi_monthly_max_span_days` | 45 | window for scattered dates to still get Flexi Bulanan (BR-3.8) |
| `weekday_routine_max_span_days` | 31 | window for a clean Mon–Fri/Sat routine to count as full Bulanan (BR-3.11) |
| `weekly_routine_max_span_days` | 14 | window for an order spanning two weeks to count as Mingguan (BR-3.12) |
| `weekly_routine_min_days_in_week` | 5 | dates one calendar week must hold for that to apply (BR-3.12) |
| `pax_table_max_pax` | 5 | above this the group table extends linearly (BR-5.4) |

**Defaults reproduce the captured page exactly**, so seeding changes nothing
about what the site charges.

### One row with typed columns, not key/value pairs

Deliberate. These values are only meaningful relative to each other, and a
key/value table cannot express a single one of these:

```sql
CONSTRAINT monthly_above_weekly CHECK (monthly_min_days > weekly_min_days),
CONSTRAINT flexi_ceiling_sane   CHECK (consecutive_flexi_weekly_max_days >= weekly_min_days
                                   AND consecutive_flexi_weekly_max_days < monthly_min_days),
CONSTRAINT flexi_span_sane      CHECK (flexi_monthly_max_span_days >= monthly_min_days),
...
```

Each rules out a combination that makes a branch of the classifier
**unreachable**. Those are silent failures: the prices still compute, they are
just computed by the wrong branch. So they are refused twice — in the domain,
with a message naming both values, and again by the database.

```bash
$ curl -X PUT .../admin/pricing-rules -d '{"weekly_min_days":20,"monthly_min_days":5,...}'
{"error":{"code":"RATE_INVARIANT_VIOLATED",
          "message":"pricing: monthly_min_days must be greater than weekly_min_days (weekly=20 monthly=5)"}}
```

### It actually moves the price

```bash
# before — 5 consecutive days
Mingguan (min. 5 hari) @ 38000 = Rp 190.000

# raise the Mingguan minimum to 6
$ curl -X PUT .../admin/pricing-rules -d '{"weekly_min_days":6, ...}'

# after — the same order, no deploy, no re-capture
Harian @ 50000 = Rp 250.000
```

The rules also ship in the **public** config document, on purpose: the page's own
calculator hard-codes the same numbers, so publishing them is what makes a drift
between the two visible from the browser.

---

## 3. The API

```
POST /api/v1/quote
```

Public, like `site-config` — the page already computes these numbers in the
browser, so the server answering the same question reveals nothing new.

The request carries **codes and quantities, never prices**. Every rupiah in the
answer is looked up from the database, so a caller cannot propose its own rate.
That is the point of moving the calculator off the client.

```bash
curl -X POST http://192.168.88.101:8082/api/v1/quote \
  -H 'Content-Type: application/json' -d '{
  "kind": "subscription",
  "plan": "healthy",
  "pax": 3,
  "dates": ["2026-08-17","2026-08-18","2026-08-19","2026-08-20","2026-08-21"],
  "addons": [{"code": "Extra Telur"}]
}'
```

`kind` is `subscription`, `tier_product` or `kantor`:

| kind | identifies with | quantity | dates |
|------|-----------------|----------|-------|
| `subscription` | `plan` slug, optional `rice` | `pax` | `dates[]` |
| `tier_product` | `product` slug + `package` | `qty` | `dates[]` |
| `kantor` | `grade` + `period` | `pax` | `start_date` |

An add-on may carry `dates[]` to narrow it to specific days, matching the day
chips the page renders (BR-6.5). The response itemises every charge, including
why one was reduced:

```json
{"code":"Extra Daging (khusus Kamis)","unit_price":20000,"days":3,
 "effective_pax":2,"total":120000,
 "note":"maks 1 porsi/5 pax → 2 porsi dari 12 pax"}
```

Admin:

```
GET /api/v1/admin/pricing-rules
PUT /api/v1/admin/pricing-rules
```

---

## 4. Making the page use it

This is where the frozen mirror bites.

The order app runs inside a closed IIFE and captures `data-rates` into a closure
at parse time. Every overlay is injected **after** that script has run. So the
overlay cannot replace the calculator — it can only sit beside it, read the same
inputs out of the DOM, ask the server, and then either report or override.

`site/overlays/pricing.html` does exactly that, in one of two modes, chosen by
the `order.pricing_mode` parameter.

### `verify` — the default

Shadow-quotes on every change and logs any disagreement. **Never blocks, never
changes what is charged.** If this script broke entirely the page would price
exactly as it does today.

```
[thenie/pricing] server-side pricing attached to 8 card(s), mode=verify
[thenie/pricing] the page and the server disagree on this card's total.
                 page=Rp 190.000 server=Rp 250.000
```

### `authoritative` — opt-in

The server's number is the one that goes in the cart. `card._current` is a plain
property on the element, so it is reachable even though the calculator that
wrote it is not; the overlay overwrites `total` and the displayed figure.

Adding to the order intercepts in the **capture phase on `document`**. That
detail is load-bearing: at the target element itself, listeners fire in
registration order and the page registered first, so a listener on the button
could never run before it. On an ancestor, capture wins.

Verified end to end — with `weekly_min_days` deliberately set to 6 so the two
engines disagreed:

```
mode                 authoritative
displayed            Rp 250.000     ← server's figure
card._current.total  250000         ← server's figure
cart bar total       Rp 250.000     ← server's figure reached the cart
```

The page's own calculator had said Rp 190.000.

### Which to run

**Start in `verify`.** Move to `authoritative` once the parity log is quiet on
real customer input. That is the honest order: `verify` is what tells you the two
engines agree on *your* data rather than on a fixture.

```bash
curl -X PUT -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H 'Content-Type: application/json' -d '{"value":"authoritative"}' \
     https://thenie.id/api/v1/admin/params/order.pricing_mode
```

### What `authoritative` costs

Honest limitations, all of them consequences of the page being frozen:

- **It is asynchronous.** A quote takes a round trip. If the API is slow or down,
  the page's own figure stands — a customer is never blocked from ordering
  because the pricing service is unwell.
- **It overrides the total, not the breakdown.** The WhatsApp message's per-item
  meta line still describes the page's own reasoning. The total is right; the
  narrative beside it may name a tier the server did not use.
- **It cannot change the Harga page's printed tables.** Those are static markup.
- **Rate *values* still come from the capture.** Changing `plan_rates` in the
  database changes what the server quotes, but the page's own calculator keeps
  the captured rates until it is re-captured. In `authoritative` mode the
  server's figure wins anyway; in `verify` mode you get a logged mismatch.

The clean fix for all of the above is the same one: inject the config into
`<head>` at build time so it lands **before** the app script. About twenty lines,
and the API already returns exactly the right shape. That is v2 work — Q-16 in
[[09-open-questions]].

---

## 5. Files

| Path | What |
|------|------|
| `server/internal/domain/pricing/rules.go` | the configurable thresholds + validation |
| `server/internal/domain/pricing/tier.go` | the ported six-tier classifier |
| `server/internal/domain/pricing/quote.go` | totals, pax table, add-ons, tier products, Kantor |
| `server/internal/domain/pricing/testdata/tier_cases.json` | 758 cases from the real engine |
| `scripts/gen-tier-cases.js` | regenerates that fixture |
| `server/internal/app/siteconfig/quote.go` | request → catalogue lookup → quote |
| `server/db/migrations/0007_pricing_rules.*.sql` | the rules table |
| `server/db/migrations/0008_pricing_mode.*.sql` | the `order.pricing_mode` parameter |
| `site/overlays/pricing.html` | the browser side |

Regenerate the fixture after any re-capture:

```bash
node scripts/gen-tier-cases.js > server/internal/domain/pricing/testdata/tier_cases.json
cd server && go test ./internal/domain/pricing/
```

Related: [[02-business-rules]] · [[04-pricing-catalogue]] · [[15-backend-engine]] · [[14-overlays]]
