# 09 — Open questions

Things the site does not answer. Each has a **proposed default** — a suggestion
to accept or overrule, never something already assumed in code. The mirror
itself is untouched regardless (see [[07-fidelity-and-verification]]).

Questions carried over from the 2026-08-22 review are marked **(still open)**;
several were re-checked against the new capture and are recorded as answered.

---

## Blocking a production rebuild

### Q-1 — Orders are lost on abandonment · (still open)
The page only opens a WhatsApp draft; the customer must press send (BR-1.7).
Abandoned orders are invisible to the business.
**Proposed default:** post the order to a real backend first, then open WhatsApp
as a *notification*, so the order exists whether or not the message is sent.

### Q-2 — The cart cannot survive a reload · (still open)
No persistence of any kind (BR-1.4). A customer who spends ten minutes building
a 20-date order loses it to an accidental refresh.
**Proposed default:** mirror the cart to `localStorage` on every change and
restore on load. Cheap, and it does not require a backend.

### Q-3 — The cut-off uses the customer's device clock · (still open)
`getTodayCutoff()` reads `new Date()` in the browser (BR-7.9), and `TODAY` is
computed once at load and never refreshed (TQ-4). A wrong device clock, a
customer abroad, or a tab left open overnight all get the wrong answer.
**Proposed default:** compute the cut-off server-side in `Asia/Jakarta` and
re-validate on submission. Never trust the client clock for a deadline.

### Q-4 — No input validation at all · (still open)
Presence is checked; format never is. No phone pattern, no length caps, no
sanitisation (BR-12.5).
**Proposed default:** validate Indonesian mobile formats (`08…` / `+628…`), cap
free text, normalise before validating, and **reject rather than silently
repair**. Validate on both client and server.

### Q-5 — Fees are displayed but never charged · (still open)
BR-9.4, BR-9.6 and BR-10.3. The WhatsApp message under-states what is owed, and
the admin corrects it by hand every time. There is also an inconsistency: a red-
rice request costs Rp5,000 as an add-on checkbox, and nothing if the customer
types it into "Catatan lain".
**Proposed default:** implement all of them in the calculator so the quoted total
is the real total.

### Q-6 — The Flexi `Extra Daging` cap is the only such rule implemented
BR-6.8 is enforced in code. Nothing else in the terms is.
**Proposed default:** decide deliberately which policy text is a *rule* (enforce
it) and which is *guidance* (say so), rather than leaving the split accidental.

---

## Product questions

### Q-7 — Is BR-3.3 intended? · (still open)
A consecutive run of 15–19 days pays the **full daily rate** — more per day than
a 5-day order. The source marks it as a deliberate revision, but it is a cliff a
customer can fall off simply by extending an order.
**Proposed default:** confirm with the business. If intended, surface it in the
UI so customers understand why extending made it more expensive.

### Q-8 — How is `Lainnya` (other area) handled? · (still open)
Accepted with no follow-up and no surcharge (BR-11.3).
**Proposed default:** ask for a free-text area and flag the order for admin
pricing.

### Q-9 — Is there a maximum order size? · (still open)
Top tiers are unbounded (`max: 99999`), and there is no capacity or stock check.
A 500-pax Kantor order silently prices at the 101–200 rate.
**Proposed default:** add a per-day kitchen capacity limit, refuse dates that
exceed it, and route very large orders to a quote instead of a price.

### Q-10 — How far ahead can a customer order? · (still open)
The calendar has no upper bound (BR-7.8).
**Proposed default:** cap at 90 days.

### Q-11 — What happens on public holidays? · (still open)
Nothing excludes them, and the weekday-routine rules (BR-3.11, BR-3.12) assume
clean Mon–Fri weeks. Indonesian holidays will silently break a customer out of
the Bulanan tier and into a more expensive one.
**Proposed default:** maintain a holiday calendar, block those dates, and exclude
them from the gap checks.

### Q-12 — Menu rotation beyond the two shown weeks · (still open)
Only weeks 34 and 35 of 2026 exist, as hard-coded markup (BR-15.1). Both are in
the past relative to nothing — they are simply the two weeks that were current
at capture. A customer ordering for September sees no menu at all.
**Proposed default:** treat the menu as data on a repeating cycle, not markup,
and show the correct week for whatever dates are selected.

### Q-20 — The two scale figures disagree · **NEW**
Beranda's stats strip says **2.000+ porsi/minggu**. Tentang Kami's service-area
card says **2.500–3.000 porsi/minggu** under the heading "Skala 2026". The
Perjalanan Kami paragraph repeats "lebih dari 2.000". A visitor reading both
pages sees two different numbers for the same claim.
**Proposed default:** pick one figure, and if the second is a forward-looking
target, label it as one.

### Q-21 — Same-day ordering: the copy and the code disagree · **NEW**
The Pesan Online home tab states *"Order untuk hari ini sudah tidak bisa dipilih
— minimal untuk besok"*, and a source comment says "Same-day ordering is
closed". **The shipped code allows today until 12:00** (BR-7.3, BR-7.4). One of
the two is wrong, and the customer-facing copy is the one people act on.
**Proposed default:** confirm the real policy. If same-day *is* allowed, fix the
copy — it is currently turning away orders the kitchen would accept.

### Q-22 — The Flexi tiers are never published · **NEW**
Harga shows only Harian / Mingguan / Bulanan. Flexi Mingguan and Flexi Bulanan —
two of the six tiers, and the ones most non-consecutive orders land in — appear
for the first time in the order form's package badge, after the customer has
already picked dates.
**Proposed default:** publish all six tiers on the Harga page. A customer who
picks scattered dates should be able to find out what that costs before doing
it.

### Q-23 — Paket Acara publishes 39 dishes that cannot be selected · **NEW**
The Harga page lists the full Menu Pilihan across eight categories, and the
order card lists each package's composition — but the form has no way to choose
any of it. The customer is told to discuss it with the admin in the notes
(BR-14.6).
**Proposed default:** either add the selectors, or say plainly on the card that
menu choice happens over WhatsApp, so the published list does not read as a
picker that is broken.

### Q-11b — The calorie figures contradict each other · **sharpened**
Three figures, two of them on the same image. The Daily Order card text says
Healthy Meal is **430–500 kcal**. The Menu poster's header says
**"~ 550-590 KKAL / PORSI"**. The five per-day totals printed on *that same
poster* read **±455, ±455, ±465, ±485, ±465 kkal** — agreeing with the card text
and contradicting the poster's own header. Verified by decoding and reading the
image (BR-15.6).
**Proposed default:** the header is the outlier and is almost certainly the
error. Confirm with the kitchen, then fix the poster.

---

## Technical questions

### Q-13 — Should the site be indexed by search engines? · **updated**
The `X-Robots-Tag: noindex, nofollow` header in the Nginx config
([[13-production-deployment-runbook]]) is what controls this — the page itself
carries no robots meta.

The new capture **improves** the position: there are now six real `<h1>`s and
per-page `<title>`s. It still ships **no meta description, no Open Graph tags,
no canonical URL and no JSON-LD**, so it would index thinly and share as a bare
link with no title card (see [[08-technical-inventory]]).
**Proposed default:** keep `noindex` while `thenie.id` is a preview. Before
removing it, add the SEO baseline — description, OG/Twitter cards, canonical,
and `LocalBusiness` + `Menu` JSON-LD. That cannot go into `site/` without
breaking the exact-mirror rule, so it is a v2 task (Q-16).

### Q-14 — Which domain should it live on? — **ANSWERED 2026-08-22**
**`thenie.id`** — its own apex domain, not a subdomain of `sunshinefood.co.id`.
`www.thenie.id` 301-redirects to the bare domain. The runbook reflects this and
**needs no change** for the new capture.

### Q-15 — Should the payload be optimised? · **sharpened**
Now 6.7 MB, up from 4.6. Three wins are available and all require editing the
page:
- **Deduplicate embedded images** — 1.06 MB, ~16% of the file, for zero visual
  change (the logo alone is embedded 7 times).
- **Extract images to cacheable files** — the whole 4.8 MB stops being
  re-downloaded on every change.
- **Lazy-load below-the-fold images** — currently impossible; they are inline.

**Proposed default:** leave the mirror alone; do all three in the rebuild. The
runbook's gzip is all that can be done without touching bytes, and it now saves
27% (up from ~4%) because the new capture carries much more text.

### Q-16 — Is `thenie_v2` a rebuild or a redesign? · **sharpened**
The 2026-08-27 capture answers half of it: upstream has already done the
**redesign** — a full marketing site around the unchanged order app. What is
still missing is the **rebuild**: a backend, persistence, validation, and the
accessibility and SEO baselines.
**Proposed default:** keep `site/` as the frozen reference and build v2
alongside it, treating this documentation set as the specification.

### Q-24 — The font is now a third-party dependency · **NEW**
The page loads Baloo 2 from Google Fonts on every visit
([[08-technical-inventory]]). Every visitor's IP and User-Agent reach Google.
Under Indonesia's **UU PDP** that is a processing question, and in the EU it is
settled case law that it is not permitted without consent.
**Proposed default:** self-host the font in the rebuild. It is two files and
removes the site's only external request. Until then, be aware that a
`Content-Security-Policy` added to the Nginx config must allow
`fonts.googleapis.com` and `fonts.gstatic.com` or the typeface breaks silently.

### Q-25 — Four of the eight order cards cannot be used by keyboard · **NEW**
The package and period selectors are `<div>`s with click handlers, the date
helper chips are `<span>`s, and the modal's close control is a `<p>` — see
A11Y-7 through A11Y-10 in [[06-design-system]]. Nasi Bento, Nasi Kuning, Paket
Acara and Catering Kantor have **no keyboard path to selecting a package**.
**Proposed default:** treat this as a launch blocker, not a polish item. Real
`<button>` elements with `aria-pressed`, a `role="dialog"` modal with a focus
trap and `Escape`, and an `aria-live` region for price changes.

### Q-26 — `:has()` is load-bearing · **NEW**
`body:has(#order.page.active)` is what lifts the nav pill clear of the cart bar.
On a browser without `:has()` — Firefox before 121, Dec 2023 — the pill sits on
top of the checkout button.
**Proposed default:** accept it (the floor is reasonable in 2026), but record it
so nobody is surprised by an old-browser bug report.

---

## Business questions

### Q-17 — Bank account is a personal name · (still open)
BR-8.2 names **R Bg Andreas Kurnianto**, not a company. It appears in the
checkout terms, on the Pesan Online home tab, and inside an image.
**Proposed default:** confirm whether a business account should be used before
this goes to real customers.

### Q-18 — Contact routing is unclear · **sharpened**
`62818100523` is the number every order and every CTA goes to. The Kontak page
introduces a **second number, `62817771123`**, which appears nowhere else and is
never explained. No business hours and no expected response time are stated
anywhere.
**Proposed default:** label what each number is for, and publish operating hours
and a response-time expectation.

### Q-19 — No terms, privacy policy or refund policy · (still open)
The page collects names, phone numbers and addresses with no privacy notice.
Under Indonesia's **UU PDP** that is a genuine compliance gap once real customer
data is involved. The reschedule terms exist, but there is no refund policy at
all.
**Proposed default:** add a privacy notice, terms of service, and a refund
policy before any public launch. Q-24 folds into this.

### Q-27 — The service-area lists do not match · **NEW**
The order form's Area dropdown offers Gading Serpong, Karawaci, BSD, Alam
Sutera, Medang, Villa Melati Mas, Park Serpong, Golden Stone, Lainnya. The
marketing pages name Gading Serpong, BSD, Karawaci, Alam Sutera and Medang, with
Bintaro, Pondok Aren and Ciledug as expansion. Villa Melati Mas, Park Serpong
and Golden Stone are orderable but never advertised; Bintaro, Pondok Aren and
Ciledug are advertised but not orderable (BR-11.4).
**Proposed default:** reconcile the two lists, and decide whether the expansion
areas are actually being served.

---

Related: [[02-business-rules]] · [[06-design-system]] · [[08-technical-inventory]] · [[PROGRESS]]
