package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
)

func listUsers(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		users, err := svc.ListUsers(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"users": users})
	}
}

func createUser(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Email    string   `json:"email"`
			Name     string   `json:"name"`
			Password string   `json:"password"`
			Roles    []string `json:"roles"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		id, err := svc.CreateUser(c.Request.Context(), body.Email, body.Name, body.Password, body.Roles)
		if err != nil {
			_ = c.Error(err)
			return
		}
		// The password is never echoed, logged or audited — only the fact that
		// an account was created, and with which roles.
		audit(c, svc, "user.create", body.Email, map[string]any{"roles": body.Roles})
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func updateUser(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name     string `json:"name"`
			IsActive *bool  `json:"is_active"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		active := true
		if body.IsActive != nil {
			active = *body.IsActive
		}
		if err := svc.UpdateUser(c.Request.Context(), actorOf(c), c.Param("id"), body.Name, active); err != nil {
			_ = c.Error(err)
			return
		}
		audit(c, svc, "user.update", c.Param("id"), map[string]any{"is_active": active})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setUserRoles(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Roles []string `json:"roles"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		if err := svc.SetUserRoles(c.Request.Context(), actorOf(c), c.Param("id"), body.Roles); err != nil {
			_ = c.Error(err)
			return
		}
		audit(c, svc, "user.roles", c.Param("id"), map[string]any{"roles": body.Roles})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func setUserPassword(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Password string `json:"password"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		if err := svc.SetPassword(c.Request.Context(), c.Param("id"), body.Password); err != nil {
			_ = c.Error(err)
			return
		}
		audit(c, svc, "user.password", c.Param("id"), nil)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func deleteUser(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := svc.DeleteUser(c.Request.Context(), actorOf(c), c.Param("id")); err != nil {
			_ = c.Error(err)
			return
		}
		audit(c, svc, "user.delete", c.Param("id"), nil)
		c.Status(http.StatusNoContent)
	}
}

func listRoles(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, err := svc.ListRoles(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		perms, err := svc.ListPermissions(c.Request.Context())
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"roles": roles, "permissions": perms})
	}
}

func setRolePermissions(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Permissions []string `json:"permissions"`
		}
		if err := bindJSON(c, &body); err != nil {
			_ = c.Error(err)
			return
		}
		if err := svc.SetRolePermissions(c.Request.Context(), c.Param("code"), body.Permissions); err != nil {
			_ = c.Error(err)
			return
		}
		audit(c, svc, "role.permissions", c.Param("code"),
			map[string]any{"permissions": body.Permissions})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

func listAudit(svc *admin.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 100
		if v := c.Query("limit"); v != "" {
			if n, err := parseLimit(v); err == nil {
				limit = n
			} else {
				_ = c.Error(apierror.Validation("limit harus angka 1–500.", nil))
				return
			}
		}
		entries, err := svc.ListAudit(c.Request.Context(), limit)
		if err != nil {
			_ = c.Error(err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"entries": entries})
	}
}

func parseLimit(s string) (int, error) {
	var n int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, apierror.Validation("limit harus angka.", nil)
		}
		n = n*10 + int(r-'0')
		if n > 500 {
			return 500, nil
		}
	}
	if n == 0 {
		return 0, apierror.Validation("limit harus lebih dari 0.", nil)
	}
	return n, nil
}
