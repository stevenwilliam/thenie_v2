# 06 — Design system

Read out of the single inline `<style>` block (22,040 characters). There is no
CSS framework, no build step, no external stylesheet, and no web font request.

## Palette

Ten CSS custom properties on `:root`:

| Token | Hex | Role |
|-------|-----|------|
| `--maroon` | `#7a1f2b` | Primary brand; header gradient start |
| `--maroon-dark` | `#5c1620` | Header gradient end |
| `--green` | `#3f6b3f` | Action / active state; selected tab fill |
| `--green-dark` | `#2e4f2e` | Section headings |
| `--cream` | `#faf5ea` | Page background |
| `--cream-2` | `#f2e9d8` | Raised surfaces |
| `--ink` | `#2b2420` | Body text |
| `--muted` | `#7a6f63` | Secondary text; inactive tabs |
| `--line` | `#e6dcc8` | Borders |
| `--white` | `#ffffff` | Cards |

Warm, earthy, food-appropriate: maroon for brand authority, green for "healthy"
and for every affirmative action, cream everywhere so white cards lift off the
background.

## Accessibility — measured, not eyeballed

Contrast ratios computed with the WCAG 2.1 relative-luminance formula.

### Passes

| Foreground | Background | Ratio | Normal text |
|------------|------------|------:|-------------|
| `--ink` | `--cream` | **14.04** | AAA |
| `--ink` | `--white` | **15.26** | AAA |
| `--ink` | `--cream-2` | **12.66** | AAA |
| `--maroon` | `--cream` | **9.38** | AAA |
| `--white` | `--maroon` | **10.20** | AAA |
| `--white` | `--maroon-dark` | **13.20** | AAA |
| `--green-dark` | `--cream` | **8.49** | AAA |
| `--white` | `--green-dark` | **9.23** | AAA |
| `--green` | `--cream` | **5.70** | AA |
| `--white` | `--green` | **6.20** | AA |
| `--muted` | `--white` | **4.90** | AA |

The core palette is genuinely strong. Body text at 14:1 and the white-on-maroon
header at 10.2:1 are comfortably above requirement.

### Findings

| Pair | Ratio | Verdict |
|------|------:|---------|
| `--muted` on `--cream-2` | **4.07** | ❌ **Fails AA** for normal text (needs 4.50) |
| `--muted` on `--cream` | **4.51** | ⚠️ Passes by **0.01** — no margin at all |
| `--line` on `--cream` | **1.25** | Fine as a hairline; **fails the 3:1 UI-component minimum** if a border is the only thing marking an interactive boundary |

**Recommendation for any rebuild** — darken `--muted` from `#7a6f63` to about
`#6b6058`. That lifts it to ~5.4:1 on cream and ~4.9:1 on cream-2, clearing AA
on both with real margin, at almost no visual cost.

Do **not** apply this to `site/index.html`. That file is a verbatim mirror
(see [[07-fidelity-and-verification]]); the fix belongs to the rebuild.

### Other accessibility observations

- **No focus-visible styling.** Keyboard users get only the browser default, and
  on custom `.pref-chip` / `.tab-btn` elements that is easy to lose.
- **Tabs are `<button>`s without `role="tab"`**, `aria-selected` or arrow-key
  navigation.
- **State is colour-only** in places — an active tab differs by fill, a disabled
  delivery window by the `.pref-disabled` class. Disabled radios do carry the
  real `disabled` attribute, which assistive tech reports.
- **No `prefers-reduced-motion` guard** on the panel fade.
- **No skip link**, and no landmark roles beyond `header` / `main`.

## Typography

```css
font-family: 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
```

System stack only — **zero webfont requests**, which is a large part of why the
page renders instantly once loaded. The logo badge uses
`'Brush Script MT', cursive`, a decorative fallback that will differ per
platform.

| Element | Size | Weight |
|---------|------|--------|
| `header h1` | 23px | 700 |
| `header p.tag` | 13.5px | italic |
| `header p.sub` | 12px | — |
| `.section-title` | 15px | 700 |
| `.tab-btn` | 13px | 600 |

Sizes are **fixed pixels**, not `rem`. They therefore ignore the user's browser
font-size preference — a real accessibility limitation and an easy rebuild fix.

## Shape and depth

```css
--radius: 16px;
--shadow: 0 6px 20px rgba(90,40,20,0.08);
```

One radius, one shadow, used consistently. The shadow is tinted warm
(`rgba(90,40,20,…)`) rather than neutral black, which keeps it from muddying the
cream background. Pills use `border-radius: 999px`.

## Components

| Class | Purpose |
|-------|---------|
| `.tab-btn` | Top-level tab pill; `.active` → green fill, white text |
| `.subtab-btn` | Product-family pill inside Menu/Order |
| `.panel` | Tab body; `.active` reveals it with a 0.2s fade |
| `.card` | White surface, radius 16, warm shadow |
| `.pref-chip` | Radio-as-chip (delivery windows); `.pref-disabled` when cut off |
| `.addon-cb` / `.addon-item` / `.addon-days` | Add-on checkbox, row, day picker |
| `.mini-calendar` + `.mc-*` | Date picker (header, label, prev/next, weekdays, grid) |
| `.date-chip` | Selected date; `.gap-break` marks a non-consecutive jump |
| `.qty-control` / `.q-dec` / `.q-inc` | Quantity stepper |
| `.menu-day` / `.kcal` | Menu row and its calorie badge |
| `.menu-photo-card` | Food photography block |
| `.cust-*-item` | Per-card recipient fields (BR-1.2) |
| `.field-label` / `.req` | Field label and required marker |
| `.form-hint` / `.section-note` / `.addon-note` | Helper text |

Frequency confirms the layout: `.pref-chip` ×108, `.addon-cb` ×54,
`.menu-day` ×40, and exactly ×8 for every per-card control — the eight order
cards of [[03-site-structure]].

## Layout

- **Mobile-first**, `main { max-width: 640px; margin: 0 auto; }` — a phone
  column, centred on desktop. It never becomes a wide desktop layout.
- **Sticky tab bar** at `top: 0`, `z-index: 20`.
- **Horizontally scrollable tab strips** with scrollbars hidden
  (`scrollbar-width: none` + `::-webkit-scrollbar { display: none }`).
- `body { padding-bottom: 120px }` clears the sticky cart bar.
- Momentum scrolling via `-webkit-overflow-scrolling: touch`.

## Motion

One keyframe:

```css
@keyframes fade { from {opacity:0; transform:translateY(4px);} to {opacity:1; transform:none;} }
```

0.2s on panel change, plus a 0.15s ease on tab-button transitions and a 1100ms
"added" flash on the add button. Restrained throughout — no reduced-motion
guard, but little to guard against.

Related: [[03-site-structure]] · [[07-fidelity-and-verification]] · [[09-open-questions]]
