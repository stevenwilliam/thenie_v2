# 02 — Business rules

Normative rules extracted from the mirror, each with a stable `BR-x.y` ID so the
rest of the documentation, the tests, and any future rebuild can cite them.

**Every rule below was read out of the shipped code or the page's own copy.**
Where the two disagree, both are recorded and the conflict is raised in
[[09-open-questions]]. Where a rule is implied rather than written, it is marked
**(inferred)**.

> **Unchanged since the previous capture.** The pricing engine — `analyze()` and
> everything around it — is byte-identical to the 2026-08-22 capture. Every rule
> in this document was re-verified against the 2026-08-27 mirror, and the test
> suite in `../tests/` passes against it unmodified.

---

## BR-1 — Platform

| ID | Rule |
|----|------|
| **BR-1.1** | The site is a single HTML file. No backend, no API, no database. |
| **BR-1.2** | Navigation between the six pages is client-side, on the URL fragment. No page ever reloads. |
| **BR-1.3** | An order exists only in a JavaScript array (`cart`) in the open tab. |
| **BR-1.4** | Nothing is persisted. No `localStorage`, no `sessionStorage`, no cookie, no IndexedDB. |
| **BR-1.5** | Closing or reloading the tab destroys the cart and every part-filled card, silently and without warning. |
| **BR-1.6** | The only outbound action the site can take is opening a `wa.me` link in a new tab. |
| **BR-1.7** | The business learns about an order only if the customer presses **send** in WhatsApp. Abandonment is invisible. |
| **BR-1.8** | All prices are Indonesian Rupiah, formatted `id-ID`, rounded to whole rupiah (`Math.round`). |

---

## BR-2 — Money

| ID | Rule |
|----|------|
| **BR-2.1** | An item's subtotal is `menuTotal + addonTotal`. Nothing else — no tax line, no service charge, no delivery fee. |
| **BR-2.2** | The cart total is the plain sum of item subtotals. No basket-level discount exists. |
| **BR-2.3** | Prices are never negative and never zero once at least one date is chosen. |
| **BR-2.4** | No VAT/PPN is shown, added, or mentioned anywhere. **(inferred: prices are gross)** |
| **BR-2.5** | The WhatsApp message quotes the computed total as final. Fees the page describes but does not calculate (BR-9.4, BR-10.3) are therefore **missing from the quoted total**. |

---

## BR-3 — The daily-subscription pricing engine

This is the heart of the system. The customer selects a **set of dates** on a
calendar; the engine classifies that set into one of six tiers and applies the
matching per-day rate. It applies to all four Daily Order cards.

### Inputs

`dates` — an array of `YYYY-MM-DD` strings, always kept **sorted and unique**.
Derived values:

- `n` — how many dates are selected.
- `consecutive` — true when every adjacent pair is exactly one calendar day apart.
- `span` — `last − first + 1` in days (the outer window, gaps included).

### The six tiers

| Tier key | Label on the page | Rate used |
|----------|-------------------|-----------|
| `daily` | Harian | `daily` |
| `weekly` | Mingguan (min. 5 hari) | `weeklyPerDay` |
| `monthly` | Bulanan (min. 20 hari) | `monthlyPerDay` |
| `flexi-weekly` | Flexi Mingguan | `flexiWeeklyPerDay` |
| `flexi-monthly` | Flexi Bulanan | `flexiMonthlyPerDay` |
| `flexi` | Flexi (tanggal acak) | `daily` — **full price** |

### The classification, in evaluation order

The engine tests these in sequence and stops at the first match. **Order
matters**: several conditions overlap.

| ID | Condition | Result |
|----|-----------|--------|
| **BR-3.1** | `consecutive` and `n ≥ 20` | **Bulanan** |
| **BR-3.2** | `consecutive` and `5 ≤ n ≤ 14` and the run **crosses a calendar-week boundary** | **Flexi Mingguan** |
| **BR-3.3** | `consecutive` and `14 < n < 20` | **Flexi — full daily rate** |
| **BR-3.4** | `consecutive` and `n ≥ 5` (so 5–14 days inside one calendar week) | **Mingguan** |
| **BR-3.5** | `consecutive` and `n < 5` | **Harian** |
| **BR-3.6** | not consecutive, but a **clean 20+ day weekday routine** (see BR-3.11) | **Bulanan** |
| **BR-3.7** | not consecutive, but a **weekly routine spanning two weeks** (see BR-3.12) | **Mingguan** |
| **BR-3.8** | not consecutive, `n ≥ 20`, `span ≤ 45` | **Flexi Bulanan** |
| **BR-3.9** | not consecutive, `5 ≤ n < 20` (any span) | **Flexi Mingguan** |
| **BR-3.10** | anything else (scattered, fewer than 5 days) | **Flexi — full daily rate** |

### The two "routine" rules

| ID | Rule |
|----|------|
| **BR-3.11** | **Weekday routine → Bulanan.** A non-consecutive selection qualifies when: every date falls on Mon–Fri *or* every date falls on Mon–Sat; `n ≥ 20`; every gap between adjacent dates is 1 or 3 days (Mon–Fri) or 1 or 2 days (Mon–Sat); and `span ≤ 31`. This is the ordinary "weekdays only for a month" customer, who would otherwise be penalised for skipping weekends. |
| **BR-3.12** | **Two-week routine → Mingguan.** When BR-3.11 does not apply: every date falls on Mon–Fri or Mon–Sat; `span ≤ 14`; and **at least one Monday-anchored calendar week contains 5 or more of the selected dates**. Gap shape is *not* checked here. A customer who starts mid-week and continues into the next week is a real weekly customer, not a one-off. |

Worked example of BR-3.12, taken from the source's own comment:
`18, 19, 20, 21, 24 Aug` = 5 days, but week 17–23 holds only 4 and week 24–30
holds only 1 → **no week reaches 5** → stays Flexi.
`18, 19, 20, 21, 24, 26, 27, 28, 29 Aug` = 9 days, and week 24–30 holds 5
(24, 26, 27, 28, 29) → **the whole 9-day order** gets Mingguan.

### Week boundaries

| ID | Rule |
|----|------|
| **BR-3.13** | A calendar week runs **Monday → Sunday**. A date's week is identified by its Monday: `offset = (dow === 0 ? 6 : dow − 1)`. Sunday belongs to the week that *started* six days earlier. |

### Notes on BR-3.3

A consecutive run of 15–19 days pays the **full daily rate** — more per day than
a 5-day order and more than a 20-day one. The source marks this as a deliberate
revision ("Flexi Mingguan cuma berlaku untuk dipakai sampai 14 hari"), so it is
intentional, but it is a cliff a customer can fall off by extending an order.
Raised as Q-7 in [[09-open-questions]].

---

## BR-4 — Daily-subscription totals

| ID | Rule |
|----|------|
| **BR-4.1** | Default: `total = tierRate × n × pax`. |
| **BR-4.2** | `pax` is clamped to a minimum of 1. |
| **BR-4.3** | The tier rate is per **person per day**, never per order. |
| **BR-4.4** | No proration, no rounding to package boundaries: 23 days at the Bulanan rate costs 23 × rate, not one "month". |

---

## BR-5 — Regular Catering's pax table

Regular Catering alone carries a `data-pax-table`: an official per-group price
for 1–5 pax that replaces the flat per-pax rate.

| ID | Rule |
|----|------|
| **BR-5.1** | The table applies **only** on the strict `weekly` and `monthly` tiers. Every Flexi tier and Harian falls back to `rate × n × pax`. |
| **BR-5.2** | The table has two variants selected by the **Dengan Nasi / Tanpa Nasi** toggle. Dengan Nasi is the default. |
| **BR-5.3** | Table values are the **per-day total for the whole group**, not per person. |
| **BR-5.4** | For `pax > 5`: `dayTotal = round(table[5] / 5) × pax`. The 5-pax group rate is extended linearly; the discount stops deepening. |
| **BR-5.5** | The displayed "effective rate" is `round(dayTotal / pax)`. |
| **BR-5.6** | The toggle is cosmetic on every other tier — it changes the item's label but not its price. **(inferred from BR-5.1)** |

Actual values are in [[04-pricing-catalogue]].

---

## BR-6 — Add-ons (Tambahan)

| ID | Rule |
|----|------|
| **BR-6.1** | An add-on costs `price × pax × dayCount`, on top of the menu subtotal. |
| **BR-6.2** | `dayCount` is the number of selected dates the add-on actually applies to — **not** the number of dates in the order. |
| **BR-6.3** | An add-on may be restricted to certain weekdays via `data-restrict-days` (digit string, `0` = Sunday … `6` = Saturday). |
| **BR-6.4** | A restricted add-on is **auto-disabled and force-unchecked** whenever no selected date falls on an allowed weekday. |
| **BR-6.5** | On the four Daily Order cards, a checked add-on renders a **per-day chip picker**. It defaults to every eligible date; the customer can tap individual days off. |
| **BR-6.6** | Once the customer edits that picker manually, the choice is remembered and only pruned as dates are removed — it never silently re-expands. |
| **BR-6.7** | Unchecking an add-on discards its day selection entirely. |
| **BR-6.8** | **The Flexi meat cap.** On any Flexi tier (`flexi`, `flexi-weekly`, `flexi-monthly`), *Extra Daging (khusus Kamis)* is charged for `floor(pax / 5)` portions rather than `pax` — at most one portion per 5 pax. Below 5 pax it charges nothing. On Harian, Mingguan and Bulanan it charges per full pax as normal. |
| **BR-6.9** | Add-on rates are flat rupiah per pax per day. They do not scale with the tier discount. |

### Add-on catalogue — the four Daily Order cards

| Add-on | Price | Allowed weekdays |
|--------|-------|------------------|
| Extra Ayam | +Rp15,000 | Mon, Tue, Wed, Fri, Sat (`12356`) |
| Extra Telur | +Rp5,000 | any |
| Extra Ikan (khusus Rabu) | +Rp15,000 | Wed (`3`) |
| Extra Seafood (khusus Rabu) | +Rp20,000 | Wed (`3`) |
| Extra Daging (khusus Kamis) | +Rp20,000 | Thu (`4`) — and see BR-6.8 |
| Extra Sayur | +Rp5,000 | Mon–Sat (`123456`) |
| Extra Lauk Pendamping | +Rp5,000 | Mon–Sat (`123456`) |
| Ganti Nasi Merah | +Rp5,000 | Mon–Sat (`123456`) |
| Packaging ganti Thinwall | +Rp2,000 | any |

### Add-on catalogue — Nasi Bento and Catering Kantor

Both carry a nine-item list. It differs from the list above:

| Add-on | Price | Restriction |
|--------|-------|-------------|
| Ganti Thinwall | +Rp2,000 | none |
| Extra Protein Ayam | +Rp15,000 | none |
| Extra Telur | +Rp5,000 | none |
| Extra Protein Ikan | +Rp15,000 | **Nasi Bento:** none · **Kantor:** Wed (`3`) |
| Extra Protein Seafood | +Rp20,000 | none |
| Extra Daging (khusus Kamis) | +Rp20,000 | Thu (`4`) |
| Extra Sayur | +Rp5,000 | none |
| Extra Lauk Pendamping | +Rp5,000 | none |
| Ganti Nasi Merah | +Rp5,000 | none |

| ID | Rule |
|----|------|
| **BR-6.10** | Nasi Bento and Catering Kantor have **no per-day chip picker**. A checked add-on applies to every eligible date, all or nothing. |
| **BR-6.11** | BR-6.8's Flexi meat cap does **not** apply to Nasi Bento or Kantor — neither has Flexi tiers. |
| **BR-6.12** | Nasi Kuning and Paket Acara have **no add-ons at all**. |

---

## BR-7 — Dates and delivery cut-offs

| ID | Rule |
|----|------|
| **BR-7.1** | Dates are picked on an inline month calendar. Multiple dates per card, in any pattern. |
| **BR-7.2** | Dates strictly before today are always disabled. |
| **BR-7.3** | **Today is selectable until 12:00 local time.** After that its calendar cell is disabled too. |
| **BR-7.4** | Delivery-slot availability for *today* depends on the clock: **Pagi (both windows) is never available same-day**; **Siang (12.00)** closes at **09:00**; **Sore (18.00)** closes at **12:00**; **Request (dikonfirmasi admin)** stays open regardless. |
| **BR-7.5** | If the customer's chosen slot becomes disabled, selection falls back to the first still-enabled slot — silently. |
| **BR-7.6** | Healthy Meal and Bulking Extra **never** deliver on Sunday. Every Sunday cell is disabled on those two cards, whatever the date. |
| **BR-7.7** | Regular Catering and Kids Meal *may* be delivered on Sunday, but the menu differs: **lunch = Nasi Goreng Kampung**, **dinner = as set by Thenie**. A warning appears when a Sunday is selected. |
| **BR-7.8** | There is **no upper bound** on how far ahead a customer may order. |
| **BR-7.9** | Every date and cut-off calculation uses `new Date()` — the **customer's device clock**, in the device's own timezone. |
| **BR-7.10** | The cut-off is re-applied at "+ Tambah ke Order" time on the tier-based and Kantor cards, in case the clock crossed 09:00 or 12:00 while the card sat open. The four Daily Order cards do not re-check. |

> **BR-7.3 contradicts the page's own copy.** The Pesan Online home tab states
> *"Order untuk hari ini sudah tidak bisa dipilih — minimal untuk besok"*, and a
> source comment agrees ("Same-day ordering is closed"). The shipped code does
> not implement that. Raised as Q-21 in [[09-open-questions]].

### Date-filling helpers (Daily Order only)

| ID | Rule |
|----|------|
| **BR-7.11** | **5 Hari / 6 Hari** anchor to the *coming* Monday and fill forward. If that Monday is today or earlier, they skip to the following Monday — these packages are for a week that has not started. |
| **BR-7.12** | **20 Hari (Sen–Jum)** and **20 Hari (Sen–Sab)** start at **tomorrow** and walk forward, skipping the excluded weekdays, until 20 dates are collected. |
| **BR-7.13** | **Isi Rentang** (from–to) fills every calendar day in the range, capped at **60 days**. Reversed ranges are swapped. |
| **BR-7.14** | Isi Rentang is **hidden** on Healthy Meal and Bulking Extra — a raw range fill would include the Sundays those plans cannot deliver. |
| **BR-7.15** | Quick-fill and range-fill **replace** the current selection; they never merge into it. |

---

## BR-8 — Payment

| ID | Rule |
|----|------|
| **BR-8.1** | Payment is **bank transfer only**. No card, no e-wallet, no QRIS, no cash-on-delivery. |
| **BR-8.2** | The account is **BCA 8660-281-402**, a/n **R Bg Andreas Kurnianto** — a personal name, not a company. |
| **BR-8.3** | Customers must write `"Catering atas nama ..."` in the transfer description. |
| **BR-8.4** | The customer must confirm the transfer to the admin manually. Nothing detects payment. |
| **BR-8.5** | The site never sees, handles, or verifies money. |

---

## BR-9 — Changes and cancellation

| ID | Rule |
|----|------|
| **BR-9.1** | All changes must be reported by **H-1, 17:00 WIB**. Past that, Thenie may refuse. |
| **BR-9.2** | Orders **cannot be cancelled on the day of delivery**. |
| **BR-9.3** | Moving a date **within the same calendar week** is free. |
| **BR-9.4** | Moving a date **to a different week** costs **Rp10,000/pax** — waived if the customer continues into a full following package. |
| **BR-9.5** | A Mingguan package covers 7 days within one week; a Bulanan package covers 30 days from the payment date. |
| **BR-9.6** | Requests outside the published menu, and red-rice substitution outside the add-on, cost **+Rp5,000/pax**, confirmed by the admin. |
| **BR-9.7** | **None of BR-9.4 or BR-9.6 is implemented in the calculator.** They are policy text only; the quoted total ignores them. |

---

## BR-10 — Delivery

| ID | Rule |
|----|------|
| **BR-10.1** | Delivery windows: **Pagi 06.00–07.00**, **Pagi 07.00–09.00**, **Siang 12.00** (default), **Sore 18.00**, **Request (dikonfirmasi admin)**. |
| **BR-10.2** | Pagi 06.00–07.00 is restricted to **Lippo Karawaci & Amarillo**. Pagi 07.00–09.00 to **PHG, BPK Penabur, and parts of BSD**, subject to admin confirmation. These restrictions are **displayed but never enforced**. |
| **BR-10.3** | Free delivery requires **≥ 5 days** *and* a menu of **≥ Rp26,000/day**. Otherwise **+Rp5,000 per delivery**, across all categories. |
| **BR-10.4** | **BR-10.3 is not implemented.** No delivery charge is ever added to any total. |
| **BR-10.5** | Free cut fruit every **Friday** on Catering Kantor packages. |
| **BR-10.6** | Free delivery is also stated on the Harga page as "minimum 1 minggu (5 hari)" — consistent with BR-10.3's day count, silent on the rupiah floor. |

---

## BR-11 — Service area

| ID | Rule |
|----|------|
| **BR-11.1** | The Area dropdown offers exactly nine values: Gading Serpong (default), Karawaci, BSD, Alam Sutera, Medang, Villa Melati Mas, Park Serpong, Golden Stone, **Lainnya**. |
| **BR-11.2** | Area is **never** used in pricing. It is passed through to the WhatsApp message as text. |
| **BR-11.3** | **Lainnya** is accepted with no follow-up question and no surcharge. |
| **BR-11.4** | The marketing pages name a different set — Gading Serpong, BSD, Karawaci, Alam Sutera, Medang, plus Bintaro/Pondok Aren/Ciledug as expansion. The order form's list and the marketing list do not match. |

---

## BR-12 — Validation and submission

| ID | Rule |
|----|------|
| **BR-12.1** | Recipient details live **per item**, not at checkout. One cart can carry several recipients at several addresses. |
| **BR-12.2** | Adding an item requires **Nama Penerima**, **No. WhatsApp**, and **Alamat**. All three are presence-checked only. |
| **BR-12.3** | Tier-based cards (Nasi Bento, Nasi Kuning, Paket Acara) additionally require **at least one delivery date**. Catering Kantor requires a **Tanggal Mulai**. |
| **BR-12.4** | Daily Order cards enforce dates by disabling "+ Tambah ke Order" until at least one is picked. |
| **BR-12.5** | **No format validation exists anywhere.** Phone numbers, names and addresses are accepted as typed — no pattern, no length cap, no normalisation, no sanitisation. |
| **BR-12.6** | Checkout requires only that the cart is non-empty. |
| **BR-12.7** | After a successful add, the card **resets itself to blank** so the next item starts fresh. |
| **BR-12.8** | **Reset / Ulang Order dari Awal** always asks for confirmation — even with an empty cart, because a card may hold unsaved picks. |
| **BR-12.9** | Submitting opens `wa.me/62818100523` in a new tab with the message pre-filled. The site's involvement ends there. |

---

## BR-13 — Catering Kantor

| ID | Rule |
|----|------|
| **BR-13.1** | Minimum **5 pax**. |
| **BR-13.2** | Two grades — **Reguler** and **Healthy Catering** — and two periods — **Mingguan (5 days)** and **Bulanan (20 days)**. |
| **BR-13.3** | `total = rate × pax × days`, where `days` is 5 or 20 by period. |
| **BR-13.4** | The rate comes from a five-band pax table: 5–10, 11–20, 21–50, 51–100, 101–200. |
| **BR-13.5** | Above 200 pax the top band applies — `max` is `99999`, so there is no upper limit and no "contact us" path. |
| **BR-13.6** | The customer picks **one start date**; the period is a fixed day count walked forward from it. |
| **BR-13.7** | That walk-forward **skips Saturday and Sunday**. A Bulanan package is therefore 20 *weekdays* — four calendar weeks. |
| **BR-13.8** | Weekday-restricted add-ons count only the Rabu/Kamis that fall inside that computed range. |
| **BR-13.9** | Healthy Catering is **exactly Rp10,000/pax/day above** Reguler in every band and both periods. |

---

## BR-14 — Tier-based cards (Nasi Bento, Nasi Kuning, Paket Acara)

| ID | Rule |
|----|------|
| **BR-14.1** | `total = tierPrice × qty × numberOfDates`. Each selected date is a **separate full delivery** of the same quantity. |
| **BR-14.2** | The tier is chosen by `qty` alone. Number of dates never affects the unit price. |
| **BR-14.3** | `qty` is clamped up to the package minimum: **20** boxes (Nasi Bento), **10** boxes (Nasi Kuning), **25** pax (Paket Acara). |
| **BR-14.4** | Top tiers are unbounded (`max: 99999`). |
| **BR-14.5** | Paket Acara's tiers are labelled explicitly (`25-50 pax`, `>50 pax`); the others derive labels from their bounds. |
| **BR-14.6** | Paket Acara publishes each package's composition, but the customer **cannot choose menu items in the form** — they are told to discuss it with the admin in the notes. |

---

## BR-15 — Menu rotation

| ID | Rule |
|----|------|
| **BR-15.1** | Weekly menus are **hard-coded markup**, not data. Two weeks exist: **week 34 (17–21 Aug 2026)** and **week 35 (24–28 Aug 2026)**. |
| **BR-15.2** | Menus are published Monday–Friday only. |
| **BR-15.3** | Thursday is the meat day, marked ⭐ in every plan's menu list. |
| **BR-15.4** | Bulking Extra uses the **same menu and the same photographs** as Healthy Meal, with the protein portion doubled. |
| **BR-15.5** | Selecting a date outside the two published weeks is allowed, and no menu is shown for it. |
| **BR-15.6** | **Three calorie figures disagree.** The Daily Order card text says Healthy Meal is **430–500 kcal**. The Menu-tab poster's header says **"~ 550-590 KKAL / PORSI"**. And the five per-day totals printed on that same poster read **±455, ±455, ±465, ±485, ±465 kkal** — matching the card text, and contradicting the poster's own header. Verified by decoding and reading the image. |

Related: [[04-pricing-catalogue]] · [[05-order-flow-and-whatsapp]] · [[09-open-questions]] · `../tests/README.md`
