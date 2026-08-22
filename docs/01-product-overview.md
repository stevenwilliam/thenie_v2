# 01 — Product overview

## What it is

A **single-page, mobile-first ordering front end** for Thenie Healthy Catering,
a meal-catering business operating in the **Tangerang / Serpong** corridor west
of Jakarta. The customer browses menus, builds an order across one or more
product families, fills in recipient details, and sends the finished order to
the business **over WhatsApp**.

The interface language is **Indonesian** (`<html lang="id">`). There is no
language switcher and no English variant.

## Who uses it

| Actor | How they use it |
|-------|-----------------|
| **Customer** | The only user of this page. Browses, builds a cart, sends via WhatsApp. |
| **Admin / owner** | Never touches this page. Receives the order as a WhatsApp message on `+62 818 100 523` and processes it manually. |

There is no staff surface, no login, and no account. Every visitor is anonymous
and every session is independent.

## The five product families

The Order tab is divided into five sub-tabs. These are the product families:

| # | Family | Sub-tab label | Pricing shape |
|---|--------|---------------|---------------|
| 1 | **Daily Order** | 🥗 Daily Order | Subscription rates, four meal types |
| 2 | **Nasi Bento Box** | 🍱 Nasi Bento Box | Quantity tiers, two packages |
| 3 | **Catering Kantor** | 🏢 Catering Kantor | Pax × period matrix, five tiers |
| 4 | **Nasi Kuning** | 🟡 Nasi Kuning | Quantity tiers, one package |
| 5 | **Paket Acara** | 🍽️ Paket Acara | Pax tiers, four packages A–D |

Daily Order splits further into **four meal types**, each its own order card
with its own rate table:

- **Healthy Meal** — the flagship healthy line
- **Bulking Extra** — larger portions, higher protein
- **Regular Catering** — the standard adult meal
- **Kids Meal** — child portions

That gives **eight order cards** in total (4 daily + 4 family-level). Each card
is self-contained: its own quantity, its own calendar, its own add-ons, its own
recipient, its own delivery time.

## The defining design decision: recipient-per-card

Most ordering sites collect the customer's name and address **once**, at
checkout. This one collects them **per order card**.

The reason is written into the source as a comment: one customer ordering for
two different people at two different addresses fills each card with that
person's own details, instead of being forced into a single shared address.
Checkout is then pure summary and re-confirmation — it re-enters nothing.

This matters for any future rebuild: the data model is
**order → many items, each item carrying its own delivery target**, not
*order → one address → many items*. See [[03-site-structure]] and
[[05-order-flow-and-whatsapp]].

## Delivery areas

Nine options, identical on all eight cards:

Gading Serpong · Karawaci · BSD · Alam Sutera · Medang · Villa Melati Mas ·
Park Serpong · Golden Stone · **Lainnya** (other)

`Lainnya` is a free escape hatch — the page does not ask for a follow-up when
it is chosen, and does not restrict or surcharge it. **(inferred: out-of-area
handling is resolved by the admin over WhatsApp.)**

## What it deliberately does not do

- No payment. Bank details are **displayed** for manual transfer; nothing is charged.
- No stock or capacity check. Any quantity on any date is accepted.
- No order history, no reorder, no saved addresses.
- No confirmation screen after sending — the customer leaves for WhatsApp.

Related: [[02-business-rules]] · [[08-technical-inventory]] · [[09-open-questions]]
