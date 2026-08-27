# 15 — The backend engine

A Go + PostgreSQL service that holds the site's content and serves it to the
page at runtime. It exists because of one line in [[02-business-rules]]:

> **BR-15.1** — Weekly menus are **hard-coded markup**, not data.

Publishing next week's menu meant editing HTML. Now it is one API call.

---

## Where it sits

```
                    ┌─────────────────────────────────────────┐
                    │  site/index.html   (frozen, byte-exact) │
                    │  + site/overlays/*.html                 │
                    │        ↓ scripts/build-site.sh          │
                    │  dist/index.html   ──▶ Nginx ──▶ visitor│
                    └──────────────────┬──────────────────────┘
                                       │  hydrate.html, on load:
                                       │  GET /api/v1/site-config
                                       ▼
                    ┌─────────────────────────────────────────┐
                    │  thenied  (Go, gin, gorm)               │
                    │    domain → app → adapter               │
                    └──────────────────┬──────────────────────┘
                                       ▼
                    ┌─────────────────────────────────────────┐
                    │  PostgreSQL  database `thenie`          │
                    └─────────────────────────────────────────┘
```

**The mirror is still never edited.** The service does not rewrite
`site/index.html`, does not regenerate it, and is not in the deploy path for it.
It serves JSON; an overlay reads that JSON and rewrites the DOM.

---

## The one thing it cannot do on its own

> **Updated 2026-08-27.** This section describes the *content* engine. The order
> calculator has since been ported to the backend as well, with an overlay that
> can make the server's figure authoritative — see [[16-server-side-pricing]].
> What follows still applies to the `data-rates` attributes themselves.

**This service cannot change what the order calculator charges.**

The order app captures `data-rates` into a closure the moment its `<script>`
runs:

```js
document.querySelectorAll('.order-card[data-rates]').forEach(card => {
  const rates = JSON.parse(card.dataset.rates);   // ← captured here, once
  ...
});
```

Overlays are injected **before `</body>`**, which is *after* that script has
already executed. Writing to `data-rates` from the overlay would change what the
page displays without changing what it calculates — a page that quotes one
number and charges another. That is worse than doing nothing.

So the overlay **checks** rates and never writes them. If the database and the
page disagree, it logs:

```
[thenie] the API's rates differ from the captured page's in 2 place(s).
         The order calculator uses the CAPTURED rates — re-capture the page to apply these.
```

**What this means in practice:**

| Content | Editable live? | How |
|---------|:--------------:|-----|
| Weekly menu cycles | ✅ | API call; live on next page load |
| Contact details (WhatsApp, email) | ✅ | API call |
| Everything else in the document | ✅ stored, ⚠️ not yet wired into the DOM | see *What is not hydrated yet* |
| **Subscription rates, tier prices, Kantor bands** | ⚠️ | stored and validated; the page's own calculator uses the captured values until re-capture, but `pricing.html` in `authoritative` mode makes the server's figure win — see [[16-server-side-pricing]] |

This is the honest cost of runtime hydration into a frozen page. The fix, if it
ever matters, is to inject the config into `<head>` at build time so it lands
before the app script — about twenty lines, and the API already returns exactly
the right shape for it.

---

## The database

`postgres://thenie@127.0.0.1:5432/thenie` — its own database and its own role,
matching the one-database-per-project pattern `healthy_catering` and `ruuma`
already use on this machine. Connection details are in `.env`
(git-ignored); `.env.example` is the documented surface.

### Schema

Six migrations, numbered, each with a matching `.down.sql`, embedded in the
binary with `go:embed`. **The migrations are the source of truth**; gorm maps
onto them and `AutoMigrate` is never used.

| # | Migration | Holds |
|---|-----------|-------|
| 0001 | `platform` | `sys_parameters`, `content_revision` + the bump trigger |
| 0002 | `catalogue` | `plans`, `plan_rates`, `plan_pax_prices`, `kantor_periods`, `kantor_rates`, `tier_products`, `tier_packages`, `tier_prices`, `addons` |
| 0003 | `menu` | `menu_cycles`, `menu_days`, `menu_components` |
| 0004 | `operations` | `service_areas`, `delivery_windows` |
| 0005 | `content` | `content_blocks` + every revision trigger |
| 0006 | `reference_data` | the 25 `sys_parameters` rows |

### Invariants the database itself enforces

Not the application — the database. A bad edit is stopped three times: in the
domain, by a `CHECK`, and by `thenied validate`.

| Constraint | What it refuses |
|------------|-----------------|
| `rates_monotonic` | `monthly ≤ weekly ≤ daily` — the shape the front end's pricing engine assumes |
| `flexi_never_cheaper` | a Flexi rate undercutting the tier it shadows |
| `flexi_never_over_list` | Flexi Mingguan above the daily list rate |
| `menu_cycles_no_overlap` | two **published** cycles claiming the same delivery date (a GiST exclusion constraint) |
| `delivery_windows_one_default` | more than one default delivery window (BR-12.7 needs exactly one) |
| `service_areas_one_catch_all` | more than one "Lainnya" |
| `addons.restrict_days ~ '^[0-6]*$'` | a weekday restriction that is not digits 0–6 |
| cycle span `≤ 6 days` | a "week" that is not a week |

One rule the database **cannot** express is enforced in the domain instead:
**quantity bands must form an unbroken ladder**. A gap is dangerous because the
front end's `tiers.find(...)` falls back to the *last* band when nothing
matches — so a table missing 51–100 does not error, it silently charges a 60-box
order at the 200+ price.

### Money

`BIGINT`, whole rupiah, integer arithmetic only. Sen is obsolete in retail, so
the rupiah *is* the minor unit. No `NUMERIC`, no floating point, anywhere near a
price — including the discount percentages, which are basis points rounded
half-up with integers.

---

## The seed is extracted, not written

`thenied seed` reads `site/index.html` and **extracts** the content:

```
$ thenied seed
seeding from mirror  path=site/index.html bytes=6983019
extracted  plans=4 tier_products=3 kantor_bands=20 addons=27
           areas=9 delivery_windows=5 menu_cycles=2
seeded from site/index.html: 4 plans, 3 tier products, 2 menu cycles
```

This matters more than it looks. The mirror is the specification
([[07-fidelity-and-verification]]), so a seed derived from it **cannot drift
from it**. A hand-typed fixture would be a second copy of every price, and the
first time the page was re-captured the two would silently disagree.

It reads `data-rates`, `data-pax-table` and `data-plans-select` as JSON; the
Kantor `RATES` constant, `TIER_MINS` and `TIER_MAXS` out of the JavaScript; the
add-on checkboxes, area `<option>`s and delivery radios out of the markup; and
the weekly menus out of the rendered `<div class="menu-day">` lines.

**Verified against [[04-pricing-catalogue]]:** every extracted value matches the
documented catalogue exactly — all four rate tables, all 20 Kantor bands, all
seven tier ladders, all 27 add-ons, all 9 areas, all 5 delivery windows, and
both menu cycles with 5 days each for all 4 plans.

And it round-trips: HTML → extractor → Postgres → API → overlay → HTML produces
markup **character-identical** to the capture for the same week.

---

## The API

### Public — one GET

```
GET  /api/v1/site-config              the whole document (~25 KB)
HEAD /api/v1/site-config
GET  /api/v1/site-config/revision     just the revision number
GET  /healthz  /readyz
```

One document, not eight tidy REST resources. The overlay runs inside a page that
is already 6.7 MB, often on a phone on mobile data — one round trip beats a
clean resource hierarchy.

`ETag: "rev-N"` comes from a database trigger that bumps on every content write,
so a browser that already has the current version pays **200 bytes instead of
25 KB**:

```
$ curl -I -H 'If-None-Match: "rev-370"' .../site-config
HTTP/1.1 304 Not Modified
```

### Admin — everything that writes

Behind `Authorization: Bearer $ADMIN_TOKEN`, compared in constant time.

```
GET    /api/v1/admin/params
PUT    /api/v1/admin/params/:key
PUT    /api/v1/admin/plans/:slug/rates
PUT    /api/v1/admin/tier-products/:slug/packages/:name/prices
PUT    /api/v1/admin/kantor/:grade/:period/rates
PUT    /api/v1/admin/menu/cycles                       ← the one that matters
POST   /api/v1/admin/menu/cycles/:year/:week/publish
POST   /api/v1/admin/menu/cycles/:year/:week/unpublish
DELETE /api/v1/admin/menu/cycles/:year/:week
GET    /api/v1/admin/validate
```

There is no user system. This service has exactly one class of caller — whoever
edits the menu — and inventing accounts for that would be scaffolding nobody
asked for. If more than one person ever needs distinct access, that is the
moment to add real identity.

### Publishing a week

```bash
curl -X PUT -H "Authorization: Bearer $ADMIN_TOKEN" \
     -H 'Content-Type: application/json' \
     -d '{
  "iso_year": 2026, "iso_week": 36,
  "starts_on": "2026-08-31", "ends_on": "2026-09-04",
  "label": "Minggu ke-36 · 31 Agustus – 4 September 2026",
  "publish": true,
  "days": {
    "healthy": [
      {"date": "2026-08-31", "kcal": 460, "items": [
        {"name": "Nasi Merah", "grams": 100},
        {"name": "Ayam Bakar Madu", "grams": 90},
        {"name": "Tumis Brokoli", "grams": 100}
      ]},
      {"date": "2026-09-03", "kcal": 490, "is_meat_day": true, "items": [
        {"name": "Kentang Rebus", "grams": 120},
        {"name": "Empal Daging Sapi", "grams": 80}
      ]}
    ]
  }
}' https://api.thenie.id/api/v1/admin/menu/cycles
```

A week lands **whole or not at all**. Half a week in the database is worse than
none: the page would render Monday to Wednesday, drop Thursday, and nothing
would look broken enough to notice.

Plan slugs are `healthy`, `bulking`, `reguler`, `kids`.

### What it refuses, and why

Every one of these was exercised against the running service:

| Request | Response |
|---------|----------|
| Sunday menu for Healthy Meal | `422 menu: this plan does not deliver on Sunday: healthy` (BR-7.6) |
| A day outside its own cycle | `422 menu day falls outside its cycle: 2026-10-01 not in 2026-09-14..2026-09-18` |
| An unknown plan | `422 CARD_KEY_UNKNOWN Unknown plan "vegan".` |
| A cycle overlapping a published one | `409 CYCLE_OVERLAP That week overlaps another published cycle.` |
| `monthly` above `weekly` | `422 RATE_INVARIANT_VIOLATED monthly=45000 weekly=38000 daily=50000` |
| A Flexi rate undercutting its tier | `422 RATE_INVARIANT_VIOLATED flexi_weekly=30000 < weekly=38000` |
| A tier ladder with a gap | `422 quantity bands leave a gap: nothing covers 51..100` |
| `"maybe"` for a boolean parameter | `400 value must be "true" or "false"` |
| Missing or wrong admin token | `401 UNAUTHENTICATED` |

Driver messages never reach a client. One JSON error model, always:

```json
{"error": {"code": "CYCLE_OVERLAP", "message": "That week overlaps another published cycle.",
           "details": {"constraint": "menu_cycles_no_overlap"}}}
```

---

## The overlay

`site/overlays/hydrate.html` — see [[14-overlays]] for the mechanism.

Three rules govern it:

1. **Never break the page.** Every failure path — no API, bad JSON, a shape that
   does not match, a thrown exception — leaves the captured content exactly as
   it was. Verified by stopping the service and reloading: the page renders
   fully, all 299 calendar cells present, one informational console line.
2. **Only touch what JavaScript does not read.** Hence rates being checked, not
   written.
3. **Same shape, same classes**, so the page's own CSS keeps working.

It also caches to `localStorage` and sends `If-None-Match`, so a returning
visitor on a flaky connection sees the last fetched menu rather than the
captured one, and a 6-second timeout means a slow API never holds the page.

`site.hydration_enabled=false` in `sys_parameters` is a kill switch that needs
no deploy.

### What is not hydrated yet

Stored and served, but not yet wired into the DOM: the Harga page's price
tables, service areas, delivery windows, testimonials and stats. They are in the
document and `window.__thenieConfig`, ready for whoever wires them. Menus and
contact details are done because they are what actually changes week to week.

---

## Running it

```bash
cd /home/dev/projects/thenie_v2

# once
sudo -u postgres psql -c "CREATE ROLE thenie LOGIN PASSWORD 'REPLACE_ME'"
sudo -u postgres createdb -O thenie thenie
sudo -u postgres createdb -O thenie thenie_test
cp .env.example .env && vi .env

cd server && go build -o bin/thenied ./cmd/thenied && cd ..

./server/bin/thenied migrate up
./server/bin/thenied seed
./server/bin/thenied validate
./server/bin/thenied serve
```

`serve` **refuses to start** with migrations pending. A service answering 500 to
every request because a table is missing is a much worse failure than one that
will not start and says why.

### Tests

```bash
cd server
TEST_DATABASE_URL=... go test ./...
```

The domain layer is exhaustively unit-tested and needs no database. The one
integration test pins something a dependency bump could silently break: that
gorm's constraint errors are still reachable as `*pgconn.PgError`. The first
version of `translate()` asserted `*pq.Error` — which never matches, because
`gorm.io/driver/postgres` is built on pgx — so every constraint violation fell
through as a 500 with a perfectly good message sitting unread in the log. That
bug is why the test exists.

---

## Deployment

The static site's deployment is **unchanged** — see
[[13-production-deployment-runbook]]. This service is additional, and the page
works without it.

Minimum viable production setup:

1. Build `thenied` and put it on the server.
2. Create the database and role; run `migrate up` then `seed`.
3. Run it under systemd with `APP_ENV=production` and a real `ADMIN_TOKEN`.
4. Give Nginx a `location /api/ { proxy_pass http://127.0.0.1:8082; }` on the
   existing `thenie.id` server block — then the overlay's same-origin default
   works and no CORS configuration is needed at all.
5. Rebuild the page once so `hydrate.html` is in `dist/index.html`.

If the API instead lives on its own host, set `CORS_ORIGINS` to the page's
origin and build with `THENIE_API_BASE=https://api.thenie.id/api/v1`.

Related: [[02-business-rules]] · [[04-pricing-catalogue]] · [[07-fidelity-and-verification]] · [[14-overlays]]
