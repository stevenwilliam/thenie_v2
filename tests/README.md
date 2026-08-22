# Tests

Boundary tests for the Daily Order pricing engine — the rules documented as
**BR-3.1 – BR-3.12** in [`../docs/02-business-rules.md`](../docs/02-business-rules.md).

## Run them

```bash
cd /home/dev/projects/thenie_v2
node --test tests/
```

Requires Node 20+ (uses the built-in `node:test` runner). No dependencies, no
`npm install`.

## What is actually being tested

`extract-analyze.js` reads `site/index.html` at run time, pulls the
`analyze()` function and its two helpers **verbatim** out of the inline
`<script>` by brace-matching, and compiles them into a callable function. The
rate tables are read from the page's `data-rates` attributes the same way.

Nothing is reimplemented or copy-pasted. The tests exercise **the real shipped
code**, so:

- if the mirror is re-captured and upstream changed the pricing logic, these
  tests test the new logic automatically;
- if a test fails, either the page changed or the documentation is wrong —
  never "the copy drifted";
- **`site/index.html` is only ever read**, never written. Fidelity is untouched
  (see [`../docs/07-fidelity-and-verification.md`](../docs/07-fidelity-and-verification.md)).

## Coverage

| Rule | What is proven |
|------|----------------|
| **BR-3.1** | 20, 25, 30 consecutive days → Bulanan |
| **BR-3.2** | Thu–Mon (a run straddling two calendar weeks) → Flexi Mingguan, and it costs **more** than an aligned Mon–Fri week |
| **BR-3.3** | 15, 16, 17, 18, 19 consecutive days → **full daily rate**; 19 days costs more per day than 14 |
| **BR-3.4** | Mon–Fri and Mon–Sat inside one calendar week → Mingguan |
| **BR-3.5** | 1–4 consecutive days → Harian |
| **BR-3.8** | 20 scattered dates within 45 days → Flexi Bulanan |
| **BR-3.10** | 3 random dates → full price |
| **BR-3.11** | 20 clean Mon–Fri weekdays, non-consecutive → Bulanan |
| **BR-3.12** | Both worked examples from the source's own comments — `18,19,20,21,24` stays Flexi; adding days until one week holds 5 promotes the **whole order** to Mingguan |
| **BR-2.1** | `unitPrice × qty × dates` — 3 pax Mon–Fri Healthy Meal = Rp 570.000 |
| — | Rate-table integrity: all four cards expose the same five keys, and `monthly ≤ weekly ≤ daily` holds for every card |

The BR-3.3 cliff is asserted explicitly, because it is the most surprising rule
in the system:

```js
assert.strictEqual(run(seq('2026-08-17', 14)).tier.rate, RATES.flexiWeeklyPerDay);
assert.strictEqual(run(seq('2026-08-17', 15)).tier.rate, RATES.daily);        // more expensive
assert.strictEqual(run(seq('2026-08-17', 20)).tier.rate, RATES.monthlyPerDay);
```

Whether the business *intends* that cliff is still an open question — Q-7 in
[`../docs/09-open-questions.md`](../docs/09-open-questions.md).

## A note on timezones

The first run of this suite failed seven tests. The cause was **the test
helper**, not the page: it built dates with `toISOString().slice(0,10)`, and
this machine runs `Asia/Jakarta` (UTC+7). Local midnight is 17:00 UTC the
previous day, so every generated date shifted back by one and a Mon–Fri run
became Sun–Thu — which really does price differently.

The page gets this right: its own `toKey()` formats from `getFullYear()` /
`getMonth()` / `getDate()` in local time. The helper now does the same, and a
sanity test asserts both that 2026-08-17 is a Monday and that `seq()` does not
drift, so a timezone change can never silently invalidate the suite.

## Last run

```
# tests 31
# suites 12
# pass 31
# fail 0
# duration_ms 233.536017
```

Executed 2026-08-22 on Node v20.20.2 — **all green**.

## Not covered

The add-on engine (BR-6), the delivery cut-off (BR-7) and the Catering Kantor
matrix (BR-5) are documented but not yet under test — they are bound to DOM
elements rather than being pure functions, so testing them needs a DOM harness
(jsdom or Playwright) rather than plain extraction. BR-7 in particular is worth
covering, because it reads the client clock (BR-7.9, Q-3).
