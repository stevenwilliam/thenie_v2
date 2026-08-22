# 04 — Pricing catalogue

Every price in the mockup, transcribed from `site/index.html`. All figures are
**whole rupiah per pax (or per box) per day**, unless stated otherwise.

Rules that govern how these are applied: [[02-business-rules]].

---

## Daily Order — the four meal types

Five rates per meal type. Which one applies is decided by the date-shape engine
(BR-3.1 – BR-3.12), never chosen by the customer.

| Meal type | `daily` | `weeklyPerDay` | `monthlyPerDay` | `flexiWeeklyPerDay` | `flexiMonthlyPerDay` |
|-----------|--------:|---------------:|----------------:|--------------------:|---------------------:|
| **Healthy Meal** | 50.000 | 38.000 | 35.000 | 40.000 | 38.000 |
| **Bulking Extra** | 70.000 | 55.000 | 50.000 | 57.000 | 52.000 |
| **Regular Catering** | 31.000 | 26.000 | 25.000 | 27.000 | 26.000 |
| **Kids Meal** | 26.000 | 21.000 | 20.000 | 23.000 | 21.000 |

Reading the columns:

- `daily` — full price. Fewer than 5 consecutive days, random dates, or the
  BR-3.3 dead zone (15–19 consecutive days).
- `weeklyPerDay` — **Mingguan**, the best short-commitment rate. Needs 5+
  consecutive days, or a clean Mon–Fri/Mon–Sat week holding 5+ dates.
- `monthlyPerDay` — **Bulanan**, the cheapest rate. Needs 20+ consecutive days,
  or a 20+ day weekday routine inside 31 days.
- `flexiWeeklyPerDay` / `flexiMonthlyPerDay` — the middle rates for customers
  who commit to volume but not to a clean pattern.

The saving is real: a Healthy Meal customer pays 50.000/day ad-hoc and
35.000/day on Bulanan — **30% off**.

## Nasi Bento Box

Priced by quantity band. Two packages.

| Package | 20–50 | 51–100 | 101–199 | 200+ |
|---------|------:|-------:|--------:|-----:|
| **Paket Ayam** | 26.000 | 24.000 | 22.000 | 20.000 |
| **Paket Daging** | 31.000 | 29.000 | 27.000 | 25.000 |

Minimum order: **20 boxes**. Paket Daging is a flat **+5.000** over Paket Ayam
at every band.

## Nasi Kuning

One package, three bands. Minimum **10**.

| Package | 10–29 | 30–59 | 60+ |
|---------|------:|------:|----:|
| **Nasi Kuning Wow** | 39.000 | 37.000 | 35.000 |

## Paket Acara

Four packages, two pax bands each. Minimum **25 pax**.

| Package | 25–50 pax | >50 pax | Saving above 50 |
|---------|----------:|--------:|----------------:|
| **Paket A** | 40.000 | 35.000 | 5.000 |
| **Paket B** | 50.000 | 45.000 | 5.000 |
| **Paket C** | 65.000 | 60.000 | 5.000 |
| **Paket D** | 80.000 | 75.000 | 5.000 |

## Catering Kantor

A full matrix: **jenis × periode × pax tier**. Committed days per period are
**Mingguan = 5**, **Bulanan = 20** (BR-5.3).

### Reguler

| Periode | 5–10 | 11–20 | 21–50 | 51–100 | 101+ |
|---------|-----:|------:|------:|-------:|-----:|
| **Mingguan** | 24.000 | 23.000 | 22.000 | 21.000 | 20.000 |
| **Bulanan** | 23.000 | 22.000 | 21.000 | 20.000 | 19.000 |

### Healthy

| Periode | 5–10 | 11–20 | 21–50 | 51–100 | 101+ |
|---------|-----:|------:|------:|-------:|-----:|
| **Mingguan** | 34.000 | 33.000 | 32.000 | 31.000 | 30.000 |
| **Bulanan** | 33.000 | 32.000 | 31.000 | 30.000 | 29.000 |

The matrix is perfectly regular — **healthy = reguler + 10.000**, and
**bulanan = mingguan − 1.000**, at every single tier (BR-5.4, BR-5.5). A rebuild
could store one base array plus two offsets, though keeping the explicit table
is safer if the business ever breaks the pattern.

## Add-ons

Priced per pax per eligible day. `Days` uses `1`=Mon … `6`=Sat; `—` means no
weekday restriction.

### Daily Order cards

| Add-on | Price | Days | Note |
|--------|------:|:----:|------|
| Extra Ayam | 15.000 | 1,2,3,5,6 | Not Thursday |
| Extra Telur | 5.000 | — | |
| Extra Ikan (khusus Rabu) | 15.000 | 3 | Wednesday only |
| Extra Seafood (khusus Rabu) | 20.000 | 3 | Wednesday only |
| Extra Daging (khusus Kamis) | 20.000 | 4 | Thursday only |
| Extra Sayur | 5.000 | 1–6 | |
| Extra Lauk Pendamping | 5.000 | 1–6 | |
| Ganti Nasi Merah | 5.000 | 1–6 | Substitution |
| Packaging Thinwall | 2.000 | — | |
| Ganti Thinwall | 2.000 | — | |

### Other cards

| Add-on | Price | Days |
|--------|------:|:----:|
| Extra Protein Ayam | 15.000 | — |
| Extra Protein Ikan | 15.000 | — |
| Extra Protein Ikan (khusus Rabu) | 15.000 | 3 |
| Extra Protein Seafood | 20.000 | — |
| Extra Sayur | 5.000 | — |
| Extra Lauk Pendamping | 5.000 | — |
| Ganti Nasi Merah | 5.000 | — |

**Flexi cap:** on all Flexi tiers, `Extra Daging (khusus Kamis)` is limited to
one portion per each multiple of 5 pax — 12 pax buys 2 portions (BR-6.8). The
calculator does not enforce this.

**Free choice, not an add-on:** Regular Catering offers a Wednesday protein
either/or — `Rabu: Ayam` or `Rabu: Seafood (termasuk ikan)` — at no charge
(BR-6.7).

## Charges stated but never calculated

| Charge | Amount | Status |
|--------|-------:|--------|
| Delivery, order ≥5 days *and* ≥Rp 26.000/day | **free** | Policy text (BR-10.1). |
| Delivery below either threshold | 5.000 per delivery | **Displayed only.** Never added to any total (BR-10.3). |
| Reschedule to a different week | 10.000 per pax | **Displayed only.** No reschedule function exists (BR-9.4). |
| Reschedule within the same week | free | Policy text. |

Both are real money the page never charges. Any rebuild must decide whether to
implement them or keep them as admin-side adjustments — see [[09-open-questions]].

## Worked example

3 pax · Healthy Meal · Mon–Fri consecutive (5 days) · Extra Telur every day:

```
base    38.000 (Mingguan, BR-3.4) × 3 pax × 5 days = 570.000
add-on   5.000 (Extra Telur, no day limit) × 3 × 5 =  75.000
                                              total = 645.000
```

Same order as 4 random scattered dates instead:

```
base    50.000 (Flexi/Harian, BR-3.10) × 3 × 4     = 600.000
add-on   5.000 × 3 × 4                             =  60.000
                                              total = 660.000
```

Fewer meals, more money — the commitment discount is the whole point.

Related: [[02-business-rules]] · [[01-product-overview]]
