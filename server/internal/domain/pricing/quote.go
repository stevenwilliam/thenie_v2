package pricing

import (
	"errors"
	"fmt"
	"time"
)

// This file ports the rest of the calculator: the totals the front end computes
// in render(), recalc() and the Kantor IIFE.
//
// Every arithmetic detail is preserved, including the roundings. JavaScript's
// Math.round is round-half-UP for positive numbers (floor(x+0.5)), NOT Go's
// banker-free math.Round on a float. Both are reproduced here with integer
// arithmetic so no float ever touches a price, and roundHalfUp exists precisely
// so the two engines cannot drift by a rupiah on a .5 boundary.

var (
	ErrNoDates      = errors.New("pricing: at least one delivery date is required")
	ErrPaxTooLow    = errors.New("pricing: pax must be at least 1")
	ErrQtyBelowMin  = errors.New("pricing: quantity is below the product minimum")
	ErrNoBandForQty = errors.New("pricing: no price band covers that quantity")
	ErrUnknownRice  = errors.New("pricing: rice option must be dengan or tanpa")
)

// roundHalfUp divides a by b rounding halves away from zero, matching
// JavaScript's Math.round for the positive values every price path uses.
func roundHalfUp(a, b int64) int64 {
	if b == 0 {
		return 0
	}
	return (2*a + b) / (2 * b)
}

// Addon is one Tambahan as the calculator sees it.
type Addon struct {
	Code  string `json:"code"`
	Price int64  `json:"price"`
	// RestrictDays is digit characters, 0=Sunday..6=Saturday. Empty = any day.
	RestrictDays string `json:"restrict_days"`
	// FlexiPortionPerPax carries BR-6.8: on a Flexi tier this add-on charges
	// floor(pax/N) portions instead of pax. Zero means charge per full pax.
	FlexiPortionPerPax int `json:"flexi_portion_per_pax,omitempty"`
	// Dates optionally narrows the add-on to specific delivery dates. Empty
	// means every eligible date, which is the front end's default until the
	// customer taps a day chip off.
	Dates []string `json:"dates,omitempty"`
}

// AddonCharge is one add-on's contribution, itemised.
type AddonCharge struct {
	Code         string   `json:"code"`
	UnitPrice    int64    `json:"unit_price"`
	Days         int      `json:"days"`
	EffectivePax int      `json:"effective_pax"`
	Total        int64    `json:"total"`
	MatchedDates []string `json:"matched_dates"`
	// Note explains a reduced EffectivePax, so a customer asking "why is Extra
	// Daging cheaper than I expected" has an answer in the response.
	Note string `json:"note,omitempty"`
}

// SubscriptionRequest quotes one Daily Order card.
type SubscriptionRequest struct {
	Dates []time.Time
	Pax   int
	Rates Rates
	// PaxTable is Regular Catering's group price table, rice -> period -> pax
	// -> whole-group day total. Nil for every other plan.
	PaxTable map[string]map[string]map[int]int64
	// Rice selects the PaxTable variant; ignored when PaxTable is nil.
	Rice   string
	Addons []Addon
}

// SubscriptionQuote is the priced result.
type SubscriptionQuote struct {
	Analysis      *Analysis     `json:"analysis"`
	Pax           int           `json:"pax"`
	Days          int           `json:"days"`
	EffectiveRate int64         `json:"effective_rate"`
	UsedPaxTable  bool          `json:"used_pax_table"`
	RiceLabel     string        `json:"rice_label,omitempty"`
	MenuTotal     int64         `json:"menu_total"`
	AddonTotal    int64         `json:"addon_total"`
	Total         int64         `json:"total"`
	AddonCharges  []AddonCharge `json:"addon_charges"`
}

// QuoteSubscription prices one Daily Order card.
func QuoteSubscription(req SubscriptionRequest, r Rules) (*SubscriptionQuote, error) {
	dates := SortUnique(req.Dates)
	if len(dates) == 0 {
		return nil, ErrNoDates
	}
	if req.Pax < 1 {
		return nil, fmt.Errorf("%w (got %d)", ErrPaxTooLow, req.Pax)
	}

	analysis := Analyze(dates, req.Rates, r)
	q := &SubscriptionQuote{Analysis: analysis, Pax: req.Pax, Days: analysis.N}

	// BR-5.1 — the group price table applies ONLY on the strict Mingguan and
	// Bulanan tiers. Every Flexi tier and Harian falls back to rate x days x pax.
	// That is not an oversight in the original: the published table only covers
	// those two commitments.
	usePaxTable := req.PaxTable != nil &&
		(analysis.TierKey == TierWeekly || analysis.TierKey == TierMonthly)

	if usePaxTable {
		rice := req.Rice
		if rice == "" {
			rice = "dengan"
		}
		if rice != "dengan" && rice != "tanpa" {
			return nil, fmt.Errorf("%w (got %q)", ErrUnknownRice, rice)
		}
		period := "weekly"
		if analysis.TierKey == TierMonthly {
			period = "monthly"
		}
		dayTotal, err := paxTableDayTotal(req.PaxTable, rice, period, req.Pax, r)
		if err != nil {
			return nil, err
		}
		q.UsedPaxTable = true
		q.RiceLabel = map[string]string{"dengan": "Dengan Nasi", "tanpa": "Tanpa Nasi"}[rice]
		q.MenuTotal = dayTotal * int64(analysis.N)
		q.EffectiveRate = roundHalfUp(dayTotal, int64(req.Pax))
	} else {
		q.EffectiveRate = analysis.Rate
		q.MenuTotal = analysis.Rate * int64(analysis.N) * int64(req.Pax)
	}

	isFlexi := analysis.TierKey == TierFlexi ||
		analysis.TierKey == TierFlexiWeekly ||
		analysis.TierKey == TierFlexiMonthly

	charges, total := chargeAddons(req.Addons, dates, req.Pax, isFlexi)
	q.AddonCharges = charges
	q.AddonTotal = total
	q.Total = q.MenuTotal + q.AddonTotal
	return q, nil
}

// paxTableDayTotal looks up the whole-group day price.
//
// Above the table's last row the rate stops deepening and extends linearly from
// it (BR-5.4): round(table[max]/max) x pax. The rounding is applied to the
// per-head rate BEFORE multiplying, exactly as the front end does -- rounding
// after would differ by a few rupiah on larger groups.
func paxTableDayTotal(table map[string]map[string]map[int]int64, rice, period string, pax int, r Rules) (int64, error) {
	byPeriod, ok := table[rice]
	if !ok {
		return 0, fmt.Errorf("pricing: no pax table for rice option %q", rice)
	}
	byPax, ok := byPeriod[period]
	if !ok {
		return 0, fmt.Errorf("pricing: no pax table for period %q", period)
	}
	if pax <= r.PaxTableMaxPax {
		v, ok := byPax[pax]
		if !ok {
			return 0, fmt.Errorf("pricing: pax table has no row for %d pax", pax)
		}
		return v, nil
	}
	top, ok := byPax[r.PaxTableMaxPax]
	if !ok {
		return 0, fmt.Errorf("pricing: pax table has no row for %d pax", r.PaxTableMaxPax)
	}
	return roundHalfUp(top, int64(r.PaxTableMaxPax)) * int64(pax), nil
}

// chargeAddons prices every add-on against the selected dates.
func chargeAddons(addons []Addon, dates []time.Time, pax int, isFlexi bool) ([]AddonCharge, int64) {
	var charges []AddonCharge
	var total int64

	for _, a := range addons {
		matched := matchingDates(a, dates)
		if len(matched) == 0 {
			// BR-6.4 -- a restricted add-on with no eligible date is
			// unavailable, not free. The front end disables and unchecks it;
			// here it simply contributes nothing and is reported as such.
			charges = append(charges, AddonCharge{
				Code: a.Code, UnitPrice: a.Price, Days: 0, EffectivePax: 0, Total: 0,
				MatchedDates: []string{},
				Note:         "no selected date falls on an allowed weekday",
			})
			continue
		}

		effectivePax := pax
		note := ""
		// BR-6.8 -- the Flexi meat cap. On any Flexi tier this add-on is capped
		// at one portion per N pax rather than one per pax.
		if a.FlexiPortionPerPax > 0 && isFlexi {
			effectivePax = pax / a.FlexiPortionPerPax
			if effectivePax > 0 {
				note = fmt.Sprintf("maks 1 porsi/%d pax → %d porsi dari %d pax",
					a.FlexiPortionPerPax, effectivePax, pax)
			} else {
				note = fmt.Sprintf("maks 1 porsi/%d pax → belum dapat porsi (min. %d pax)",
					a.FlexiPortionPerPax, a.FlexiPortionPerPax)
			}
		}

		labels := make([]string, 0, len(matched))
		for _, d := range matched {
			labels = append(labels, d.Format(time.DateOnly))
		}
		amount := a.Price * int64(effectivePax) * int64(len(matched))
		charges = append(charges, AddonCharge{
			Code: a.Code, UnitPrice: a.Price, Days: len(matched),
			EffectivePax: effectivePax, Total: amount, MatchedDates: labels, Note: note,
		})
		total += amount
	}
	return charges, total
}

// matchingDates returns the dates an add-on actually charges for: those on an
// allowed weekday, narrowed to the customer's own day selection if they made one.
func matchingDates(a Addon, dates []time.Time) []time.Time {
	allowed := map[int]bool{}
	for _, c := range a.RestrictDays {
		if c >= '0' && c <= '6' {
			allowed[int(c-'0')] = true
		}
	}
	chosen := map[string]bool{}
	for _, s := range a.Dates {
		chosen[s] = true
	}

	var out []time.Time
	for _, d := range dates {
		if len(allowed) > 0 && !allowed[int(d.Weekday())] {
			continue
		}
		if len(chosen) > 0 && !chosen[d.Format(time.DateOnly)] {
			continue
		}
		out = append(out, d)
	}
	return out
}

// ---- tier-priced products: Nasi Bento, Nasi Kuning, Paket Acara ----

// Band is one quantity band of a tier-priced product.
type Band struct {
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Price int64  `json:"price"`
	Label string `json:"label,omitempty"`
}

// TierProductRequest quotes one tier-priced card.
type TierProductRequest struct {
	Dates  []time.Time
	Qty    int
	MinQty int
	Bands  []Band
	Addons []Addon
}

// TierProductQuote is the priced result.
type TierProductQuote struct {
	Qty          int           `json:"qty"`
	Dates        int           `json:"dates"`
	Band         Band          `json:"band"`
	UnitPrice    int64         `json:"unit_price"`
	MenuTotal    int64         `json:"menu_total"`
	AddonTotal   int64         `json:"addon_total"`
	Total        int64         `json:"total"`
	AddonCharges []AddonCharge `json:"addon_charges"`
}

// QuoteTierProduct prices a Nasi Bento / Nasi Kuning / Paket Acara card.
//
// Each selected date is a full separate delivery of the same quantity
// (BR-14.1), so the total scales with how many dates were picked. The band is
// chosen by quantity alone; the number of dates never changes the unit price.
func QuoteTierProduct(req TierProductRequest, _ Rules) (*TierProductQuote, error) {
	dates := SortUnique(req.Dates)
	if len(dates) == 0 {
		return nil, ErrNoDates
	}
	qty := req.Qty
	if req.MinQty > 0 && qty < req.MinQty {
		// The front end clamps up rather than erroring, and the two must agree,
		// or the server would reject an order the page happily accepted.
		qty = req.MinQty
	}
	if qty < 1 {
		return nil, fmt.Errorf("%w (got %d)", ErrQtyBelowMin, req.Qty)
	}

	band, ok := bandFor(req.Bands, qty)
	if !ok && len(req.Bands) == 0 {
		return nil, ErrNoBandForQty
	}

	q := &TierProductQuote{
		Qty: qty, Dates: len(dates), Band: band, UnitPrice: band.Price,
		MenuTotal: band.Price * int64(qty) * int64(len(dates)),
	}
	// These cards have no Flexi tiers, so the meat cap never applies here
	// (BR-6.11) -- isFlexi is false, always.
	charges, total := chargeAddons(req.Addons, dates, qty, false)
	q.AddonCharges = charges
	q.AddonTotal = total
	q.Total = q.MenuTotal + q.AddonTotal
	return q, nil
}

// bandFor mirrors the front end's lookup INCLUDING its fallback to the last
// band when nothing matches. Diverging here would mean the server quoting one
// price and the page showing another for an out-of-range quantity.
func bandFor(bands []Band, qty int) (Band, bool) {
	if len(bands) == 0 {
		return Band{}, false
	}
	sorted := make([]Band, len(bands))
	copy(sorted, bands)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j].Min < sorted[j-1].Min; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	for _, b := range sorted {
		if qty >= b.Min && qty <= b.Max {
			return b, true
		}
	}
	return sorted[len(sorted)-1], false
}

// ---- Catering Kantor ----

// KantorRequest quotes the office-catering card.
type KantorRequest struct {
	StartDate time.Time
	Pax       int
	Days      int    // 5 for mingguan, 20 for bulanan
	Period    string // for the label only
	Bands     []Band
	Addons    []Addon
}

// KantorQuote is the priced result.
type KantorQuote struct {
	Pax          int           `json:"pax"`
	Days         int           `json:"days"`
	Period       string        `json:"period"`
	Band         Band          `json:"band"`
	RatePerPax   int64         `json:"rate_per_pax_day"`
	ServedDates  []string      `json:"served_dates"`
	MenuTotal    int64         `json:"menu_total"`
	AddonTotal   int64         `json:"addon_total"`
	Total        int64         `json:"total"`
	AddonCharges []AddonCharge `json:"addon_charges"`
}

// QuoteKantor prices Catering Kantor.
//
// The card stores only a start date and a fixed day count, so the actual
// delivery dates are walked forward from it, SKIPPING Saturday and Sunday
// (BR-13.7). A "Bulanan" package is therefore 20 weekdays -- four calendar
// weeks -- not 20 calendar days. Weekday-restricted add-ons count only the
// Rabu/Kamis that fall inside that walk.
func QuoteKantor(req KantorRequest, _ Rules) (*KantorQuote, error) {
	if req.Pax < 1 {
		return nil, fmt.Errorf("%w (got %d)", ErrPaxTooLow, req.Pax)
	}
	if req.Days < 1 {
		return nil, fmt.Errorf("pricing: kantor period must cover at least one day (got %d)", req.Days)
	}
	band, ok := bandFor(req.Bands, req.Pax)
	if !ok && len(req.Bands) == 0 {
		return nil, ErrNoBandForQty
	}

	served := kantorDates(req.StartDate, req.Days)
	labels := make([]string, 0, len(served))
	for _, d := range served {
		labels = append(labels, d.Format(time.DateOnly))
	}

	q := &KantorQuote{
		Pax: req.Pax, Days: req.Days, Period: req.Period, Band: band,
		RatePerPax:  band.Price,
		ServedDates: labels,
		MenuTotal:   band.Price * int64(req.Pax) * int64(req.Days),
	}

	// Unrestricted add-ons apply to every day of the package, not just the
	// weekdays walked out -- that is what the front end does, and the walk
	// exists only to answer "is there a Rabu/Kamis in here".
	var charges []AddonCharge
	var addonTotal int64
	for _, a := range req.Addons {
		var days int
		var matchedLabels []string
		if a.RestrictDays == "" {
			days = req.Days
		} else {
			m := matchingDates(a, served)
			days = len(m)
			for _, d := range m {
				matchedLabels = append(matchedLabels, d.Format(time.DateOnly))
			}
		}
		if days == 0 {
			charges = append(charges, AddonCharge{
				Code: a.Code, UnitPrice: a.Price, MatchedDates: []string{},
				Note: "no day in this period falls on an allowed weekday",
			})
			continue
		}
		amount := a.Price * int64(req.Pax) * int64(days)
		charges = append(charges, AddonCharge{
			Code: a.Code, UnitPrice: a.Price, Days: days, EffectivePax: req.Pax,
			Total: amount, MatchedDates: matchedLabels,
		})
		addonTotal += amount
	}
	q.AddonCharges = charges
	q.AddonTotal = addonTotal
	q.Total = q.MenuTotal + q.AddonTotal
	return q, nil
}

// kantorDates walks forward from start collecting Mon-Fri dates until it has
// `days` of them. The guard mirrors the front end's, so a pathological input
// terminates instead of looping.
func kantorDates(start time.Time, days int) []time.Time {
	if start.IsZero() || days < 1 {
		return nil
	}
	cursor := dateOnly(start)
	out := make([]time.Time, 0, days)
	for guard := 0; len(out) < days && guard < days*3+14; guard++ {
		if w := cursor.Weekday(); w != time.Saturday && w != time.Sunday {
			out = append(out, cursor)
		}
		cursor = cursor.AddDate(0, 0, 1)
	}
	return out
}
