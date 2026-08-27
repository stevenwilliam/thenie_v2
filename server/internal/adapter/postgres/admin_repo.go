package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/id"
)

// AdminRepo stores identity, authorisation and the audit trail.
type AdminRepo struct{ db *gorm.DB }

func NewAdminRepo(db *gorm.DB) *AdminRepo { return &AdminRepo{db: db} }

// The interface lives in the app layer; these unexported row types are how the
// repository hands rows back without the app importing gorm.
type userRow struct {
	ID             string
	Email          string
	Name           string
	PasswordHash   string
	IsActive       bool
	FailedAttempts int
	LockedUntil    *time.Time
}

func (r *AdminRepo) loadUser(ctx context.Context, where string, arg any) (*admin.StoredUser, error) {
	var row userRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, email, name, password_hash, is_active, failed_attempts, locked_until
		  FROM admin_users WHERE `+where, arg).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("load user: %w", err)
	}
	if row.ID == "" {
		return nil, nil
	}
	perms, roles, err := r.rolesAndPermissions(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return &admin.StoredUser{
		ID: row.ID, Email: row.Email, Name: row.Name,
		PasswordHash: row.PasswordHash, IsActive: row.IsActive,
		FailedAttempts: row.FailedAttempts, LockedUntil: row.LockedUntil,
		Permissions: perms, Roles: roles,
	}, nil
}

// rolesAndPermissions resolves the union of every permission across a user's
// roles. The union is the whole point of roles: holding two of them grants the
// sum, never the intersection.
func (r *AdminRepo) rolesAndPermissions(ctx context.Context, userID string) (perms, roles []string, err error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT rp.permission_code
		  FROM admin_user_roles ur
		  JOIN admin_role_permissions rp ON rp.role_id = ur.role_id
		 WHERE ur.user_id = ?
		 ORDER BY 1`, userID).Rows()
	if err != nil {
		return nil, nil, fmt.Errorf("permissions: %w", err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			_ = rows.Close()
			return nil, nil, err
		}
		perms = append(perms, c)
	}
	_ = rows.Close()

	rrows, err := r.db.WithContext(ctx).Raw(`
		SELECT ro.code FROM admin_user_roles ur
		  JOIN admin_roles ro ON ro.id = ur.role_id
		 WHERE ur.user_id = ? ORDER BY ro.sort_order, ro.code`, userID).Rows()
	if err != nil {
		return nil, nil, fmt.Errorf("roles: %w", err)
	}
	defer func() { _ = rrows.Close() }()
	for rrows.Next() {
		var c string
		if err := rrows.Scan(&c); err != nil {
			return nil, nil, err
		}
		roles = append(roles, c)
	}
	return perms, roles, rrows.Err()
}

func (r *AdminRepo) FindUserByEmail(ctx context.Context, email string) (*admin.StoredUser, error) {
	return r.loadUser(ctx, "email = ?", email)
}

func (r *AdminRepo) FindUserByID(ctx context.Context, uid string) (*admin.StoredUser, error) {
	return r.loadUser(ctx, "id = ?", uid)
}

func (r *AdminRepo) RecordLoginFailure(ctx context.Context, userID string, attempts int, lockUntil *time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE admin_users SET failed_attempts = ?, locked_until = ?, updated_at = now()
		 WHERE id = ?`, attempts, lockUntil, userID).Error
}

func (r *AdminRepo) RecordLoginSuccess(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE admin_users
		   SET failed_attempts = 0, locked_until = NULL, last_login_at = now(), updated_at = now()
		 WHERE id = ?`, userID).Error
}

func (r *AdminRepo) CreateSession(ctx context.Context, userID, tokenHash string, expires time.Time, ua, ip string) (string, error) {
	sid := id.NewString()
	// Bound the stored user agent: it is attacker-controlled and there is no
	// reason to keep a megabyte of it.
	if len(ua) > 400 {
		ua = ua[:400]
	}
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO admin_sessions (id, user_id, token_hash, expires_at, user_agent, ip)
		VALUES (?, ?, ?, ?, ?, ?)`, sid, userID, tokenHash, expires, ua, ip).Error
	if err != nil {
		return "", translate(err)
	}
	return sid, nil
}

func (r *AdminRepo) FindSession(ctx context.Context, tokenHash string) (admin.SessionRow, error) {
	var row admin.SessionRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, user_id, expires_at FROM admin_sessions WHERE token_hash = ?`,
		tokenHash).Scan(&row).Error
	if err != nil {
		return row, fmt.Errorf("find session: %w", err)
	}
	return row, nil
}

func (r *AdminRepo) TouchSession(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Exec(
		`UPDATE admin_sessions SET last_seen_at = now() WHERE id = ?`, sessionID).Error
}

func (r *AdminRepo) DeleteSession(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM admin_sessions WHERE token_hash = ?`, tokenHash).Error
}

func (r *AdminRepo) DeleteUserSessions(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Exec(
		`DELETE FROM admin_sessions WHERE user_id = ?`, userID).Error
}

func (r *AdminRepo) PurgeExpiredSessions(ctx context.Context) (int64, error) {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM admin_sessions WHERE expires_at < now()`)
	return res.RowsAffected, res.Error
}

func (r *AdminRepo) ListUsers(ctx context.Context) ([]admin.User, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, email, name, is_active, last_login_at, locked_until, created_at
		  FROM admin_users ORDER BY name`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []admin.User
	for rows.Next() {
		var u admin.User
		var last, locked sql.NullTime
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.IsActive, &last, &locked, &u.CreatedAt); err != nil {
			return nil, err
		}
		if last.Valid {
			t := last.Time
			u.LastLoginAt = &t
		}
		if locked.Valid && locked.Time.After(time.Now()) {
			t := locked.Time
			u.LockedUntil = &t
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		perms, roles, err := r.rolesAndPermissions(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Permissions, out[i].Roles = perms, roles
	}
	return out, nil
}

func (r *AdminRepo) CreateUser(ctx context.Context, email, name, passwordHash string, roleCodes []string) (string, error) {
	uid := id.NewString()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO admin_users (id, email, name, password_hash)
			VALUES (?, ?, ?, ?)`, uid, email, name, passwordHash).Error; err != nil {
			return translate(err)
		}
		return assignRoles(tx, uid, roleCodes)
	})
	if err != nil {
		return "", err
	}
	return uid, nil
}

func (r *AdminRepo) UpdateUser(ctx context.Context, uid, name string, isActive bool) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE admin_users SET name = ?, is_active = ?, updated_at = now() WHERE id = ?`,
		name, isActive, uid)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound("Pengguna tidak ditemukan.")
	}
	return nil
}

func (r *AdminRepo) SetUserRoles(ctx context.Context, uid string, roleCodes []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM admin_user_roles WHERE user_id = ?`, uid).Error; err != nil {
			return err
		}
		return assignRoles(tx, uid, roleCodes)
	})
}

func (r *AdminRepo) SetPassword(ctx context.Context, uid, passwordHash string) error {
	res := r.db.WithContext(ctx).Exec(`
		UPDATE admin_users SET password_hash = ?, failed_attempts = 0, locked_until = NULL,
		       updated_at = now() WHERE id = ?`, passwordHash, uid)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound("Pengguna tidak ditemukan.")
	}
	return nil
}

func (r *AdminRepo) DeleteUser(ctx context.Context, uid string) error {
	res := r.db.WithContext(ctx).Exec(`DELETE FROM admin_users WHERE id = ?`, uid)
	if res.Error != nil {
		return translate(res.Error)
	}
	if res.RowsAffected == 0 {
		return apierror.NotFound("Pengguna tidak ditemukan.")
	}
	return nil
}

// CountUsersWithPermission counts ACTIVE users holding a permission, optionally
// excluding one. It is what stops the last administrator being removed.
func (r *AdminRepo) CountUsersWithPermission(ctx context.Context, perm, excludeUserID string) (int, error) {
	// The exclusion is a NULLable uuid parameter rather than a string compared
	// with '': Postgres evaluates the ''::uuid cast even when the OR would
	// short-circuit past it, so the empty case raised
	// "invalid input syntax for type uuid" instead of counting everyone.
	// IS DISTINCT FROM handles the NULL case without a second query.
	var exclude any
	if excludeUserID != "" {
		exclude = excludeUserID
	}
	var n int
	err := r.db.WithContext(ctx).Raw(`
		SELECT count(DISTINCT u.id)
		  FROM admin_users u
		  JOIN admin_user_roles ur ON ur.user_id = u.id
		  JOIN admin_role_permissions rp ON rp.role_id = ur.role_id
		 WHERE u.is_active
		   AND rp.permission_code = ?
		   AND u.id IS DISTINCT FROM ?::uuid`, perm, exclude).Scan(&n).Error
	if err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

func (r *AdminRepo) ListRoles(ctx context.Context) ([]admin.Role, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT id, code, name, description, is_system FROM admin_roles ORDER BY sort_order, code`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []admin.Role
	for rows.Next() {
		var role admin.Role
		if err := rows.Scan(&role.ID, &role.Code, &role.Name, &role.Description, &role.IsSystem); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		prows, err := r.db.WithContext(ctx).Raw(`
			SELECT permission_code FROM admin_role_permissions WHERE role_id = ? ORDER BY 1`,
			out[i].ID).Rows()
		if err != nil {
			return nil, err
		}
		for prows.Next() {
			var c string
			if err := prows.Scan(&c); err != nil {
				_ = prows.Close()
				return nil, err
			}
			out[i].Permissions = append(out[i].Permissions, c)
		}
		_ = prows.Close()
	}
	return out, nil
}

func (r *AdminRepo) ListPermissions(ctx context.Context) ([]admin.Permission, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT code, label, group_name FROM admin_permissions ORDER BY group_name, sort_order, code`).Rows()
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []admin.Permission
	for rows.Next() {
		var p admin.Permission
		if err := rows.Scan(&p.Code, &p.Label, &p.Group); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *AdminRepo) SetRolePermissions(ctx context.Context, roleCode string, perms []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var roleID string
		if err := tx.Raw(`SELECT id FROM admin_roles WHERE code = ?`, roleCode).Scan(&roleID).Error; err != nil {
			return err
		}
		if roleID == "" {
			return apierror.NotFound(fmt.Sprintf("Peran %q tidak ditemukan.", roleCode))
		}
		if err := tx.Exec(`DELETE FROM admin_role_permissions WHERE role_id = ?`, roleID).Error; err != nil {
			return err
		}
		for _, p := range perms {
			if err := tx.Exec(`
				INSERT INTO admin_role_permissions (role_id, permission_code) VALUES (?, ?)`,
				roleID, p).Error; err != nil {
				return translate(err)
			}
		}
		return nil
	})
}

func (r *AdminRepo) WriteAudit(ctx context.Context, userID, actor, action, target string, detail map[string]any, ip string) error {
	raw := []byte("{}")
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			raw = b
		}
	}
	var uid any
	if userID != "" {
		uid = userID
	}
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO admin_audit_log (id, user_id, actor, action, target, detail, ip)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id.NewString(), uid, actor, action, target, string(raw), ip).Error
}

func (r *AdminRepo) ListAudit(ctx context.Context, limit int, before *time.Time) ([]admin.AuditEntry, error) {
	q := `SELECT id, actor, action, target, detail, ip, created_at
	        FROM admin_audit_log`
	args := []any{}
	if before != nil {
		q += ` WHERE created_at < ?`
		args = append(args, *before)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.WithContext(ctx).Raw(q, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []admin.AuditEntry
	for rows.Next() {
		var e admin.AuditEntry
		var raw []byte
		if err := rows.Scan(&e.ID, &e.Actor, &e.Action, &e.Target, &raw, &e.IP, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &e.Detail)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// assignRoles attaches roles by code, refusing an unknown one rather than
// silently creating an account with fewer permissions than intended.
func assignRoles(tx *gorm.DB, userID string, codes []string) error {
	for _, code := range codes {
		var roleID string
		if err := tx.Raw(`SELECT id FROM admin_roles WHERE code = ?`, code).Scan(&roleID).Error; err != nil {
			return err
		}
		if roleID == "" {
			return apierror.Validation(fmt.Sprintf("Peran %q tidak ditemukan.", code),
				map[string]any{"role": code})
		}
		if err := tx.Exec(`
			INSERT INTO admin_user_roles (user_id, role_id) VALUES (?, ?)
			ON CONFLICT DO NOTHING`, userID, roleID).Error; err != nil {
			return translate(err)
		}
	}
	return nil
}

var _ = errors.Is
