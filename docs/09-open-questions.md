# 09 — Open questions

Things the mockup does not answer. Each has a **proposed default** — a
suggestion to accept or overrule, never something already assumed in code. The
mirror itself is untouched regardless (see [[07-fidelity-and-verification]]).

## Blocking a production rebuild

### Q-1 — Orders are lost on abandonment
The page only opens a WhatsApp draft; the customer must press send (BR-12.2).
Abandoned orders are invisible to the business.
**Proposed default:** post the order to a real backend first, then open WhatsApp
as a *notification*, so the order exists whether or not the message is sent.

### Q-2 — The cart cannot survive a reload
No persistence of any kind (BR-1.7). A customer who spends ten minutes building
a 20-date order loses it to an accidental refresh.
**Proposed default:** mirror the cart to `localStorage` on every change and
restore on load. Cheap, and it does not require a backend.

### Q-3 — The cut-off uses the customer's device clock
`getTodayCutoff()` reads `new Date()` in the browser (BR-7.9). A wrong device
clock, or a customer abroad, gets the wrong cut-off.
**Proposed default:** compute the cut-off server-side in `Asia/Jakarta` and
re-validate on submission. Never trust the client clock for a deadline.

### Q-4 — No input validation at all
Presence is checked; format never is. No phone pattern, no length caps, no
sanitisation (see [[05-order-flow-and-whatsapp]]).
**Proposed default:** validate Indonesian mobile formats
(`08…` / `+628…`), cap free text, normalise before validating, and **reject
rather than silently repair**. Validate on both client and server.

### Q-5 — Delivery charge and reschedule fees are displayed but never charged
BR-10.3 and BR-9.4. The WhatsApp message under-states what is actually owed on
small orders, and the admin must correct it by hand every time.
**Proposed default:** implement both in the calculator so the quoted total is
the real total.

### Q-6 — The Flexi `Extra Daging` cap is policy only
BR-6.8 caps it at one portion per 5 pax, but the calculator ignores it.
**Proposed default:** enforce it in the add-on maths.

## Product questions

### Q-7 — Is BR-3.3 intended?
A consecutive run of 15–19 days pays the **full daily rate** — more per day than
a 5-day order. The source marks it as a deliberate revision, but it is
surprising enough to be worth confirming.
**Proposed default:** confirm with the business before reproducing it. If
intended, surface it in the UI so customers understand why extending an order
made it more expensive.

### Q-8 — How is `Lainnya` (other area) handled?
Accepted with no follow-up and no surcharge (BR-11.3).
**Proposed default:** ask for a free-text area and flag the order for admin
pricing.

### Q-9 — Is there a maximum order size?
Top tiers are unbounded (`max: 99999`), and there is no capacity or stock check.
**Proposed default:** add a per-day kitchen capacity limit and refuse dates
that exceed it.

### Q-10 — How far ahead can a customer order?
The calendar has no upper bound.
**Proposed default:** cap at 90 days.

### Q-11 — What happens on public holidays?
Nothing excludes them; the weekday-routine rules assume clean Mon–Fri weeks.
**Proposed default:** maintain a holiday calendar, block those dates, and
exclude them from the BR-3.11/3.12 gap checks.

### Q-11b — The calorie figures contradict each other
The HTML text gives Healthy Meal as **430–500 kcal**, and each day's line reads
±450–485 kkal. The Menu-tab **poster image says 550–590 KKAL / PORSI**. One of
the two is wrong, and the poster is the one customers actually look at.
**Proposed default:** confirm the real figure with the kitchen and make the
text authoritative, since it is the one a rebuild can keep in sync.

### Q-12 — Menu rotation beyond the two shown weeks
Only weeks 34 and 35 of 2026 are present, hard-coded.
**Proposed default:** treat the menu as data on a repeating cycle, not markup.

## Technical questions

### Q-13 — Should the mockup be indexed by search engines?
It has no SEO baseline at all (see [[08-technical-inventory]]).
**Proposed default:** **`noindex`** while it is a mockup on a staging
subdomain, so it cannot outrank or be confused with the real site. The runbook
ships `noindex` by default for exactly this reason.

### Q-14 — Which subdomain should it live on?
Not specified in the brief.
**Proposed default:** `thenie.sunshinefood.co.id` on the existing server, so it
sits alongside `meals.` / `api.` / `cdn.` without touching them.

### Q-15 — Should the 4.6 MB payload be optimised?
Extracting the 13 images to cacheable files would cut repeat visits enormously —
and is forbidden by the exact-mirror requirement.
**Proposed default:** leave the mirror alone; do it in the rebuild. The runbook
applies server-side compression, which is all that can be done without touching
bytes.

### Q-16 — Is `thenie_v2` a rebuild or a redesign?
The name implies a v2, but this repo currently contains only a v1 mirror.
**Proposed default:** keep `site/` as the frozen v1 reference and build v2
alongside it in a separate directory, so the original stays available for
comparison.

## Business questions

### Q-17 — Bank account is a personal name
BR-8.2 names **R Bg Andreas Kurnianto**, not a company.
**Proposed default:** confirm whether a business account should be used before
this goes to real customers.

### Q-18 — Only one WhatsApp number
`62818100523` is the single point of contact. No fallback, no queue, no hours.
**Proposed default:** add business hours and an expected response time to the
page so customers know when to expect a reply.

### Q-19 — No terms, privacy policy or refund policy
The page collects names, phone numbers and addresses with no privacy notice.
Under Indonesia's **UU PDP** that is a genuine compliance gap once real customer
data is involved.
**Proposed default:** add a privacy notice and terms before any public launch.

Related: [[02-business-rules]] · [[08-technical-inventory]] · [[PROGRESS]]
