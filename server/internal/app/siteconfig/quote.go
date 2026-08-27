package siteconfig

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stevenwilliam/thenie_v2/server/internal/domain/pricing"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
)

// QuoteRequest asks the server to price one order card.
//
// It carries CODES, never prices. Every rupiah in the answer is looked up from
// the database, so a caller cannot propose its own rate — which is the whole
// reason a server-side calculator is worth having.
type QuoteRequest struct {
	// Kind is "subscription", "tier_product" or "kantor".
	Kind string `json:"kind"`

	// subscription
	Plan string `json:"plan,omitempty"` // slug, e.g. "healthy"
	Pax  int    `json:"pax,omitempty"`
	Rice string `json:"rice,omitempty"` // "dengan" | "tanpa"

	// tier_product
	Product string `json:"product,omitempty"` // slug, e.g. "nasibox"
	Package string `json:"package,omitempty"` // e.g. "Paket Ayam"
	Qty     int    `json:"qty,omitempty"`

	// kantor
	Grade  string `json:"grade,omitempty"`  // "reguler" | "healthy"
	Period string `json:"period,omitempty"` // "mingguan" | "bulanan"

	Dates     []string `json:"dates,omitempty"`
	StartDate string   `json:"start_date,omitempty"` // kantor only

	Addons []QuoteAddon `json:"addons,omitempty"`
}

// QuoteAddon selects one add-on by code, optionally narrowed to certain dates.
type QuoteAddon struct {
	Code  string   `json:"code"`
	Dates []string `json:"dates,omitempty"`
}

// QuoteResponse is the priced answer.
type QuoteResponse struct {
	Kind         string                     `json:"kind"`
	Revision     int64                      `json:"revision"`
	Rules        pricing.Rules              `json:"rules"`
	Currency     string                     `json:"currency"`
	Subscription *pricing.SubscriptionQuote `json:"subscription,omitempty"`
	TierProduct  *pricing.TierProductQuote  `json:"tier_product,omitempty"`
	Kantor       *pricing.KantorQuote       `json:"kantor,omitempty"`
	Total        int64                      `json:"total"`
	Formatted    string                     `json:"formatted"`
}

// Quote prices a request against the current catalogue.
func (s *Service) Quote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	doc, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	rules := doc.PricingRules
	if rules.Validate() != nil {
		rules = pricing.Default()
	}

	dates, err := parseDates(req.Dates)
	if err != nil {
		return nil, err
	}

	resp := &QuoteResponse{Kind: req.Kind, Revision: doc.Revision, Rules: rules, Currency: "IDR"}

	switch req.Kind {
	case "subscription":
		plan := doc.findPlan(req.Plan)
		if plan == nil {
			return nil, apierror.NotFound(fmt.Sprintf("No plan %q.", req.Plan))
		}
		q, err := pricing.QuoteSubscription(pricing.SubscriptionRequest{
			Dates:    dates,
			Pax:      req.Pax,
			Rates:    pricing.Rates(plan.Rates),
			PaxTable: convertPaxTable(plan.PaxPrices),
			Rice:     req.Rice,
			Addons:   doc.resolveAddons("daily", req.Addons),
		}, rules)
		if err != nil {
			return nil, badQuote(err)
		}
		resp.Subscription, resp.Total = q, q.Total

	case "tier_product":
		product, pkg := doc.findTierPackage(req.Product, req.Package)
		if product == nil {
			return nil, apierror.NotFound(fmt.Sprintf("No product %q.", req.Product))
		}
		if pkg == nil {
			return nil, apierror.NotFound(
				fmt.Sprintf("No package %q on product %q.", req.Package, req.Product))
		}
		// Only Nasi Bento carries add-ons; the other two have none (BR-6.12),
		// so an unknown code simply resolves to nothing rather than erroring.
		q, err := pricing.QuoteTierProduct(pricing.TierProductRequest{
			Dates:  dates,
			Qty:    req.Qty,
			MinQty: product.MinQty,
			Bands:  toPricingBands(pkg.Tiers),
			Addons: doc.resolveAddons("bento", req.Addons),
		}, rules)
		if err != nil {
			return nil, badQuote(err)
		}
		resp.TierProduct, resp.Total = q, q.Total

	case "kantor":
		bands, days, err := doc.kantorInputs(req.Grade, req.Period)
		if err != nil {
			return nil, err
		}
		start, err := parseDate(req.StartDate, "start_date")
		if err != nil {
			return nil, err
		}
		q, err := pricing.QuoteKantor(pricing.KantorRequest{
			StartDate: start,
			Pax:       req.Pax,
			Days:      days,
			Period:    req.Period,
			Bands:     bands,
			Addons:    doc.resolveAddons("kantor", req.Addons),
		}, rules)
		if err != nil {
			return nil, badQuote(err)
		}
		resp.Kantor, resp.Total = q, q.Total

	default:
		return nil, apierror.Validation(
			`kind must be "subscription", "tier_product" or "kantor".`,
			map[string]any{"kind": req.Kind})
	}

	resp.Formatted = FormatIDR(resp.Total)
	return resp, nil
}

// FormatIDR renders whole rupiah the way the page does: "Rp 1.234.567".
func FormatIDR(v int64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	digits := fmt.Sprintf("%d", v)
	var b strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	if neg {
		return "-Rp " + b.String()
	}
	return "Rp " + b.String()
}

// ---- lookups on the document ----

func (d *Document) findPlan(slug string) *Plan {
	for i := range d.Plans {
		if d.Plans[i].Slug == slug || d.Plans[i].CardKey == slug {
			return &d.Plans[i]
		}
	}
	return nil
}

func (d *Document) findTierPackage(productSlug, pkgName string) (*TierProduct, *TierPackage) {
	for i := range d.TierProducts {
		p := &d.TierProducts[i]
		if p.Slug != productSlug && p.CardKey != productSlug {
			continue
		}
		// A single-package product (Nasi Kuning) does not need the caller to
		// name the package it obviously means.
		if pkgName == "" && len(p.Packages) == 1 {
			return p, &p.Packages[0]
		}
		for j := range p.Packages {
			if p.Packages[j].Name == pkgName {
				return p, &p.Packages[j]
			}
		}
		return p, nil
	}
	return nil, nil
}

func (d *Document) kantorInputs(grade, period string) ([]pricing.Band, int, error) {
	byPeriod, ok := d.Kantor.Rates[grade]
	if !ok {
		return nil, 0, apierror.NotFound(fmt.Sprintf("No Kantor grade %q.", grade))
	}
	bands, ok := byPeriod[period]
	if !ok {
		return nil, 0, apierror.NotFound(fmt.Sprintf("No Kantor period %q for grade %q.", period, grade))
	}
	days, ok := d.Kantor.Periods[period]
	if !ok || days < 1 {
		return nil, 0, apierror.NotFound(fmt.Sprintf("No day count configured for period %q.", period))
	}
	return toPricingBands(bands), days, nil
}

// resolveAddons turns caller-supplied codes into priced add-ons from the
// catalogue. An unknown code is ignored rather than rejected: the front end
// sends whatever is checked, and a card that has since lost an add-on should
// still get a quote instead of an error the customer cannot act on.
func (d *Document) resolveAddons(scope string, wanted []QuoteAddon) []pricing.Addon {
	if len(wanted) == 0 {
		return nil
	}
	byCode := map[string]Addon{}
	for _, a := range d.Addons[scope] {
		byCode[a.Code] = a
	}
	out := make([]pricing.Addon, 0, len(wanted))
	for _, w := range wanted {
		def, ok := byCode[w.Code]
		if !ok {
			continue
		}
		out = append(out, pricing.Addon{
			Code:               def.Code,
			Price:              def.Price,
			RestrictDays:       def.RestrictDays,
			FlexiPortionPerPax: def.FlexiPortionPerPax,
			Dates:              w.Dates,
		})
	}
	return out
}

func convertPaxTable(in map[string]map[string]map[string]int64) map[string]map[string]map[int]int64 {
	if in == nil {
		return nil
	}
	out := map[string]map[string]map[int]int64{}
	for rice, periods := range in {
		out[rice] = map[string]map[int]int64{}
		for period, byPax := range periods {
			out[rice][period] = map[int]int64{}
			for paxStr, total := range byPax {
				var pax int
				if _, err := fmt.Sscanf(paxStr, "%d", &pax); err == nil {
					out[rice][period][pax] = total
				}
			}
		}
	}
	return out
}

func toPricingBands(in []Band) []pricing.Band {
	out := make([]pricing.Band, 0, len(in))
	for _, b := range in {
		out = append(out, pricing.Band{Min: b.Min, Max: b.Max, Price: b.Price, Label: b.Label})
	}
	return out
}

func parseDates(in []string) ([]time.Time, error) {
	out := make([]time.Time, 0, len(in))
	for _, s := range in {
		d, err := parseDate(s, "dates")
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func parseDate(s, field string) (time.Time, error) {
	if s == "" {
		return time.Time{}, apierror.Validation(fmt.Sprintf("%s is required.", field), nil)
	}
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return time.Time{}, apierror.Validation(
			fmt.Sprintf("%s must be YYYY-MM-DD.", field), map[string]any{field: s})
	}
	return d, nil
}

// badQuote turns a domain error into a client-actionable 422 rather than a 500.
func badQuote(err error) error {
	return apierror.New(http.StatusUnprocessableEntity, apierror.CodeValidation, err.Error())
}
