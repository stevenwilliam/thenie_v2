package admin

import (
	"context"
	"fmt"

	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/security"
)

// ListUsers returns every admin account.
func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	u, err := s.repo.ListUsers(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return u, nil
}

// CreateUser adds an account.
func (s *Service) CreateUser(ctx context.Context, email, name, password string, roles []string) (string, error) {
	email = NormalizeEmail(email)
	if err := ValidateEmail(email); err != nil {
		return "", apierror.Validation(err.Error(), map[string]any{"email": email})
	}
	if name == "" {
		return "", apierror.Validation("Nama wajib diisi.", nil)
	}
	if err := security.ValidatePassword(password); err != nil {
		return "", apierror.Validation(err.Error(), nil)
	}
	if len(roles) == 0 {
		// An account with no role can log in and do nothing, which looks like a
		// broken system rather than a deliberate choice.
		return "", apierror.Validation("Pilih minimal satu peran.", nil)
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return "", apierror.Internal(err)
	}
	id, err := s.repo.CreateUser(ctx, email, name, hash, roles)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpdateUser changes a name or activation state.
func (s *Service) UpdateUser(ctx context.Context, actor *Actor, id, name string, isActive bool) error {
	if name == "" {
		return apierror.Validation("Nama wajib diisi.", nil)
	}
	// Deactivating yourself is the fastest way to lock yourself out of the
	// system you are administering.
	if actor != nil && actor.UserID == id && !isActive {
		return apierror.Unprocessable(apierror.CodeValidation,
			"Anda tidak bisa menonaktifkan akun Anda sendiri.")
	}
	if !isActive {
		if err := s.guardLastAdmin(ctx, id); err != nil {
			return err
		}
	}
	if err := s.repo.UpdateUser(ctx, id, name, isActive); err != nil {
		return err
	}
	if !isActive {
		// A deactivated account's live sessions must die with it, or the person
		// stays logged in until their session happens to expire.
		_ = s.repo.DeleteUserSessions(ctx, id)
	}
	return nil
}

// SetUserRoles replaces an account's roles.
func (s *Service) SetUserRoles(ctx context.Context, actor *Actor, id string, roles []string) error {
	if len(roles) == 0 {
		return apierror.Validation("Pilih minimal satu peran.", nil)
	}
	// Removing your own user.manage is a one-way door if you are the last one
	// holding it; guard it the same way as deletion.
	if err := s.guardLastAdminAfterRoleChange(ctx, id, roles); err != nil {
		return err
	}
	return s.repo.SetUserRoles(ctx, id, roles)
}

// SetPassword changes an account's password and revokes its sessions.
func (s *Service) SetPassword(ctx context.Context, id, password string) error {
	if err := security.ValidatePassword(password); err != nil {
		return apierror.Validation(err.Error(), nil)
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return apierror.Internal(err)
	}
	if err := s.repo.SetPassword(ctx, id, hash); err != nil {
		return err
	}
	// A password change means "I no longer trust what was open before".
	_ = s.repo.DeleteUserSessions(ctx, id)
	return nil
}

// DeleteUser removes an account.
func (s *Service) DeleteUser(ctx context.Context, actor *Actor, id string) error {
	if actor != nil && actor.UserID == id {
		return apierror.Unprocessable(apierror.CodeValidation,
			"Anda tidak bisa menghapus akun Anda sendiri.")
	}
	if err := s.guardLastAdmin(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteUser(ctx, id)
}

// guardLastAdmin refuses to remove the last account that can manage users.
//
// Without this, one careless click leaves a system nobody can administer, and
// the only way back is a hand-written SQL statement against production.
func (s *Service) guardLastAdmin(ctx context.Context, excludeUserID string) error {
	n, err := s.repo.CountUsersWithPermission(ctx, string(security.PermUserManage), excludeUserID)
	if err != nil {
		return apierror.Internal(err)
	}
	if n == 0 {
		return apierror.Unprocessable(apierror.CodeValidation,
			"Ini satu-satunya akun yang bisa mengelola pengguna. Buat atau aktifkan akun lain dulu.")
	}
	return nil
}

// guardLastAdminAfterRoleChange applies the same rule to a role edit that would
// drop user.manage.
func (s *Service) guardLastAdminAfterRoleChange(ctx context.Context, userID string, newRoles []string) error {
	roles, err := s.repo.ListRoles(ctx)
	if err != nil {
		return apierror.Internal(err)
	}
	keeps := false
	wanted := map[string]bool{}
	for _, r := range newRoles {
		wanted[r] = true
	}
	for _, r := range roles {
		if !wanted[r.Code] {
			continue
		}
		for _, p := range r.Permissions {
			if p == string(security.PermUserManage) {
				keeps = true
			}
		}
	}
	if keeps {
		return nil
	}
	return s.guardLastAdmin(ctx, userID)
}

// ListRoles returns every role with its permissions.
func (s *Service) ListRoles(ctx context.Context) ([]Role, error) {
	r, err := s.repo.ListRoles(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return r, nil
}

// ListPermissions returns the catalogue the UI renders as checkboxes.
func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	p, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return p, nil
}

// SetRolePermissions replaces a role's permissions.
func (s *Service) SetRolePermissions(ctx context.Context, roleCode string, perms []string) error {
	known := map[string]bool{}
	for _, p := range security.AllPermissions() {
		known[string(p)] = true
	}
	for _, p := range perms {
		if !known[p] {
			return apierror.Validation(fmt.Sprintf("Permission tidak dikenal: %q.", p),
				map[string]any{"permission": p})
		}
	}
	// The owner role is the floor of the system. Editing it is how an
	// administrator removes their own ability to fix the mistake.
	if roleCode == "owner" {
		return apierror.Unprocessable(apierror.CodeValidation,
			"Peran Owner tidak bisa diubah — peran ini adalah jaring pengaman sistem.")
	}
	return s.repo.SetRolePermissions(ctx, roleCode, perms)
}

// ListAudit returns recent activity.
func (s *Service) ListAudit(ctx context.Context, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	e, err := s.repo.ListAudit(ctx, limit, nil)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	return e, nil
}

// PurgeExpiredSessions drops sessions past their expiry.
func (s *Service) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	return s.repo.PurgeExpiredSessions(ctx)
}
