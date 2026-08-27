// Package menu is the pure logic of the weekly menu rotation.
//
// The captured page hard-codes two weeks of menus as markup (BR-15.1), which is
// why publishing next week's menu currently means editing HTML. This package
// holds the rules that let the same content live as data instead: what a cycle
// is, which cycle covers a given delivery date, and what makes a cycle valid.
//
// No I/O, no SQL, no HTTP. Everything here is a function of its arguments.
package menu

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Weekday digits match the front end's data-restrict-days encoding and
// JavaScript's Date.getDay(): 0 = Sunday .. 6 = Saturday.
const (
	Sunday    = 0
	Monday    = 1
	Wednesday = 3
	Thursday  = 4
	Friday    = 5
	Saturday  = 6
)

// Cycle is one Monday-anchored week of menus.
type Cycle struct {
	ID        string
	ISOYear   int
	ISOWeek   int
	StartsOn  time.Time // a date; time-of-day is ignored
	EndsOn    time.Time
	Label     string
	Published bool
}

// Day is one plan's menu for one date.
type Day struct {
	PlanSlug   string
	ServeDate  time.Time
	IsMeatDay  bool
	Kcal       int
	Components []Component
}

// Component is one item on the plate, e.g. "Nasi Merah" at 100g.
type Component struct {
	Name  string
	Grams int // 0 means "no stated weight", which the page does use
}

var (
	ErrEmptyLabel      = errors.New("menu: cycle label is required")
	ErrRangeInverted   = errors.New("menu: cycle ends before it starts")
	ErrRangeTooWide    = errors.New("menu: a cycle covers at most seven days")
	ErrDayOutOfCycle   = errors.New("menu: menu day falls outside its cycle")
	ErrNoComponents    = errors.New("menu: a menu day needs at least one component")
	ErrSundayNotServed = errors.New("menu: this plan does not deliver on Sunday")
)

// DateOnly strips the time of day and pins to UTC, so two dates that mean the
// same calendar day compare equal regardless of where they were parsed.
//
// Every date in this package is a business calendar date, not an instant. The
// distinction matters: 2026-08-24 is the same menu day in Jakarta and in UTC,
// and treating it as an instant is how a menu silently shifts by one day.
func DateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// MondayOf returns the Monday of the calendar week containing t.
//
// This is the same rule the front end's analyze() uses for BR-3.13:
// offset = (dow == 0 ? 6 : dow - 1). Sunday belongs to the week that STARTED
// six days earlier, not the one about to start. Getting this backwards is the
// classic off-by-one in week arithmetic, so it is written once, here.
func MondayOf(t time.Time) time.Time {
	d := DateOnly(t)
	dow := int(d.Weekday()) // 0 = Sunday
	offset := dow - 1
	if dow == Sunday {
		offset = 6
	}
	return d.AddDate(0, 0, -offset)
}

// NewCycle builds a cycle for the week containing anchor, running Monday to
// Friday -- which is what the business actually publishes (BR-15.2).
func NewCycle(anchor time.Time, label string) Cycle {
	monday := MondayOf(anchor)
	year, week := monday.ISOWeek()
	return Cycle{
		ISOYear:  year,
		ISOWeek:  week,
		StartsOn: monday,
		EndsOn:   monday.AddDate(0, 0, 4), // Friday
		Label:    label,
	}
}

// Covers reports whether the cycle contains the given delivery date, inclusive
// at both ends.
func (c Cycle) Covers(date time.Time) bool {
	d := DateOnly(date)
	return !d.Before(DateOnly(c.StartsOn)) && !d.After(DateOnly(c.EndsOn))
}

// Validate checks the cycle's own consistency.
func (c Cycle) Validate() error {
	if c.Label == "" {
		return ErrEmptyLabel
	}
	start, end := DateOnly(c.StartsOn), DateOnly(c.EndsOn)
	if end.Before(start) {
		return ErrRangeInverted
	}
	if end.Sub(start) > 6*24*time.Hour {
		return ErrRangeTooWide
	}
	return nil
}

// ValidateDay checks one menu day against its cycle and its plan.
//
// deliversSunday comes from the plan: Healthy Meal and Bulking Extra never
// deliver on Sunday (BR-7.6), so a Sunday menu for those plans is data that can
// never be served and is rejected rather than stored.
func (c Cycle) ValidateDay(d Day, deliversSunday bool) error {
	if !c.Covers(d.ServeDate) {
		return fmt.Errorf("%w: %s not in %s..%s", ErrDayOutOfCycle,
			DateOnly(d.ServeDate).Format(time.DateOnly),
			DateOnly(c.StartsOn).Format(time.DateOnly),
			DateOnly(c.EndsOn).Format(time.DateOnly))
	}
	if len(d.Components) == 0 {
		return ErrNoComponents
	}
	if !deliversSunday && DateOnly(d.ServeDate).Weekday() == time.Sunday {
		return fmt.Errorf("%w: %s", ErrSundayNotServed, d.PlanSlug)
	}
	return nil
}

// Pick returns the cycle covering date, and whether one was found.
//
// Cycles are expected to be non-overlapping -- the schema enforces that with an
// exclusion constraint for published cycles -- but this does not assume it:
// it returns the earliest-starting match so the result is deterministic even if
// bad data gets in.
func Pick(cycles []Cycle, date time.Time) (Cycle, bool) {
	var found []Cycle
	for _, c := range cycles {
		if c.Covers(date) {
			found = append(found, c)
		}
	}
	if len(found) == 0 {
		return Cycle{}, false
	}
	sort.Slice(found, func(i, j int) bool {
		return DateOnly(found[i].StartsOn).Before(DateOnly(found[j].StartsOn))
	})
	return found[0], true
}

// Current returns the cycle covering today, else the next one starting after
// today, else the most recent past one. The front end shows "menu minggu ini"
// and "menu minggu depan", so it needs both a current and a following cycle
// even when today falls in a gap -- a weekend, or a week nobody published.
func Current(cycles []Cycle, today time.Time) (current Cycle, next Cycle, ok bool) {
	if len(cycles) == 0 {
		return Cycle{}, Cycle{}, false
	}
	sorted := append([]Cycle(nil), cycles...)
	sort.Slice(sorted, func(i, j int) bool {
		return DateOnly(sorted[i].StartsOn).Before(DateOnly(sorted[j].StartsOn))
	})

	t := DateOnly(today)
	idx := -1
	for i, c := range sorted {
		if c.Covers(t) {
			idx = i
			break
		}
	}
	if idx == -1 {
		// No cycle covers today. Take the first one starting in the future.
		for i, c := range sorted {
			if DateOnly(c.StartsOn).After(t) {
				idx = i
				break
			}
		}
	}
	if idx == -1 {
		// Everything is in the past; the newest is the best we can offer.
		idx = len(sorted) - 1
	}

	current = sorted[idx]
	if idx+1 < len(sorted) {
		next = sorted[idx+1]
		return current, next, true
	}
	return current, Cycle{}, true
}
