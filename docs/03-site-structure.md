# 03 — Site structure

One HTML document, three top-level tabs, no routing. Switching tabs toggles a
CSS class — the URL never changes, there is no history entry, and the back
button leaves the site. See [[08-technical-inventory]].

```
Thenie Healthy Catering (single page)
│
├── 🏠 Home    #panel-home
├── 📖 Menu    #panel-menu
└── 🛒 Order   #panel-order
```

## Header

Fixed brand block on a maroon gradient: circular logo badge, title, italic
tagline, sub-line. Present on every tab.

## 🏠 Home

Marketing and policy, with jump buttons into the Order tab. Sections:

| Section | Content |
|---------|---------|
| **📋 Cara Order** | How to order, step by step |
| **🏦 Rekening & Ketentuan Perubahan Jadwal** | Bank details (BR-8.2) and the reschedule policy (BR-9) |
| **Mau order apa?** | Five CTA buttons, one per product family |

Further CTAs deep-link to a specific card: *Lihat & Order Healthy / Bulking*,
*… Reguler Dewasa*, *… Kids Meal*, *… Nasi Bento*, *… Nasi Kuning*,
*… Paket Acara*, plus *🛒 Order Catering Kantor*.

These use `data-goto-subtab` / `data-goto-dailytab` to switch tab, sub-tab and
daily-tab in one press.

## 📖 Menu

Five sub-tabs (`data-menusubtab`): `daily`, `kantor`, `bento`, `nasikuning`,
`acara`.

### Daily menu rotation

The Daily sub-tab splits again into **Healthy · Bulking · Reguler Dewasa ·
Kids Meal**, each showing **two calendar weeks, Monday–Friday**:

- *Minggu ke-34, 17–21 Agustus 2026* — this week
- *Minggu ke-35, 24–28 Agustus 2026* — next week

Each day lists the full composition with gram weights and an approximate
calorie count, e.g.

> **Senin 24 Agu** — Nasi Merah (100g), Ayam Goreng Serundeng (90g), Tumis Sawi
> Putih (100g), Sup Jagung (50g), Sambal Merah (10g) · **±455 kkal**

**Thursday is marked ⭐** in both weeks and is the beef day (Rawon Daging Sapi,
Semur Daging Sapi). This lines up with the Thursday-only `Extra Daging` add-on
(BR-6.3).

Calorie range observed: **±450 – ±685 kkal**. Healthy Meal is stated as
430–500 kcal with a fixed macro split (Karbo 100–120g, Protein 100g, Sayur 100g,
Side 50g, Sambal 10g).

**Bulking Extra shares Healthy Meal's menu and photos exactly** — the page says
so outright; only the protein portion is larger. A rebuild should model Bulking
as a portion variant of Healthy, not a separate menu.

Dietary claim shown site-wide: **tanpa santan, tanpa gorengan, tanpa tepung,
minim minyak** (no coconut milk, nothing deep-fried, no flour, minimal oil).

### Photo cards

Eleven `menu-photo-card` blocks carry the food photography. All images are
embedded as base64 data URIs — see [[08-technical-inventory]].

## 🛒 Order

Five sub-tabs (`data-subtab`), and the Daily sub-tab splits into four daily-tabs
(`data-dailytab`) — giving the **eight order cards**:

| Sub-tab | Card | Card key | Pricing model |
|---------|------|----------|---------------|
| 🥗 Daily Order | Healthy Meal | `healthy` | Rate tiers (BR-3) |
| 🥗 Daily Order | Bulking Extra | `bulking` | Rate tiers (BR-3) |
| 🥗 Daily Order | Regular Catering | `regular` | Rate tiers (BR-3) |
| 🥗 Daily Order | Kids Meal | `kids` | Rate tiers (BR-3) |
| 🍱 Nasi Bento Box | Paket Ayam / Daging | `nasibox` | Quantity bands (BR-4) |
| 🏢 Catering Kantor | — | `kantor` | Pax × period matrix (BR-5) |
| 🟡 Nasi Kuning | Nasi Kuning Wow | `nasikuning` | Quantity bands (BR-4) |
| 🍽️ Paket Acara | Paket A–D | `acara` | Pax bands (BR-4) |

Card keys are the `dtime-*` radio group names — the reliable identifier in the
source.

### Anatomy of an order card

Every card carries the same controls, in this order:

1. **Package / type selector** — where the product has variants
2. **Quantity** — `−` / `+` stepper, clamped to the tier minimum (BR-2.2)
3. **Mini-calendar** — multi-date picker with `‹` `›` month navigation, plus
   *quick-fill week* and *skip-days* helpers; today and past dates are
   unselectable (BR-2.5)
4. **Date chips** — removable, with gap-break marks for non-consecutive runs
5. **Tier table** — live, highlighting the band the current quantity lands in
6. **Add-ons** — checkboxes with per-add-on day pickers (BR-6.5)
7. **Waktu Pengantaran** — five windows, cut-off-aware (BR-7)
8. **Recipient block** — Nama, No. HP, Alamat, Area, Catatan (BR-1.2)
9. **Subtotal** — live
10. **Tambah ke Order** — validates, then adds to cart

### Catering Kantor differs

It is the only card with an `id` (`kantor-card`) and the only one driven by a
pax × period matrix rather than a date-shape engine. It exposes *Pilih Jenis &
Periode*, a pax input, and a tier table, with committed days fixed by period
(BR-5.3).

## Cart and checkout

A sticky bottom bar shows item count and running total, and opens a modal.

- **Modal** — lists every item with its recipient, dates, add-ons and subtotal
- **Hapus** — deletes an item (BR-1.6)
- **Add more** — closes the modal, cart intact (BR-1.6)
- **Send** — builds the WhatsApp message (BR-12) — disabled while the cart is
  empty (BR-1.5)

Checkout collects **no new fields** (BR-1.4).

Related: [[02-business-rules]] · [[05-order-flow-and-whatsapp]] · [[06-design-system]]
