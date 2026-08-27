// Package http is the gin adapter: routing, request/response mapping, and the
// middleware that turns a typed error into the one JSON error model.
//
// The public surface is exactly one GET. Everything that writes sits behind a
// bearer token and is grouped under /api/v1/admin.
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/ports"
	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/logging"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/security"
)

// Deps are what the router needs wired in.
type Deps struct {
	Config       *siteconfig.Service
	Menu         ports.MenuRepository
	Params       ports.ParamRepository
	Rates        ports.RateRepository
	Rules        ports.RulesRepository
	Admin        *admin.Service
	Log          *slog.Logger
	AdminToken   string
	SecureCookie bool
	AdminUI      http.Handler
	CORSOrigins  []string
	Version      string
}

// New builds the router.
func New(d Deps) *gin.Engine {
	r := gin.New()
	r.Use(requestID(), recovery(d.Log), accessLog(d.Log), errorRenderer(d.Log))
	if len(d.CORSOrigins) > 0 {
		r.Use(cors(d.CORSOrigins))
	}

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "version": d.Version})
	})
	r.GET("/readyz", func(c *gin.Context) {
		if _, err := d.Config.Revision(c.Request.Context()); err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	v1 := r.Group("/api/v1")
	{
		v1.GET("/site-config", getSiteConfig(d.Config))
		v1.HEAD("/site-config", getSiteConfig(d.Config))
		v1.GET("/site-config/revision", getRevision(d.Config))
		// Pricing is a POST because the request carries a date set, not because
		// it mutates anything. It is public for the same reason site-config is:
		// the page already computes these numbers in the browser, so the server
		// answering the same question reveals nothing new.
		v1.POST("/quote", postQuote(d.Config))
	}

	// Every actor is resolved once, up front. Authorisation is then per-route
	// and deny-by-default: a handler is only reachable through a requirePerm
	// gate, so adding a route without one makes it unreachable rather than
	// public.
	authed := r.Group("/api/v1/admin", resolveActor(d.Admin))
	{
		authed.POST("/auth/login", login(d.Admin, d.SecureCookie))
		authed.POST("/auth/logout", logout(d.Admin, d.SecureCookie))
		authed.GET("/auth/me", me())

		authed.GET("/params", requirePerm(security.PermContentRead), listParams(d.Params))
		authed.PUT("/params/:key", requirePerm(security.PermContentWrite), setParam(d.Params, d.Config, d.Admin))

		authed.GET("/pricing-rules", requirePerm(security.PermRulesRead), getPricingRules(d.Config))
		authed.PUT("/pricing-rules", requirePerm(security.PermRulesWrite), setPricingRules(d.Rules, d.Config, d.Admin))

		authed.PUT("/plans/:slug/rates", requirePerm(security.PermPriceWrite), setPlanRates(d.Rates, d.Config, d.Admin))
		authed.PUT("/tier-products/:slug/packages/:name/prices", requirePerm(security.PermPriceWrite), setTierPrices(d.Rates, d.Config, d.Admin))
		authed.PUT("/kantor/:grade/:period/rates", requirePerm(security.PermPriceWrite), setKantorRates(d.Rates, d.Config, d.Admin))

		authed.PUT("/menu/cycles", requirePerm(security.PermMenuWrite), upsertCycle(d.Menu, d.Config, d.Admin))
		authed.POST("/menu/cycles/:year/:week/publish", requirePerm(security.PermMenuPublish), publishCycle(d.Menu, d.Config, d.Admin, true))
		authed.POST("/menu/cycles/:year/:week/unpublish", requirePerm(security.PermMenuPublish), publishCycle(d.Menu, d.Config, d.Admin, false))
		authed.DELETE("/menu/cycles/:year/:week", requirePerm(security.PermMenuWrite), deleteCycle(d.Menu, d.Config, d.Admin))

		authed.GET("/validate", requirePerm(security.PermContentRead), validateDocument(d.Config))

		authed.GET("/users", requirePerm(security.PermUserManage), listUsers(d.Admin))
		authed.POST("/users", requirePerm(security.PermUserManage), createUser(d.Admin))
		authed.PUT("/users/:id", requirePerm(security.PermUserManage), updateUser(d.Admin))
		authed.PUT("/users/:id/roles", requirePerm(security.PermUserManage), setUserRoles(d.Admin))
		authed.PUT("/users/:id/password", requirePerm(security.PermUserManage), setUserPassword(d.Admin))
		authed.DELETE("/users/:id", requirePerm(security.PermUserManage), deleteUser(d.Admin))

		authed.GET("/roles", requirePerm(security.PermUserManage), listRoles(d.Admin))
		authed.PUT("/roles/:code/permissions", requirePerm(security.PermUserManage), setRolePermissions(d.Admin))

		authed.GET("/audit", requirePerm(security.PermAuditRead), listAudit(d.Admin))
	}

	// The admin UI itself. Static assets only — every byte of authority is in
	// the API above, so serving the page to an anonymous visitor reveals
	// nothing but a login form.
	if d.AdminUI != nil {
		r.GET("/admin", func(c *gin.Context) { c.Redirect(http.StatusFound, "/admin/") })
		r.Any("/admin/*filepath", gin.WrapH(http.StripPrefix("/admin/", d.AdminUI)))
	}

	r.NoRoute(func(c *gin.Context) {
		_ = c.Error(apierror.NotFound("No such endpoint."))
	})
	return r
}

// getSiteConfig serves the document, honouring If-None-Match.
//
// The ETag is the content revision, so a browser that already has the current
// version pays 200 bytes instead of 60 KB. On a page this heavy that is not a
// micro-optimisation -- it is the difference between the overlay being free and
// the overlay being a second download.
func getSiteConfig(svc *siteconfig.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rev, err := svc.Revision(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		etag := fmt.Sprintf(`"rev-%d"`, rev)
		c.Header("ETag", etag)
		// The document is public and changes rarely, but a stale menu is worse
		// than a re-fetch, so revalidate every time and let the ETag make that
		// cheap.
		c.Header("Cache-Control", "public, max-age=0, must-revalidate")
		c.Header("Vary", "Origin")

		if match := c.GetHeader("If-None-Match"); match != "" && strings.Contains(match, etag) {
			c.Status(http.StatusNotModified)
			return
		}

		doc, err := svc.Get(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, doc)
	}
}

func getRevision(svc *siteconfig.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		rev, err := svc.Revision(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"revision": rev})
	}
}

// validateDocument re-runs every domain rule over what is actually in the
// database and reports whatever is wrong. It is the endpoint to hit after a
// restore or a hand-run UPDATE.
func validateDocument(svc *siteconfig.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		doc, err := svc.Get(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		problems := doc.Validate()
		msgs := make([]string, 0, len(problems))
		for _, p := range problems {
			msgs = append(msgs, p.Error())
		}
		status := http.StatusOK
		if len(msgs) > 0 {
			status = http.StatusUnprocessableEntity
		}
		c.JSON(status, gin.H{"revision": doc.Revision, "ok": len(msgs) == 0, "problems": msgs})
	}
}

// ---- middleware ----

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = uuid.NewString()
		}
		c.Header("X-Request-Id", id)
		c.Request = c.Request.WithContext(logging.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

func accessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if log == nil {
			return
		}
		log.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", logging.RequestID(c.Request.Context()),
		)
	}
}

func recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if log != nil {
					log.Error("panic recovered", "panic", fmt.Sprint(rec), "path", c.Request.URL.Path)
				}
				_ = c.Error(apierror.Internal(fmt.Errorf("panic: %v", rec)))
				c.Abort()
				renderErrors(c, log)
			}
		}()
		c.Next()
	}
}

// errorRenderer turns whatever the handlers pushed onto gin's error list into
// the single JSON error model. Handlers never write an error body themselves.
func errorRenderer(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		renderErrors(c, log)
	}
}

func renderErrors(c *gin.Context, log *slog.Logger) {
	if len(c.Errors) == 0 || c.Writer.Written() {
		return
	}
	ae := apierror.From(c.Errors.Last().Err)
	if ae.Status >= 500 && log != nil {
		// The cause is logged here and nowhere else; the client never sees it.
		log.Error("request failed",
			"code", string(ae.Code),
			"error", ae.Error(),
			"path", c.Request.URL.Path,
			"request_id", logging.RequestID(c.Request.Context()),
		)
	}
	body := gin.H{"code": ae.Code, "message": ae.Message}
	if len(ae.Details) > 0 {
		body["details"] = ae.Details
	}
	c.AbortWithStatusJSON(ae.Status, gin.H{"error": body})
}

func cors(origins []string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[strings.ToLower(strings.TrimRight(o, "/"))] = true
	}
	return func(c *gin.Context) {
		origin := strings.ToLower(strings.TrimRight(c.GetHeader("Origin"), "/"))
		// Never reflect an arbitrary Origin: this API is read-anonymous but
		// write-authenticated, and a reflected origin plus a leaked token is a
		// working cross-site write.
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", c.GetHeader("Origin"))
			c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "If-None-Match, Content-Type")
			c.Header("Access-Control-Expose-Headers", "ETag")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// bindJSON decodes a request body into v, turning a decode failure into a
// typed validation error rather than a 500.
func bindJSON(c *gin.Context, v any) error {
	if err := c.ShouldBindJSON(v); err != nil {
		return apierror.Validation("The request body could not be read.",
			map[string]any{"reason": err.Error()})
	}
	return nil
}

func pathInt(c *gin.Context, key string) (int, error) {
	n, err := strconv.Atoi(c.Param(key))
	if err != nil {
		return 0, apierror.Validation(fmt.Sprintf("%s must be a number.", key),
			map[string]any{key: c.Param(key)})
	}
	return n, nil
}
