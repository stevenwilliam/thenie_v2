// Package catalogue is the pure logic of the price catalogue.
//
// Its job is to refuse a rate table that would break the front end. The order
// page ships a pricing engine (analyze(), documented as BR-3.1 – BR-3.13) that
// assumes a shape: the discounted tiers are cheaper than list, and the Flexi
// tiers sit between the discounted tier and list. Feeding it a table that
// violates that produces a page where committing to twenty days costs more than
// buying five, with no error anywhere.
//
// tests/pricing.test.js asserts these invariants against the captured page. The
// same rules are enforced here, and again as CHECK constraints in migration
// 0002, so a bad edit is stopped at the API, at the domain, and at the database.
package catalogue

import (
	"errors"
	"fmt"
	"sort"
)

// Rates are the five per-pax-per-day rates a subscription plan is sold at.
// All values are whole rupiah.
type Rates struct {
	Daily              int64
	WeeklyPerDay       int64
	MonthlyPerDay      int64
	FlexiWeeklyPerDay  int64
	FlexiMonthlyPerDay int64
}

var (
	ErrRatePositive   = errors.New("catalogue: every rate must be greater than zero")
	ErrRateMonotonic  = errors.New("catalogue: monthly must not exceed weekly, and weekly must not exceed daily")
	ErrFlexiCheaper   = errors.New("catalogue: a Flexi rate must not be cheaper than the tier it shadows")
	ErrFlexiOverList  = errors.New("catalogue: Flexi Mingguan must not exceed the daily list rate")
	ErrBandInverted   = errors.New("catalogue: a quantity band ends below where it starts")
	ErrBandOverlap    = errors.New("catalogue: quantity bands overlap")
	ErrBandGap        = errors.New("catalogue: quantity bands leave a gap")
	ErrBandEmpty      = errors.New("catalogue: at least one quantity band is required")
	ErrBandNotFromMin = errors.New("catalogue: the first quantity band must start at the product minimum")
	ErrRestrictDays   = errors.New("catalogue: restrict_days must contain only the digits 0-6")
	ErrRestrictDupes  = errors.New("catalogue: restrict_days repeats a weekday")
)

// Validate enforces the invariants the front end's pricing engine depends on.
func (r Rates) Validate() error {
	for name, v := range map[string]int64{
		"daily":                 r.Daily,
		"weekly_per_day":        r.WeeklyPerDay,
		"monthly_per_day":       r.MonthlyPerDay,
		"flexi_weekly_per_day":  r.FlexiWeeklyPerDay,
		"flexi_monthly_per_day": r.FlexiMonthlyPerDay,
	} {
		if v <= 0 {
			return fmt.Errorf("%w: %s = %d", ErrRatePositive, name, v)
		}
	}
	if r.MonthlyPerDay > r.WeeklyPerDay || r.WeeklyPerDay > r.Daily {
		return fmt.Errorf("%w: monthly=%d weekly=%d daily=%d",
			ErrRateMonotonic, r.MonthlyPerDay, r.WeeklyPerDay, r.Daily)
	}
	// A Flexi tier is the consolation rate for a less committed order. If it
	// undercut the tier it shadows, a customer would be better off scattering
	// their dates -- the opposite of what the discount is for.
	if r.FlexiMonthlyPerDay < r.MonthlyPerDay {
		return fmt.Errorf("%w: flexi_monthly=%d < monthly=%d",
			ErrFlexiCheaper, r.FlexiMonthlyPerDay, r.MonthlyPerDay)
	}
	if r.FlexiWeeklyPerDay < r.WeeklyPerDay {
		return fmt.Errorf("%w: flexi_weekly=%d < weekly=%d",
			ErrFlexiCheaper, r.FlexiWeeklyPerDay, r.WeeklyPerDay)
	}
	if r.FlexiWeeklyPerDay > r.Daily {
		return fmt.Errorf("%w: flexi_weekly=%d > daily=%d",
			ErrFlexiOverList, r.FlexiWeeklyPerDay, r.Daily)
	}
	return nil
}

// DiscountBps reports each tier's discount off the daily list rate, in basis
// points, rounded half-up with integer arithmetic only.
//
// Percentages are basis points and the rounding is explicit:
// floor((amount * 10000 + half) / daily). No float ever touches a price.
func (r Rates) DiscountBps() map[string]int64 {
	off := func(rate int64) int64 {
		if r.Daily == 0 {
			return 0
		}
		// (1 - rate/daily) in bps, rounded half-up.
		return (10000*(r.Daily-rate)*2 + r.Daily) / (2 * r.Daily)
	}
	return map[string]int64{
		"weekly":        off(r.WeeklyPerDay),
		"monthly":       off(r.MonthlyPerDay),
		"flexi_weekly":  off(r.FlexiWeeklyPerDay),
		"flexi_monthly": off(r.FlexiMonthlyPerDay),
	}
}

// Band is one quantity tier of a tier-priced product (Nasi Bento, Nasi Kuning,
// Paket Acara) or of Catering Kantor's pax table.
type Band struct {
	Min   int
	Max   int
	Price int64
	Label string
}

// ValidateBands checks that bands form one unbroken, non-overlapping ladder
// starting at minQty.
//
// A gap is the dangerous case and the reason this exists: the front end picks a
// band with `tiers.find(t => qty >= t.min && qty <= t.max)` and falls back to
// the LAST band when nothing matches. So a table missing 51-100 does not error
// -- it silently charges a 60-box order at the 200+ price. The database cannot
// express "no gaps" as a CHECK, so it is enforced here.
func ValidateBands(bands []Band, minQty int) error {
	if len(bands) == 0 {
		return ErrBandEmpty
	}
	sorted := append([]Band(nil), bands...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Min < sorted[j].Min })

	if sorted[0].Min != minQty {
		return fmt.Errorf("%w: first band starts at %d, product minimum is %d",
			ErrBandNotFromMin, sorted[0].Min, minQty)
	}
	for i, b := range sorted {
		if b.Max < b.Min {
			return fmt.Errorf("%w: [%d..%d]", ErrBandInverted, b.Min, b.Max)
		}
		if b.Price <= 0 {
			return fmt.Errorf("%w: band [%d..%d] price %d", ErrRatePositive, b.Min, b.Max, b.Price)
		}
		if i == 0 {
			continue
		}
		prev := sorted[i-1]
		if b.Min <= prev.Max {
			return fmt.Errorf("%w: [%d..%d] and [%d..%d]",
				ErrBandOverlap, prev.Min, prev.Max, b.Min, b.Max)
		}
		if b.Min != prev.Max+1 {
			return fmt.Errorf("%w: nothing covers %d..%d", ErrBandGap, prev.Max+1, b.Min-1)
		}
	}
	return nil
}

// BandFor returns the band covering qty, mirroring the front end's lookup
// exactly -- including its fallback to the last band -- so the server and the
// page can never disagree about which price applies.
func BandFor(bands []Band, qty int) (Band, bool) {
	if len(bands) == 0 {
		return Band{}, false
	}
	sorted := append([]Band(nil), bands...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Min < sorted[j].Min })
	for _, b := range sorted {
		if qty >= b.Min && qty <= b.Max {
			return b, true
		}
	}
	return sorted[len(sorted)-1], false
}

// ValidateRestrictDays checks an add-on's weekday restriction string. The
// encoding is the front end's: digit characters, 0 = Sunday .. 6 = Saturday,
// empty meaning "any day".
func ValidateRestrictDays(s string) error {
	seen := map[rune]bool{}
	for _, c := range s {
		if c < '0' || c > '6' {
			return fmt.Errorf("%w: %q", ErrRestrictDays, s)
		}
		if seen[c] {
			return fmt.Errorf("%w: %q", ErrRestrictDupes, s)
		}
		seen[c] = true
	}
	return nil
}
