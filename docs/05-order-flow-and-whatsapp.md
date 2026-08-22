# 05 — Order flow and the WhatsApp handoff

## The flow end to end

```
Home  ─CTA─▶  Order tab ─▶ sub-tab ─▶ card
                              │
                              ├─ pick package / type
                              ├─ set quantity        (clamped to tier minimum)
                              ├─ pick date(s)        (calendar → chips)
                              ├─ tick add-ons        (+ per-add-on day chips)
                              ├─ pick delivery time  (cut-off aware)
                              ├─ fill recipient      (name, phone, address, area, note)
                              └─ Tambah ke Order ──▶ VALIDATE ──▶ cart
                                                        │
                                     ┌──────────────────┘
                                     ▼
                          sticky bar (count + total)
                                     │  open
                                     ▼
                             checkout modal  ── summary only, no new fields
                                     │  send
                                     ▼
                       wa.me/62818100523?text=<encoded order>
                                     │
                                     ▼
                       WhatsApp opens — CUSTOMER MUST PRESS SEND
```

## Where validation happens

Validation is **front-loaded onto the card**, not the checkout:

| Stage | Checks |
|-------|--------|
| **Tambah ke Order** | Nama, No. HP, Alamat present; at least one date picked; quantity ≥ tier minimum |
| **Checkout** | Cart is not empty — and nothing else (BR-1.5) |

By the time an item reaches the cart its recipient data is complete, so checkout
has nothing left to verify. The send button stays disabled with the hint
*"Lengkapi dulu: minimal 1 item menu."* while the cart is empty.

**There is no format validation.** No phone-number pattern, no length limits, no
character restrictions, no address sanity check. A single space passes as a
name. This is the mockup's weakest point for a rebuild — see
[[09-open-questions]].

## The item record

Each cart item holds:

| Field | Meaning |
|-------|---------|
| `sub` | Card / product family (`card.dataset.sub`) |
| `itemName` | Package or meal type |
| `qty` | Boxes or pax |
| `unitPrice` | The resolved tier price |
| `n` | Number of delivery dates |
| `total` | `unitPrice × qty × n` (BR-2.1) |
| `recipient` | `{name, phone, address, area}` |
| `dtime` | Delivery window |
| `note` | Free-text note |
| `addonText` | Rendered add-on summary |

Cart total is the plain sum of item totals — no discount, tax or delivery
charge is applied at cart level (BR-10.3).

The rendered detail line reads:

```
3 box/pax × Rp 38.000 × 5 tanggal · Tanggal: 24 – 28 Agu 2026 · Jam: Siang (12.00) · Tambahan: Extra Telur
```

## Recipient de-duplication

Before building the message the page tests whether **every** item shares the
same `name`, `phone`, `address`, `area`, `dtime` **and** `note`.

- **All identical** → recipient details appear **once**, in a combined block near
  the bottom. The message stays short.
- **Any difference** → **every item repeats its own** Penerima / Alamat /
  No. HP / Waktu Pengantaran / Catatan lines, so nothing is lost when one
  customer orders for several people at several addresses (BR-12.3).

This is the customer-facing consequence of the recipient-per-card decision in
[[01-product-overview]].

## Message construction

The message is built as an **array of plain-text lines**, then each line is
percent-encoded **individually** and joined with `%0A`.

The reason is in the source: encoding the whole blob at once would escape the
newlines too, and WhatsApp would receive one unreadable run-on line. Encoding
per line keeps every `%0A` a real line break while still safely escaping
user-typed names, addresses and notes. Some stored lines carry an embedded `\n`
(the multi-date "Tanggal" line) and are split before being pushed.

The message includes the bank block:

```
BCA a.n. R Bg Andreas Kurnianto
No. Rek: 8660-281-402
```

Final URL shape:

```
https://wa.me/62818100523?text=<encoded>
```

## The handoff gap

**The page does not send anything.** It opens WhatsApp with a draft. The
customer must still press send.

| Consequence | Detail |
|-------------|--------|
| **Silent abandonment** | A customer who reaches WhatsApp and stops is invisible. The business never learns the order existed. |
| **No confirmation** | Nothing tells the customer the order arrived. |
| **No order number** | Nothing identifies the order on either side. |
| **No copy for the customer** | Leaving the page loses the cart (BR-1.7). |
| **Editable in transit** | The draft is plain text in the customer's own app — quantities, dates and prices can be altered before sending. |
| **URL length** | A large multi-item order produces a very long URL. No truncation guard exists. |

For a mockup this is fine — it is the cheapest possible order channel and it
matches how the business already works. For a rebuild it is the first thing to
replace; see [[09-open-questions]].

Related: [[02-business-rules]] · [[03-site-structure]]
