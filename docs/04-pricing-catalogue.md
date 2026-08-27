# 04 — Pricing catalogue

Every price on the site, transcribed in full. All figures are Indonesian Rupiah.

Two sources are recorded, because they are not always the same thing:

- **The calculator** — the `data-rates`, `data-pax-table`, `data-plans-select`
  and `RATES` values the order app actually computes with.
- **The Harga page** — the published tables a customer reads.

Where they differ it is called out explicitly.

---

## 1. Daily subscriptions — the calculator rates

Read straight out of each card's `data-rates`. Every value is **per pax per
day**. Which one applies is decided by `analyze()` — see `BR-3.x` in
[[02-business-rules]].

| Plan | `daily` | `weeklyPerDay` | `monthlyPerDay` | `flexiWeeklyPerDay` | `flexiMonthlyPerDay` |
|------|--------:|---------------:|----------------:|--------------------:|---------------------:|
| **Healthy Meal** | 50,000 | 38,000 | 35,000 | 40,000 | 38,000 |
| **Bulking Extra** | 70,000 | 55,000 | 50,000 | 57,000 | 52,000 |
| **Regular Catering** | 31,000 | 26,000 | 25,000 | 27,000 | 26,000 |
| **Kids Meal** | 26,000 | 21,000 | 20,000 | 23,000 | 21,000 |

Two invariants hold across all four plans, and the test suite asserts them:
`monthlyPerDay ≤ weeklyPerDay ≤ daily`.

Note that `flexiMonthlyPerDay` **equals `weeklyPerDay` exactly** for Healthy
Meal, Regular Catering and Kids Meal — a scattered 20-day order gets the same
per-day price as a clean 5-day week. Bulking Extra is the only plan where it
sits between the two (52,000, against 55,000 weekly and 50,000 monthly). And
Flexi Bulanan is never cheaper than Bulanan on any plan.

### The same rates as a discount off list

| Plan | Mingguan | Bulanan | Flexi Mingguan | Flexi Bulanan |
|------|---------:|--------:|---------------:|--------------:|
| Healthy Meal | −24% | −30% | −20% | −24% |
| Bulking Extra | −21% | −29% | −19% | −26% |
| Regular Catering | −16% | −19% | −13% | −16% |
| Kids Meal | −19% | −23% | −12% | −19% |

---

## 2. Daily subscriptions — the published table

From the Harga page, tab **Langganan Harian**:

| Paket | Harian | Mingguan | Bulanan |
|-------|-------:|---------:|--------:|
| Regular Catering (1 Pax) | 31,000 | 26,000/hari | 25,000/hari |
| **Healthy Meal (430–500 kkal)** | 50,000 | **190,000** (@38,000, min 5 hari) | **700,000** (@35,000, min 20 hari) |
| Bulking Extra (700–800 kkal) | 70,000 | **275,000** (@55,000, min 5 hari) | **1,000,000** (@50,000, min 20 hari) |
| Kids Meal | 26,000 | 21,000/hari | 20,000/hari |

The bundle prices are exactly consistent with the calculator:
5 × 38,000 = 190,000 · 20 × 35,000 = 700,000 ·
5 × 55,000 = 275,000 · 20 × 50,000 = 1,000,000.

> The published table shows only Harian / Mingguan / Bulanan. **The two Flexi
> tiers are never published** — a customer meets them for the first time in the
> order form's package badge. See Q-22 in [[09-open-questions]].

Terms printed with the table:

- 🚚 Free delivery for orders of at least 1 week (5 days).
- Changes of time, date or address up to **H-1**.
- Orders cannot be cancelled on the day.
- Off-menu requests, or a switch to red rice, cost **+Rp5,000/pax**.

---

## 3. Regular Catering — the 1–5 pax table

Only Regular Catering carries this. It replaces the flat per-pax rate on the
**Mingguan** and **Bulanan** tiers only (BR-5.1). Values are the **per-day total
for the whole group**.

### Dengan Nasi (default)

| Pax | Mingguan | per pax | Bulanan | per pax |
|----:|---------:|--------:|--------:|--------:|
| 1 | 26,000 | 26,000 | 25,000 | 25,000 |
| 2 | 52,000 | 26,000 | 50,000 | 25,000 |
| 3 | 76,000 | 25,333 | 74,000 | 24,667 |
| 4 | 98,000 | 24,500 | 95,000 | 23,750 |
| 5 | 118,000 | 23,600 | 113,000 | 22,600 |

### Tanpa Nasi

| Pax | Mingguan | per pax | Bulanan | per pax |
|----:|---------:|--------:|--------:|--------:|
| 1 | 26,000 | 26,000 | 25,000 | 25,000 |
| 2 | 52,000 | 26,000 | 50,000 | 25,000 |
| 3 | 73,000 | 24,333 | 71,000 | 23,667 |
| 4 | 92,000 | 23,000 | 90,000 | 22,500 |
| 5 | 111,000 | 22,200 | 108,000 | 21,600 |

**Tanpa Nasi costs the same as Dengan Nasi at 1 and 2 pax**, and only diverges
from 3 pax up. Removing rice saves nothing for one or two people.

### Above 5 pax

`dayTotal = round(table[5] / 5) × pax` (BR-5.4). The per-pax rate freezes at the
5-pax level:

| Variant | Period | Frozen per-pax rate |
|---------|--------|--------------------:|
| Dengan Nasi | Mingguan | 23,600 |
| Dengan Nasi | Bulanan | 22,600 |
| Tanpa Nasi | Mingguan | 22,200 |
| Tanpa Nasi | Bulanan | 21,600 |

The card's own copy points 6+ pax customers at Catering Kantor instead.

---

## 4. Catering Kantor

From the `RATES` constant in the Kantor IIFE — and identical to the Harga page's
"Paket Kantor (Multi-Pax)" tables. Per pax per day.

### Paket Reguler

| Jumlah Pax | Mingguan | Bulanan |
|------------|---------:|--------:|
| 5–10 | 24,000 | 23,000 |
| 11–20 | 23,000 | 22,000 |
| 21–50 | 22,000 | 21,000 |
| 51–100 | 21,000 | 20,000 |
| **101–200** | **20,000** | **19,000** |

### Healthy Catering — exactly +Rp10,000 on every cell

| Jumlah Pax | Mingguan | Bulanan |
|------------|---------:|--------:|
| 5–10 | 34,000 | 33,000 |
| 11–20 | 33,000 | 32,000 |
| 21–50 | 32,000 | 31,000 |
| 51–100 | 31,000 | 30,000 |
| **101–200** | **30,000** | **29,000** |

Period lengths: **Mingguan = 5 days**, **Bulanan = 20 days** (BR-13.3). Those 20
days are 20 *weekdays* — the date walk skips weekends (BR-13.7).

Free cut fruit every Friday.

> The tables stop at 200 pax, but the code's top band is unbounded
> (`TIER_MAXS` ends at `99999`). A 500-pax order silently prices at the
> 101–200 rate.

---

## 5. Nasi Bento / Nasi Box

From `data-plans-select`, and matching the Harga page's Nasi Box tab. Per box.
Minimum 20 boxes.

| Jumlah Order | 🐔 Paket Ayam | 🥩 Paket Daging |
|--------------|-------------:|---------------:|
| 20–50 box | 26,000 | 31,000 |
| 51–100 box | 24,000 | 29,000 |
| 101–199 box | 22,000 | 27,000 |
| **200+ box** | **20,000** | **25,000** |

Daging is **+Rp5,000** on every tier. Each box contains rice, main protein,
vegetable, side dish and fruit.

The Menu & Layanan page advertises "Mulai Rp20.000/box" and "Mulai
Rp25.000/box" — the 200+ tier, correctly labelled as such.

---

## 6. Nasi Kuning Wow

Per box. Minimum 10 boxes. One package only.

| Jumlah Order | Harga/Box |
|--------------|----------:|
| 10–29 box | 39,000 |
| 30–59 box | 37,000 |
| **60+ box** | **35,000** |

Contents: nasi kuning · ayam suwir · sambal goreng tempe kacang manis ·
telur dadar iris · mie goreng · sambal balado kentang · timun.

---

## 7. Paket Acara / Buffet Korporat

Per pax. Minimum 25 pax. Two bands.

| Paket | 25–50 Pax | >50 Pax |
|-------|----------:|--------:|
| Paket A | 40,000 | 35,000 |
| Paket B | 50,000 | 45,000 |
| Paket C | 65,000 | 60,000 |
| **Paket D** | **80,000** | **75,000** |

### What each package contains

| Paket | Nasi | Protein A | Protein B | Protein C | Soup | Lauk A | Lauk B | Buah |
|-------|:----:|:---------:|:---------:|:---------:|:----:|:------:|:------:|:----:|
| A | 1 | 1 | – | – | 1 | 1 | 1 | Potong |
| B | 1 | 1 | – | – | 1 | 2 | 1 | Potong |
| C | 2* | 1 | 1 | – | 1 | 2 | 1 | Potong |
| D | 2* | 1 | 1 | 1 | 1 | 2 | 1 | Potong |

\* Paket C and D may mix two rice menus.

> The order card's `includes` list and the Harga page's matrix agree, with one
> wording difference: the card lists "Lauk Pendamping A / B" where the matrix
> column headers read "Lauk A / Lauk B", and the matrix has a separate **Sayur**
> category that the card's `includes` never mentions. The Menu Pilihan list
> below does publish Sayur options.

### Menu Pilihan — the full published list

| Category | Options |
|----------|---------|
| **Nasi** | Nasi Putih · Nasi Goreng |
| **Soup** | Soup Bakso · Soup Jagung · Soup Batam Jagung · Soup Sayur |
| **Protein A — Ayam** | Ayam Goreng Lengkuas · Ayam Goreng Mentega · Ayam Asam Manis · Ayam Lada Garam · Ayam Tumis Cabe Hijau · Ayam Goreng Sambal Matah · Ayam Kung Pao · Ayam Kareaage |
| **Protein B — Dori** | Dori Asam Manis · Dori Lada Garam · Dori Goreng Sambal Matah |
| **Protein C — Daging Sapi** | Semur Daging · Daging Rendang · Daging Sapi Rica · Sop Daging Sapi · Sapi Lada Hitam |
| **Lauk Pendamping A** | Mun Tahu · Tahu Lada Garam · Tempe Bacem · Tempe Orek · Bakwan Jagung · Bakwan Sayur · Orek Tahu Pedas · Telur Balado · Martabak Mini · Perkedel Kentang |
| **Sayur** | Capcay · Kangkung Cah Bawang Putih · Tumis Buncis Bawang Putih · Caisim Cah Bawang Putih · Tumis Jagung Muda Wortel · Tumis Labu Siam |
| **Lauk Pendamping B** | Bakmi Goreng · Bihun Goreng · Soun Goreng |

39 dishes in total. **None of them is selectable in the order form** — the card
tells the customer to discuss the menu with the admin in the notes (BR-14.6).

---

## 8. Add-ons

See BR-6 in [[02-business-rules]] for the day restrictions and the Flexi meat
cap. Prices, per pax per day:

| Add-on | Price |
|--------|------:|
| Extra Seafood / Extra Protein Seafood | 20,000 |
| Extra Daging (khusus Kamis) | 20,000 |
| Extra Ayam / Extra Protein Ayam | 15,000 |
| Extra Ikan / Extra Protein Ikan | 15,000 |
| Extra Telur | 5,000 |
| Extra Sayur | 5,000 |
| Extra Lauk Pendamping | 5,000 |
| Ganti Nasi Merah | 5,000 |
| Packaging ganti Thinwall | 2,000 |

---

## 9. Fees the page states but never charges

| Fee | Amount | Rule | Charged? |
|-----|--------|------|:--------:|
| Delivery, below the free threshold | +5,000 per delivery | BR-10.3 | ❌ |
| Reschedule across weeks | +10,000 per pax | BR-9.4 | ❌ |
| Off-menu request | +5,000 per pax | BR-9.6 | ❌ |
| Red rice, outside the add-on | +5,000 per pax | BR-9.6 | ❌ |

The last two are reachable as a **paid add-on** (Ganti Nasi Merah, +5,000) —
so a customer who uses the checkbox is charged, and a customer who types the
same request into "Catatan lain" is not.

The WhatsApp message therefore under-states the amount owed on any order that
triggers one of these. The admin corrects it by hand. See Q-5 in
[[09-open-questions]].

---

## 10. Price consistency check

Every published figure was compared against the calculator. Results:

| Source | Consistent? |
|--------|-------------|
| Harga → Langganan Harian vs `data-rates` | ✅ exact |
| Harga → Paket Kantor vs `RATES` | ✅ exact |
| Harga → Nasi Box vs `data-plans-select` | ✅ exact |
| Harga → Nasi Kuning vs `data-plans-select` | ✅ exact |
| Harga → Buffet A–D vs `data-plans-select` | ✅ exact |
| Menu & Layanan "Mulai 35K / 50K" vs monthly rates | ✅ exact |
| Menu & Layanan "Mulai Rp20.000 / Rp25.000 per box" vs 200+ tier | ✅ exact |
| Menu poster image "Hanya 35K / porsi / hari" | ✅ matches `monthlyPerDay` |
| Flexi tiers | ⚠️ **not published anywhere** |

No price contradiction was found between the marketing pages and the order
engine. The only divergences are the Flexi tiers (unpublished) and the
uncharged fees in section 9.

Related: [[02-business-rules]] · [[05-order-flow-and-whatsapp]] · [[09-open-questions]]
