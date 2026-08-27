# 03 — Site structure

One HTML file. Six client-side pages. Inside the sixth, a complete order
application with three further levels of tabs.

```
index.html
│
├── <nav> floating pill (fixed, bottom-centre) ──▶ 6 page links + "Pesan Online" CTA
│
├── <main id="main">
│   ├── #home     Beranda            ← active on load
│   ├── #about    Tentang Kami
│   ├── #menu     Menu & Layanan
│   ├── #pricing  Harga
│   ├── #contact  Kontak
│   └── #order    Pesan Online  ── the order application ──┐
│                                                          │
├── <footer>                                               │
└── .fab-wa  floating WhatsApp button (fixed, bottom-right) │
                                                            │
   ┌────────────────────────────────────────────────────────┘
   │  #order
   ├── page hero
   ├── #tabs ─────────── 🏠 Home │ 📖 Menu │ 🛒 Order      (level 1)
   ├── .order-app-main
   │   ├── #panel-home   how to order, bank details, shortcuts
   │   ├── #panel-menu   photo menus            (level 2: 5 sub-tabs)
   │   └── #panel-order  the eight order cards  (level 2: 5 sub-tabs)
   │       └── #subpanel-daily                  (level 3: 4 sub-sub-tabs)
   ├── .cart-bar         fixed bottom, count + total + checkout
   └── #modal-overlay    the order summary sheet
```

---

## The router

| Aspect | Behaviour |
|--------|-----------|
| Mechanism | `location.hash` → `showPage(id)`; listens on `hashchange` |
| Page switching | Toggles `.active` on `.page` elements. `.page{display:none}` / `.page.active{display:block}` |
| Unknown hash | Falls back to `#home` |
| No hash | Falls back to `#home` |
| Document title | Rewritten per page from a lookup table |
| Active link | `aria-current="page"` on the matching `[data-nav]` |
| Scroll | `window.scrollTo({top:0})` — `smooth`, or `auto` under `prefers-reduced-motion` |
| Side effect | Entering `#pricing` re-measures the tab highlight on the next frame |

Because routing is on the **fragment**, the server never sees a path other than
`/`. No rewrite rules are needed — see [[13-production-deployment-runbook]].

### Page titles

| Hash | `document.title` |
|------|------------------|
| `#home` | Thenie Healthy Catering |
| `#about` | Tentang Kami — Thenie |
| `#menu` | Menu & Layanan — Thenie |
| `#pricing` | Harga — Thenie |
| `#order` | Pesan Online — Thenie |
| `#contact` | Kontak — Thenie |

---

## Navigation

There is **no top header**. All navigation is a fixed pill at the bottom centre:

- **`#navToggle`** — a hamburger that opens `#navlinks`, a vertical panel of the
  six page links. `aria-expanded` is maintained.
- **`.nav-cta`** — a permanent "Pesan Online" button straight to `#order`.
- The pill lifts from `bottom:20px` to `bottom:96px` while `#order` is active,
  using `body:has(#order.page.active)`, to clear the cart bar.
- The menu closes automatically on every page change.

The footer repeats five of the six links (it omits Pesan Online).

---

## The five content pages

### `#home` — Beranda

| Section | Content |
|---------|---------|
| Hero | Full-viewport photo, logo, "Katering Sehat, Rasa *Restoran.*" |
| Stats strip | 6 Thn · 2.000+ porsi/minggu · 20+ event · 60+ varian — animated counters |
| Perjalanan Kami | Origin story, link to `#about` |
| Kenapa Thenie | Five pillar cards with icons |
| Filosofi Kuliner | Dark "Anatomi Satu Porsi Sehat" panel — macro list + four badges |
| Testimoni | Three customer quotes beside a photo |
| CTA band | "Pesan Sekarang" → `#order`, "Chat WhatsApp" → `wa.me` |

The counters use an `IntersectionObserver` at `threshold: 0.4`, run once, and
ease with `1 − (1−p)³` over 1,100 ms. Under `prefers-reduced-motion` they are
written straight to their final value and the observer is never created.

### `#about` — Tentang Kami

Page hero · four-step timeline (2020, 2021, 2023, 2024–2026) · Visi / Misi /
Komitmen cards · four service-area cards.

### `#menu` — Menu & Layanan

Page hero · three service-line photo cards (Korporat, Personal, Acara) · two
subscription plan cards (Healthy Meal, Bulking Extra) with spec grids · two
Nasi Box photo cards · the Nasi Kuning signature section · the event-buffet
section. Cross-links to `#pricing` throughout.

### `#pricing` — Harga

Page hero, then a **tab bar with a sliding highlight** over four panels:

| Tab | Panel | Contents |
|-----|-------|----------|
| Langganan Harian | `#panel-sub` | 4-plan × 3-period table, terms, and the Paket Kantor multi-pax tables (Reguler + Healthy, Mingguan + Bulanan) |
| Nasi Box | `#panel-box` | Ayam and Daging tier tables |
| Nasi Kuning | `#panel-kuning` | Three-tier table plus contents |
| Buffet Korporat | `#panel-buffet` | A–D price table, composition matrix, and the full Menu Pilihan lists |

The highlight is a positioned `<div>` moved by measuring the active button's
`offsetLeft`/`offsetTop`/`offsetWidth`/`offsetHeight`. It is re-measured on
`resize`, on page entry, on `load`, and on `document.fonts.ready` — because the
web font changes button widths after first paint.

### `#contact` — Kontak

A **quick-chat segmented control** (Korporat / Personal / Acara-Event) that
previews the message text and rewrites the WhatsApp link; four contact cards
(email, two phone numbers, Instagram, service area); and a closing CTA band.

---

## `#order` — the order application

This is the whole of the previous project, carried over. Three tab levels.

### Level 1 — `#tabs`

| Tab | Panel | Purpose |
|-----|-------|---------|
| 🏠 Home | `#panel-home` | Halal certificate, six-step how-to-order, bank details and reschedule terms, five shortcut buttons |
| 📖 Menu | `#panel-menu` | Photo menus only — no ordering |
| 🛒 Order | `#panel-order` | The eight order cards |

Every tab click scrolls the window to the top.

### Level 2 — Menu sub-tabs (`#menu-subtabs`)

🥗 Daily Order · 🍱 Nasi Bento Box · 🏢 Catering Kantor · 🟡 Nasi Kuning ·
🍽️ Paket Acara

Catering Kantor holds **no images of its own** — it carries a button that
programmatically clicks the Daily Order sub-tab, because the menus are the same.

### Level 2 — Order sub-tabs (`#order-subtabs`)

🥗 Daily Order · 🏢 Catering Kantor · 🍱 Nasi Bento · 🟡 Nasi Kuning ·
🍽️ Paket Acara

### Level 3 — Daily Order sub-tabs (`#daily-subtabs`)

Healthy · Bulking · Reguler Dewasa · Kids Meal

### Cross-tab shortcuts

`.goto-order-btn` carries `data-goto-subtab` and optionally
`data-goto-dailytab`. It works by **synthesising clicks** on the real tab
buttons in order — level 1, then 2, then 3 — so all the normal tab side effects
fire. `.goto-menu-subtab-btn` does the same one level down.

---

## The eight order cards

| # | Card | `data-sub` | Engine | Minimum |
|---|------|-----------|--------|---------|
| 1 | Healthy Meal | `Healthy Meal` | `analyze()` + `data-rates` | 1 pax |
| 2 | Bulking Extra | `Bulking Extra` | `analyze()` + `data-rates` | 1 pax |
| 3 | Regular Catering | `Regular Catering` | `analyze()` + `data-rates` + `data-pax-table` | 1 pax |
| 4 | Kids Meal | `Kids Meal` | `analyze()` + `data-rates` | 1 pax |
| 5 | Nasi Bento | `Nasi Bento` | `data-plans-select` tiers | 20 boxes |
| 6 | Nasi Kuning Wow | `Nasi Kuning Wow` | `data-plans-select` tiers | 10 boxes |
| 7 | Paket Acara | `Paket Acara` | `data-plans-select` tiers | 25 pax |
| 8 | Catering Kantor | `#kantor-card` | its own IIFE, `RATES` constant | 5 pax |

### Which widgets each card gets

| Widget | 1–4 | 5–7 | 8 |
|--------|:---:|:---:|:-:|
| Inline month calendar | ✅ multi-date | ✅ multi-date | ✅ single date |
| Quick-fill buttons (5/6/20 hari) | ✅ | — | — |
| Isi Rentang (from–to) | ✅ *(hidden on 1 & 2)* | — | — |
| Date chips with × | ✅ | ✅ | — |
| Package badge / tier read-out | ✅ | ✅ table | ✅ table |
| Weekly menu `<details>` | ✅ | — | — |
| Preferensi Menu | ✅ | — | — |
| Add-ons | ✅ *with per-day chips* | Bento only | ✅ |
| Dengan/Tanpa Nasi toggle | Regular only | — | — |
| Recipient fields | ✅ | ✅ | ✅ |
| Waktu Pengantaran radios | ✅ | ✅ | ✅ |

All three date widgets share the same `.mini-calendar` markup and the same
disabled-date rules (BR-7.2, BR-7.3).

---

## Cart and checkout

- **`.cart-bar`** — fixed to the bottom of the viewport, `z-index:150`. Item
  count, running total, and "Lihat Order & Checkout". It lives inside `#order`,
  so it disappears with the page.
- **`#modal-overlay`** — `z-index:300`, bottom-anchored sheet. Item list with
  per-item "Hapus", grand total, "+ Tambah Menu Lain", "↺ Reset / Ulang Order
  dari Awal", an optional start-date field, a collapsible terms block, and
  "📲 Kirim Order via WhatsApp".
- Closes on the × text, on "Tambah Menu Lain", and on a click on the backdrop
  itself. **Not** on `Escape`.

Full behaviour in [[05-order-flow-and-whatsapp]].

---

## Stacking order

| Layer | `z-index` |
|-------|-----------|
| `#modal-overlay` | 300 |
| `.float-nav` | 260 |
| `#navlinks` | 259 |
| `.fab-wa` | 200 |
| `.cart-bar` | 150 |
| in-page decoration | ≤ 20 |

The stack is correct; the **positions** collide. The WhatsApp button sits inside
the cart bar's band on the order page, and inside the pill nav's on narrow
screens. That is what the one overlay in this repo fixes — see [[14-overlays]].

Related: [[01-product-overview]] · [[05-order-flow-and-whatsapp]] · [[06-design-system]]
