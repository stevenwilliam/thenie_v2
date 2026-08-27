// Package siteconfig assembles the single document the front end hydrates from.
//
// The contract is deliberately one GET. The overlay runs inside a page that is
// already 6.7 MB; making it issue eight requests to eight tidy REST resources
// would trade a clean API surface for a slower page, on a connection that is
// often a phone on mobile data in Tangerang. One document, one round trip, one
// ETag.
//
// Field names are snake_case to match what the front end's own data attributes
// already use, so the overlay never has to translate between two conventions.
package siteconfig

import "time"

// Document is the whole public configuration surface.
type Document struct {
	// Revision is bumped by a database trigger on every content write. The
	// overlay sends it back as an ETag, so an unchanged site answers 304 and
	// the payload never crosses the wire.
	Revision    int64     `json:"revision"`
	GeneratedAt time.Time `json:"generated_at"`
	Timezone    string    `json:"timezone"`

	Plans           []Plan             `json:"plans"`
	Kantor          Kantor             `json:"kantor"`
	TierProducts    []TierProduct      `json:"tier_products"`
	Addons          map[string][]Addon `json:"addons"`
	Areas           []Area             `json:"areas"`
	DeliveryWindows []DeliveryWindow   `json:"delivery_windows"`
	Menu            Menu               `json:"menu"`
	Content         map[string][]Block `json:"content"`
	Params          map[string]string  `json:"params"`
}

// Plan is one Daily Order subscription card.
type Plan struct {
	Slug           string `json:"slug"`
	CardKey        string `json:"card_key"` // the page's data-sub value
	Name           string `json:"name"`
	Description    string `json:"description"`
	KcalMin        int    `json:"kcal_min,omitempty"`
	KcalMax        int    `json:"kcal_max,omitempty"`
	DeliversSunday bool   `json:"delivers_sunday"`
	SortOrder      int    `json:"sort_order"`

	// Rates is shaped exactly like the page's data-rates attribute, so the
	// overlay can compare or write it without remapping a single key.
	Rates Rates `json:"rates"`

	// PaxPrices is Regular Catering's 1-5 pax table and is omitted for every
	// other plan, mirroring data-pax-table being present on one card only.
	// Shape: rice -> period -> pax -> group day total.
	PaxPrices map[string]map[string]map[string]int64 `json:"pax_prices,omitempty"`
}

// Rates carries the five per-pax-per-day rates. The JSON keys are the front
// end's, camelCase and all: this object is written straight into data-rates.
type Rates struct {
	Daily              int64 `json:"daily"`
	WeeklyPerDay       int64 `json:"weeklyPerDay"`
	MonthlyPerDay      int64 `json:"monthlyPerDay"`
	FlexiWeeklyPerDay  int64 `json:"flexiWeeklyPerDay"`
	FlexiMonthlyPerDay int64 `json:"flexiMonthlyPerDay"`
}

// Kantor is Catering Kantor's pax ladder.
type Kantor struct {
	// Periods maps mingguan/bulanan to their fixed day count (5 / 20).
	Periods map[string]int `json:"periods"`
	// Rates is grade -> period -> ordered bands.
	Rates map[string]map[string][]Band `json:"rates"`
}

// TierProduct is Nasi Bento, Nasi Kuning or Paket Acara.
type TierProduct struct {
	Slug      string        `json:"slug"`
	CardKey   string        `json:"card_key"`
	Name      string        `json:"name"`
	Unit      string        `json:"unit"`
	MinQty    int           `json:"min_qty"`
	SortOrder int           `json:"sort_order"`
	Packages  []TierPackage `json:"packages"`
}

// TierPackage is one selectable option within a product, e.g. "Paket Ayam".
type TierPackage struct {
	Name     string   `json:"name"`
	Includes []string `json:"includes,omitempty"`
	Tiers    []Band   `json:"tiers"`
}

// Band is one quantity band. The JSON keys match the page's
// data-plans-select entries.
type Band struct {
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Price int64  `json:"price"`
	Label string `json:"label,omitempty"`
}

// Addon is one Tambahan checkbox.
type Addon struct {
	Code  string `json:"code"`  // the checkbox `value`
	Label string `json:"label"` // the visible chip text
	Price int64  `json:"price"`
	// RestrictDays is digit characters, 0=Sunday..6=Saturday. Empty = any day.
	RestrictDays string `json:"restrict_days"`
	// FlexiPortionPerPax carries BR-6.8's meat cap: 5 means "at most one
	// portion per 5 pax on a Flexi tier". Zero means charge per full pax.
	FlexiPortionPerPax int `json:"flexi_portion_per_pax,omitempty"`
	SortOrder          int `json:"sort_order"`
}

// Area is one service area.
type Area struct {
	Name string `json:"name"`
	// Orderable drives the order form's dropdown; Advertised drives marketing
	// copy. They differ in the captured page (BR-11.4, Q-27) and modelling both
	// keeps the mismatch visible instead of hidden in two places in the markup.
	Orderable  bool `json:"orderable"`
	Advertised bool `json:"advertised"`
	CatchAll   bool `json:"catch_all,omitempty"`
	SortOrder  int  `json:"sort_order"`
}

// DeliveryWindow is one Waktu Pengantaran option.
type DeliveryWindow struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	Value   string `json:"value"` // written verbatim into the WhatsApp message
	Note    string `json:"note,omitempty"`
	Default bool   `json:"default,omitempty"`
	// SameDayCutoffHour is the hour in the operating timezone after which this
	// window closes for today. nil means never available same-day, which is
	// what both Pagi windows do (BR-7.4).
	SameDayCutoffHour *int `json:"same_day_cutoff_hour"`
	SortOrder         int  `json:"sort_order"`
}

// Menu carries the cycles the page renders.
type Menu struct {
	// Current and Next are what the page's two <details> blocks show as
	// "menu minggu ini" and "menu minggu depan".
	Current *Cycle  `json:"current"`
	Next    *Cycle  `json:"next"`
	Cycles  []Cycle `json:"cycles"`
}

// Cycle is one published week.
type Cycle struct {
	ISOYear  int    `json:"iso_year"`
	ISOWeek  int    `json:"iso_week"`
	StartsOn string `json:"starts_on"` // YYYY-MM-DD, a calendar date not an instant
	EndsOn   string `json:"ends_on"`
	Label    string `json:"label"`
	// Days is plan slug -> that plan's days, in date order.
	Days map[string][]Day `json:"days"`
}

// Day is one plan's menu for one date.
type Day struct {
	Date      string      `json:"date"`    // YYYY-MM-DD
	Weekday   int         `json:"weekday"` // 0=Sunday..6=Saturday
	IsMeatDay bool        `json:"is_meat_day"`
	Kcal      int         `json:"kcal,omitempty"`
	Items     []Component `json:"items"`
}

// Component is one item on the plate.
type Component struct {
	Name  string `json:"name"`
	Grams int    `json:"grams,omitempty"`
}

// Block is one piece of marketing copy.
type Block struct {
	Kind      string         `json:"kind"`
	Heading   string         `json:"heading,omitempty"`
	Body      string         `json:"body,omitempty"`
	Meta      map[string]any `json:"meta,omitempty"`
	SortOrder int            `json:"sort_order"`
}
