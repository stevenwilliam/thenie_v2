# 05 — Order flow and the WhatsApp handoff

How an order is assembled, validated, and handed off. The site has no backend,
so this whole document describes what happens inside one browser tab
(`BR-1.x` in [[02-business-rules]]).

---

## The flow, end to end

```
   Pesan Online (#order)
        │
   ┌────▼──────────────────────────────────────────────┐
   │ 1. Pick a card                                    │
   │      Order tab → sub-tab → (Daily) sub-sub-tab    │
   ├───────────────────────────────────────────────────┤
   │ 2. Configure it                                   │
   │      dates · qty/pax · package · add-ons · prefs  │
   │      → subtotal recomputes on every change        │
   ├───────────────────────────────────────────────────┤
   │ 3. Fill the recipient — ON THE CARD               │
   │      name* · WhatsApp* · address* · area · notes  │
   │      · delivery window                            │
   ├───────────────────────────────────────────────────┤
   │ 4. "+ Tambah ke Order"                            │
   │      validate → push to cart → flash → RESET card │
   ├───────────────────────────────────────────────────┤
   │ 5. Repeat for more items / people / addresses     │
   ├───────────────────────────────────────────────────┤
   │ 6. "Lihat Order & Checkout" → summary sheet       │
   │      review · remove items · add more · reset     │
   ├───────────────────────────────────────────────────┤
   │ 7. "Kirim Order via WhatsApp"                     │
   │      build message → window.open(wa.me/…)         │
   └────┬──────────────────────────────────────────────┘
        │
        ▼   the site's job is over
   WhatsApp draft — the customer still has to press send
        │
        ▼
   Bank transfer to BCA 8660-281-402, then confirm with the admin
```

---

## Why recipient details live on the card

The obvious design puts one name/phone/address at checkout. This site puts
**one set per item**, and the reason is stated in the source: a customer
ordering for two people at two addresses fills each card with that person's own
details, instead of placing two separate orders.

The consequence is that **checkout collects almost nothing**. It is a review
screen: a list of what is already complete, a grand total, an optional overall
start date, the terms, and the send button.

---

## The cart

An in-memory array. Each entry:

| Field | Type | Purpose |
|-------|------|---------|
| `name` | string | Item title, e.g. `Healthy Meal (Dengan Nasi)` or `Nasi Bento — Paket Ayam` |
| `detail` | string | One-line summary shown in the sheet |
| `itemLines` | string[] | The multi-line form used in the WhatsApp message |
| `recipient` | object | `{name, phone, address, area}` |
| `dtime` | string | Delivery window |
| `note` | string | Free-text note |
| `total` | number | Item subtotal in rupiah |

There is no item ID, no quantity-merge, and no edit. Removing is by array index;
changing an item means removing it and building it again.

---

## Validation

Everything is **presence-only**. No format is ever checked (BR-12.5).

### At "+ Tambah ke Order"

| Card group | Required |
|------------|----------|
| Daily Order (4 cards) | Nama Penerima · No. WhatsApp · Alamat. Dates are enforced by disabling the button. |
| Nasi Bento / Nasi Kuning / Paket Acara | The three above **plus** at least one delivery date. |
| Catering Kantor | The three above **plus** Tanggal Mulai. |

Failure writes `Lengkapi dulu: <fields>.` into the card's own hint line. Nothing
is focused, nothing scrolls, no field is marked. On a long card the message can
appear off-screen.

### At checkout

Only that the cart is non-empty. The send button carries `disabled` until then.

### What is never validated

- Phone number format — `abc` is accepted as a WhatsApp number.
- Any length cap on names, addresses or notes.
- Any normalisation — `0818…`, `+62818…` and `62818…` all pass through as typed.
- Any sanitisation. Text is `encodeURIComponent`-escaped on the way into the
  URL, which makes it safe *for the URL*, but it is also interpolated into the
  summary sheet's `innerHTML` **unescaped** — see [[08-technical-inventory]].

---

## Reset behaviour

| Trigger | Effect |
|---------|--------|
| Successful add | That one card resets to blank — dates, qty, add-ons, prefs, recipient, delivery window back to Siang (12.00) |
| **↺ Reset / Ulang Order dari Awal** | `confirm()`, then empties the cart **and** resets every one of the eight cards |
| Reload / close tab | Everything is lost, silently |

The reset button always confirms, even with an empty cart — because a card may
hold picks that never reached the cart (BR-12.8).

---

## The WhatsApp message

### Construction

The message is built as an **array of plain-text lines**, then each line is
`encodeURIComponent`-ed individually and joined with `%0A`. That is deliberate:
encoding the whole string at once would escape the newlines too and produce one
run-on line.

Lines carrying an embedded `\n` — the "Tanggal" line for scattered dates — are
split before being pushed, so they render as two lines.

```js
const msg = lines.map(l => encodeURIComponent(l)).join('%0A');
window.open(`https://wa.me/62818100523?text=${msg}`, '_blank');
```

### The `sameRecipient` optimisation

Before building, the code checks whether **every** item shares the same
recipient name, phone, address and area, *and* the same delivery time, *and*
the same note.

- **All identical** → the recipient block is printed **once**, near the bottom.
- **Any difference** → each item carries its own Penerima / Alamat / Area /
  No. HP / Waktu Pengantaran / Catatan block.

### Structure

```
Halo Thenie, saya mau order:

1. <item name>
<itemLines…>
Paket: <tier label>
<n> hari (berturutan|acak) · <pax> pax · Rp <rate>/hari/pax
Tanggal: <period>
Tambahan:
- <add-on> (+Rp x/pax/hari × n hari) (Senin, Kamis)
Preferensi: <prefs>
                                    ← only when recipients differ:
Penerima: <name>
Alamat: <address>
Area: <area>
No. HP: <phone>
Waktu Pengantaran: <slot>
*Catatan: <note>*

*Subtotal: Rp <n>*

2. <next item…>

*Total: Rp <n>*
                                    ← only when every recipient matches:
Nama: <name>
Alamat: <address>
Area: <area>
No. HP: <phone>
Tanggal mulai/kirim: <YYYY-MM-DD>
Waktu Pengantaran: <slot>
*Catatan: <note>*

Pembayaran:
BCA a.n. R Bg Andreas Kurnianto
No. Rek: 8660-281-402

*Silahkan lakukan pembayaran sesuai dengan total yang disebutkan di atas.*
*Mohon dapat dibantu untuk diberikan keterangan pada saat transfer "Catering atas nama ..." agar mempermudah pengecekan. Thanks*
```

`*asterisks*` are WhatsApp bold markers. Subtotals, the total, and notes are
bolded; the payment instructions are bolded in full.

### The optional start date

`#cust-date` at checkout is the one field the summary sheet still collects. It
is optional, and it is **never used in pricing**. It appears as
`Tanggal mulai/kirim:` in the shared-recipient block, or as
`Tanggal mulai/kirim (umum):` when recipients differ. Note it is emitted as the
raw `YYYY-MM-DD` input value, not formatted like every other date in the
message.

---

## Delivery windows

| Option | Value written to the message |
|--------|------------------------------|
| Pagi 06.00–07.00 | `Pagi (06.00–07.00)` |
| Pagi 07.00–09.00 | `Pagi (07.00–09.00)` |
| **Siang (12.00)** — default | `Siang (12.00)` |
| Sore (18.00) | `Sore (18.00)` |
| Request (dikonfirmasi) | `Request (dikonfirmasi admin)` |

Same-day availability is governed by BR-7.4. The area restrictions on the two
Pagi windows are printed under the radios and never enforced (BR-10.2).

---

## The other WhatsApp links

The marketing pages have their own, entirely separate, WhatsApp path — a small
IIFE with four canned messages:

| `data-msg` | Message |
|------------|---------|
| `home` | Halo Thenie, saya ingin tanya-tanya soal katering. |
| `korporat` | Halo Thenie, saya ingin tanya soal katering korporat untuk kantor kami. |
| `personal` | Halo Thenie, saya ingin tanya soal langganan katering harian untuk personal/keluarga. |
| `event` | Halo Thenie, saya ingin tanya soal katering untuk acara/event. |

Any `.wa-link` element gets its `href` rewritten on load from its `data-msg`.
The Kontak page's segmented control switches between `korporat`, `personal` and
`event`, previewing the text before sending. `korporat` is preselected.

Both the footer and Kontak page also carry **plain, un-prefilled**
`https://wa.me/…` links, including the only appearance of the second number,
`62817771123`.

---

## What the flow does not do

| Missing | Consequence |
|---------|-------------|
| No persistence | A reload destroys a 20-date order with no warning |
| No submission | The business sees nothing unless the customer presses send in WhatsApp |
| No confirmation | No order number, no receipt, no email |
| No availability check | Any date, any quantity, no capacity limit |
| No payment integration | The transfer is manual and unverified |
| No edit | Changing an item means deleting and rebuilding it |
| No message length guard | A large multi-item order can produce a very long `wa.me` URL |

Raised in [[09-open-questions]].

Related: [[02-business-rules]] · [[03-site-structure]] · [[08-technical-inventory]]
