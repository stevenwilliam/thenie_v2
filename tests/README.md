# Tests

Boundary tests for the Daily Order pricing engine — the rules documented as
**BR-3.1 – BR-3.13** in [`../docs/02-business-rules.md`](../docs/02-business-rules.md).

## Run them

```bash
cd /home/dev/projects/thenie_v2
node --test tests/
```

Requires Node 20+ (uses the built-in `node:test` runner). No dependencies, no
`npm install`.

## What is actually being tested

`extract-analyze.js` reads `site/index.html` at run time, pulls the `analyze()`
function and its two helpers **verbatim** out of the inline `<script>` by
brace-matching, and compiles them into a callable function. The rate tables are
read from the page's `data-rates` attributes the same way.

Nothing is reimplemented or copy-pasted. The tests exercise **the real shipped
code**, so:

- if the mirror is re-captured and upstream changed the pricing logic, these
  tests test the new logic automatically;
- if a test fails, either the page changed or the documentation is wrong —
  never "the copy drifted";
- **`site/index.html` is only ever read**, never written. Fidelity is untouched
  (see [`../docs/07-fidelity-and-verification.md`](../docs/07-fidelity-and-verification.md)).

## Status against the 2026-08-27 capture

**31 tests, 31 passing, 0 assertions changed.**

The re-capture rewrote the entire site around the order app, but the pricing
engine came through byte-identical — `diff` on the extracted `analyze()` from
both captures returns nothing. The tests confirmed that independently, and they
are the reason it can be stated as fact rather than hoped.

Two things in the harness did need updating, both structural rather than
behavioural:

| Change | Why |
|--------|-----|
| Find the `<script>` block that **defines `analyze()`**, instead of taking the first one | The page now ships two inline scripts. The site router comes first; the order app second. The old regex grabbed the router and failed to find `analyze()`. |
| Match `data-sub` and `data-rates` **adjacently** rather than within 400 characters | Cards without rates (Nasi Bento, Nasi Kuning, Paket Acara) could otherwise borrow the *next* card's `data-rates` and silently produce a wrong rate table. |

## Coverage

| Rule | What is proven |
|------|----------------|
| **BR-3.1** | 20, 25, 30 consecutive days → Bulanan |
| **BR-3.2** | Thu–Mon (a run straddling two calendar weeks) → Flexi Mingguan, and it costs **more** than an aligned Mon–Fri week |
| **BR-3.3** | 15, 16, 17, 18, 19 consecutive days → **full daily rate**; 19 days costs more per day than 14 |
| **BR-3.4** | Mon–Fri and Mon–Sat inside one calendar week → Mingguan |
| **BR-3.5** | 1, 2, 3, 4 consecutive days → Harian |
| **BR-3.8** | 20+ scattered days within 45 → Flexi Bulanan |
| **BR-3.9** | 5–19 scattered days → Flexi Mingguan, at any span |
| **BR-3.10** | Fewer than 5 scattered days → full daily rate |
| **BR-3.11** | A clean 20-day Mon–Fri routine → Bulanan, not Flexi Bulanan |
| **BR-3.12** | The two-week worked example from the source's own comment, both the qualifying and the non-qualifying case |
| **BR-2.1** | `total = rate × days × pax` |
| — | Rate-table integrity: all four cards, the same five keys, and `monthly ≤ weekly ≤ daily` |
| — | Timezone safety: the helper itself is asserted, so a TZ change cannot silently invalidate every date-shape test |

## The timezone trap

Every date in these tests is formatted **locally**, never with
`toISOString()`. This machine runs `Asia/Jakarta` (UTC+7), so local midnight is
17:00 UTC the previous day — `toISOString()` would shift every date back by one
and quietly turn a Mon–Fri run into Sun–Thu. The page's own `toKey()` formats
locally for exactly this reason, and the test helper is asserted directly so
the trap cannot reopen unnoticed.

## What is not covered

- The tier-based cards (Nasi Bento, Nasi Kuning, Paket Acara) — simple
  `price × qty × dates`, no branching worth testing.
- Catering Kantor's pax bands and weekday walk.
- Add-on arithmetic, including the Flexi meat cap (BR-6.8).
- The Regular Catering 1–5 pax table (BR-5.x).
- The WhatsApp message builder.
- Anything in the DOM — these are pure-function tests.

The engine was chosen because it is where the branching, and therefore the risk,
actually is: ten ordered conditions, two "routine" heuristics, and several
overlapping ranges.
