# 06 — Design system

Everything below is read out of the mirror's two `<style>` blocks. Contrast
ratios are **calculated** from the actual hex values using the WCAG 2.x relative
luminance formula, not estimated.

> **Completely replaced on 2026-08-27.** The previous capture was deep maroon
> `#7a1f2b` on cream `#faf5ea` in system fonts. This one is coral `#E1614A` and
> olive `#5F7A33` on **white**, set in **Baloo 2**. Nothing carried over but the
> radius scale.

---

## 1. Tokens

All design values are CSS custom properties on `:root`. There is exactly one
theme — a comment states it explicitly: *"single committed dark theme: no
OS/toggle variants needed"* (the wording is left over; the theme is light).

### Surfaces

| Token | Value | Use |
|-------|-------|-----|
| `--cream` | `#FFFFFF` | Page background |
| `--cream-soft` | `#FAF5E9` | Alternating section bands, table stripes |
| `--cream-raised` | `#FFFFFF` | Cards, nav pill, modal |
| `--paper-line` | `rgba(150,135,115,0.45)` | Hairline borders |
| `--shadow` | `rgba(43,38,32,0.16)` | Card shadow |
| `--overlay-scrim` | `rgba(9,7,5,0.7)` | Image overlays |

### Ink

| Token | Value | Use |
|-------|-------|-----|
| `--ink` | `#2B2620` | Headings, primary text |
| `--ink-soft` | `#615848` | Body copy |
| `--ink-faint` | `#8B8271` | Captions, meta |

`--surface-ink`, `--surface-ink-soft` and `--surface-ink-faint` hold the same
three values. They exist so nested "surface" scopes (the nav panel, the cart
bar, the modal) can re-declare `--ink` locally without losing the real value.

### Brand

| Token | Value | Use |
|-------|-------|-----|
| `--maroon` | `#E1614A` | Primary action, eyebrows, accents |
| `--maroon-deep` | `#B84B39` | Hover, deeper accent |
| `--maroon-tint` | `#FCE3DE` | Tinted backgrounds |
| `--maroon-tint-strong` | `#F6C7BC` | Stronger tint |
| `--olive` | `#5F7A33` | Secondary action |
| `--olive-deep` | `#3E5321` | Olive text on tint |
| `--olive-tint` | `#E7EEDA` | Ghost-button background |
| `--olive-tint-strong` | `#d6e3c2` | Stronger tint |
| `--gold` | `#D9A867` | Hero accent, brand sub-label |
| `--ribbon` | `#E4D3AA` | Eyebrow on dark panels |
| `--focus-ring` | `#8FC4EC` | Focus outline |

### Aliases

The order app was written against a different token set, so its names are
aliased rather than rewritten — this is how the two codebases were merged
without editing the app's CSS:

```css
--green: var(--olive);        --green-dark: var(--olive-deep);
--maroon-dark: var(--maroon-deep);
--white: var(--cream-raised); --muted: var(--ink-soft);
--line: var(--paper-line);    --cream-2: var(--cream-soft);
--radius: 16px;
--amber-text:#8a5a26; --amber-border:#b5793a; --amber-bg-soft:rgba(181,121,58,0.12);
```

---

## 2. Typography

**One family for everything: `"Baloo 2"`**, loaded from Google Fonts at
weights 400/500/600/700/800, with `&display=swap`.

```css
font-family:"Baloo 2", system-ui, -apple-system, "Segoe UI", sans-serif;
```

| Role | Size | Weight | Notes |
|------|------|--------|-------|
| Body | 16px / 1.6 | 400 | `--ink-soft` |
| `h1`–`h4` | fluid | 600 | `text-wrap: balance` |
| `.hero h1` | `clamp()` | 600 | `.accent` span is italic gold |
| `.eyebrow` | 0.72rem | 700 | uppercase, `0.14em` tracking, with a 20×2px rule before it |
| `.brand-word` | 1.3rem | 700 | |
| `.brand-sub` | 0.62rem | 500 | uppercase, gold |
| `.btn` | 0.98rem | 700 | |
| `.nav-cta` | 0.92rem | 800 | 0.86rem below 520px |
| `.cart-btn` | 14.5px | 700 | |

`.script` is declared as `"Baloo 2", cursive` italic — a decorative alias to the
same family, not a second typeface.

`.tnum` applies `font-variant-numeric: tabular-nums` so price columns align.

### The font is the site's only external request

Everything else — all 44 images — is inlined. If `fonts.googleapis.com` is
unreachable the page still renders, in the browser's default sans-serif, and
`display=swap` means text is never invisible while waiting. This has deployment
consequences; see [[13-production-deployment-runbook]].

---

## 3. Components

| Component | Class | Notes |
|-----------|-------|-------|
| Floating nav pill | `.float-nav` | Fixed bottom-centre, `z-index:260`, lifts to 96px on `#order` |
| Nav panel | `.navlinks` | Opens upward, fades + slides, `aria-expanded` maintained |
| Buttons | `.btn` + `.btn-primary` / `.btn-outline` / `.btn-ghost-olive` / `.btn-sm` | Pill, `999px` radius, lift on hover |
| Card | `.card` | White, hairline border, 16px radius |
| Order card | `.order-card` | Renamed from `.card` to avoid colliding with the above |
| Photo card | `.photo-card` | Image + body, fixed aspect ratio |
| Plan card | `.plan-card` | Image header with a calorie badge, spec grid, tags |
| Tilt frame | `.tilt-frame` / `.tilt-frame.right` | Rotated photo frame |
| Timeline | `.timeline` / `.tl-item` / `.tl-dot` | Four-step, olive dots |
| Stats strip | `.stats-strip` | 4 → 2 columns below 620px, animated counters |
| Philosophy panel | `.philosophy` | Dark panel with `.grain` texture |
| Price tabs | `.tabbar` + `.tab-highlight` | Measured, animated sliding highlight |
| Price table | `.price-table` | Coral header, `.highlight` row, `.num` right-aligned |
| Mini calendar | `.mini-calendar` | Shared by all three date widgets |
| Date chip | `.date-chip` / `.gap-break` | `.gap-break` marks a discontinuity |
| Package badge | `.package-badge` + `.tier-*` | Colour-coded per tier |
| Add-on chips | `.addon-chip` / `.addon-day-chip` | Per-day opt-out chips |
| Cart bar | `.cart-bar` | Fixed bottom, `z-index:150` |
| Modal | `.modal-overlay` / `.modal` | Bottom sheet, `z-index:300` |
| WhatsApp FAB | `.fab-wa` | Fixed bottom-right, 60px, `#25D366` |

### The grain texture

`.grain::after` overlays an inline SVG `feTurbulence` fractal-noise filter as a
`data:` URI. It is declared twice — the second rule overrides the first, so the
effective values are `opacity:0.08; mix-blend-mode:overlay` and the first
declaration (`0.05` / `multiply`) is dead.

---

## 4. Layout

| Aspect | Value |
|--------|-------|
| Container | `max-width: 1180px`, `padding: 0 24px` |
| Section rhythm | `padding: 76px 0`; `.tight` → `48px 0` |
| Grids | `.g2` `.g3` `.g4` `.g5` helpers |
| Radius | 16px cards · 999px pills · 12–13px small controls |
| Order app | `max-width: 640px` — a phone-first column inside the wider site |

### Breakpoints

`520px` · `560px` · `620px` · `780px` — plus component-local ones. There is no
single declared scale; each component picks the width at which it breaks.

---

## 5. Motion

| Effect | Detail |
|--------|--------|
| Page transition | `fadeIn` 0.35s — 6px rise + fade |
| Counters | 1,100ms, eased `1 − (1−p)³`, fired once by `IntersectionObserver` |
| Tab highlight | `0.28s cubic-bezier(.5,0,.2,1)` on transform, width and height |
| Buttons | `translateY(-2px)` on hover, 0.15s |
| FAB | `scale(1.08)` on hover, 0.2s |
| Smooth scroll | `html { scroll-behavior: smooth }` |

### Reduced motion

A global block handles most of it well:

```css
@media (prefers-reduced-motion: reduce){
  html{scroll-behavior:auto;}
  *{animation-duration:0.001ms !important;
    animation-iteration-count:1 !important;
    transition-duration:0.001ms !important;}
}
```

JavaScript honours it too — counters jump to their final value, and
`window.scrollTo` uses `auto` instead of `smooth`.

**One gap:** the rule zeroes *durations*, not *transforms*. Hover effects that
change `transform` still jump instantly rather than being suppressed. The
overlay in [[14-overlays]] neutralises this for `.fab-wa`, the most prominent
one.

---

## 6. Accessibility

### What is done well

- `<html lang="id">`.
- A visible focus style is defined globally:
  `outline: 3px solid var(--focus-ring); outline-offset: 2px` on
  `a`, `button` and `input` `:focus-visible`.
- `aria-expanded` on the nav toggle, `aria-current="page"` on the active link.
- `role="tablist"` / `role="tab"` / `aria-selected` on the Harga tab bar.
- Calendar days are real `<button type="button">` with `disabled`.
- The floating WhatsApp button has an `aria-label`.
- Decorative images carry `alt=""`; content images have descriptive Indonesian
  alt text.
- `prefers-reduced-motion` is respected in both CSS and JS.

### Contrast — measured

| Foreground | Background | Ratio | AA normal (4.5) | AA large (3.0) |
|------------|-----------|------:|:---------------:|:--------------:|
| `--ink` `#2B2620` | white | **14.99** | ✅ | ✅ |
| `--ink-soft` `#615848` | white | **7.01** | ✅ | ✅ |
| `--ink-faint` `#8B8271` | white | **3.80** | ❌ | ✅ |
| `--maroon` `#E1614A` | white | **3.49** | ❌ | ✅ |
| `--maroon-deep` `#B84B39` | white | **5.12** | ✅ | ✅ |
| `--olive` `#5F7A33` | white | **4.86** | ✅ | ✅ |
| `--olive-deep` `#3E5321` | white | **8.52** | ✅ | ✅ |
| `--gold` `#D9A867` | white | **2.15** | ❌ | ❌ |
| white | `--maroon` `#E1614A` | **3.49** | ❌ | ✅ |
| white | `--maroon-deep` | **5.12** | ✅ | ✅ |
| white | `--olive` | **4.86** | ✅ | ✅ |
| `--maroon-deep` | `--maroon-tint` | **4.19** | ❌ | ✅ |
| `--olive-deep` | `--olive-tint` | **7.16** | ✅ | ✅ |
| `#e7ded0` | footer `#0f0c09` | **14.63** | ✅ | ✅ |
| `#cfc6b4` | footer | **11.51** | ✅ | ✅ |
| `#a89d89` | footer | **7.29** | ✅ | ✅ |

Sections alternate between white and `--cream-soft` `#FAF5E9`, so every pairing
also has to hold on the soft band — and each one loses a little there:

| Foreground | On `--cream-soft` | vs white |
|------------|------------------:|---------:|
| `--ink` | **13.78** ✅ | 14.99 |
| `--ink-soft` | **6.44** ✅ | 7.01 |
| `--ink-faint` | **3.49** ❌ | 3.80 |
| `--maroon` | **3.21** ❌ | 3.49 |
| `--olive` | **4.47** ❌ *(just short of 4.5)* | 4.86 |

Reproduce any of these with:

```bash
python3 scripts/contrast.py                    # the standing pairings
python3 scripts/contrast.py '#E1614A' '#FFFFFF'
```

The footer is the most accessible part of the page. The problems are all in the
brand colours on white:

| # | Finding | Severity |
|---|---------|----------|
| **A11Y-1** | **`--maroon` on white fails AA at 3.49.** It is the colour of every `.eyebrow` (0.72rem), `.page-hero-tag`, `.tl-year`, `.plan-price`, `.stat-item b`, and every `.contact-card` link. None of these is large text. | High — affects most section labels |
| **A11Y-2** | **White on `--maroon` also fails AA at 3.49.** This is the *primary call to action*: `.btn-primary` (0.98rem/700), `.nav-cta` (0.92rem/800), `.cart-btn` (14.5px/700). All are below the 18.66px-bold "large text" threshold, so 4.5:1 applies and none reaches it. | High — the buttons that matter most |
| **A11Y-3** | **`--gold` on white is 2.15** — fails even the large-text threshold. Used for `.brand-sub` at 0.62rem uppercase. On the hero it sits on a dark photo scrim instead and is fine there. | Medium |
| **A11Y-4** | `--ink-faint` at 3.80 fails AA for the small text it is used on (`.testi-meta` 0.82rem, price footnotes 0.84rem). | Medium |
| **A11Y-5** | The focus ring `#8FC4EC` against white is **1.86:1** — below the 3:1 WCAG 2.2 minimum for a focus indicator. It is clearly visible against dark controls but weak against white ones. | Medium |
| **A11Y-6** | `--maroon-deep` on `--maroon-tint` is 4.19 — the nav panel's hover state, marginally short. | Low |
| **A11Y-16** | **`--olive` on `--cream-soft` is 4.47** — three hundredths short of AA. On white it passes at 4.86. Any olive text that lands on an alternating band fails; the same text one section up passes. | Low |

`--maroon-deep` is already in the palette and passes at 5.12. Swapping the
foreground/background use of `--maroon` for `--maroon-deep` would fix A11Y-1 and
A11Y-2 without introducing a new colour.

### Semantics — findings

| # | Finding | Impact |
|---|---------|--------|
| **A11Y-7** | The package and period selectors (`.plan-opt`) are **`<div>`s with click handlers**. Not focusable, no `role`, no `aria-selected`. They are the primary control on Nasi Bento, Nasi Kuning, Paket Acara and Catering Kantor — **four of the eight order cards cannot be operated by keyboard**. | High |
| **A11Y-8** | The date helpers — `.quick-fill-week`, `.quick-fill-skip`, `.clear-dates`, `.addon-day-chip` — are **`<span>`s with click handlers**. Same problem. | High |
| **A11Y-9** | "Tutup & lanjut pilih menu" is a **`<p>`**, not a button. The modal's close control is not keyboard-reachable. | High |
| **A11Y-10** | The modal has **no `role="dialog"`, no `aria-modal`, no focus trap, no restore-focus, and no `Escape` handler**. Background content stays reachable behind it. | High |
| **A11Y-11** | Price changes, add-to-cart, and validation messages are **not announced**. No `aria-live` region exists anywhere. | Medium |
| **A11Y-12** | The Harga tab bar has `role="tablist"` and `role="tab"` but the panels have **no `role="tabpanel"` and no `aria-controls`**, and the tabs are not wired for arrow-key navigation. | Medium |
| **A11Y-13** | The order app's three tab levels have **no ARIA roles at all**. | Medium |
| **A11Y-14** | The six `<h1>`s are all present in the DOM at once; five are inside `display:none` pages. Assistive tech that walks the document rather than the render tree sees six top-level headings. | Low |
| **A11Y-15** | **11 of the 44 `<img>` elements carry no `alt` attribute at all.** They are the order app's menu posters and the bank-details card — images that carry the entire weekly menu, and the account number, as pixels. (5 more are correctly `alt=""` as decoration; the remaining 28 have descriptive Indonesian alt text.) | Medium |

None of this is fixable in `site/` without breaking the exact-mirror rule
([[07-fidelity-and-verification]]). It is rebuild work — see
[[09-open-questions]].

Related: [[03-site-structure]] · [[08-technical-inventory]] · [[09-open-questions]]
