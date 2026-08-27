# 01 — Product overview

## What it is

**Thenie Healthy Catering** is a catering business in Gading Serpong,
Tangerang, Indonesia. This repository holds a byte-exact capture of its website:
a single HTML file that is both the company's marketing site and its
self-service ordering tool.

The entire site is in Indonesian. No language switch exists.

## The business, as the site tells it

| Fact | Value | Where it is stated |
|------|-------|--------------------|
| Founded | Gidae restaurant 2020 → **Thenie Restaurant** 2021 → catering 2023 | Beranda, Tentang Kami timeline |
| Base of operations | Gading Serpong | Tentang Kami, Kontak |
| Active service area | BSD · Karawaci · Alam Sutera · Medang | Tentang Kami |
| Expansion area | Bintaro · Pondok Aren · Ciledug | Tentang Kami |
| Current scale | 2,000+ portions/week (Beranda) · 2,500–3,000/week "Skala 2026" (Tentang Kami) | both pages |
| Events handled | 20+ | Beranda stats strip |
| Menu variants | 60+ | Beranda stats strip |
| Halal certification | BPJPH, No. **ID36210079132750826** | Pesan Online → Home tab |
| WhatsApp | **0818-100-523** (`62818100523`) — primary | everywhere |
| WhatsApp | 0817-771-123 (`62817771123`) — Kontak page only | Kontak |
| Email | thenie.resto@gmail.com | Kontak, footer |
| Instagram | @thenie.id | Kontak, footer |
| Bank | BCA **8660-281-402**, a/n R Bg Andreas Kurnianto | checkout, Pesan Online home |

> The two portions-per-week figures (2,000+ and 2,500–3,000) sit on different
> pages and do not agree. See Q-20 in [[09-open-questions]].

## Who it serves

The site sorts customers into three routes, and the Kontak page's quick-chat
control makes that split explicit:

- **Korporat** — staff lunches for offices, hotels, hospitals, schools. Daily
  delivery, priced per pax on a sliding scale from 5 to 200 pax.
- **Personal** — individuals and families. Daily subscriptions, kids' menus,
  diet plans.
- **Acara / Event** — buffets and special menus for celebrations, corporate
  events, and *syukuran*.

## The five product families

Everything the business sells falls into one of five families. The Harga page
prices them; the Pesan Online page sells them.

### 1. Daily subscriptions (Langganan Harian)
Four plans, ordered by picking individual dates on a calendar. Price per day
falls as the commitment grows — this is where the whole pricing engine lives
([[02-business-rules]], `BR-3.x`).

| Plan | Calories | From | Notes |
|------|----------|------|-------|
| **Healthy Meal** | 430–500 kcal | Rp35,000/day | No Sunday delivery |
| **Bulking Extra** | 700–800 kcal | Rp50,000/day | Same menu, protein doubled. No Sunday delivery |
| **Regular Catering** | — | Rp25,000/day | Has a 1–5 pax price table, and a Dengan/Tanpa Nasi toggle |
| **Kids Meal** | ±320–375 kcal | Rp20,000/day | Portions and seasoning adjusted for children |

### 2. Catering Kantor (office catering)
For 5+ pax per day. Two grades (Reguler / Healthy Catering) × two periods
(Mingguan 5 days / Bulanan 20 days), priced across five pax bands from 5–10 up
to 101–200. Free fruit every Friday.

### 3. Nasi Bento / Nasi Box
Boxed meals for meetings and gatherings. Two packages (Ayam / Daging), four
volume tiers from 20 boxes to 200+. Each box: rice, main protein, vegetable,
side dish, fruit.

### 4. Nasi Kuning ("The Wow Experience")
The signature product — a single package, three volume tiers from 10 to 60+
boxes. Contents: yellow rice, shredded chicken, sweet peanut-tempeh sambal
goreng, sliced omelette, fried noodles, potato balado sambal, cucumber.

### 5. Paket Acara (event buffets)
Four packages A–D, minimum 25 pax, two price bands (25–50 pax, >50 pax).
Customers compose the menu from a published list of options across eight
categories (Nasi, Soup, Protein A/B/C, Lauk Pendamping A/B, Sayur).

## The kitchen's stated standards

Repeated across Beranda, Menu & Layanan and the order app:

- **Tanpa santan** — no coconut milk
- **Tanpa digoreng** — nothing deep-fried
- **Tanpa tepung** — no flour coating
- **Minim minyak** — minimal oil

And the portion philosophy, stated on Beranda as "Anatomi Satu Porsi Sehat":

| Component | Portion |
|-----------|---------|
| Carbohydrate | 100 g red rice / 150 g potato |
| Animal protein | 100–150 g |
| Vegetable | minimum 100 g |
| Side dish | 50 g |
| **Total energy** | **±400–500 kcal per portion** |

## The five reasons ("Kenapa Thenie")

Free delivery · 60+ menu variants · affordable · flexible delivery windows
(morning/noon/evening) · trusted by corporates and schools.

## How an order actually happens

There is no checkout, no payment gateway, and no order database. The site's
job ends at composing a WhatsApp message:

1. Customer picks a package and dates on the Pesan Online page.
2. They fill recipient details **on the menu card itself** — so one order can
   go to several people at several addresses.
3. "+ Tambah ke Order" pushes the item into an in-memory cart.
4. "Kirim Order via WhatsApp" opens `wa.me/62818100523` with the full order
   pre-typed.
5. The customer transfers to the BCA account and confirms with the admin.

Full detail in [[05-order-flow-and-whatsapp]].

Related: [[02-business-rules]] · [[03-site-structure]] · [[04-pricing-catalogue]]
