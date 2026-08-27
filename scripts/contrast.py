#!/usr/bin/env python3
"""WCAG contrast for the Thenie palette.

Contrast is calculated, not eyeballed. The ratios in docs/06-design-system.md
come from this script; re-run it after any colour change and update that table,
because the ratio beside a token IS the record.

The palette below must stay in step with the :root block in site/index.html.
That file is the frozen mirror and is never edited (docs/07-fidelity-and-
verification.md), so in practice these values only move when the site is
re-captured.

    python3 scripts/contrast.py                    # the standing pairings
    python3 scripts/contrast.py '#E1614A' '#FFFFFF'
"""
import sys

# Keep in step with the :root block in site/index.html.
WHITE = '#FFFFFF'        # --cream, --cream-raised
CREAM_SOFT = '#FAF5E9'   # --cream-soft: alternating bands, table stripes
INK = '#2B2620'
INK_SOFT = '#615848'
INK_FAINT = '#8B8271'
MAROON = '#E1614A'       # --maroon (coral, despite the name)
MAROON_DEEP = '#B84B39'
MAROON_TINT = '#FCE3DE'
OLIVE = '#5F7A33'
OLIVE_DEEP = '#3E5321'
OLIVE_TINT = '#E7EEDA'
GOLD = '#D9A867'
FOCUS = '#8FC4EC'
FOOTER = '#0f0c09'


def _lin(c):
    c /= 255
    return c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4


def luminance(hexcolour):
    h = hexcolour.lstrip('#')
    r, g, b = (int(h[i:i + 2], 16) for i in (0, 2, 4))
    return 0.2126 * _lin(r) + 0.7152 * _lin(g) + 0.0722 * _lin(b)


def ratio(a, b):
    la, lb = luminance(a), luminance(b)
    hi, lo = max(la, lb), min(la, lb)
    return (hi + 0.05) / (lo + 0.05)


def verdict(v):
    if v >= 7:      return 'AAA body'
    if v >= 4.5:    return 'AA body'
    if v >= 3:      return 'AA large / UI boundary'
    return 'FAILS'


PAIRS = [
    ('headings on white', INK, WHITE),
    ('body copy on white', INK_SOFT, WHITE),
    ('captions/meta on white', INK_FAINT, WHITE),
    ('coral as text on white', MAROON, WHITE),
    ('coral-deep as text on white', MAROON_DEEP, WHITE),
    ('olive as text on white', OLIVE, WHITE),
    ('olive-deep as text on white', OLIVE_DEEP, WHITE),
    ('gold as text on white', GOLD, WHITE),
    ('white on the coral fill (primary CTA)', WHITE, MAROON),
    ('white on the coral-deep fill', WHITE, MAROON_DEEP),
    ('white on the olive fill', WHITE, OLIVE),
    ('the focus ring against white', FOCUS, WHITE),
    ('headings on the soft band', INK, CREAM_SOFT),
    ('body copy on the soft band', INK_SOFT, CREAM_SOFT),
    ('captions on the soft band', INK_FAINT, CREAM_SOFT),
    ('coral as text on the soft band', MAROON, CREAM_SOFT),
    ('olive as text on the soft band', OLIVE, CREAM_SOFT),
    ('coral-deep on the coral tint', MAROON_DEEP, MAROON_TINT),
    ('olive-deep on the olive tint', OLIVE_DEEP, OLIVE_TINT),
    ('gold on the footer', GOLD, FOOTER),
]

if len(sys.argv) == 3:
    a, b = sys.argv[1], sys.argv[2]
    v = ratio(a, b)
    print(f'{a} on {b}  {v:.2f}  {verdict(v)}')
else:
    width = max(len(n) for n, _, _ in PAIRS)
    for name, a, b in PAIRS:
        v = ratio(a, b)
        print(f'{name:<{width}}  {a} on {b}  {v:5.2f}  {verdict(v)}')
