# 02 — Business rules

**Normative.** Every rule carries a `BR-x.y` ID. A rebuild should reference
these IDs from its code and tests. Every rule here was read out of
`site/index.html`; nothing has been invented. Rules marked **(inferred)** are
consistent with the code but not stated on the page.

Money in this document is **whole rupiah**. The mockup does all arithmetic in
plain integers — there are no cents and no floating-point prices.

---

## BR-1 — Cart and items

| ID | Rule |
|----|------|
| **BR-1.1** | An order is a **cart of items**. Each item is one product, at one quantity, for one or more delivery dates. |
| **BR-1.2** | Each item carries **its own recipient** — name, phone, address, area, notes — and **its own delivery time**. Two items in one cart may go to two different people at two different addresses. |
| **BR-1.3** | Recipient name, phone and address are validated when the item is **added to the cart**, not at checkout. |
| **BR-1.4** | Checkout is **summary and re-confirmation only**. It collects no new field. |
| **BR-1.5** | Checkout requires **at least one item**. That is its only validation. |
| **BR-1.6** | An item may be deleted from the cart (`Hapus`). Editing is done by deleting and re-adding. |
| **BR-1.7** | The cart lives in **browser memory only**. Reloading or closing the tab discards it. There is no persistence of any kind — no `localStorage`, no cookie, no server draft. |

## BR-2 — Quantity, dates and the item total

| ID | Rule |
|----|------|
| **BR-2.1** | Every item's total is **`unit price × quantity × number of selected dates`**. Each selected date is a full separate delivery of the same quantity. |
| **BR-2.2** | Quantity may never fall below the product's **minimum tier quantity**. The input clamps up to that minimum. |
| **BR-2.3** | An item cannot be added until **at least one date** is selected. |
| **BR-2.4** | Dates are picked on a per-card mini-calendar and shown as removable chips. Non-consecutive runs are visually marked with a gap break. |
| **BR-2.5** | **Same-day ordering is closed.** Today's date cannot be selected — the earliest deliverable date is tomorrow. |
| **BR-2.6** | **Healthy Meal and Bulking Extra do not deliver on Sundays.** Stated on the page: *"Khusus Healthy Meal & Bulking Extra tidak melayani hari Minggu."* The other six cards carry no such restriction. |

## BR-3 — Daily Order pricing (the rate-tier engine)

Applies to the four Daily Order cards: Healthy Meal, Bulking Extra, Regular
Catering, Kids Meal. Each has five rates — see [[04-pricing-catalogue]].

The tier is **derived from the shape of the selected dates**, never chosen by
the customer. Let `n` = number of dates, `span` = calendar days from first to
last inclusive, `consecutive` = the dates form an unbroken run.

Evaluated **in order**; first match wins:

| ID | Condition | Tier | Rate |
|----|-----------|------|------|
| **BR-3.1** | `consecutive` and `n ≥ 20` | Bulanan | `monthlyPerDay` |
| **BR-3.2** | `consecutive`, `5 ≤ n ≤ 14`, run crosses a calendar week | Flexi Mingguan | `flexiWeeklyPerDay` |
| **BR-3.3** | `consecutive`, `14 < n < 20` | Flexi | `daily` *(full price)* |
| **BR-3.4** | `consecutive`, `n ≥ 5` | Mingguan | `weeklyPerDay` |
| **BR-3.5** | `consecutive`, `n < 5` | Harian | `daily` |
| **BR-3.6** | Weekday routine (below) | Bulanan | `monthlyPerDay` |
| **BR-3.7** | Weekly routine (below) | Mingguan | `weeklyPerDay` |
| **BR-3.8** | `n ≥ 20` and `span ≤ 45` | Flexi Bulanan | `flexiMonthlyPerDay` |
| **BR-3.9** | `5 ≤ n < 20` | Flexi Mingguan | `flexiWeeklyPerDay` |
| **BR-3.10** | otherwise | Flexi | `daily` |

**BR-3.3 is deliberate and counter-intuitive.** A consecutive run longer than 14
days but short of 20 pays the **full daily rate** — more per day than a 5-day
order. The discount returns only at 20 days (BR-3.1). The source marks this as a
revision, not an oversight.

### BR-3.11 — Weekday routine (non-consecutive → Bulanan)

All of: not consecutive · `n ≥ 20` · every weekday used falls in **Mon–Fri** or
every one in **Mon–Sat** · consecutive gaps are clean for that pattern
(Mon–Fri: 1 or 3 days; Mon–Sat: 1 or 2 days) · `span ≤ 31`.

### BR-3.12 — Weekly routine (non-consecutive → Mingguan)

All of: not consecutive · `n ≥ 5` · not already a weekday routine · clean
Mon–Fri or Mon–Sat weekday set · `span ≤ 14` · **at least one Monday–Sunday
calendar week contains 5 or more of the selected dates**.

The last clause is the sharp edge. `18,19,20,21,24 Aug` is five days but no
single week holds five of them, so it stays Flexi. Adding days until one week
holds five promotes the **whole order** to Mingguan.

## BR-4 — Tiered products (quantity-banded)

| ID | Rule |
|----|------|
| **BR-4.1** | Nasi Bento Box, Nasi Kuning and Paket Acara price by **quantity band**. The band is chosen by quantity; the customer does not pick it. |
| **BR-4.2** | Quantity below the lowest band's minimum is clamped up to it. |
| **BR-4.3** | Quantity above the highest band stays in the highest band — the top band's maximum is effectively unbounded. |
| **BR-4.4** | The active band is highlighted in the on-card rate table as quantity changes. |

## BR-5 — Catering Kantor

| ID | Rule |
|----|------|
| **BR-5.1** | Priced on a matrix of **jenis** (`reguler` / `healthy`) × **periode** (`mingguan` / `bulanan`) × **pax tier**. |
| **BR-5.2** | Pax tiers: `5–10`, `11–20`, `21–50`, `51–100`, `101+`. |
| **BR-5.3** | Committed days per period: **Mingguan = 5 days**, **Bulanan = 20 days**. |
| **BR-5.4** | `healthy` costs exactly **Rp 10.000 more per pax per day** than `reguler` at every tier and period. |
| **BR-5.5** | `bulanan` is exactly **Rp 1.000 less per pax per day** than `mingguan` at every tier. |
| **BR-5.6** | Minimum **5 pax per day**. |
| **BR-5.7** | **Free fruit every Friday** is included with Catering Kantor. Stated on the card; no price impact. |

## BR-6 — Add-ons

| ID | Rule |
|----|------|
| **BR-6.1** | Add-ons are per-item, priced **per pax per selected day**. |
| **BR-6.2** | An add-on may be **restricted to specific weekdays**. It then applies only to selected dates falling on those weekdays. |
| **BR-6.3** | Day codes are ISO-style digits: `1`=Mon … `6`=Sat. `Extra Ikan` and `Extra Seafood` are Wednesday-only (`3`); `Extra Daging` is Thursday-only (`4`). |
| **BR-6.4** | By default a checked add-on applies to **every eligible date**. |
| **BR-6.5** | The customer may **opt an add-on out of individual days** via day chips. Once touched, that choice is remembered and only pruned as dates change. |
| **BR-6.6** | If no selected date matches an add-on's allowed weekdays, the add-on is unavailable and cannot contribute. |
| **BR-6.7** | Regular Catering additionally offers a **Wednesday protein choice** — `Rabu: Ayam` or `Rabu: Seafood (termasuk ikan)` — as a free either/or, not a paid add-on. |
| **BR-6.8** | On **all Flexi tiers**, `Extra Daging (khusus Kamis)` is capped at **one portion per each multiple of 5 pax** (12 pax → 2 portions). Stated on the page as policy; **not enforced by the calculator**. |

## BR-7 — Delivery time and the ordering cut-off

| ID | Rule |
|----|------|
| **BR-7.1** | Five delivery windows: `Pagi (06.00–07.00)`, `Pagi (07.00–09.00)`, `Siang (12.00)`, `Sore (18.00)`, `Request (dikonfirmasi admin)`. |
| **BR-7.2** | The default when nothing is chosen is **`Siang (12.00)`**. |
| **BR-7.3** | Cut-offs apply **only when the selected dates include today**. For future-only dates every window is enabled. |
| **BR-7.4** | When today is included: **both Pagi windows are always disabled** — same-day morning delivery is never possible. |
| **BR-7.5** | `Siang (12.00)` is available same-day only **before 09:00**. |
| **BR-7.6** | `Sore (18.00)` is available same-day only **before 12:00**. |
| **BR-7.7** | `Request (dikonfirmasi admin)` is **never disabled** — it is the always-open escape hatch, resolved by the admin. |
| **BR-7.8** | If the customer's chosen window becomes disabled, selection **falls back to the first still-enabled window**. It is never left empty. |
| **BR-7.9** | The clock used is the **browser's local time**, not a server or Asia/Jakarta time. A customer with a wrong device clock or in another timezone gets the wrong cut-off. |

**BR-7.9 is a real defect for a production rebuild**, not a mockup quirk — see
[[09-open-questions]]. Note also that BR-7.3–7.8 can only fire on cards where
today is selectable, which BR-2.5 largely forecloses.

## BR-8 — Payment

| ID | Rule |
|----|------|
| **BR-8.1** | Payment is **manual bank transfer**. Nothing is charged online. |
| **BR-8.2** | Account: **BCA**, a.n. **R Bg Andreas Kurnianto**, No. Rek **8660-281-402**. |
| **BR-8.3** | The transfer description must read `Catering atas nama …` with the customer's name. |
| **BR-8.4** | Bank details are repeated in the outgoing WhatsApp message, not only on the page. |

## BR-9 — Schedule changes

| ID | Rule |
|----|------|
| **BR-9.1** | Moving a delivery to another date **in the same week** is **free**. |
| **BR-9.2** | Moving it to a **different week** costs **Rp 10.000 per pax**. |
| **BR-9.3** | Continuing to a full package incurs **no additional charge**. |
| **BR-9.4** | These are stated on the page as policy text. **The page does not implement them** — there is no reschedule function. They are handled by the admin over WhatsApp. |

## BR-10 — Delivery charge

| ID | Rule |
|----|------|
| **BR-10.1** | Delivery is **free** when the order is for **at least 5 days** *and* the menu is at least **Rp 26.000/day**. The Menu-tab poster words the same offer as *"free ongkir, min. order 1 minggu"*. |
| **BR-10.2** | Below either threshold — fewer than 5 days, or under Rp 26.000/day — delivery costs **Rp 5.000 per delivery**, across all categories. |
| **BR-10.3** | **Not implemented in the page.** It is displayed as policy text and never added to any total. The WhatsApp message the admin receives therefore under-states the amount due for small orders. |

## BR-11 — Delivery areas

| ID | Rule |
|----|------|
| **BR-11.1** | Nine areas, identical on every card: Gading Serpong, Karawaci, BSD, Alam Sutera, Medang, Villa Melati Mas, Park Serpong, Golden Stone, Lainnya. |
| **BR-11.2** | Area is a required per-item field. |
| **BR-11.3** | `Lainnya` is accepted with no follow-up prompt and no surcharge. **(inferred: resolved by the admin.)** |

## BR-12 — Order submission

| ID | Rule |
|----|------|
| **BR-12.1** | Submission opens a **WhatsApp deep link** to `wa.me/62818100523` with the whole order pre-filled as message text. |
| **BR-12.2** | The order is **not sent by the page**. The customer must press send inside WhatsApp. An abandoned message means a lost order, invisible to the business. |
| **BR-12.3** | If every item shares the same recipient, delivery time and notes, those appear **once** in a combined block. If any differs, **every item carries its own** recipient lines. |
| **BR-12.4** | Message lines are percent-encoded individually so newlines survive as real `%0A` line breaks. |

Related: [[04-pricing-catalogue]] · [[05-order-flow-and-whatsapp]] · [[09-open-questions]]
