package pricing

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

type fixture struct {
	Rates map[string]Rates `json:"rates"`
	Cases []struct {
		Name  string   `json:"name"`
		Plan  string   `json:"plan"`
		Dates []string `json:"dates"`
		Want  struct {
			N           int     `json:"n"`
			Consecutive bool    `json:"consecutive"`
			Span        int     `json:"span"`
			TierKey     TierKey `json:"tier_key"`
			Rate        int64   `json:"rate"`
		} `json:"want"`
	} `json:"cases"`
}

func loadFixture(t *testing.T) fixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/tier_cases.json")
	if err != nil {
		t.Fatalf("read fixture: %v (regenerate: node scripts/gen-tier-cases.js > server/internal/domain/pricing/testdata/tier_cases.json)", err)
	}
	var f fixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	if len(f.Cases) == 0 {
		t.Fatal("fixture has no cases")
	}
	return f
}

func mustDates(t *testing.T, in []string) []time.Time {
	t.Helper()
	out := make([]time.Time, 0, len(in))
	for _, s := range in {
		d, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t.Fatalf("bad date %q: %v", s, err)
		}
		out = append(out, d)
	}
	return out
}

// TestMatchesShippedEngine is the whole reason this package can be trusted.
//
// The fixture was produced by running the REAL analyze() out of site/index.html
// — the code the customer's browser actually executes — over 758 date shapes
// spanning every branch of the ladder. This asserts the Go port agrees on all
// of them: the tier, the rate, the day count, the consecutive flag and the span.
//
// If this fails after a re-capture, upstream changed the pricing rules and the
// port has to follow. If it fails after a change here, the port drifted. Either
// way it fails loudly instead of quoting one number and charging another.
func TestMatchesShippedEngine(t *testing.T) {
	f := loadFixture(t)
	rules := Default()
	var checked int

	for _, tc := range f.Cases {
		rates, ok := f.Rates[tc.Plan]
		if !ok {
			t.Fatalf("%s: fixture has no rates for plan %q", tc.Name, tc.Plan)
		}
		got := Analyze(mustDates(t, tc.Dates), rates, rules)
		if got == nil {
			t.Errorf("%s: got nil analysis", tc.Name)
			continue
		}
		if got.TierKey != tc.Want.TierKey {
			t.Errorf("%s (%d dates): tier = %s, shipped engine says %s",
				tc.Name, len(tc.Dates), got.TierKey, tc.Want.TierKey)
		}
		if got.Rate != tc.Want.Rate {
			t.Errorf("%s: rate = %d, shipped engine says %d", tc.Name, got.Rate, tc.Want.Rate)
		}
		if got.N != tc.Want.N || got.Consecutive != tc.Want.Consecutive || got.Span != tc.Want.Span {
			t.Errorf("%s: got n=%d consecutive=%v span=%d, shipped engine says n=%d consecutive=%v span=%d",
				tc.Name, got.N, got.Consecutive, got.Span,
				tc.Want.N, tc.Want.Consecutive, tc.Want.Span)
		}
		checked++
	}
	t.Logf("agreed with the shipped engine on %d/%d cases", checked, len(f.Cases))
}

// The fixture must keep exercising every branch. A regeneration that silently
// stopped covering, say, flexi-monthly would make the test above pass while
// proving much less.
func TestFixtureCoversEveryTier(t *testing.T) {
	f := loadFixture(t)
	seen := map[TierKey]int{}
	for _, tc := range f.Cases {
		seen[tc.Want.TierKey]++
	}
	for _, want := range []TierKey{TierDaily, TierWeekly, TierMonthly, TierFlexi, TierFlexiWeekly, TierFlexiMonthly} {
		if seen[want] == 0 {
			t.Errorf("fixture never exercises tier %q", want)
		}
	}
	t.Logf("coverage: %v", seen)
}

func TestAnalyzeEmpty(t *testing.T) {
	if got := Analyze(nil, Rates{Daily: 1}, Default()); got != nil {
		t.Errorf("no dates must produce no analysis, got %+v", got)
	}
}

func TestMondayOf(t *testing.T) {
	d := func(s string) time.Time {
		v, _ := time.Parse(time.DateOnly, s)
		return v
	}
	// The whole week 17-23 Aug 2026 resolves to Monday the 17th, Sunday
	// included: Sunday belongs to the week that started six days earlier.
	for _, day := range []string{
		"2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20",
		"2026-08-21", "2026-08-22", "2026-08-23",
	} {
		if got := mondayOf(d(day)); !got.Equal(d("2026-08-17")) {
			t.Errorf("mondayOf(%s) = %s, want 2026-08-17", day, got.Format(time.DateOnly))
		}
	}
}

func TestSortUnique(t *testing.T) {
	in := mustDates(t, []string{"2026-08-20", "2026-08-17", "2026-08-20", "2026-08-18"})
	got := SortUnique(in)
	want := []string{"2026-08-17", "2026-08-18", "2026-08-20"}
	if len(got) != len(want) {
		t.Fatalf("got %d dates, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Format(time.DateOnly) != w {
			t.Errorf("[%d] = %s, want %s", i, got[i].Format(time.DateOnly), w)
		}
	}
}

// Changing a rule must actually change the outcome -- otherwise the whole
// point of lifting the thresholds out of the code is lost.
func TestRulesActuallyDrive(t *testing.T) {
	dates := mustDates(t, []string{
		"2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20", "2026-08-21",
	})
	rates := Rates{Daily: 50000, WeeklyPerDay: 38000, MonthlyPerDay: 35000,
		FlexiWeeklyPerDay: 40000, FlexiMonthlyPerDay: 38000}

	if got := Analyze(dates, rates, Default()); got.TierKey != TierWeekly {
		t.Fatalf("5 consecutive days should be weekly by default, got %s", got.TierKey)
	}

	// Raise the Mingguan minimum to 6 and the same order drops to Harian.
	stricter := Default()
	stricter.WeeklyMinDays = 6
	if got := Analyze(dates, rates, stricter); got.TierKey != TierDaily {
		t.Errorf("with weekly_min_days=6, 5 days should be daily, got %s", got.TierKey)
	}
	// Lower the Bulanan minimum to 5 and it jumps to Bulanan.
	looser := Default()
	looser.MonthlyMinDays = 5
	looser.ConsecutiveFlexiWeeklyMaxDays = 4
	if got := Analyze(dates, rates, looser); got.TierKey != TierMonthly {
		t.Errorf("with monthly_min_days=5, 5 days should be monthly, got %s", got.TierKey)
	}
}

func TestRulesValidate(t *testing.T) {
	if err := Default().Validate(); err != nil {
		t.Fatalf("the shipped defaults must validate: %v", err)
	}
	for name, mut := range map[string]func(*Rules){
		"weekly below 1":              func(r *Rules) { r.WeeklyMinDays = 0 },
		"monthly not above weekly":    func(r *Rules) { r.MonthlyMinDays = r.WeeklyMinDays },
		"flexi ceiling below weekly":  func(r *Rules) { r.ConsecutiveFlexiWeeklyMaxDays = 2 },
		"flexi ceiling above monthly": func(r *Rules) { r.ConsecutiveFlexiWeeklyMaxDays = 25 },
		"flexi span below monthly":    func(r *Rules) { r.FlexiMonthlyMaxSpanDays = 10 },
		"weekday span below monthly":  func(r *Rules) { r.WeekdayRoutineMaxSpanDays = 10 },
		"weekly span below weekly":    func(r *Rules) { r.WeeklyRoutineMaxSpanDays = 2 },
		"week count out of range":     func(r *Rules) { r.WeeklyRoutineMinDaysInWeek = 8 },
		"pax table below 1":           func(r *Rules) { r.PaxTableMaxPax = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			r := Default()
			mut(&r)
			if err := r.Validate(); err == nil {
				t.Errorf("%s must be rejected", name)
			}
		})
	}
}
