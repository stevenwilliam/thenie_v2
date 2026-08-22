# 14 — Floating WhatsApp button (overlay)

The first thing in this repository that is **ours** rather than captured. Every
other document describes the mirror; this one describes something added on top
of it.

## Why it is an overlay and not an edit

`site/index.html` is a byte-for-byte capture (see [[07-fidelity-and-verification]]).
Editing it would break `scripts/verify-mirror.sh`, break the ability to diff
against upstream, and quietly turn "the mirror" into "the mirror, plus whatever
we bolted on since". So the mirror is never touched. Instead:

```
site/index.html            the capture — untouched, hash still 9d4cfeb…
site/overlays/*.html       our additions, one file each
scripts/build-site.sh      mirror + overlays  →  dist/index.html
dist/index.html            what Nginx actually serves (git-ignored)
```

`build-site.sh` re-checks the mirror's SHA-256 before it builds and refuses to
run if the capture has drifted, so the invariant is enforced, not just written
down.

## What the button does

`site/overlays/whatsapp-fab.html` adds a round green WhatsApp button pinned to
the bottom-right of every screen. Tapping it opens

```
https://wa.me/62818100523?text=Halo Thenie Healthy Catering, saya mau tanya-tanya soal menu & order 🙏
```

in a new tab — the same number the checkout flow already sends orders to
(see [[05-order-flow-and-whatsapp]]), so there is one WhatsApp destination for
the whole site. This button is for **questions before ordering**; the cart's
"Kirim Order" button remains the way an actual order is sent, and nothing about
that flow changed.

## Placement decisions

| Choice | Value | Why |
|---|---|---|
| Size | 72 px, 80 px from 560 px up | "Quite big" as asked — comfortably above the 44 px touch-target minimum noted in [[06-design-system]] |
| Bottom offset | `calc(110px + env(safe-area-inset-bottom))` | `.cart-bar` is `position:fixed` and ~97 px tall; the button clears it instead of covering "Lihat Order & Checkout" |
| `z-index` | 60 | Above `.cart-bar` (50), below `.modal-overlay` (100) — so the checkout sheet still covers it |
| Colour | `#25d366` | WhatsApp's own green. It reads as a WhatsApp button at a glance, which the maroon/green site palette would not |
| Label | "Chat Admin" pill, ≥560 px only | No room for it beside the button on a phone |
| Motion | Pulse ring, disabled under `prefers-reduced-motion` | |

Accessibility: it is a real `<a>` (keyboard-reachable, middle-clickable), carries
an `aria-label`, and has a visible maroon `:focus-visible` ring.

## Changing it

Edit `site/overlays/whatsapp-fab.html`, run `scripts/build-site.sh`, redeploy.
To change the number, change it in the overlay **and** in the mirror's checkout
handler — but the mirror cannot be hand-edited, so a number change is really a
request for a new upstream capture. See [[09-open-questions]].

To add a second overlay, drop another `.html` file in `site/overlays/`; the
build injects every file in that directory, in sorted filename order.

## Deploying it

[[13-production-deployment-runbook]], Part 9 — the short version is
`git pull && ./scripts/build-site.sh && sudo cp dist/index.html /var/www/thenie/`.
