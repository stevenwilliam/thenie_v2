package menu

import (
	"errors"
	"testing"
	"time"
)

func d(s string) time.Time {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return t
}

// 2026-08-17 is a Monday -- the captured page labels it "Senin 17 Agu", and
// tests/pricing.test.js asserts the same fact. Every date below hangs off it.
func TestAnchorDateIsAMonday(t *testing.T) {
	if got := d("2026-08-17").Weekday(); got != time.Monday {
		t.Fatalf("2026-08-17 must be a Monday, got %v", got)
	}
}

func TestMondayOf(t *testing.T) {
	// The whole week 17-23 Aug 2026 resolves to Monday the 17th, INCLUDING
	// Sunday the 23rd. Sunday belonging to the week that started six days
	// earlier is BR-3.13, and it is the easiest thing in this file to get
	// backwards.
	for _, day := range []string{
		"2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20",
		"2026-08-21", "2026-08-22", "2026-08-23",
	} {
		if got := MondayOf(d(day)); !got.Equal(d("2026-08-17")) {
			t.Errorf("MondayOf(%s) = %s, want 2026-08-17", day, got.Format(time.DateOnly))
		}
	}
	// And Monday the 24th starts the next week, not the previous one.
	if got := MondayOf(d("2026-08-24")); !got.Equal(d("2026-08-24")) {
		t.Errorf("MondayOf(2026-08-24) = %s, want 2026-08-24", got.Format(time.DateOnly))
	}
}

// MondayOf must not depend on the machine's timezone. This project runs in
// Asia/Jakarta (UTC+7), where a naive implementation that round-trips through
// UTC shifts every date back by one -- the exact trap documented in
// tests/README.md for the JavaScript side.
func TestMondayOfIsTimezoneIndependent(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Skipf("tzdata unavailable: %v", err)
	}
	for _, loc := range []*time.Location{time.UTC, jakarta, time.FixedZone("UTC-11", -11*3600)} {
		local := time.Date(2026, 8, 23, 0, 30, 0, 0, loc) // a Sunday, just after midnight
		if got := MondayOf(local); !got.Equal(d("2026-08-17")) {
			t.Errorf("in %s: got %s, want 2026-08-17", loc, got.Format(time.DateOnly))
		}
	}
}

func TestNewCycleSpansMondayToFriday(t *testing.T) {
	// Built from a Wednesday: the cycle must still start on that week's Monday.
	c := NewCycle(d("2026-08-19"), "Minggu ke-34 · 17–21 Agustus 2026")
	if !c.StartsOn.Equal(d("2026-08-17")) {
		t.Errorf("StartsOn = %s, want 2026-08-17", c.StartsOn.Format(time.DateOnly))
	}
	if !c.EndsOn.Equal(d("2026-08-21")) {
		t.Errorf("EndsOn = %s, want 2026-08-21", c.EndsOn.Format(time.DateOnly))
	}
	if c.ISOWeek != 34 || c.ISOYear != 2026 {
		t.Errorf("got %d-W%02d, want 2026-W34", c.ISOYear, c.ISOWeek)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("a freshly built cycle must validate: %v", err)
	}
}

func TestCycleCovers(t *testing.T) {
	c := NewCycle(d("2026-08-17"), "week 34")
	for _, in := range []string{"2026-08-17", "2026-08-19", "2026-08-21"} {
		if !c.Covers(d(in)) {
			t.Errorf("%s must be covered", in)
		}
	}
	for _, out := range []string{"2026-08-16", "2026-08-22", "2026-08-24"} {
		if c.Covers(d(out)) {
			t.Errorf("%s must not be covered", out)
		}
	}
}

func TestCycleValidate(t *testing.T) {
	good := NewCycle(d("2026-08-17"), "week 34")

	if err := (Cycle{StartsOn: good.StartsOn, EndsOn: good.EndsOn}).Validate(); !errors.Is(err, ErrEmptyLabel) {
		t.Errorf("empty label: got %v", err)
	}
	inverted := good
	inverted.EndsOn = d("2026-08-16")
	if err := inverted.Validate(); !errors.Is(err, ErrRangeInverted) {
		t.Errorf("inverted range: got %v", err)
	}
	wide := good
	wide.EndsOn = d("2026-08-25") // 9 days
	if err := wide.Validate(); !errors.Is(err, ErrRangeTooWide) {
		t.Errorf("over-wide range: got %v", err)
	}
}

func TestValidateDay(t *testing.T) {
	c := NewCycle(d("2026-08-17"), "week 34")
	ok := Day{
		PlanSlug:   "healthy",
		ServeDate:  d("2026-08-20"),
		IsMeatDay:  true,
		Kcal:       470,
		Components: []Component{{Name: "Kentang Rebus", Grams: 120}},
	}
	if err := c.ValidateDay(ok, false); err != nil {
		t.Fatalf("a valid Thursday menu must pass: %v", err)
	}

	outside := ok
	outside.ServeDate = d("2026-08-24")
	if err := c.ValidateDay(outside, false); !errors.Is(err, ErrDayOutOfCycle) {
		t.Errorf("out-of-cycle: got %v", err)
	}

	bare := ok
	bare.Components = nil
	if err := c.ValidateDay(bare, false); !errors.Is(err, ErrNoComponents) {
		t.Errorf("no components: got %v", err)
	}

	// BR-7.6: Healthy Meal and Bulking Extra never deliver on Sunday, so a
	// Sunday menu for them is unservable data. A plan that does deliver on
	// Sunday is unaffected.
	sundayCycle := Cycle{Label: "wk", StartsOn: d("2026-08-17"), EndsOn: d("2026-08-23")}
	sunday := ok
	sunday.ServeDate = d("2026-08-23")
	if err := sundayCycle.ValidateDay(sunday, false); !errors.Is(err, ErrSundayNotServed) {
		t.Errorf("Sunday for a no-Sunday plan must be rejected: got %v", err)
	}
	if err := sundayCycle.ValidateDay(sunday, true); err != nil {
		t.Errorf("Sunday for a Sunday-serving plan must be accepted: got %v", err)
	}
}

func TestPick(t *testing.T) {
	w34 := NewCycle(d("2026-08-17"), "34")
	w35 := NewCycle(d("2026-08-24"), "35")
	cycles := []Cycle{w35, w34} // deliberately out of order

	got, ok := Pick(cycles, d("2026-08-25"))
	if !ok || got.Label != "35" {
		t.Errorf("25 Aug should pick week 35, got %q ok=%v", got.Label, ok)
	}
	if _, ok := Pick(cycles, d("2026-08-22")); ok {
		t.Error("a Saturday outside both Mon-Fri cycles must find nothing")
	}
	if _, ok := Pick(nil, d("2026-08-20")); ok {
		t.Error("no cycles must find nothing")
	}
}

func TestCurrent(t *testing.T) {
	w34 := NewCycle(d("2026-08-17"), "34")
	w35 := NewCycle(d("2026-08-24"), "35")
	cycles := []Cycle{w35, w34}

	// Inside week 34: current is 34, next is 35.
	cur, next, ok := Current(cycles, d("2026-08-19"))
	if !ok || cur.Label != "34" || next.Label != "35" {
		t.Fatalf("got cur=%q next=%q ok=%v, want 34/35", cur.Label, next.Label, ok)
	}

	// On the weekend gap between them, no cycle covers today, so the next one
	// starting in the future becomes current. This is the case that makes the
	// page still show a menu on a Saturday.
	cur, next, ok = Current(cycles, d("2026-08-22"))
	if !ok || cur.Label != "35" {
		t.Fatalf("weekend gap: got cur=%q ok=%v, want 35", cur.Label, ok)
	}
	if next.Label != "" {
		t.Errorf("nothing follows week 35, got next=%q", next.Label)
	}

	// Everything in the past: fall back to the most recent rather than nothing,
	// so the page degrades to a stale menu instead of an empty one.
	cur, _, ok = Current(cycles, d("2027-01-01"))
	if !ok || cur.Label != "35" {
		t.Fatalf("all past: got cur=%q ok=%v, want 35", cur.Label, ok)
	}

	if _, _, ok := Current(nil, d("2026-08-19")); ok {
		t.Error("no cycles must report not-ok")
	}
}
