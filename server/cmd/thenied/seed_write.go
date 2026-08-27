package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/platform/id"
)

// write loads the extracted content into the database in one transaction.
//
// All or nothing: a half-seeded database that still answers requests is the
// worst outcome, because the page would render with some prices missing and
// nothing would look broken enough to investigate.
func (d *seedData) write(ctx context.Context, db *gorm.DB, force bool) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if force {
			// Order matters only where there is no cascade; menu_components and
			// menu_days cascade from menu_cycles, tier_prices from
			// tier_packages, and plan_rates from plans.
			for _, table := range []string{
				"menu_cycles", "tier_products", "kantor_rates", "kantor_periods",
				"addons", "service_areas", "delivery_windows", "content_blocks",
				"plan_pax_prices", "plans",
			} {
				if err := tx.Exec("DELETE FROM " + table).Error; err != nil {
					return fmt.Errorf("clear %s: %w", table, err)
				}
			}
		}

		planID := map[string]string{}
		for _, p := range d.Plans {
			pid := id.NewString()
			planID[p.Slug] = pid
			var kmin, kmax any
			if p.KcalMin > 0 {
				kmin, kmax = p.KcalMin, p.KcalMax
			}
			if err := tx.Exec(`
				INSERT INTO plans (id, slug, card_key, name, description, kcal_min, kcal_max,
				                   delivers_sunday, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				pid, p.Slug, p.CardKey, p.Name, p.Description, kmin, kmax,
				p.DeliversSunday, p.Sort).Error; err != nil {
				return fmt.Errorf("plan %s: %w", p.Slug, err)
			}
			if err := tx.Exec(`
				INSERT INTO plan_rates (plan_id, daily, weekly_per_day, monthly_per_day,
				                        flexi_weekly_per_day, flexi_monthly_per_day)
				VALUES (?, ?, ?, ?, ?, ?)`,
				pid, p.Rates["daily"], p.Rates["weeklyPerDay"], p.Rates["monthlyPerDay"],
				p.Rates["flexiWeeklyPerDay"], p.Rates["flexiMonthlyPerDay"]).Error; err != nil {
				return fmt.Errorf("plan %s rates: %w", p.Slug, err)
			}
			for rice, periods := range p.PaxTable {
				for period, byPax := range periods {
					for pax, total := range byPax {
						if err := tx.Exec(`
							INSERT INTO plan_pax_prices (id, plan_id, rice, period, pax, day_total)
							VALUES (?, ?, ?, ?, ?, ?)`,
							id.NewString(), pid, rice, period, pax, total).Error; err != nil {
							return fmt.Errorf("plan %s pax table: %w", p.Slug, err)
						}
					}
				}
			}
		}

		// BR-13.3 — the fixed day counts. Read from the page's PERIOD_DAYS.
		for _, kp := range []struct {
			period string
			days   int
			label  string
			sort   int
		}{
			{"mingguan", 5, "Mingguan", 0},
			{"bulanan", 20, "Bulanan", 1},
		} {
			if err := tx.Exec(`
				INSERT INTO kantor_periods (period, days, label, sort_order) VALUES (?, ?, ?, ?)`,
				kp.period, kp.days, kp.label, kp.sort).Error; err != nil {
				return fmt.Errorf("kantor period %s: %w", kp.period, err)
			}
		}
		for _, k := range d.Kantor {
			if err := tx.Exec(`
				INSERT INTO kantor_rates (id, grade, period, pax_min, pax_max, rate_per_pax_day, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				id.NewString(), k.Grade, k.Period, k.Band.Min, k.Band.Max,
				k.Band.Price, k.Sort).Error; err != nil {
				return fmt.Errorf("kantor rate %s/%s: %w", k.Grade, k.Period, err)
			}
		}

		for _, tp := range d.TierProducts {
			tid := id.NewString()
			if err := tx.Exec(`
				INSERT INTO tier_products (id, slug, card_key, name, unit, min_qty, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				tid, tp.Slug, tp.CardKey, tp.Name, tp.Unit, tp.MinQty, tp.Sort).Error; err != nil {
				return fmt.Errorf("tier product %s: %w", tp.Slug, err)
			}
			for pi, pkg := range tp.Packages {
				pkgID := id.NewString()
				if err := tx.Exec(`
					INSERT INTO tier_packages (id, product_id, name, includes, sort_order)
					VALUES (?, ?, ?, ?, ?)`,
					pkgID, tid, pkg.Name, pgTextArray(pkg.Includes), pi).Error; err != nil {
					return fmt.Errorf("tier package %s/%s: %w", tp.Slug, pkg.Name, err)
				}
				for bi, b := range pkg.Tiers {
					if err := tx.Exec(`
						INSERT INTO tier_prices (id, package_id, qty_min, qty_max, label, price, sort_order)
						VALUES (?, ?, ?, ?, ?, ?, ?)`,
						id.NewString(), pkgID, b.Min, b.Max, b.Label, b.Price, bi).Error; err != nil {
						return fmt.Errorf("tier price %s/%s: %w", tp.Slug, pkg.Name, err)
					}
				}
			}
		}

		for _, a := range d.Addons {
			var flexi any
			if a.FlexiPortion > 0 {
				flexi = a.FlexiPortion
			}
			if err := tx.Exec(`
				INSERT INTO addons (id, scope, code, label, price, restrict_days,
				                    flexi_portion_per_pax, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id.NewString(), a.Scope, a.Code, a.Label, a.Price,
				a.RestrictDays, flexi, a.Sort).Error; err != nil {
				return fmt.Errorf("addon %s/%s: %w", a.Scope, a.Code, err)
			}
		}

		// BR-11.4 / Q-27 — the order form's list and the marketing list differ.
		// Both facts are recorded: everything in the dropdown is orderable, and
		// the five the marketing pages actually name are advertised.
		advertised := map[string]bool{
			"Gading Serpong": true, "BSD": true, "Karawaci": true,
			"Alam Sutera": true, "Medang": true,
		}
		for i, name := range d.Areas {
			catchAll := name == "Lainnya"
			if err := tx.Exec(`
				INSERT INTO service_areas (id, name, is_orderable, is_advertised, is_catch_all, sort_order)
				VALUES (?, ?, TRUE, ?, ?, ?)`,
				id.NewString(), name, advertised[name] && !catchAll, catchAll, i).Error; err != nil {
				return fmt.Errorf("area %s: %w", name, err)
			}
		}

		for _, w := range d.Windows {
			if err := tx.Exec(`
				INSERT INTO delivery_windows (id, code, label, value, note, is_default,
				                              same_day_cutoff_hour, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				id.NewString(), w.Code, w.Label, w.Value, w.Note, w.Default,
				nullableInt(w.Cutoff), w.Sort).Error; err != nil {
				return fmt.Errorf("delivery window %s: %w", w.Code, err)
			}
		}

		for i, t := range d.Testimonials {
			if err := tx.Exec(`
				INSERT INTO content_blocks (id, kind, slot, body, sort_order)
				VALUES (?, 'testimonial', 'home.testimonials', ?, ?)`,
				id.NewString(), t, i).Error; err != nil {
				return fmt.Errorf("testimonial %d: %w", i, err)
			}
		}
		for i, s := range d.Stats {
			meta, _ := json.Marshal(map[string]any{"target": s.Target, "suffix": s.Suffix})
			if err := tx.Exec(`
				INSERT INTO content_blocks (id, kind, slot, heading, meta, sort_order)
				VALUES (?, 'stat', 'home.stats', ?, ?, ?)`,
				id.NewString(), s.Label, string(meta), i).Error; err != nil {
				return fmt.Errorf("stat %d: %w", i, err)
			}
		}

		// Menu cycles last: they reference plans, and publishing them trips the
		// overlap exclusion constraint if anything above went wrong.
		for _, c := range d.Cycles {
			cid := id.NewString()
			if err := tx.Exec(`
				INSERT INTO menu_cycles (id, iso_year, iso_week, starts_on, ends_on, label, is_published)
				VALUES (?, ?, ?, ?, ?, ?, TRUE)`,
				cid, c.ISOYear, c.ISOWeek, c.StartsOn, c.EndsOn, c.Label).Error; err != nil {
				return fmt.Errorf("cycle %d-W%02d: %w", c.ISOYear, c.ISOWeek, err)
			}
			for slug, days := range c.Days {
				pid, ok := planID[slug]
				if !ok {
					return fmt.Errorf("cycle %d-W%02d references unknown plan %q", c.ISOYear, c.ISOWeek, slug)
				}
				for _, day := range days {
					dayID := id.NewString()
					var kcal any
					if day.Kcal > 0 {
						kcal = day.Kcal
					}
					if err := tx.Exec(`
						INSERT INTO menu_days (id, cycle_id, plan_id, serve_date, is_meat_day, kcal)
						VALUES (?, ?, ?, ?, ?, ?)`,
						dayID, cid, pid, day.Date, day.IsMeatDay, kcal).Error; err != nil {
						return fmt.Errorf("menu day %s %s: %w", slug, day.Date.Format("2006-01-02"), err)
					}
					for ci, comp := range day.Items {
						var grams any
						if comp.Grams > 0 {
							grams = comp.Grams
						}
						if err := tx.Exec(`
							INSERT INTO menu_components (id, menu_day_id, name, grams, sort_order)
							VALUES (?, ?, ?, ?, ?)`,
							id.NewString(), dayID, comp.Name, grams, ci).Error; err != nil {
							return fmt.Errorf("menu component %s: %w", comp.Name, err)
						}
					}
				}
			}
		}
		return nil
	})
}

// pgTextArray renders a Go slice as a PostgreSQL array literal. The values here
// come from the mirror's own JSON, but the quoting is done properly anyway:
// a menu item containing a comma or a quote would otherwise corrupt the array.
func pgTextArray(items []string) string {
	if len(items) == 0 {
		return "{}"
	}
	quoted := make([]string, 0, len(items))
	for _, s := range items {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		quoted = append(quoted, `"`+s+`"`)
	}
	return "{" + strings.Join(quoted, ",") + "}"
}

func nullableInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
