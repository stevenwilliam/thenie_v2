package catalogue

import (
	"errors"
	"testing"
)

// The four rate tables exactly as they appear in the captured page's
// data-rates attributes (docs/04-pricing-catalogue.md §1). If a future edit
// breaks one of these, the front end's pricing engine breaks with it.
var captured = map[string]Rates{
	"Healthy Meal":     {Daily: 50000, WeeklyPerDay: 38000, MonthlyPerDay: 35000, FlexiWeeklyPerDay: 40000, FlexiMonthlyPerDay: 38000},
	"Bulking Extra":    {Daily: 70000, WeeklyPerDay: 55000, MonthlyPerDay: 50000, FlexiWeeklyPerDay: 57000, FlexiMonthlyPerDay: 52000},
	"Regular Catering": {Daily: 31000, WeeklyPerDay: 26000, MonthlyPerDay: 25000, FlexiWeeklyPerDay: 27000, FlexiMonthlyPerDay: 26000},
	"Kids Meal":        {Daily: 26000, WeeklyPerDay: 21000, MonthlyPerDay: 20000, FlexiWeeklyPerDay: 23000, FlexiMonthlyPerDay: 21000},
}

func TestCapturedRatesAreValid(t *testing.T) {
	for name, r := range captured {
		if err := r.Validate(); err != nil {
			t.Errorf("%s: the shipped rate table must validate, got %v", name, err)
		}
	}
}

func TestRatesValidate(t *testing.T) {
	base := captured["Healthy Meal"]

	tests := []struct {
		name string
		mut  func(*Rates)
		want error
	}{
		{"monthly above weekly", func(r *Rates) { r.MonthlyPerDay = r.WeeklyPerDay + 1 }, ErrRateMonotonic},
		{"weekly above daily", func(r *Rates) { r.WeeklyPerDay = r.Daily + 1 }, ErrRateMonotonic},
		{"zero daily", func(r *Rates) { r.Daily = 0 }, ErrRatePositive},
		{"negative flexi", func(r *Rates) { r.FlexiWeeklyPerDay = -1 }, ErrRatePositive},
		{"flexi monthly undercuts monthly", func(r *Rates) { r.FlexiMonthlyPerDay = r.MonthlyPerDay - 1 }, ErrFlexiCheaper},
		{"flexi weekly undercuts weekly", func(r *Rates) { r.FlexiWeeklyPerDay = r.WeeklyPerDay - 1 }, ErrFlexiCheaper},
		{"flexi weekly above list", func(r *Rates) { r.FlexiWeeklyPerDay = r.Daily + 1 }, ErrFlexiOverList},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := base
			tc.mut(&r)
			err := r.Validate()
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// The boundary case: equal rates are legal. Three of the four shipped plans
// have flexi_monthly exactly equal to weekly, and Regular Catering has
// flexi_monthly equal to weekly too. A stricter ">" would reject the real
// price list.
func TestEqualRatesAreLegal(t *testing.T) {
	r := Rates{Daily: 100, WeeklyPerDay: 100, MonthlyPerDay: 100, FlexiWeeklyPerDay: 100, FlexiMonthlyPerDay: 100}
	if err := r.Validate(); err != nil {
		t.Fatalf("a flat table is degenerate but not invalid: %v", err)
	}
}

func TestDiscountBps(t *testing.T) {
	// Healthy Meal: 38000 off a 50000 list is 24.00%, i.e. 2400 bps.
	got := captured["Healthy Meal"].DiscountBps()
	for tier, want := range map[string]int64{
		"weekly": 2400, "monthly": 3000, "flexi_weekly": 2000, "flexi_monthly": 2400,
	} {
		if got[tier] != want {
			t.Errorf("Healthy Meal %s: got %d bps, want %d", tier, got[tier], want)
		}
	}
	// Bulking Extra 55000/70000 = 21.428...% -> 2143 bps, rounded half-up.
	if b := captured["Bulking Extra"].DiscountBps()["weekly"]; b != 2143 {
		t.Errorf("Bulking Extra weekly: got %d bps, want 2143", b)
	}
}

// The Nasi Bento ladder exactly as shipped (docs/04 §5).
func bentoAyam() []Band {
	return []Band{
		{Min: 20, Max: 50, Price: 26000},
		{Min: 51, Max: 100, Price: 24000},
		{Min: 101, Max: 199, Price: 22000},
		{Min: 200, Max: 99999, Price: 20000},
	}
}

func TestCapturedBandsAreValid(t *testing.T) {
	for name, tc := range map[string]struct {
		bands  []Band
		minQty int
	}{
		"Nasi Bento Ayam": {bentoAyam(), 20},
		"Nasi Kuning": {[]Band{
			{Min: 10, Max: 29, Price: 39000},
			{Min: 30, Max: 59, Price: 37000},
			{Min: 60, Max: 99999, Price: 35000},
		}, 10},
		"Paket Acara A": {[]Band{
			{Min: 25, Max: 50, Price: 40000, Label: "25-50 pax"},
			{Min: 51, Max: 99999, Price: 35000, Label: ">50 pax"},
		}, 25},
		"Kantor reguler mingguan": {[]Band{
			{Min: 5, Max: 10, Price: 24000},
			{Min: 11, Max: 20, Price: 23000},
			{Min: 21, Max: 50, Price: 22000},
			{Min: 51, Max: 100, Price: 21000},
			{Min: 101, Max: 99999, Price: 20000},
		}, 5},
	} {
		if err := ValidateBands(tc.bands, tc.minQty); err != nil {
			t.Errorf("%s: the shipped ladder must validate, got %v", name, err)
		}
	}
}

func TestValidateBands(t *testing.T) {
	tests := []struct {
		name   string
		bands  []Band
		minQty int
		want   error
	}{
		{"empty", nil, 20, ErrBandEmpty},
		{"does not start at the minimum", bentoAyam()[1:], 20, ErrBandNotFromMin},
		{"inverted", []Band{{Min: 20, Max: 10, Price: 1}}, 20, ErrBandInverted},
		{"zero price", []Band{{Min: 20, Max: 50, Price: 0}}, 20, ErrRatePositive},
		{"overlap", []Band{
			{Min: 20, Max: 60, Price: 26000},
			{Min: 51, Max: 100, Price: 24000},
		}, 20, ErrBandOverlap},
		{"gap", []Band{
			{Min: 20, Max: 50, Price: 26000},
			{Min: 101, Max: 199, Price: 22000},
		}, 20, ErrBandGap},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateBands(tc.bands, tc.minQty); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

// Unsorted input must validate the same as sorted input: the admin API has no
// reason to guarantee order, and a false rejection here would be baffling.
func TestValidateBandsIgnoresInputOrder(t *testing.T) {
	b := bentoAyam()
	shuffled := []Band{b[2], b[0], b[3], b[1]}
	if err := ValidateBands(shuffled, 20); err != nil {
		t.Fatalf("order must not matter: %v", err)
	}
}

func TestBandFor(t *testing.T) {
	b := bentoAyam()
	for _, tc := range []struct {
		qty       int
		wantPrice int64
		wantOK    bool
	}{
		{20, 26000, true},
		{50, 26000, true},
		{51, 24000, true},
		{100, 24000, true},
		{101, 22000, true},
		{199, 22000, true},
		{200, 20000, true},
		{5000, 20000, true},
		// Below the ladder: no band matches. The front end falls back to the
		// last band, so this mirrors it -- and reports ok=false so a caller
		// that cares can tell the difference.
		{1, 20000, false},
	} {
		got, ok := BandFor(b, tc.qty)
		if got.Price != tc.wantPrice || ok != tc.wantOK {
			t.Errorf("qty %d: got %d (ok=%v), want %d (ok=%v)", tc.qty, got.Price, ok, tc.wantPrice, tc.wantOK)
		}
	}
}

func TestValidateRestrictDays(t *testing.T) {
	// The shipped values, from docs/02-business-rules.md BR-6.
	for _, ok := range []string{"", "3", "4", "12356", "123456", "0123456"} {
		if err := ValidateRestrictDays(ok); err != nil {
			t.Errorf("%q must be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"7", "a", "3,4", " 3", "-1"} {
		if err := ValidateRestrictDays(bad); !errors.Is(err, ErrRestrictDays) {
			t.Errorf("%q must be rejected as a bad digit, got %v", bad, err)
		}
	}
	if err := ValidateRestrictDays("33"); !errors.Is(err, ErrRestrictDupes) {
		t.Errorf("a repeated weekday must be rejected, got %v", err)
	}
}
