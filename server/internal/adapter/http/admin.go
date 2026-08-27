package http

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/ports"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
	"github.com/stevenwilliam/thenie_v2/server/internal/domain/catalogue"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
)

// Every write handler validates in the domain BEFORE it touches the database,
// so a rejected edit produces a precise message rather than a constraint
// violation, and invalidates the read cache AFTER, so the next page load sees
// the change without waiting for a TTL.

func listParams(repo ports.ParamRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		params, err := repo.List(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"params": params})
	}
}

func setParam(repo ports.ParamRepository, svc *siteconfig.Service, aud *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Value *string `json:"value"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		if body.Value == nil {
			_ = c.Error(apierror.Validation("value is required.", nil))
			return
		}
		if err := repo.Set(c.Request.Context(), c.Param("key"), *body.Value); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "param.set", c.Param("key"), map[string]any{"value": *body.Value})
		c.JSON(http.StatusOK, gin.H{"key": c.Param("key"), "value": *body.Value})
	}
}

func setPlanRates(repo ports.RateRepository, svc *siteconfig.Service, aud *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in siteconfig.Rates
		if err := bindJSON(c, &in); err != nil {
			_ = c.Error(err)
			return
		}
		// The domain says no before the database has to.
		r := catalogue.Rates{
			Daily:              in.Daily,
			WeeklyPerDay:       in.WeeklyPerDay,
			MonthlyPerDay:      in.MonthlyPerDay,
			FlexiWeeklyPerDay:  in.FlexiWeeklyPerDay,
			FlexiMonthlyPerDay: in.FlexiMonthlyPerDay,
		}
		if err := r.Validate(); err != nil {
			_ = c.Error(apierror.Unprocessable(apierror.CodeRateInvariant, err.Error()).
				WithDetails(map[string]any{"plan": c.Param("slug")}))
			return
		}
		if err := repo.SetPlanRates(c.Request.Context(), c.Param("slug"), in); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "price.plan_rates", c.Param("slug"), map[string]any{"rates": in})
		c.JSON(http.StatusOK, gin.H{"plan": c.Param("slug"), "rates": in})
	}
}

func setTierPrices(repo ports.RateRepository, svc *siteconfig.Service, aud *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			MinQty int               `json:"min_qty"`
			Tiers  []siteconfig.Band `json:"tiers"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		bands := make([]catalogue.Band, 0, len(body.Tiers))
		for _, b := range body.Tiers {
			bands = append(bands, catalogue.Band{Min: b.Min, Max: b.Max, Price: b.Price, Label: b.Label})
		}
		if err := catalogue.ValidateBands(bands, body.MinQty); err != nil {
			_ = c.Error(apierror.Unprocessable(apierror.CodeValidation, err.Error()))
			return
		}
		if err := repo.SetTierPrices(c.Request.Context(), c.Param("slug"), c.Param("name"), body.Tiers); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "price.tier_prices", c.Param("slug")+"/"+c.Param("name"),
			map[string]any{"tiers": body.Tiers})
		c.JSON(http.StatusOK, gin.H{"product": c.Param("slug"), "package": c.Param("name"), "tiers": body.Tiers})
	}
}

func setKantorRates(repo ports.RateRepository, svc *siteconfig.Service, aud *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Bands []siteconfig.Band `json:"bands"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		bands := make([]catalogue.Band, 0, len(body.Bands))
		for _, b := range body.Bands {
			bands = append(bands, catalogue.Band{Min: b.Min, Max: b.Max, Price: b.Price, Label: b.Label})
		}
		// BR-13.1: Catering Kantor starts at 5 pax.
		if err := catalogue.ValidateBands(bands, 5); err != nil {
			_ = c.Error(apierror.Unprocessable(apierror.CodeValidation, err.Error()))
			return
		}
		if err := repo.SetKantorRates(c.Request.Context(), c.Param("grade"), c.Param("period"), body.Bands); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "price.kantor_rates", c.Param("grade")+"/"+c.Param("period"),
			map[string]any{"bands": body.Bands})
		c.JSON(http.StatusOK, gin.H{"grade": c.Param("grade"), "period": c.Param("period"), "bands": body.Bands})
	}
}

// upsertCycle writes one whole week of menus as a unit.
//
// A week arrives complete or not at all. Half a week in the database is worse
// than none: the page would render Monday to Wednesday and silently drop
// Thursday, and nothing would look broken enough to notice.
func upsertCycle(repo ports.MenuRepository, svc *siteconfig.Service, aud *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ISOYear  int    `json:"iso_year"`
			ISOWeek  int    `json:"iso_week"`
			StartsOn string `json:"starts_on"`
			EndsOn   string `json:"ends_on"`
			Label    string `json:"label"`
			Publish  bool   `json:"publish"`
			Days     map[string][]struct {
				Date      string `json:"date"`
				IsMeatDay bool   `json:"is_meat_day"`
				Kcal      int    `json:"kcal"`
				Items     []struct {
					Name  string `json:"name"`
					Grams int    `json:"grams"`
				} `json:"items"`
			} `json:"days"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		if body.Label == "" {
			_ = c.Error(apierror.Validation("label is required.", nil))
			return
		}

		in := ports.UpsertCycleInput{
			ISOYear:  body.ISOYear,
			ISOWeek:  body.ISOWeek,
			StartsOn: body.StartsOn,
			EndsOn:   body.EndsOn,
			Label:    body.Label,
			Publish:  body.Publish,
			Days:     map[string][]ports.DayInput{},
		}
		for slug, days := range body.Days {
			for _, d := range days {
				items := make([]ports.ComponentInput, 0, len(d.Items))
				for _, it := range d.Items {
					items = append(items, ports.ComponentInput{Name: it.Name, Grams: it.Grams})
				}
				in.Days[slug] = append(in.Days[slug], ports.DayInput{
					Date:      d.Date,
					IsMeatDay: d.IsMeatDay,
					Kcal:      d.Kcal,
					Items:     items,
				})
			}
		}

		id, err := repo.UpsertCycle(c.Request.Context(), in)
		if err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "menu.upsert", fmt.Sprintf("%d-W%02d", body.ISOYear, body.ISOWeek),
			map[string]any{"label": body.Label, "publish": body.Publish, "plans": len(in.Days)})
		c.JSON(http.StatusOK, gin.H{"id": id, "iso_year": body.ISOYear, "iso_week": body.ISOWeek})
	}
}

func publishCycle(repo ports.MenuRepository, svc *siteconfig.Service, aud *admin.Service, publish bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		year, err := pathInt(c, "year")
		if err != nil {
			_ = c.Error(err)
			return
		}
		week, err := pathInt(c, "week")
		if err != nil {
			_ = c.Error(err)
			return
		}
		if err := repo.PublishCycle(c.Request.Context(), year, week, publish); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "menu.publish", fmt.Sprintf("%d-W%02d", year, week),
			map[string]any{"published": publish})
		c.JSON(http.StatusOK, gin.H{"iso_year": year, "iso_week": week, "published": publish})
	}
}

func deleteCycle(repo ports.MenuRepository, svc *siteconfig.Service, aud *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		year, err := pathInt(c, "year")
		if err != nil {
			_ = c.Error(err)
			return
		}
		week, err := pathInt(c, "week")
		if err != nil {
			_ = c.Error(err)
			return
		}
		if err := repo.DeleteCycle(c.Request.Context(), year, week); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		audit(c, aud, "menu.delete", fmt.Sprintf("%d-W%02d", year, week), nil)
		c.Status(http.StatusNoContent)
	}
}
