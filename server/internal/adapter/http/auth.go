package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/security"
)

const (
	sessionCookie = "thenie_admin"
	actorKey      = "thenie.actor"

	// csrfHeader is required on every state-changing admin request.
	//
	// A browser cannot set a custom header on a cross-origin request without a
	// successful CORS preflight, and this API's CORS allows only GET/HEAD from
	// a fixed origin list. Combined with SameSite=Strict on the cookie, that
	// closes CSRF without a token round-trip. It is documented rather than
	// clever so nobody "simplifies" it away later.
	csrfHeader = "X-Admin-Request"
)

// resolveActor attaches whoever is asking to the request context. It never
// rejects: deciding what an anonymous request may do is requirePerm's job.
func resolveActor(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if svc == nil {
			c.Next()
			return
		}
		var sessionToken string
		if ck, err := c.Request.Cookie(sessionCookie); err == nil {
			sessionToken = ck.Value
		}
		bearer := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")

		actor, err := svc.Resolve(c.Request.Context(), sessionToken, bearer)
		if err != nil {
			_ = c.Error(err)
			c.Abort()
			return
		}
		if actor != nil {
			c.Set(actorKey, actor)
		}
		c.Next()
	}
}

// actorOf returns the current actor, or nil.
func actorOf(c *gin.Context) *admin.Actor {
	if v, ok := c.Get(actorKey); ok {
		if a, ok := v.(*admin.Actor); ok {
			return a
		}
	}
	return nil
}

// requirePerm is the authorisation gate. Deny by default: a route that declares
// no permission is never reachable, because it is never wired to this.
func requirePerm(p security.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := actorOf(c)
		if actor == nil {
			_ = c.Error(apierror.Unauthorized("Silakan masuk terlebih dahulu."))
			c.Abort()
			return
		}
		// CSRF: a state-changing request must carry the custom header. The
		// service token is exempt — it is used by scripts, not browsers, and
		// there is no ambient credential for an attacker to ride.
		if !isSafeMethod(c.Request.Method) && !actor.IsService {
			if c.GetHeader(csrfHeader) == "" {
				_ = c.Error(apierror.Forbidden("Permintaan ditolak: header " + csrfHeader + " wajib ada."))
				c.Abort()
				return
			}
		}
		if !actor.Can(p) {
			_ = c.Error(apierror.Forbidden("Anda tidak punya akses untuk tindakan ini.").
				WithDetails(map[string]any{"required": string(p)}))
			c.Abort()
			return
		}
		c.Next()
	}
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// audit records a completed write. Called by handlers AFTER the work succeeded,
// so the log never claims something that did not happen.
func audit(c *gin.Context, svc *admin.Service, action, target string, detail map[string]any) {
	if svc == nil {
		return
	}
	svc.Audit(c.Request.Context(), actorOf(c), action, target, detail, c.ClientIP())
}

// ---- handlers ----

func login(svc *admin.Service, secureCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		token, user, err := svc.Login(c.Request.Context(), body.Email, body.Password,
			c.Request.UserAgent(), c.ClientIP())
		if err != nil {
			_ = c.Error(err)
			return
		}
		setSessionCookie(c, token, secureCookie)
		c.JSON(http.StatusOK, gin.H{"user": user})
	}
}

func logout(svc *admin.Service, secureCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if ck, err := c.Request.Cookie(sessionCookie); err == nil {
			_ = svc.Logout(c.Request.Context(), ck.Value)
		}
		audit(c, svc, "auth.logout", "", nil)
		clearSessionCookie(c, secureCookie)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// me is what the UI calls on load to decide which screens to show. It is the
// only admin route with no permission requirement — it answers "who am I", and
// an anonymous caller gets a truthful null rather than a 401.
func me() gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := actorOf(c)
		if actor == nil {
			c.JSON(http.StatusOK, gin.H{"user": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": gin.H{
			"user_id":     actor.UserID,
			"name":        actor.Name,
			"email":       actor.Email,
			"is_service":  actor.IsService,
			"permissions": actor.Permissions.Codes(),
		}})
	}
}

func setSessionCookie(c *gin.Context, token string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(admin.SessionTTL.Seconds()),
		HttpOnly: true, // JavaScript must never be able to read it
		Secure:   secure,
		// Strict, not Lax: nothing links into the admin UI from elsewhere, so
		// there is no usability cost, and it removes a whole class of attack.
		SameSite: http.SameSiteStrictMode,
	})
}

func clearSessionCookie(c *gin.Context, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteStrictMode,
	})
}
