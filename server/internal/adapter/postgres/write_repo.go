package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/ports"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
	"github.com/stevenwilliam/thenie_v2/server/internal/domain/menu"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/id"
)

// MenuRepo is the write side of the menu rotation.
type MenuRepo struct{ db *gorm.DB }

func NewMenuRepo(db *gorm.DB) *MenuRepo { return &MenuRepo{db: db} }

// UpsertCycle replaces one week of menus atomically.
//
// Replace, not merge: the admin sends the week as it should end up looking, and
// the old days are deleted first. Merging would leave a Thursday from last
// week's draft sitting in this week's menu with nothing in the payload to
// reveal it.
func (r *MenuRepo) UpsertCycle(ctx context.Context, in ports.UpsertCycleInput) (string, error) {
	starts, err := time.Parse(time.DateOnly, in.StartsOn)
	if err != nil {
		return "", apierror.Validation("starts_on must be YYYY-MM-DD.",
			map[string]any{"starts_on": in.StartsOn})
	}
	ends, err := time.Parse(time.DateOnly, in.EndsOn)
	if err != nil {
		return "", apierror.Validation("ends_on must be YYYY-MM-DD.",
			map[string]any{"ends_on": in.EndsOn})
	}
	cycle := menu.Cycle{Label: in.Label, StartsOn: starts, EndsOn: ends}
	if err := cycle.Validate(); err != nil {
		return "", apierror.Unprocessable(apierror.CodeValidation, err.Error())
	}

	var cycleID string
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Which plans exist, and which of them refuse Sunday (BR-7.6). Read
		// inside the transaction so a plan deleted mid-write cannot orphan days.
		type planRow struct {
			ID             string
			Slug           string
			DeliversSunday bool
		}
		var plans []planRow
		if err := tx.Raw(`SELECT id, slug, delivers_sunday FROM plans`).Scan(&plans).Error; err != nil {
			return fmt.Errorf("read plans: %w", err)
		}
		planID := map[string]string{}
		sunday := map[string]bool{}
		for _, p := range plans {
			planID[p.Slug] = p.ID
			sunday[p.Slug] = p.DeliversSunday
		}

		// Validate everything before writing anything.
		for slug, days := range in.Days {
			if _, ok := planID[slug]; !ok {
				return apierror.Unprocessable(apierror.CodeCardKeyUnknown,
					fmt.Sprintf("Unknown plan %q.", slug))
			}
			for _, d := range days {
				date, err := time.Parse(time.DateOnly, d.Date)
				if err != nil {
					return apierror.Validation("A menu day date must be YYYY-MM-DD.",
						map[string]any{"plan": slug, "date": d.Date})
				}
				items := make([]menu.Component, 0, len(d.Items))
				for _, it := range d.Items {
					if strings.TrimSpace(it.Name) == "" {
						return apierror.Validation("A menu component needs a name.",
							map[string]any{"plan": slug, "date": d.Date})
					}
					items = append(items, menu.Component{Name: it.Name, Grams: it.Grams})
				}
				md := menu.Day{PlanSlug: slug, ServeDate: date, Components: items}
				if err := cycle.ValidateDay(md, sunday[slug]); err != nil {
					return apierror.Unprocessable(apierror.CodeValidation, err.Error())
				}
			}
		}

		newID := id.NewString()
		if err := tx.Exec(`
			INSERT INTO menu_cycles (id, iso_year, iso_week, starts_on, ends_on, label, is_published)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (iso_year, iso_week) DO UPDATE
			   SET starts_on = EXCLUDED.starts_on,
			       ends_on   = EXCLUDED.ends_on,
			       label     = EXCLUDED.label,
			       is_published = EXCLUDED.is_published,
			       updated_at = now()`,
			newID, in.ISOYear, in.ISOWeek, starts, ends, in.Label, in.Publish).Error; err != nil {
			return translate(err)
		}
		if err := tx.Raw(`SELECT id FROM menu_cycles WHERE iso_year = ? AND iso_week = ?`,
			in.ISOYear, in.ISOWeek).Scan(&cycleID).Error; err != nil {
			return fmt.Errorf("read cycle id: %w", err)
		}

		// menu_components cascades from menu_days, so one delete is enough.
		if err := tx.Exec(`DELETE FROM menu_days WHERE cycle_id = ?`, cycleID).Error; err != nil {
			return fmt.Errorf("clear days: %w", err)
		}

		for slug, days := range in.Days {
			for _, d := range days {
				dayID := id.NewString()
				var kcal any
				if d.Kcal > 0 {
					kcal = d.Kcal
				}
				if err := tx.Exec(`
					INSERT INTO menu_days (id, cycle_id, plan_id, serve_date, is_meat_day, kcal)
					VALUES (?, ?, ?, ?, ?, ?)`,
					dayID, cycleID, planID[slug], d.Date, d.IsMeatDay, kcal).Error; err != nil {
					return translate(err)
				}
				for i, it := range d.Items {
					var grams any
					if it.Grams > 0 {
						grams = it.Grams
					}
					if err := tx.Exec(`
						INSERT INTO menu_components (id, menu_day_id, name, grams, sort_order)
						VALUES (?, ?, ?, ?, ?)`,
						id.NewString(), dayID, it.Name, grams, i).Error; err != nil {
						return translate(err)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return cycleID, nil
}

func (r *MenuRepo) PublishCycle(ctx context.Context, isoYear, isoWeek int, published bool) error {
	res := r.db.WithContext(ctx).Exec(
		`UPDATE menu_cycles SET is_published = ?, updated_at = now()
		  WHERE iso_year = ? AND iso_week = ?`, published, isoYear, isoWeek)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound(fmt.Sprintf("No cycle for %d-W%02d.", isoYear, isoWeek))
	}
	return nil
}

func (r *MenuRepo) DeleteCycle(ctx context.Context, isoYear, isoWeek int) error {
	res := r.db.WithContext(ctx).Exec(
		`DELETE FROM menu_cycles WHERE iso_year = ? AND iso_week = ?`, isoYear, isoWeek)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound(fmt.Sprintf("No cycle for %d-W%02d.", isoYear, isoWeek))
	}
	return nil
}

// ParamRepo is CRUD over sys_parameters.
type ParamRepo struct{ db *gorm.DB }

func NewParamRepo(db *gorm.DB) *ParamRepo { return &ParamRepo{db: db} }

func (r *ParamRepo) List(ctx context.Context) ([]ports.Param, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT key, value, value_type, label, description, group_name, sort_order
		  FROM sys_parameters ORDER BY group_name, sort_order, key`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list params: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ports.Param
	for rows.Next() {
		var p ports.Param
		if err := rows.Scan(&p.Key, &p.Value, &p.ValueType, &p.Label,
			&p.Description, &p.Group, &p.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Set updates a parameter, checking the value against its declared type.
//
// Typing matters here because these values are read by the front end: shipping
// "yes" where the overlay expects a boolean, or "26.000" where it expects an
// integer, produces a page that misbehaves with nothing in any log.
func (r *ParamRepo) Set(ctx context.Context, key, value string) error {
	var valueType string
	if err := r.db.WithContext(ctx).
		Raw(`SELECT value_type FROM sys_parameters WHERE key = ?`, key).
		Scan(&valueType).Error; err != nil {
		return fmt.Errorf("read param: %w", err)
	}
	if valueType == "" {
		return apierror.NotFound(fmt.Sprintf("No parameter %q.", key))
	}
	if err := checkParamValue(valueType, value); err != nil {
		return apierror.Validation(err.Error(),
			map[string]any{"key": key, "value_type": valueType, "value": value})
	}
	res := r.db.WithContext(ctx).Exec(
		`UPDATE sys_parameters SET value = ?, updated_at = now() WHERE key = ?`, value, key)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound(fmt.Sprintf("No parameter %q.", key))
	}
	return nil
}

// RateRepo is the write side of the price catalogue.
type RateRepo struct{ db *gorm.DB }

func NewRateRepo(db *gorm.DB) *RateRepo { return &RateRepo{db: db} }

func (r *RateRepo) SetPlanRates(ctx context.Context, planSlug string, in siteconfig.Rates) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE plan_rates SET daily = ?, weekly_per_day = ?, monthly_per_day = ?,
		       flexi_weekly_per_day = ?, flexi_monthly_per_day = ?, updated_at = now()
		 WHERE plan_id = (SELECT id FROM plans WHERE slug = ?)`,
		in.Daily, in.WeeklyPerDay, in.MonthlyPerDay,
		in.FlexiWeeklyPerDay, in.FlexiMonthlyPerDay, planSlug)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound(fmt.Sprintf("No plan %q.", planSlug))
	}
	return nil
}

func (r *RateRepo) SetTierPrices(ctx context.Context, productSlug, packageName string, bands []siteconfig.Band) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pkgID string
		if err := tx.Raw(`
			SELECT pk.id FROM tier_packages pk
			  JOIN tier_products tp ON tp.id = pk.product_id
			 WHERE tp.slug = ? AND pk.name = ?`, productSlug, packageName).
			Scan(&pkgID).Error; err != nil {
			return fmt.Errorf("find package: %w", err)
		}
		if pkgID == "" {
			return apierror.NotFound(fmt.Sprintf("No package %q on product %q.", packageName, productSlug))
		}
		if err := tx.Exec(`DELETE FROM tier_prices WHERE package_id = ?`, pkgID).Error; err != nil {
			return fmt.Errorf("clear tiers: %w", err)
		}
		for i, b := range bands {
			if err := tx.Exec(`
				INSERT INTO tier_prices (id, package_id, qty_min, qty_max, label, price, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				id.NewString(), pkgID, b.Min, b.Max, b.Label, b.Price, i).Error; err != nil {
				return translate(err)
			}
		}
		return nil
	})
}

func (r *RateRepo) SetKantorRates(ctx context.Context, grade, period string, bands []siteconfig.Band) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM kantor_rates WHERE grade = ? AND period = ?`,
			grade, period).Error; err != nil {
			return fmt.Errorf("clear kantor rates: %w", err)
		}
		for i, b := range bands {
			if err := tx.Exec(`
				INSERT INTO kantor_rates (id, grade, period, pax_min, pax_max, rate_per_pax_day, sort_order)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				id.NewString(), grade, period, b.Min, b.Max, b.Price, i).Error; err != nil {
				return translate(err)
			}
		}
		return nil
	})
}

func checkParamValue(valueType, value string) error {
	switch valueType {
	case "int":
		for _, c := range value {
			if c < '0' || c > '9' {
				return errors.New("value must be a whole number")
			}
		}
		if value == "" {
			return errors.New("value must be a whole number")
		}
	case "bool":
		if value != "true" && value != "false" {
			return errors.New(`value must be "true" or "false"`)
		}
	case "json":
		if !strings.HasPrefix(strings.TrimSpace(value), "{") &&
			!strings.HasPrefix(strings.TrimSpace(value), "[") {
			return errors.New("value must be a JSON object or array")
		}
	case "string":
	default:
		return fmt.Errorf("unknown value_type %q", valueType)
	}
	return nil
}

// translate maps the constraint violations we deliberately rely on onto typed
// errors, so a caller gets an actionable 409/422 instead of an opaque 500.
//
// The exclusion constraint is the important one: it is how the schema refuses
// two published cycles claiming the same day, and that is a conflict the admin
// can fix, not a server fault.
//
// The error type here is *pgconn.PgError, NOT *pq.Error. gorm.io/driver/postgres
// is built on pgx/v5, so a lib/pq type assertion silently never matches and
// every constraint violation falls through as a 500 with a perfectly good
// message sitting unread in the log. That is exactly what happened the first
// time this was written, and it is why the SQLSTATE codes below are spelled out
// rather than looked up by name.
func translate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23P01": // exclusion_violation
			return apierror.Conflict(apierror.CodeCycleOverlap,
				"That week overlaps another published cycle.").
				WithDetails(map[string]any{"constraint": pgErr.ConstraintName}).
				WithCause(err)
		case "23505": // unique_violation
			return apierror.Conflict(apierror.CodeConflict, "That record already exists.").
				WithDetails(map[string]any{"constraint": pgErr.ConstraintName}).
				WithCause(err)
		case "23514": // check_violation
			return apierror.Unprocessable(apierror.CodeRateInvariant,
				fmt.Sprintf("The database rejected this value (%s).", pgErr.ConstraintName)).
				WithCause(err)
		case "23503": // foreign_key_violation
			return apierror.Unprocessable(apierror.CodeValidation,
				"That references something that does not exist.").
				WithDetails(map[string]any{"constraint": pgErr.ConstraintName}).
				WithCause(err)
		}
	}
	// gorm's TranslateError surfaces its own duplicate-key sentinel too.
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apierror.Conflict(apierror.CodeConflict, "That record already exists.")
	}
	return err
}
