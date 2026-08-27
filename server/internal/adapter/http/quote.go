package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/ports"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
	"github.com/stevenwilliam/thenie_v2/server/internal/domain/pricing"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
)

// postQuote prices one order card server-side.
//
// The request carries codes and quantities; every price comes from the
// database. A caller cannot propose a rate, which is the point of moving the
// calculator off the client in the first place.
func postQuote(svc *siteconfig.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req siteconfig.QuoteRequest
		if err := bindJSON(c, &req); err != nil {
			_ = c.Error(err)
			return
		}
		// A quote over a thousand dates is not a real order, it is someone
		// probing. The front end's own range fill caps at 60.
		if len(req.Dates) > 400 {
			_ = c.Error(apierror.Validation("Too many dates in one quote.",
				map[string]any{"dates": len(req.Dates), "max": 400}))
			return
		}
		resp, err := svc.Quote(c.Request.Context(), req)
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, resp)
	}
}

func getPricingRules(svc *siteconfig.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		doc, err := svc.Get(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"pricing_rules": doc.PricingRules, "revision": doc.Revision})
	}
}

// setPricingRules changes how the tier classifier decides.
//
// This is the most consequential write in the service: it does not adjust a
// price, it changes which price applies. So it validates in the domain first,
// and the database has its own CHECKs behind that — a rule set that makes a
// branch unreachable is refused twice.
func setPricingRules(repo ports.RulesRepository, svc *siteconfig.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in pricing.Rules
		if err := bindJSON(c, &in); err != nil {
			_ = c.Error(err)
			return
		}
		if err := in.Validate(); err != nil {
			_ = c.Error(apierror.Unprocessable(apierror.CodeRateInvariant, err.Error()))
			return
		}
		if err := repo.SetPricingRules(c.Request.Context(), in); err != nil {
			_ = c.Error(err)
			return
		}
		svc.Invalidate()
		c.JSON(http.StatusOK, gin.H{"pricing_rules": in})
	}
}
