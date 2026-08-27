package pricing

import (
	"testing"
	"time"
)

func dates(t *testing.T, ss ...string) []time.Time { return mustDates(t, ss) }

var healthy = Rates{Daily: 50000, WeeklyPerDay: 38000, MonthlyPerDay: 35000,
	FlexiWeeklyPerDay: 40000, FlexiMonthlyPerDay: 38000}

// docs/04-pricing-catalogue.md §1 and the JS suite's BR-2.1 case.
func TestSubscriptionTotal(t *testing.T) {
	q, err := QuoteSubscription(SubscriptionRequest{
		Dates: dates(t, "2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20", "2026-08-21"),
		Pax:   3, Rates: healthy,
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if q.Analysis.TierKey != TierWeekly {
		t.Errorf("tier = %s, want weekly", q.Analysis.TierKey)
	}
	// 38,000 x 5 days x 3 pax
	if q.Total != 570000 {
		t.Errorf("total = %d, want 570000", q.Total)
	}
	if q.EffectiveRate != 38000 {
		t.Errorf("effective rate = %d, want 38000", q.EffectiveRate)
	}
}

// The published bundle prices: 20 x 35,000 = 700,000 for one pax.
func TestSubscriptionMonthlyMatchesPublishedBundle(t *testing.T) {
	var ds []string
	d, _ := time.Parse(time.DateOnly, "2026-08-17")
	for i := 0; i < 20; i++ {
		ds = append(ds, d.AddDate(0, 0, i).Format(time.DateOnly))
	}
	q, err := QuoteSubscription(SubscriptionRequest{Dates: dates(t, ds...), Pax: 1, Rates: healthy}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if q.Analysis.TierKey != TierMonthly || q.Total != 700000 {
		t.Errorf("got %s / %d, want monthly / 700000", q.Analysis.TierKey, q.Total)
	}
}

// docs/04 §3 — Regular Catering's group table, and the linear extension above
// its last row (BR-5.4).
func TestRegularCateringPaxTable(t *testing.T) {
	table := map[string]map[string]map[int]int64{
		"dengan": {
			"weekly":  {1: 26000, 2: 52000, 3: 76000, 4: 98000, 5: 118000},
			"monthly": {1: 25000, 2: 50000, 3: 74000, 4: 95000, 5: 113000},
		},
		"tanpa": {
			"weekly":  {1: 26000, 2: 52000, 3: 73000, 4: 92000, 5: 111000},
			"monthly": {1: 25000, 2: 50000, 3: 71000, 4: 90000, 5: 108000},
		},
	}
	reguler := Rates{Daily: 31000, WeeklyPerDay: 26000, MonthlyPerDay: 25000,
		FlexiWeeklyPerDay: 27000, FlexiMonthlyPerDay: 26000}
	week := dates(t, "2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20", "2026-08-21")

	for _, tc := range []struct {
		name          string
		pax           int
		rice          string
		wantDayTotal  int64
		wantEffective int64
	}{
		{"5 pax dengan nasi", 5, "dengan", 118000, 23600},
		{"5 pax tanpa nasi", 5, "tanpa", 111000, 22200},
		{"2 pax — both rice options cost the same", 2, "tanpa", 52000, 26000},
		// BR-5.4: round(118000/5)=23600, x 6 pax = 141,600 per day.
		{"6 pax extends linearly", 6, "dengan", 141600, 23600},
		{"20 pax extends linearly", 20, "dengan", 472000, 23600},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q, err := QuoteSubscription(SubscriptionRequest{
				Dates: week, Pax: tc.pax, Rates: reguler, PaxTable: table, Rice: tc.rice,
			}, Default())
			if err != nil {
				t.Fatal(err)
			}
			if !q.UsedPaxTable {
				t.Fatal("the group table must apply on the weekly tier")
			}
			if got := q.MenuTotal / 5; got != tc.wantDayTotal {
				t.Errorf("day total = %d, want %d", got, tc.wantDayTotal)
			}
			if q.EffectiveRate != tc.wantEffective {
				t.Errorf("effective rate = %d, want %d", q.EffectiveRate, tc.wantEffective)
			}
		})
	}
}

// BR-5.1 — the table applies ONLY on the strict weekly/monthly tiers.
func TestPaxTableIgnoredOnFlexi(t *testing.T) {
	table := map[string]map[string]map[int]int64{
		"dengan": {"weekly": {1: 26000, 5: 118000}, "monthly": {1: 25000, 5: 113000}},
	}
	reguler := Rates{Daily: 31000, WeeklyPerDay: 26000, MonthlyPerDay: 25000,
		FlexiWeeklyPerDay: 27000, FlexiMonthlyPerDay: 26000}
	// Scattered: 5 days, not consecutive, no full week -> flexi-weekly.
	q, err := QuoteSubscription(SubscriptionRequest{
		Dates: dates(t, "2026-08-17", "2026-08-19", "2026-08-21", "2026-08-25", "2026-08-27"),
		Pax:   5, Rates: reguler, PaxTable: table,
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if q.UsedPaxTable {
		t.Fatal("the group table must not apply on a Flexi tier")
	}
	// 27,000 x 5 days x 5 pax
	if q.Total != 675000 {
		t.Errorf("total = %d, want 675000", q.Total)
	}
}

// BR-6.8 — the Flexi meat cap: one portion per 5 pax on a Flexi tier, per full
// pax otherwise. This is the single most surprising rule in the calculator.
func TestFlexiMeatCap(t *testing.T) {
	meat := Addon{Code: "Extra Daging (khusus Kamis)", Price: 20000,
		RestrictDays: "4", FlexiPortionPerPax: 5}

	// Scattered dates including two Thursdays -> flexi-weekly.
	flexiDates := dates(t, "2026-08-20", "2026-08-24", "2026-08-27", "2026-09-01", "2026-09-03")
	q, err := QuoteSubscription(SubscriptionRequest{
		Dates: flexiDates, Pax: 12, Rates: healthy, Addons: []Addon{meat},
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if q.Analysis.TierKey != TierFlexiWeekly {
		t.Fatalf("expected a Flexi tier, got %s", q.Analysis.TierKey)
	}
	c := q.AddonCharges[0]
	// 12 pax -> floor(12/5) = 2 portions. Thursdays in the set: 20 Aug, 27 Aug, 3 Sep = 3.
	if c.EffectivePax != 2 {
		t.Errorf("effective pax = %d, want 2 (12 pax capped at 1 per 5)", c.EffectivePax)
	}
	if c.Days != 3 {
		t.Errorf("charged days = %d, want 3 Thursdays", c.Days)
	}
	if c.Total != 20000*2*3 {
		t.Errorf("addon total = %d, want %d", c.Total, 20000*2*3)
	}

	// Below the cap divisor, a Flexi order gets no portion at all.
	q, _ = QuoteSubscription(SubscriptionRequest{
		Dates: flexiDates, Pax: 4, Rates: healthy, Addons: []Addon{meat},
	}, Default())
	if q.AddonCharges[0].Total != 0 {
		t.Errorf("4 pax on Flexi must get no meat portion, got %d", q.AddonCharges[0].Total)
	}

	// On a NON-Flexi tier the cap does not apply: Mon-Fri including a Thursday.
	q, err = QuoteSubscription(SubscriptionRequest{
		Dates: dates(t, "2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20", "2026-08-21"),
		Pax:   12, Rates: healthy, Addons: []Addon{meat},
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if q.Analysis.TierKey != TierWeekly {
		t.Fatalf("expected weekly, got %s", q.Analysis.TierKey)
	}
	c = q.AddonCharges[0]
	if c.EffectivePax != 12 || c.Days != 1 || c.Total != 20000*12 {
		t.Errorf("on Mingguan the cap must not apply: pax=%d days=%d total=%d",
			c.EffectivePax, c.Days, c.Total)
	}
}

// BR-6.2/6.3 — a restricted add-on charges only for matching weekdays, and an
// unrestricted one for every selected date.
func TestAddonDayMatching(t *testing.T) {
	week := dates(t, "2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20", "2026-08-21")
	q, err := QuoteSubscription(SubscriptionRequest{
		Dates: week, Pax: 2, Rates: healthy,
		Addons: []Addon{
			{Code: "Extra Telur", Price: 5000},                                  // any day
			{Code: "Extra Ikan (khusus Rabu)", Price: 15000, RestrictDays: "3"}, // Wed only
			{Code: "Extra Sayur", Price: 5000, RestrictDays: "123456"},          // Mon-Sat
		},
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		days  int
		total int64
	}{
		"Extra Telur":              {5, 5000 * 2 * 5},
		"Extra Ikan (khusus Rabu)": {1, 15000 * 2 * 1},
		"Extra Sayur":              {5, 5000 * 2 * 5},
	}
	for _, c := range q.AddonCharges {
		w := want[c.Code]
		if c.Days != w.days || c.Total != w.total {
			t.Errorf("%s: days=%d total=%d, want days=%d total=%d", c.Code, c.Days, c.Total, w.days, w.total)
		}
	}
}

// BR-6.5/6.6 — the customer can opt an add-on out of specific days.
func TestAddonExplicitDates(t *testing.T) {
	week := dates(t, "2026-08-17", "2026-08-18", "2026-08-19", "2026-08-20", "2026-08-21")
	q, _ := QuoteSubscription(SubscriptionRequest{
		Dates: week, Pax: 1, Rates: healthy,
		Addons: []Addon{{Code: "Extra Telur", Price: 5000,
			Dates: []string{"2026-08-17", "2026-08-19"}}},
	}, Default())
	if c := q.AddonCharges[0]; c.Days != 2 || c.Total != 10000 {
		t.Errorf("days=%d total=%d, want 2 / 10000", c.Days, c.Total)
	}
}

// BR-6.4 — a restricted add-on with no eligible date contributes nothing.
func TestAddonUnavailable(t *testing.T) {
	q, _ := QuoteSubscription(SubscriptionRequest{
		Dates: dates(t, "2026-08-17", "2026-08-18"), Pax: 3, Rates: healthy,
		Addons: []Addon{{Code: "Extra Daging (khusus Kamis)", Price: 20000, RestrictDays: "4"}},
	}, Default())
	if q.AddonTotal != 0 {
		t.Errorf("addon total = %d, want 0", q.AddonTotal)
	}
	if q.AddonCharges[0].Note == "" {
		t.Error("an unavailable add-on should say why")
	}
}

// BR-14.1 — each date is a separate full delivery.
func TestTierProduct(t *testing.T) {
	bands := []Band{
		{Min: 20, Max: 50, Price: 26000}, {Min: 51, Max: 100, Price: 24000},
		{Min: 101, Max: 199, Price: 22000}, {Min: 200, Max: 99999, Price: 20000},
	}
	q, err := QuoteTierProduct(TierProductRequest{
		Dates: dates(t, "2026-09-01", "2026-09-08"), Qty: 60, MinQty: 20, Bands: bands,
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	// 60 boxes lands in 51-100 at 24,000, over 2 delivery dates.
	if q.UnitPrice != 24000 || q.Total != 24000*60*2 {
		t.Errorf("unit=%d total=%d, want 24000 / %d", q.UnitPrice, q.Total, 24000*60*2)
	}
	// Below the minimum, quantity is clamped up rather than rejected -- the
	// front end does the same, and disagreeing would let the page accept an
	// order the server refuses.
	q, _ = QuoteTierProduct(TierProductRequest{
		Dates: dates(t, "2026-09-01"), Qty: 5, MinQty: 20, Bands: bands,
	}, Default())
	if q.Qty != 20 {
		t.Errorf("qty = %d, want it clamped up to 20", q.Qty)
	}
}

// BR-13.3/13.7 — Kantor walks weekdays only, so "Bulanan" is 20 working days.
func TestKantor(t *testing.T) {
	bands := []Band{
		{Min: 5, Max: 10, Price: 24000}, {Min: 11, Max: 20, Price: 23000},
		{Min: 21, Max: 50, Price: 22000}, {Min: 51, Max: 100, Price: 21000},
		{Min: 101, Max: 99999, Price: 20000},
	}
	start, _ := time.Parse(time.DateOnly, "2026-08-17") // a Monday
	q, err := QuoteKantor(KantorRequest{
		StartDate: start, Pax: 15, Days: 20, Period: "bulanan", Bands: bands,
	}, Default())
	if err != nil {
		t.Fatal(err)
	}
	if q.RatePerPax != 23000 {
		t.Errorf("rate = %d, want 23000 for 15 pax", q.RatePerPax)
	}
	if q.Total != 23000*15*20 {
		t.Errorf("total = %d, want %d", q.Total, 23000*15*20)
	}
	if len(q.ServedDates) != 20 {
		t.Fatalf("served %d dates, want 20", len(q.ServedDates))
	}
	// 20 weekdays from Mon 17 Aug runs to Fri 11 Sep — four calendar weeks.
	if q.ServedDates[19] != "2026-09-11" {
		t.Errorf("last served date = %s, want 2026-09-11", q.ServedDates[19])
	}
	for _, s := range q.ServedDates {
		d, _ := time.Parse(time.DateOnly, s)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			t.Errorf("%s is a weekend; the walk must skip those", s)
		}
	}
}

// BR-13.8 — a Rabu/Kamis add-on counts only the matching days inside the walk,
// while an unrestricted one applies to the whole period.
func TestKantorAddons(t *testing.T) {
	bands := []Band{{Min: 5, Max: 99999, Price: 24000}}
	start, _ := time.Parse(time.DateOnly, "2026-08-17")
	q, _ := QuoteKantor(KantorRequest{
		StartDate: start, Pax: 10, Days: 5, Period: "mingguan", Bands: bands,
		Addons: []Addon{
			{Code: "Extra Protein Ikan (khusus Rabu)", Price: 15000, RestrictDays: "3"},
			{Code: "Ganti Thinwall", Price: 2000},
		},
	}, Default())
	byCode := map[string]AddonCharge{}
	for _, c := range q.AddonCharges {
		byCode[c.Code] = c
	}
	// One Wednesday in Mon 17 - Fri 21.
	if c := byCode["Extra Protein Ikan (khusus Rabu)"]; c.Days != 1 || c.Total != 15000*10 {
		t.Errorf("Rabu addon: days=%d total=%d, want 1 / %d", c.Days, c.Total, 15000*10)
	}
	// Unrestricted applies to all 5 days of the package.
	if c := byCode["Ganti Thinwall"]; c.Days != 5 || c.Total != 2000*10*5 {
		t.Errorf("thinwall: days=%d total=%d, want 5 / %d", c.Days, c.Total, 2000*10*5)
	}
}

func TestQuoteRejectsBadInput(t *testing.T) {
	if _, err := QuoteSubscription(SubscriptionRequest{Pax: 1, Rates: healthy}, Default()); err == nil {
		t.Error("no dates must be rejected")
	}
	if _, err := QuoteSubscription(SubscriptionRequest{
		Dates: dates(t, "2026-08-17"), Pax: 0, Rates: healthy,
	}, Default()); err == nil {
		t.Error("zero pax must be rejected")
	}
}

// roundHalfUp must match JavaScript's Math.round on the .5 boundary, or the two
// engines drift by a rupiah on exactly the inputs nobody tests by hand.
func TestRoundHalfUp(t *testing.T) {
	for _, tc := range []struct{ a, b, want int64 }{
		{5, 2, 3}, // 2.5 -> 3
		{7, 2, 4}, // 3.5 -> 4
		{118000, 5, 23600},
		{113000, 5, 22600},
		{1, 3, 0},
		{2, 3, 1},
	} {
		if got := roundHalfUp(tc.a, tc.b); got != tc.want {
			t.Errorf("roundHalfUp(%d,%d) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
