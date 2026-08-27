// Package postgres implements the application's ports against PostgreSQL.
//
// gorm opens and pools the connection; the queries themselves are explicit SQL.
// This is a read-mostly service assembling one denormalised document from a
// dozen tables, which is exactly the shape an ORM is worst at and plain SQL is
// best at. Every money column is BIGINT and is read into int64 -- no float ever
// touches a price.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
)

// ConfigRepo reads the public configuration surface.
type ConfigRepo struct{ db *gorm.DB }

func NewConfigRepo(db *gorm.DB) *ConfigRepo { return &ConfigRepo{db: db} }

// Revision returns the current content revision -- one row, one column.
func (r *ConfigRepo) Revision(ctx context.Context) (int64, error) {
	var rev int64
	err := r.db.WithContext(ctx).
		Raw(`SELECT revision FROM content_revision WHERE only_row`).
		Scan(&rev).Error
	if err != nil {
		return 0, fmt.Errorf("revision: %w", err)
	}
	return rev, nil
}

// Load assembles the whole document.
//
// It runs inside one repeatable-read transaction. Without that, a menu edit
// landing between the plans query and the menu query would produce a document
// that is internally inconsistent -- and, worse, would be cached under a
// revision that matches neither state.
func (r *ConfigRepo) Load(ctx context.Context) (*siteconfig.Document, error) {
	doc := &siteconfig.Document{
		Addons:  map[string][]siteconfig.Addon{},
		Content: map[string][]siteconfig.Block{},
		Params:  map[string]string{},
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SET TRANSACTION ISOLATION LEVEL REPEATABLE READ`).Error; err != nil {
			return fmt.Errorf("set isolation: %w", err)
		}
		if err := tx.Raw(`SELECT revision FROM content_revision WHERE only_row`).
			Scan(&doc.Revision).Error; err != nil {
			return fmt.Errorf("revision: %w", err)
		}
		for _, step := range []struct {
			name string
			fn   func(*gorm.DB, *siteconfig.Document) error
		}{
			{"params", loadParams},
			{"plans", loadPlans},
			{"kantor", loadKantor},
			{"tier products", loadTierProducts},
			{"addons", loadAddons},
			{"areas", loadAreas},
			{"delivery windows", loadDeliveryWindows},
			{"menu", loadMenu},
			{"content", loadContent},
		} {
			if err := step.fn(tx, doc); err != nil {
				return fmt.Errorf("%s: %w", step.name, err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	doc.Timezone = doc.Params["site.timezone"]
	if doc.Timezone == "" {
		doc.Timezone = "Asia/Jakarta"
	}
	return doc, nil
}

func loadParams(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`SELECT key, value FROM sys_parameters ORDER BY group_name, sort_order`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return err
		}
		doc.Params[k] = v
	}
	return rows.Err()
}

func loadPlans(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT p.slug, p.card_key, p.name, p.description,
		       COALESCE(p.kcal_min, 0), COALESCE(p.kcal_max, 0),
		       p.delivers_sunday, p.sort_order,
		       r.daily, r.weekly_per_day, r.monthly_per_day,
		       r.flexi_weekly_per_day, r.flexi_monthly_per_day
		  FROM plans p
		  JOIN plan_rates r ON r.plan_id = p.id
		 WHERE p.is_active
		 ORDER BY p.sort_order, p.slug`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	bySlug := map[string]int{}
	for rows.Next() {
		var p siteconfig.Plan
		if err := rows.Scan(&p.Slug, &p.CardKey, &p.Name, &p.Description,
			&p.KcalMin, &p.KcalMax, &p.DeliversSunday, &p.SortOrder,
			&p.Rates.Daily, &p.Rates.WeeklyPerDay, &p.Rates.MonthlyPerDay,
			&p.Rates.FlexiWeeklyPerDay, &p.Rates.FlexiMonthlyPerDay); err != nil {
			return err
		}
		bySlug[p.Slug] = len(doc.Plans)
		doc.Plans = append(doc.Plans, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Regular Catering's 1-5 pax table. Attached to whichever plan owns rows
	// rather than hard-coding the slug, so renaming the plan cannot orphan it.
	prows, err := tx.Raw(`
		SELECT p.slug, x.rice, x.period, x.pax, x.day_total
		  FROM plan_pax_prices x
		  JOIN plans p ON p.id = x.plan_id
		 ORDER BY p.slug, x.rice, x.period, x.pax`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = prows.Close() }()
	for prows.Next() {
		var slug, rice, period string
		var pax int
		var total int64
		if err := prows.Scan(&slug, &rice, &period, &pax, &total); err != nil {
			return err
		}
		i, ok := bySlug[slug]
		if !ok {
			continue // an inactive plan; its table is not published
		}
		if doc.Plans[i].PaxPrices == nil {
			doc.Plans[i].PaxPrices = map[string]map[string]map[string]int64{}
		}
		if doc.Plans[i].PaxPrices[rice] == nil {
			doc.Plans[i].PaxPrices[rice] = map[string]map[string]int64{}
		}
		if doc.Plans[i].PaxPrices[rice][period] == nil {
			doc.Plans[i].PaxPrices[rice][period] = map[string]int64{}
		}
		doc.Plans[i].PaxPrices[rice][period][fmt.Sprint(pax)] = total
	}
	return prows.Err()
}

func loadKantor(tx *gorm.DB, doc *siteconfig.Document) error {
	doc.Kantor.Periods = map[string]int{}
	doc.Kantor.Rates = map[string]map[string][]siteconfig.Band{}

	prows, err := tx.Raw(`SELECT period, days FROM kantor_periods ORDER BY sort_order`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = prows.Close() }()
	for prows.Next() {
		var period string
		var days int
		if err := prows.Scan(&period, &days); err != nil {
			return err
		}
		doc.Kantor.Periods[period] = days
	}
	if err := prows.Err(); err != nil {
		return err
	}

	rows, err := tx.Raw(`
		SELECT grade, period, pax_min, pax_max, rate_per_pax_day
		  FROM kantor_rates ORDER BY grade, period, sort_order, pax_min`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var grade, period string
		var b siteconfig.Band
		if err := rows.Scan(&grade, &period, &b.Min, &b.Max, &b.Price); err != nil {
			return err
		}
		if doc.Kantor.Rates[grade] == nil {
			doc.Kantor.Rates[grade] = map[string][]siteconfig.Band{}
		}
		doc.Kantor.Rates[grade][period] = append(doc.Kantor.Rates[grade][period], b)
	}
	return rows.Err()
}

func loadTierProducts(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT slug, card_key, name, unit, min_qty, sort_order
		  FROM tier_products WHERE is_active ORDER BY sort_order, slug`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	index := map[string]int{}
	for rows.Next() {
		var p siteconfig.TierProduct
		if err := rows.Scan(&p.Slug, &p.CardKey, &p.Name, &p.Unit, &p.MinQty, &p.SortOrder); err != nil {
			return err
		}
		index[p.Slug] = len(doc.TierProducts)
		doc.TierProducts = append(doc.TierProducts, p)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// Packages and their bands in one pass, ordered so a package's bands arrive
	// contiguously and in ladder order.
	prows, err := tx.Raw(`
		SELECT tp.slug, pk.name, pk.includes, pk.sort_order,
		       pr.qty_min, pr.qty_max, pr.price, pr.label
		  FROM tier_packages pk
		  JOIN tier_products tp ON tp.id = pk.product_id
		  LEFT JOIN tier_prices pr ON pr.package_id = pk.id
		 ORDER BY tp.sort_order, pk.sort_order, pr.qty_min`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = prows.Close() }()

	pkgIndex := map[string]int{} // productSlug + "\x00" + packageName -> position
	for prows.Next() {
		var slug, pkgName string
		var includes pq.StringArray
		var pkgSort int
		var qMin, qMax sql.NullInt64
		var price sql.NullInt64
		var label sql.NullString
		if err := prows.Scan(&slug, &pkgName, &includes, &pkgSort, &qMin, &qMax, &price, &label); err != nil {
			return err
		}
		pi, ok := index[slug]
		if !ok {
			continue
		}
		key := slug + "\x00" + pkgName
		ki, seen := pkgIndex[key]
		if !seen {
			doc.TierProducts[pi].Packages = append(doc.TierProducts[pi].Packages, siteconfig.TierPackage{
				Name:     pkgName,
				Includes: []string(includes),
			})
			ki = len(doc.TierProducts[pi].Packages) - 1
			pkgIndex[key] = ki
		}
		// LEFT JOIN: a package with no bands yet still appears, with none.
		if qMin.Valid && price.Valid {
			doc.TierProducts[pi].Packages[ki].Tiers = append(
				doc.TierProducts[pi].Packages[ki].Tiers, siteconfig.Band{
					Min:   int(qMin.Int64),
					Max:   int(qMax.Int64),
					Price: price.Int64,
					Label: label.String,
				})
		}
	}
	return prows.Err()
}

func loadAddons(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT scope, code, label, price, restrict_days,
		       COALESCE(flexi_portion_per_pax, 0), sort_order
		  FROM addons WHERE is_active ORDER BY scope, sort_order, code`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var scope string
		var a siteconfig.Addon
		if err := rows.Scan(&scope, &a.Code, &a.Label, &a.Price,
			&a.RestrictDays, &a.FlexiPortionPerPax, &a.SortOrder); err != nil {
			return err
		}
		doc.Addons[scope] = append(doc.Addons[scope], a)
	}
	return rows.Err()
}

func loadAreas(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT name, is_orderable, is_advertised, is_catch_all, sort_order
		  FROM service_areas ORDER BY sort_order, name`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var a siteconfig.Area
		if err := rows.Scan(&a.Name, &a.Orderable, &a.Advertised, &a.CatchAll, &a.SortOrder); err != nil {
			return err
		}
		doc.Areas = append(doc.Areas, a)
	}
	return rows.Err()
}

func loadDeliveryWindows(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT code, label, value, note, is_default, same_day_cutoff_hour, sort_order
		  FROM delivery_windows ORDER BY sort_order, code`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var w siteconfig.DeliveryWindow
		var cutoff sql.NullInt64
		if err := rows.Scan(&w.Code, &w.Label, &w.Value, &w.Note, &w.Default, &cutoff, &w.SortOrder); err != nil {
			return err
		}
		if cutoff.Valid {
			h := int(cutoff.Int64)
			w.SameDayCutoffHour = &h
		}
		doc.DeliveryWindows = append(doc.DeliveryWindows, w)
	}
	return rows.Err()
}

func loadContent(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT kind, slot, heading, body, meta, sort_order
		  FROM content_blocks WHERE is_active ORDER BY slot, sort_order`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var b siteconfig.Block
		var slot string
		var raw []byte
		if err := rows.Scan(&b.Kind, &slot, &b.Heading, &b.Body, &raw, &b.SortOrder); err != nil {
			return err
		}
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &b.Meta); err != nil {
				return fmt.Errorf("content_blocks.meta for slot %s: %w", slot, err)
			}
		}
		doc.Content[slot] = append(doc.Content[slot], b)
	}
	return rows.Err()
}

func loadMenu(tx *gorm.DB, doc *siteconfig.Document) error {
	rows, err := tx.Raw(`
		SELECT c.iso_year, c.iso_week, c.starts_on, c.ends_on, c.label,
		       p.slug, d.serve_date, d.is_meat_day, COALESCE(d.kcal, 0),
		       m.name, COALESCE(m.grams, 0)
		  FROM menu_cycles c
		  LEFT JOIN menu_days       d ON d.cycle_id = c.id
		  LEFT JOIN plans           p ON p.id = d.plan_id
		  LEFT JOIN menu_components m ON m.menu_day_id = d.id
		 WHERE c.is_published
		 ORDER BY c.starts_on, p.sort_order, d.serve_date, m.sort_order`).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	cycleIndex := map[string]int{}
	dayIndex := map[string]int{}

	for rows.Next() {
		var isoYear, isoWeek int
		var startsOn, endsOn time.Time
		var label string
		var slug sql.NullString
		var serveDate sql.NullTime
		var isMeat sql.NullBool
		var kcal sql.NullInt64
		var compName sql.NullString
		var grams sql.NullInt64

		if err := rows.Scan(&isoYear, &isoWeek, &startsOn, &endsOn, &label,
			&slug, &serveDate, &isMeat, &kcal, &compName, &grams); err != nil {
			return err
		}

		ckey := fmt.Sprintf("%d-%02d", isoYear, isoWeek)
		ci, ok := cycleIndex[ckey]
		if !ok {
			doc.Menu.Cycles = append(doc.Menu.Cycles, siteconfig.Cycle{
				ISOYear:  isoYear,
				ISOWeek:  isoWeek,
				StartsOn: startsOn.Format(time.DateOnly),
				EndsOn:   endsOn.Format(time.DateOnly),
				Label:    label,
				Days:     map[string][]siteconfig.Day{},
			})
			ci = len(doc.Menu.Cycles) - 1
			cycleIndex[ckey] = ci
		}
		// A published cycle with no days yet is legitimate -- the admin created
		// the week before filling it in.
		if !slug.Valid || !serveDate.Valid {
			continue
		}

		dkey := ckey + "\x00" + slug.String + "\x00" + serveDate.Time.Format(time.DateOnly)
		di, ok := dayIndex[dkey]
		if !ok {
			doc.Menu.Cycles[ci].Days[slug.String] = append(doc.Menu.Cycles[ci].Days[slug.String],
				siteconfig.Day{
					Date:      serveDate.Time.Format(time.DateOnly),
					Weekday:   int(serveDate.Time.Weekday()),
					IsMeatDay: isMeat.Bool,
					Kcal:      int(kcal.Int64),
				})
			di = len(doc.Menu.Cycles[ci].Days[slug.String]) - 1
			dayIndex[dkey] = di
		}
		if compName.Valid {
			days := doc.Menu.Cycles[ci].Days[slug.String]
			days[di].Items = append(days[di].Items, siteconfig.Component{
				Name:  compName.String,
				Grams: int(grams.Int64),
			})
			doc.Menu.Cycles[ci].Days[slug.String] = days
		}
	}
	return rows.Err()
}
