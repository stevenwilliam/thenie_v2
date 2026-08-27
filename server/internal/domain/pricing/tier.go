package pricing

import (
	"sort"
	"time"
)

// TierKey identifies which of the six rates an order qualifies for.
type TierKey string

const (
	TierDaily        TierKey = "daily"
	TierWeekly       TierKey = "weekly"
	TierMonthly      TierKey = "monthly"
	TierFlexi        TierKey = "flexi"
	TierFlexiWeekly  TierKey = "flexi-weekly"
	TierFlexiMonthly TierKey = "flexi-monthly"
)

// Rates are the five per-pax-per-day rates a plan is sold at.
type Rates struct {
	Daily              int64 `json:"daily"`
	WeeklyPerDay       int64 `json:"weeklyPerDay"`
	MonthlyPerDay      int64 `json:"monthlyPerDay"`
	FlexiWeeklyPerDay  int64 `json:"flexiWeeklyPerDay"`
	FlexiMonthlyPerDay int64 `json:"flexiMonthlyPerDay"`
}

// Analysis is what the classifier decided and why.
type Analysis struct {
	N           int     `json:"n"`
	Consecutive bool    `json:"consecutive"`
	Span        int     `json:"span"`
	TierKey     TierKey `json:"tier_key"`
	TierLabel   string  `json:"tier_label"`
	Rate        int64   `json:"rate"`
}

const dayHours = 24 * time.Hour

// Analyze classifies a set of delivery dates into a pricing tier.
//
// This is a direct port of analyze() in the captured page. The branch ORDER is
// load-bearing and is preserved exactly: several conditions overlap, and the
// first match wins. Reordering them silently reprices orders.
//
// dates must be sorted ascending and unique -- the front end guarantees that by
// construction (it sorts on every add), and Quote() enforces it before calling.
func Analyze(dates []time.Time, rates Rates, r Rules) *Analysis {
	n := len(dates)
	if n == 0 {
		return nil
	}

	consecutive := true
	for i := 1; i < n; i++ {
		if daysBetween(dates[i-1], dates[i]) != 1 {
			consecutive = false
			break
		}
	}
	span := daysBetween(dates[0], dates[n-1]) + 1

	// A consecutive 5-19 day run that straddles two calendar weeks is not one
	// clean Mingguan commitment even though every date is back-to-back -- it is
	// the tail of one week plus the start of the next. It still gets a discount,
	// just not the full Mingguan rate. Only computed for the 5..(monthly-1)
	// range: a run at or above monthly_min_days is Bulanan by definition and
	// naturally spans weeks, so it is exempt.
	consecutiveCrossesWeek := false
	if consecutive && n >= r.WeeklyMinDays && n < r.MonthlyMinDays {
		consecutiveCrossesWeek = !mondayOf(dates[0]).Equal(mondayOf(dates[n-1]))
	}

	isWeekdayRoutine := false
	isWeeklyRoutine := false
	if !consecutive && n >= r.WeeklyMinDays {
		isMonFri, isMonSat := true, true
		for _, d := range dates {
			w := int(d.Weekday())
			if w < 1 || w > 5 {
				isMonFri = false
			}
			if w < 1 || w > 6 {
				isMonSat = false
			}
		}
		cleanWeekdaySet := isMonFri || isMonSat

		// A clean Mon-Fri / Mon-Sat routine of monthly_min_days or more, whose
		// gaps are only ever the next working day or the expected weekend skip,
		// inside one month: that is an ordinary weekday customer, and charging
		// them the scattered-order rate for skipping weekends would be absurd.
		if cleanWeekdaySet && n >= r.MonthlyMinDays {
			cleanGaps := true
			for i := 1; i < n; i++ {
				diff := daysBetween(dates[i-1], dates[i])
				var ok bool
				if isMonFri {
					ok = diff == 1 || diff == 3 // Fri -> Mon is 3
				} else {
					ok = diff == 1 || diff == 2 // Sat -> Mon is 2
				}
				if !ok {
					cleanGaps = false
					break
				}
			}
			if cleanGaps && span <= r.WeekdayRoutineMaxSpanDays {
				isWeekdayRoutine = true
			}
		}

		// One tier down: an order that starts mid-week and continues into the
		// next. Mingguan additionally requires that at least one real calendar
		// week inside the order holds weekly_routine_min_days_in_week dates --
		// otherwise 18,19,20,21,24 Aug (4 days in one week, 1 in the next) would
		// qualify on day count alone without any week actually being full.
		if !isWeekdayRoutine && cleanWeekdaySet && span <= r.WeeklyRoutineMaxSpanDays {
			counts := map[time.Time]int{}
			for _, d := range dates {
				counts[mondayOf(d)]++
			}
			max := 0
			for _, c := range counts {
				if c > max {
					max = c
				}
			}
			if max >= r.WeeklyRoutineMinDaysInWeek {
				isWeeklyRoutine = true
			}
		}
	}

	// The branch ladder, in the shipped order. Do not reorder.
	switch {
	case consecutive && n >= r.MonthlyMinDays:
		return mk(n, consecutive, span, TierMonthly,
			label("Bulanan (min. %d hari)", r.MonthlyMinDays), rates.MonthlyPerDay)

	case consecutive && n >= r.WeeklyMinDays &&
		n <= r.ConsecutiveFlexiWeeklyMaxDays && consecutiveCrossesWeek:
		return mk(n, consecutive, span, TierFlexiWeekly,
			label("Flexi Mingguan (berturutan tapi lintas minggu kalender, maks %d hari)",
				r.ConsecutiveFlexiWeeklyMaxDays), rates.FlexiWeeklyPerDay)

	case consecutive && n > r.ConsecutiveFlexiWeeklyMaxDays && n < r.MonthlyMinDays:
		// Flexi Mingguan only applies up to its ceiling. Past it, and short of
		// Bulanan, the price goes back to full daily rate -- so a 19-day run
		// costs more per day than a 5-day one. Surprising, deliberate, and
		// raised as Q-7 in docs/09-open-questions.md.
		return mk(n, consecutive, span, TierFlexi,
			label("Flexi (berturutan >%d hari, belum genap Bulanan)",
				r.ConsecutiveFlexiWeeklyMaxDays), rates.Daily)

	case consecutive && n >= r.WeeklyMinDays:
		return mk(n, consecutive, span, TierWeekly,
			label("Mingguan (min. %d hari)", r.WeeklyMinDays), rates.WeeklyPerDay)

	case consecutive:
		return mk(n, consecutive, span, TierDaily, "Harian", rates.Daily)

	case isWeekdayRoutine:
		return mk(n, consecutive, span, TierMonthly,
			label("Bulanan (%d hari Sen–Jum/Sen–Sab dalam 1 bulan)", r.MonthlyMinDays),
			rates.MonthlyPerDay)

	case isWeeklyRoutine:
		return mk(n, consecutive, span, TierWeekly,
			label("Mingguan (%d+ hari Sen–Jum/Sen–Sab, nyambung antar minggu)", r.WeeklyMinDays),
			rates.WeeklyPerDay)

	case n >= r.MonthlyMinDays && span <= r.FlexiMonthlyMaxSpanDays:
		return mk(n, consecutive, span, TierFlexiMonthly,
			label("Flexi Bulanan (%d+ hari / %d hari)", r.MonthlyMinDays, r.FlexiMonthlyMaxSpanDays),
			rates.FlexiMonthlyPerDay)

	case n >= r.WeeklyMinDays && n < r.MonthlyMinDays:
		return mk(n, consecutive, span, TierFlexiWeekly,
			label("Flexi Mingguan (%d–%d hari, tanggal tidak berurutan)",
				r.WeeklyMinDays, r.MonthlyMinDays-1), rates.FlexiWeeklyPerDay)

	default:
		return mk(n, consecutive, span, TierFlexi, "Flexi (tanggal acak)", rates.Daily)
	}
}

// mondayOf returns the Monday of the calendar week containing d.
//
// Sunday belongs to the week that STARTED six days earlier (BR-3.13). This is
// the single easiest thing in the file to get backwards, so it is written once
// and shared by both routine checks.
func mondayOf(d time.Time) time.Time {
	d = dateOnly(d)
	dow := int(d.Weekday())
	offset := dow - 1
	if dow == 0 {
		offset = 6
	}
	return d.AddDate(0, 0, -offset)
}

// dateOnly pins a value to a calendar date in UTC. Every date here is a business
// date, not an instant: 2026-08-24 is the same delivery day in Jakarta and in
// UTC, and treating it as an instant is how a whole schedule shifts by a day.
func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// daysBetween counts whole calendar days from a to b.
func daysBetween(a, b time.Time) int {
	return int(dateOnly(b).Sub(dateOnly(a)) / dayHours)
}

// SortUnique returns the dates sorted ascending with duplicates removed, which
// is the shape Analyze requires. The front end maintains this invariant as the
// customer taps; an API caller may not, so Quote() normalises first.
func SortUnique(in []time.Time) []time.Time {
	if len(in) == 0 {
		return nil
	}
	out := make([]time.Time, 0, len(in))
	for _, d := range in {
		out = append(out, dateOnly(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	deduped := out[:1]
	for _, d := range out[1:] {
		if !d.Equal(deduped[len(deduped)-1]) {
			deduped = append(deduped, d)
		}
	}
	return deduped
}

func mk(n int, consecutive bool, span int, key TierKey, lbl string, rate int64) *Analysis {
	return &Analysis{N: n, Consecutive: consecutive, Span: span,
		TierKey: key, TierLabel: lbl, Rate: rate}
}

func label(format string, args ...any) string { return sprintf(format, args...) }
