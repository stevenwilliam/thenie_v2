package siteconfig

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stevenwilliam/thenie_v2/server/internal/domain/catalogue"
	"github.com/stevenwilliam/thenie_v2/server/internal/domain/menu"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
)

// Repository is the read port. It is declared here rather than imported from
// app/ports to keep this package free of an import cycle -- ports imports this
// package for the document types.
type Repository interface {
	Revision(ctx context.Context) (int64, error)
	Load(ctx context.Context) (*Document, error)
}

// Service serves the public configuration document.
//
// It caches: the document is read on every page load of a public site, changes
// a few times a week, and costs a dozen queries to assemble. The cache is
// invalidated by revision, not by a timer, so an edit is live on the next
// request rather than after a TTL.
type Service struct {
	repo Repository

	mu       sync.RWMutex
	cached   *Document
	cachedAt time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Revision returns the current content revision.
func (s *Service) Revision(ctx context.Context) (int64, error) {
	rev, err := s.repo.Revision(ctx)
	if err != nil {
		return 0, apierror.Internal(fmt.Errorf("siteconfig: revision: %w", err))
	}
	return rev, nil
}

// Get returns the document, assembling it only when the revision has moved.
func (s *Service) Get(ctx context.Context) (*Document, error) {
	rev, err := s.Revision(ctx)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	cached := s.cached
	s.mu.RUnlock()
	if cached != nil && cached.Revision == rev {
		return cached, nil
	}

	doc, err := s.repo.Load(ctx)
	if err != nil {
		return nil, apierror.Internal(fmt.Errorf("siteconfig: load: %w", err))
	}
	doc.GeneratedAt = time.Now().UTC()
	doc.resolveCurrentCycle(time.Now())

	s.mu.Lock()
	s.cached, s.cachedAt = doc, doc.GeneratedAt
	s.mu.Unlock()
	return doc, nil
}

// Invalidate drops the cache. Writes call it so an admin edit shows up
// immediately even if the revision read is served by a replica that has not
// caught up yet.
func (s *Service) Invalidate() {
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
}

// Validate re-checks a whole document against the domain rules.
//
// This is belt-and-braces on purpose. The write endpoints validate their own
// input and the database has CHECK constraints, but this catches the third
// case: data that entered the database some other way -- a hand-run UPDATE, a
// restored backup, a migration written in a hurry -- and would otherwise be
// served to the front end and silently mis-price an order.
//
// It returns every problem, not the first: an operator fixing a bad import
// wants the whole list.
func (d *Document) Validate() []error {
	var problems []error

	for _, p := range d.Plans {
		r := catalogue.Rates{
			Daily:              p.Rates.Daily,
			WeeklyPerDay:       p.Rates.WeeklyPerDay,
			MonthlyPerDay:      p.Rates.MonthlyPerDay,
			FlexiWeeklyPerDay:  p.Rates.FlexiWeeklyPerDay,
			FlexiMonthlyPerDay: p.Rates.FlexiMonthlyPerDay,
		}
		if err := r.Validate(); err != nil {
			problems = append(problems, fmt.Errorf("plan %s: %w", p.Slug, err))
		}
	}

	for _, tp := range d.TierProducts {
		for _, pkg := range tp.Packages {
			bands := toBands(pkg.Tiers)
			if err := catalogue.ValidateBands(bands, tp.MinQty); err != nil {
				problems = append(problems, fmt.Errorf("%s / %s: %w", tp.Slug, pkg.Name, err))
			}
		}
	}

	for grade, periods := range d.Kantor.Rates {
		for period, bands := range periods {
			// Catering Kantor's minimum is 5 pax (BR-13.1).
			if err := catalogue.ValidateBands(toBands(bands), 5); err != nil {
				problems = append(problems, fmt.Errorf("kantor %s/%s: %w", grade, period, err))
			}
		}
	}

	for scope, addons := range d.Addons {
		for _, a := range addons {
			if err := catalogue.ValidateRestrictDays(a.RestrictDays); err != nil {
				problems = append(problems, fmt.Errorf("addon %s/%s: %w", scope, a.Code, err))
			}
		}
	}

	// Exactly one default delivery window, or the front end's reset has nothing
	// deterministic to reset to (BR-12.7).
	defaults := 0
	for _, w := range d.DeliveryWindows {
		if w.Default {
			defaults++
		}
	}
	if len(d.DeliveryWindows) > 0 && defaults != 1 {
		problems = append(problems, fmt.Errorf("delivery windows: %d marked default, want exactly 1", defaults))
	}

	problems = append(problems, d.validateMenu()...)
	return problems
}

// validateMenu checks the cycles against the domain rules, including the one
// the schema cannot express for unpublished cycles: two cycles claiming the
// same date.
func (d *Document) validateMenu() []error {
	var problems []error
	deliversSunday := map[string]bool{}
	for _, p := range d.Plans {
		deliversSunday[p.Slug] = p.DeliversSunday
	}

	seen := map[string]string{} // date -> cycle label
	for _, c := range d.Menu.Cycles {
		start, err := time.Parse(time.DateOnly, c.StartsOn)
		if err != nil {
			problems = append(problems, fmt.Errorf("cycle %s: bad starts_on %q", c.Label, c.StartsOn))
			continue
		}
		end, err := time.Parse(time.DateOnly, c.EndsOn)
		if err != nil {
			problems = append(problems, fmt.Errorf("cycle %s: bad ends_on %q", c.Label, c.EndsOn))
			continue
		}
		dc := menu.Cycle{Label: c.Label, StartsOn: start, EndsOn: end}
		if err := dc.Validate(); err != nil {
			problems = append(problems, fmt.Errorf("cycle %s: %w", c.Label, err))
		}

		for slug, days := range c.Days {
			for _, day := range days {
				date, err := time.Parse(time.DateOnly, day.Date)
				if err != nil {
					problems = append(problems, fmt.Errorf("cycle %s / %s: bad date %q", c.Label, slug, day.Date))
					continue
				}
				items := make([]menu.Component, 0, len(day.Items))
				for _, it := range day.Items {
					items = append(items, menu.Component{Name: it.Name, Grams: it.Grams})
				}
				md := menu.Day{PlanSlug: slug, ServeDate: date, Components: items}
				if err := dc.ValidateDay(md, deliversSunday[slug]); err != nil {
					problems = append(problems, fmt.Errorf("cycle %s / %s: %w", c.Label, slug, err))
				}
				key := slug + "@" + day.Date
				if other, dup := seen[key]; dup {
					problems = append(problems, fmt.Errorf(
						"%s on %s appears in both %q and %q", slug, day.Date, other, c.Label))
				}
				seen[key] = c.Label
			}
		}
	}
	return problems
}

func toBands(in []Band) []catalogue.Band {
	out := make([]catalogue.Band, 0, len(in))
	for _, b := range in {
		out = append(out, catalogue.Band{Min: b.Min, Max: b.Max, Price: b.Price, Label: b.Label})
	}
	return out
}

// resolveCurrentCycle picks the "this week" and "next week" cycles the page's
// two <details> blocks render.
//
// now is converted to the operating timezone before the date is taken. That
// conversion is the whole point: at 23:30 WIB on a Sunday the server's UTC
// clock still says Sunday 16:30, but at 07:30 WIB on a Monday it says Sunday
// again -- and picking the cycle off a UTC date would show last week's menu for
// the first seven hours of every Monday.
func (d *Document) resolveCurrentCycle(now time.Time) {
	if len(d.Menu.Cycles) == 0 {
		return
	}
	loc, err := time.LoadLocation(d.Timezone)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)

	cycles := make([]menu.Cycle, 0, len(d.Menu.Cycles))
	byLabel := map[string]int{}
	for i, c := range d.Menu.Cycles {
		start, err1 := time.Parse(time.DateOnly, c.StartsOn)
		end, err2 := time.Parse(time.DateOnly, c.EndsOn)
		if err1 != nil || err2 != nil {
			continue
		}
		key := fmt.Sprintf("%d-%02d", c.ISOYear, c.ISOWeek)
		byLabel[key] = i
		cycles = append(cycles, menu.Cycle{
			ID:       key,
			ISOYear:  c.ISOYear,
			ISOWeek:  c.ISOWeek,
			StartsOn: start,
			EndsOn:   end,
			Label:    c.Label,
		})
	}

	cur, next, ok := menu.Current(cycles, today)
	if !ok {
		return
	}
	if i, found := byLabel[cur.ID]; found {
		d.Menu.Current = &d.Menu.Cycles[i]
	}
	if next.ID != "" {
		if i, found := byLabel[next.ID]; found {
			d.Menu.Next = &d.Menu.Cycles[i]
		}
	}
}
