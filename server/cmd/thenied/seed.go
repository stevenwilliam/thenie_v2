package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/platform/config"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/database"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/id"
)

// runSeed loads the captured page's content into an empty database.
//
// It reads site/index.html and EXTRACTS the content rather than carrying a
// hand-written fixture. That is the whole point: the mirror is the
// specification (docs/07-fidelity-and-verification.md), so a seed derived from
// it cannot drift from it. A hand-typed fixture would be a second copy of every
// price, and the first time someone re-captured the page the two would silently
// disagree.
//
// It refuses to run against a database that already has content unless --force
// is given, because seeding twice would duplicate every menu cycle.
func runSeed(ctx context.Context, cfg *config.Config, log *slogLogger, args []string) error {
	force := false
	mirror := os.Getenv("MIRROR_PATH")
	if mirror == "" {
		mirror = "../site/index.html"
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			force = true
		case "--mirror":
			if i+1 >= len(args) {
				return errors.New("seed: --mirror needs a path")
			}
			i++
			mirror = args[i]
		}
	}

	raw, err := os.ReadFile(mirror) // #nosec G304 -- operator-supplied path
	if err != nil {
		return fmt.Errorf("seed: read mirror: %w (set MIRROR_PATH or pass --mirror)", err)
	}
	page := string(raw)
	log.Info("seeding from mirror", "path", mirror, "bytes", len(raw))

	extracted, err := extract(page)
	if err != nil {
		return fmt.Errorf("seed: extract: %w", err)
	}
	log.Info("extracted",
		"plans", len(extracted.Plans),
		"tier_products", len(extracted.TierProducts),
		"kantor_bands", len(extracted.Kantor),
		"addons", len(extracted.Addons),
		"areas", len(extracted.Areas),
		"delivery_windows", len(extracted.Windows),
		"menu_cycles", len(extracted.Cycles))

	db, err := database.Open(ctx, database.Options{URL: cfg.DatabaseURL}, log)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	var existing int64
	if err := db.WithContext(ctx).Raw(`SELECT count(*) FROM plans`).Scan(&existing).Error; err != nil {
		return fmt.Errorf("seed: check existing: %w (run: thenied migrate up)", err)
	}
	if existing > 0 && !force {
		return fmt.Errorf("seed: database already has %d plan(s); pass --force to replace", existing)
	}

	if err := extracted.write(ctx, db, force); err != nil {
		return err
	}
	fmt.Printf("seeded from %s: %d plans, %d tier products, %d menu cycles\n",
		mirror, len(extracted.Plans), len(extracted.TierProducts), len(extracted.Cycles))
	return nil
}

// ---- the extracted shape ----

type seedData struct {
	Plans        []seedPlan
	TierProducts []seedTierProduct
	Kantor       []seedKantorBand
	Addons       []seedAddon
	Areas        []string
	Windows      []seedWindow
	Cycles       []seedCycle
	Testimonials []string
	Stats        []seedStat
}

type seedPlan struct {
	Slug           string
	CardKey        string
	Name           string
	Description    string
	KcalMin        int
	KcalMax        int
	DeliversSunday bool
	Sort           int
	Rates          map[string]int64
	PaxTable       map[string]map[string]map[string]int64
}

type seedTierProduct struct {
	Slug     string
	CardKey  string
	Name     string
	Unit     string
	MinQty   int
	Sort     int
	Packages []seedTierPackage
}

type seedTierPackage struct {
	Name     string
	Includes []string
	Tiers    []seedBand
}

type seedBand struct {
	Min   int
	Max   int
	Price int64
	Label string
}

type seedKantorBand struct {
	Grade  string
	Period string
	Band   seedBand
	Sort   int
}

type seedAddon struct {
	Scope        string
	Code         string
	Label        string
	Price        int64
	RestrictDays string
	FlexiPortion int
	Sort         int
}

type seedWindow struct {
	Code    string
	Label   string
	Value   string
	Note    string
	Default bool
	Cutoff  *int
	Sort    int
}

type seedCycle struct {
	ISOYear  int
	ISOWeek  int
	StartsOn time.Time
	EndsOn   time.Time
	Label    string
	Days     map[string][]seedDay // plan slug -> days
}

type seedDay struct {
	Date      time.Time
	IsMeatDay bool
	Kcal      int
	Items     []seedComponent
}

type seedComponent struct {
	Name  string
	Grams int
}

// ---- extraction ----

var (
	reDailyPanel  = regexp.MustCompile(`<div class="daily-subpanel[^"]*" data-dailypanel="([a-z]+)">`)
	reOrderCard   = regexp.MustCompile(`<div class="order-card"([^>]*)>`)
	reAttrSub     = regexp.MustCompile(`data-sub="([^"]*)"`)
	reAttrRates   = regexp.MustCompile(`data-rates='([^']*)'`)
	reAttrPaxTbl  = regexp.MustCompile(`data-pax-table='([^']*)'`)
	reH3          = regexp.MustCompile(`<h3>([^<]*)</h3>`)
	reDesc        = regexp.MustCompile(`<div class="desc">([\s\S]*?)</div>`)
	reKcalRange   = regexp.MustCompile(`(\d{3})[–-](\d{3})\s*kcal`)
	reDetails     = regexp.MustCompile(`<details class="menu-week">([\s\S]*?)</details>`)
	reSummaryWeek = regexp.MustCompile(`Minggu ke-(\d+),\s*(\d+)[–-](\d+)\s+(\p{L}+)\s+(\d{4})`)
	reMenuDay     = regexp.MustCompile(`<div class="menu-day"><b>([^<]*)</b>([^—]*)—([\s\S]*?)(?:<span class="kcal">([^<]*)</span>)?</div>`)
	rePlansSelect = regexp.MustCompile(`data-plans-select='([\s\S]*?)'`)
	reAddonInput  = regexp.MustCompile(`<input type="checkbox" class="addon-cb"([^>]*)>([^<]*)`)
	reAttrPrice   = regexp.MustCompile(`data-price="(\d+)"`)
	reAttrRestr   = regexp.MustCompile(`data-restrict-days="(\d*)"`)
	reAttrValue   = regexp.MustCompile(`value="([^"]*)"`)
	reAreaSelect  = regexp.MustCompile(`<select class="cust-area-item">([\s\S]*?)</select>`)
	reOption      = regexp.MustCompile(`<option value="([^"]*)"`)
	reDtimeRow    = regexp.MustCompile(`<div class="delivery-time-row">([\s\S]*?)</div>`)
	reDtimeInput  = regexp.MustCompile(`<input type="radio" name="dtime-[^"]*" value="([^"]*)"([^>]*)>([^<]*)`)
	reKantorRates = regexp.MustCompile(`const RATES = \{([\s\S]*?)\};`)
	reKantorLine  = regexp.MustCompile(`(reguler|healthy):\s*\{\s*mingguan:\s*\[([\d,\s]*)\],\s*bulanan:\s*\[([\d,\s]*)\]`)
	reTierMins    = regexp.MustCompile(`const TIER_MINS\s*= \[([\d,\s]*)\]`)
	reTierMaxs    = regexp.MustCompile(`const TIER_MAXS\s*= \[([\d,\s]*)\]`)
	rePeriodDays  = regexp.MustCompile(`const PERIOD_DAYS = \{ mingguan: (\d+), bulanan: (\d+) \}`)
	reTestimonial = regexp.MustCompile(`<div class="testi-card">\s*<p>([\s\S]*?)</p>`)
	reStatItem    = regexp.MustCompile(`<div class="stat-item"><b>(?:<span class="count" data-target="(\d+)">0</span>)?\s*([^<]*)</b><span>([^<]*)</span>`)
)

// bulanID maps the page's Indonesian month names, both the long form used in
// week labels and the three-letter form used in menu day lines.
var bulanID = map[string]time.Month{
	"januari": time.January, "februari": time.February, "maret": time.March,
	"april": time.April, "mei": time.May, "juni": time.June,
	"juli": time.July, "agustus": time.August, "september": time.September,
	"oktober": time.October, "november": time.November, "desember": time.December,
	"jan": time.January, "feb": time.February, "mar": time.March,
	"apr": time.April, "jun": time.June, "jul": time.July,
	"agu": time.August, "sep": time.September, "okt": time.October,
	"nov": time.November, "des": time.December,
}

func extract(page string) (*seedData, error) {
	d := &seedData{}
	if err := d.extractPlans(page); err != nil {
		return nil, err
	}
	if err := d.extractTierProducts(page); err != nil {
		return nil, err
	}
	if err := d.extractKantor(page); err != nil {
		return nil, err
	}
	d.extractAddons(page)
	d.extractAreas(page)
	d.extractWindows(page)
	d.extractContent(page)
	if len(d.Plans) == 0 {
		return nil, errors.New("no Daily Order plans found — is this the right mirror?")
	}
	return d, nil
}

// extractPlans reads the four Daily Order cards, each of which lives inside a
// .daily-subpanel that names its slug.
func (d *seedData) extractPlans(page string) error {
	panels := reDailyPanel.FindAllStringSubmatchIndex(page, -1)
	if len(panels) == 0 {
		return errors.New("no .daily-subpanel found")
	}
	cycles := map[string]*seedCycle{} // week key -> cycle

	for i, loc := range panels {
		slug := page[loc[2]:loc[3]]
		end := len(page)
		if i+1 < len(panels) {
			end = panels[i+1][0]
		}
		body := page[loc[1]:end]

		card := reOrderCard.FindStringSubmatchIndex(body)
		if card == nil {
			continue
		}
		attrs := body[card[2]:card[3]]

		p := seedPlan{Slug: slug, Sort: i, DeliversSunday: true}
		if m := reAttrSub.FindStringSubmatch(attrs); m != nil {
			p.CardKey = m[1]
		}
		if m := reAttrRates.FindStringSubmatch(attrs); m != nil {
			if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &p.Rates); err != nil {
				return fmt.Errorf("plan %s: data-rates: %w", slug, err)
			}
		} else {
			return fmt.Errorf("plan %s: no data-rates", slug)
		}
		if m := reAttrPaxTbl.FindStringSubmatch(attrs); m != nil {
			if err := json.Unmarshal([]byte(html.UnescapeString(m[1])), &p.PaxTable); err != nil {
				return fmt.Errorf("plan %s: data-pax-table: %w", slug, err)
			}
		}
		if m := reH3.FindStringSubmatch(body); m != nil {
			p.Name = strings.TrimSpace(m[1])
		}
		if m := reDesc.FindStringSubmatch(body); m != nil {
			p.Description = cleanText(m[1])
		}
		if m := reKcalRange.FindStringSubmatch(p.Description); m != nil {
			p.KcalMin, _ = strconv.Atoi(m[1])
			p.KcalMax, _ = strconv.Atoi(m[2])
		}
		// BR-7.6 — the two plans whose calendars block every Sunday.
		if p.CardKey == "Healthy Meal" || p.CardKey == "Bulking Extra" {
			p.DeliversSunday = false
		}
		d.Plans = append(d.Plans, p)

		// This card's weekly menus.
		for _, det := range reDetails.FindAllStringSubmatch(body, -1) {
			block := det[1]
			wk := reSummaryWeek.FindStringSubmatch(block)
			if wk == nil {
				continue
			}
			week, _ := strconv.Atoi(wk[1])
			dayFrom, _ := strconv.Atoi(wk[2])
			dayTo, _ := strconv.Atoi(wk[3])
			month, ok := bulanID[strings.ToLower(wk[4])]
			if !ok {
				return fmt.Errorf("plan %s: unknown month %q", slug, wk[4])
			}
			year, _ := strconv.Atoi(wk[5])

			key := fmt.Sprintf("%d-%02d", year, week)
			c, seen := cycles[key]
			if !seen {
				c = &seedCycle{
					ISOYear:  year,
					ISOWeek:  week,
					StartsOn: time.Date(year, month, dayFrom, 0, 0, 0, 0, time.UTC),
					EndsOn:   time.Date(year, month, dayTo, 0, 0, 0, 0, time.UTC),
					Label:    fmt.Sprintf("Minggu ke-%d · %d–%d %s %d", week, dayFrom, dayTo, wk[4], year),
					Days:     map[string][]seedDay{},
				}
				cycles[key] = c
				d.Cycles = append(d.Cycles, seedCycle{}) // placeholder, filled below
			}

			for _, md := range reMenuDay.FindAllStringSubmatch(block, -1) {
				day, err := parseMenuDay(md, year)
				if err != nil {
					return fmt.Errorf("plan %s week %d: %w", slug, week, err)
				}
				c.Days[slug] = append(c.Days[slug], day)
			}
		}
	}

	// Rebuild the cycle slice in a deterministic order rather than map order.
	d.Cycles = d.Cycles[:0]
	keys := make([]string, 0, len(cycles))
	for k := range cycles {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		d.Cycles = append(d.Cycles, *cycles[k])
	}
	return nil
}

// parseMenuDay turns one rendered line back into structured data:
//
//	<b>Kamis 27 Agu</b> ⭐ — Kentang Rebus (120g), Rawon Daging Sapi (80g), …
//	<span class="kcal">±485 kkal</span>
func parseMenuDay(m []string, year int) (seedDay, error) {
	head := cleanText(m[1])             // "Kamis 27 Agu"
	star := strings.Contains(m[2], "⭐") // BR-15.3, the meat day
	bodyText := cleanText(m[3])
	kcalText := cleanText(m[4])

	fields := strings.Fields(head)
	if len(fields) < 3 {
		return seedDay{}, fmt.Errorf("cannot read date from %q", head)
	}
	dayNum, err := strconv.Atoi(fields[1])
	if err != nil {
		return seedDay{}, fmt.Errorf("cannot read day number from %q", head)
	}
	month, ok := bulanID[strings.ToLower(fields[2])]
	if !ok {
		return seedDay{}, fmt.Errorf("unknown month %q in %q", fields[2], head)
	}

	day := seedDay{
		Date:      time.Date(year, month, dayNum, 0, 0, 0, 0, time.UTC),
		IsMeatDay: star,
	}
	if n := regexp.MustCompile(`(\d+)`).FindStringSubmatch(kcalText); n != nil {
		day.Kcal, _ = strconv.Atoi(n[1])
	}
	for _, part := range strings.Split(bodyText, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		comp := seedComponent{Name: part}
		// "Nasi Merah (100g)" -> name + grams. Kids Meal lines carry no grams
		// at all, which is why grams is optional rather than required.
		if g := regexp.MustCompile(`^(.*?)\s*\((\d+)\s*g(?:r)?\)$`).FindStringSubmatch(part); g != nil {
			comp.Name = strings.TrimSpace(g[1])
			comp.Grams, _ = strconv.Atoi(g[2])
		}
		day.Items = append(day.Items, comp)
	}
	if len(day.Items) == 0 {
		return seedDay{}, fmt.Errorf("no components in %q", bodyText)
	}
	return day, nil
}

func (d *seedData) extractTierProducts(page string) error {
	// Each data-plans-select sits on the card whose data-sub names it.
	for _, loc := range rePlansSelect.FindAllStringSubmatchIndex(page, -1) {
		raw := html.UnescapeString(page[loc[2]:loc[3]])
		var packages []struct {
			Name     string   `json:"name"`
			Includes []string `json:"includes"`
			Tiers    []struct {
				Min   int    `json:"min"`
				Max   int    `json:"max"`
				Price int64  `json:"price"`
				Label string `json:"label"`
			} `json:"tiers"`
		}
		if err := json.Unmarshal([]byte(raw), &packages); err != nil {
			return fmt.Errorf("data-plans-select: %w", err)
		}

		// Walk backwards to the owning card for its data-sub and minimum.
		before := page[:loc[0]]
		cardStart := strings.LastIndex(before, `<div class="order-card"`)
		if cardStart < 0 {
			continue
		}
		attrs := page[cardStart:loc[0]]
		sub := ""
		if m := reAttrSub.FindStringSubmatch(attrs); m != nil {
			sub = m[1]
		}

		tp := seedTierProduct{
			CardKey: sub,
			Name:    sub,
			Sort:    len(d.TierProducts),
			Unit:    "box",
		}
		switch sub {
		case "Nasi Bento":
			tp.Slug = "nasibox"
		case "Nasi Kuning Wow":
			tp.Slug = "nasikuning"
		case "Paket Acara":
			tp.Slug, tp.Unit = "acara", "pax"
		default:
			tp.Slug = strings.ToLower(strings.ReplaceAll(sub, " ", "-"))
		}

		minQty := 0
		for _, p := range packages {
			pkg := seedTierPackage{Name: p.Name, Includes: p.Includes}
			for _, t := range p.Tiers {
				pkg.Tiers = append(pkg.Tiers, seedBand{Min: t.Min, Max: t.Max, Price: t.Price, Label: t.Label})
				if minQty == 0 || t.Min < minQty {
					minQty = t.Min
				}
			}
			tp.Packages = append(tp.Packages, pkg)
		}
		tp.MinQty = minQty
		d.TierProducts = append(d.TierProducts, tp)
	}
	return nil
}

// extractKantor reads the RATES constant and the pax bands out of the Kantor
// IIFE. They are JavaScript literals, not markup, so this is the one place the
// extractor reads code rather than DOM.
func (d *seedData) extractKantor(page string) error {
	block := reKantorRates.FindStringSubmatch(page)
	if block == nil {
		return errors.New("kantor RATES constant not found")
	}
	mins := parseIntList(reTierMins.FindStringSubmatch(page))
	maxs := parseIntList(reTierMaxs.FindStringSubmatch(page))
	if len(mins) == 0 || len(mins) != len(maxs) {
		return fmt.Errorf("kantor pax bands: %d mins, %d maxs", len(mins), len(maxs))
	}

	sort := 0
	for _, m := range reKantorLine.FindAllStringSubmatch(block[1], -1) {
		grade := m[1]
		for periodIdx, period := range []string{"mingguan", "bulanan"} {
			values := parseInt64List(m[2+periodIdx])
			if len(values) != len(mins) {
				return fmt.Errorf("kantor %s/%s: %d rates for %d bands", grade, period, len(values), len(mins))
			}
			for i, price := range values {
				d.Kantor = append(d.Kantor, seedKantorBand{
					Grade:  grade,
					Period: period,
					Band:   seedBand{Min: mins[i], Max: maxs[i], Price: price},
					Sort:   sort,
				})
				sort++
			}
		}
	}
	if len(d.Kantor) == 0 {
		return errors.New("kantor rates parsed to nothing")
	}
	return nil
}

// extractAddons reads every Tambahan checkbox. The same add-on appears on all
// four Daily Order cards, so they are de-duplicated by (scope, code).
func (d *seedData) extractAddons(page string) {
	regions := []struct {
		scope string
		from  string
		to    string
	}{
		{"daily", `<div class="daily-subpanel active" data-dailypanel="healthy">`, `<div class="daily-subpanel" data-dailypanel="bulking">`},
		{"bento", `<div class="subpanel" id="subpanel-nasibox">`, `<div class="subpanel" id="subpanel-nasikuning">`},
		{"kantor", `<div class="order-card" id="kantor-card">`, `<div class="cart-bar">`},
	}
	for _, r := range regions {
		start := strings.Index(page, r.from)
		if start < 0 {
			continue
		}
		end := strings.Index(page[start:], r.to)
		body := page[start:]
		if end > 0 {
			body = page[start : start+end]
		}
		seen := map[string]bool{}
		for _, m := range reAddonInput.FindAllStringSubmatch(body, -1) {
			attrs, label := m[1], cleanText(m[2])
			a := seedAddon{Scope: r.scope, Label: label, Sort: len(seen)}
			if v := reAttrValue.FindStringSubmatch(attrs); v != nil {
				a.Code = v[1]
			}
			if p := reAttrPrice.FindStringSubmatch(attrs); p != nil {
				n, _ := strconv.ParseInt(p[1], 10, 64)
				a.Price = n
			}
			if rd := reAttrRestr.FindStringSubmatch(attrs); rd != nil {
				a.RestrictDays = rd[1]
			}
			// BR-6.8 — the only add-on with a Flexi portion cap.
			if a.Code == "Extra Daging (khusus Kamis)" {
				a.FlexiPortion = 5
			}
			if a.Code == "" || seen[a.Code] {
				continue
			}
			seen[a.Code] = true
			if a.Label == "" {
				a.Label = a.Code
			}
			d.Addons = append(d.Addons, a)
		}
	}
}

func (d *seedData) extractAreas(page string) {
	m := reAreaSelect.FindStringSubmatch(page)
	if m == nil {
		return
	}
	for _, o := range reOption.FindAllStringSubmatch(m[1], -1) {
		d.Areas = append(d.Areas, o[1])
	}
}

func (d *seedData) extractWindows(page string) {
	m := reDtimeRow.FindStringSubmatch(page)
	if m == nil {
		return
	}
	// BR-7.4 — the cut-offs, read from getTodayCutoff(): Pagi is never
	// same-day; Siang closes at 09.00; Sore at 12.00; Request stays open.
	cutoff := func(h int) *int { return &h }
	rules := map[string]*int{
		"Pagi (06.00–07.00)":           nil,
		"Pagi (07.00–09.00)":           nil,
		"Siang (12.00)":                cutoff(9),
		"Sore (18.00)":                 cutoff(12),
		"Request (dikonfirmasi admin)": cutoff(24),
	}
	for i, w := range reDtimeInput.FindAllStringSubmatch(m[1], -1) {
		value, rest, label := w[1], w[2], cleanText(w[3])
		code := slugify(value)
		win := seedWindow{
			Code:    code,
			Label:   label,
			Value:   value,
			Default: strings.Contains(rest, "checked"),
			Sort:    i,
		}
		if c, ok := rules[value]; ok {
			win.Cutoff = c
		}
		if win.Label == "" {
			win.Label = value
		}
		d.Windows = append(d.Windows, win)
	}
}

func (d *seedData) extractContent(page string) {
	for _, m := range reTestimonial.FindAllStringSubmatch(page, -1) {
		d.Testimonials = append(d.Testimonials, cleanText(m[1]))
	}
	for _, m := range reStatItem.FindAllStringSubmatch(page, -1) {
		target, _ := strconv.Atoi(m[1])
		d.Stats = append(d.Stats, seedStat{
			Target: target,
			Suffix: cleanText(m[2]),
			Label:  cleanText(m[3]),
		})
	}
}

type seedStat struct {
	Target int
	Suffix string
	Label  string
}

// ---- helpers ----

var reTag = regexp.MustCompile(`<[^>]*>`)

// cleanText strips tags, unescapes entities and collapses whitespace. The
// captured markup wraps values in <b>/<span> freely, so every extracted string
// goes through this.
func cleanText(s string) string {
	s = reTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, " ", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func parseIntList(m []string) []int {
	if m == nil {
		return nil
	}
	var out []int
	for _, p := range strings.Split(m[1], ",") {
		if p = strings.TrimSpace(p); p != "" {
			if n, err := strconv.Atoi(p); err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}

func parseInt64List(s string) []int64 {
	var out []int64
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			if n, err := strconv.ParseInt(p, 10, 64); err == nil {
				out = append(out, n)
			}
		}
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

var _ = gorm.ErrRecordNotFound
var _ = id.NewString
