// Package admin is identity and authorisation for the admin surface: who is
// asking, what they may do, and a record of what they did.
package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/security"
)

// SessionTTL is how long a login lasts. Long enough not to interrupt a menu
// edit, short enough that a forgotten browser session on a shared machine is
// not a standing invitation.
const SessionTTL = 12 * time.Hour

// Lockout policy. Five wrong passwords buys a fifteen-minute pause, which
// makes online guessing pointless without ever permanently locking a real
// person out of their own account.
const (
	MaxFailedAttempts = 5
	LockoutDuration   = 15 * time.Minute
)

// Actor is whoever is making the current request.
type Actor struct {
	UserID      string                 `json:"user_id,omitempty"`
	Name        string                 `json:"name"`
	Email       string                 `json:"email,omitempty"`
	IsService   bool                   `json:"is_service"`
	Permissions security.PermissionSet `json:"-"`
	SessionID   string                 `json:"-"`
}

// Can reports whether the actor holds a permission.
func (a *Actor) Can(p security.Permission) bool {
	return a != nil && a.Permissions.Has(p)
}

// Label is how the actor appears in the audit log.
func (a *Actor) Label() string {
	if a == nil {
		return "anonymous"
	}
	if a.IsService {
		return "service-token"
	}
	if a.Email != "" {
		return a.Email
	}
	return a.Name
}

// User is an admin account.
type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	IsActive    bool       `json:"is_active"`
	Roles       []string   `json:"roles"`
	Permissions []string   `json:"permissions"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	LockedUntil *time.Time `json:"locked_until,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Role is a named bundle of permissions.
type Role struct {
	ID          string   `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
}

// AuditEntry is one recorded action.
type AuditEntry struct {
	ID        string         `json:"id"`
	Actor     string         `json:"actor"`
	Action    string         `json:"action"`
	Target    string         `json:"target,omitempty"`
	Detail    map[string]any `json:"detail,omitempty"`
	IP        string         `json:"ip,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// Repository is what the admin service needs from storage.
type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*StoredUser, error)
	FindUserByID(ctx context.Context, id string) (*StoredUser, error)
	RecordLoginFailure(ctx context.Context, userID string, attempts int, lockUntil *time.Time) error
	RecordLoginSuccess(ctx context.Context, userID string) error

	CreateSession(ctx context.Context, userID, tokenHash string, expires time.Time, ua, ip string) (string, error)
	FindSession(ctx context.Context, tokenHash string) (SessionRow, error)
	TouchSession(ctx context.Context, sessionID string) error
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	PurgeExpiredSessions(ctx context.Context) (int64, error)

	ListUsers(ctx context.Context) ([]User, error)
	CreateUser(ctx context.Context, email, name, passwordHash string, roleCodes []string) (string, error)
	UpdateUser(ctx context.Context, id, name string, isActive bool) error
	SetUserRoles(ctx context.Context, id string, roleCodes []string) error
	SetPassword(ctx context.Context, id, passwordHash string) error
	DeleteUser(ctx context.Context, id string) error
	CountUsersWithPermission(ctx context.Context, perm string, excludeUserID string) (int, error)

	ListRoles(ctx context.Context) ([]Role, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	SetRolePermissions(ctx context.Context, roleCode string, perms []string) error

	WriteAudit(ctx context.Context, userID, actor, action, target string, detail map[string]any, ip string) error
	ListAudit(ctx context.Context, limit int, before *time.Time) ([]AuditEntry, error)
}

// Permission is one capability, as the UI lists it.
type Permission struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Group string `json:"group"`
}

// StoredUser is the raw row, including the fields no caller should ever see.
// It is exported only so the postgres adapter can construct it; nothing outside
// this package and that adapter should touch it.
type StoredUser struct {
	ID             string
	Email          string
	Name           string
	PasswordHash   string
	IsActive       bool
	FailedAttempts int
	LockedUntil    *time.Time
	Permissions    []string
	Roles          []string
}

// SessionRow is a session lookup result, exported for the same reason.
type SessionRow struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
}

// Service is the admin identity service.
type Service struct {
	repo  Repository
	token string // the machine credential; empty disables it
	now   func() time.Time
}

func NewService(repo Repository, serviceToken string) *Service {
	return &Service{repo: repo, token: serviceToken, now: time.Now}
}

var (
	// ErrInvalidLogin is deliberately the SAME error for an unknown email, a
	// wrong password and a deactivated account. Distinguishing them would turn
	// the login form into an account-enumeration oracle.
	ErrInvalidLogin = errors.New("admin: invalid credentials")
	ErrLockedOut    = errors.New("admin: account temporarily locked")
)

// Login verifies credentials and issues a session token.
func (s *Service) Login(ctx context.Context, email, password, ua, ip string) (token string, user *User, err error) {
	email = NormalizeEmail(email)
	if email == "" || password == "" {
		return "", nil, apierror.Unauthorized("Email dan kata sandi wajib diisi.")
	}

	u, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return "", nil, apierror.Internal(err)
	}
	if u == nil {
		// Hash a dummy password anyway so a missing account and a wrong
		// password take the same time. Without this, response timing tells an
		// attacker which addresses are registered.
		_, _ = security.HashPassword("timing-equaliser-" + password)
		return "", nil, apierror.Unauthorized("Email atau kata sandi salah.")
	}
	if u.LockedUntil != nil && u.LockedUntil.After(s.now()) {
		return "", nil, apierror.New(429, apierror.CodeForbidden,
			fmt.Sprintf("Terlalu banyak percobaan. Coba lagi setelah %s.",
				u.LockedUntil.In(time.Local).Format("15:04")))
	}

	ok, verr := security.VerifyPassword(password, u.PasswordHash)
	if verr != nil || !ok || !u.IsActive {
		attempts := u.FailedAttempts + 1
		var lock *time.Time
		if attempts >= MaxFailedAttempts {
			t := s.now().Add(LockoutDuration)
			lock = &t
			attempts = 0
		}
		_ = s.repo.RecordLoginFailure(ctx, u.ID, attempts, lock)
		_ = s.repo.WriteAudit(ctx, "", email, "auth.login.failed", "", nil, ip)
		return "", nil, apierror.Unauthorized("Email atau kata sandi salah.")
	}

	tok, hash, err := security.NewSessionToken()
	if err != nil {
		return "", nil, apierror.Internal(err)
	}
	if _, err := s.repo.CreateSession(ctx, u.ID, hash, s.now().Add(SessionTTL), ua, ip); err != nil {
		return "", nil, apierror.Internal(err)
	}
	_ = s.repo.RecordLoginSuccess(ctx, u.ID)
	_ = s.repo.WriteAudit(ctx, u.ID, u.Email, "auth.login", "", nil, ip)

	return tok, &User{
		ID: u.ID, Email: u.Email, Name: u.Name, IsActive: u.IsActive,
		Roles: u.Roles, Permissions: u.Permissions,
	}, nil
}

// Logout revokes one session.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.DeleteSession(ctx, security.HashSessionToken(token))
}

// Resolve turns a request's credentials into an Actor, or nil when there are
// none. It never returns an error for "not logged in" — that is the caller's
// decision to make.
func (s *Service) Resolve(ctx context.Context, sessionToken, bearer string) (*Actor, error) {
	// The machine credential. Constant-time, and only when one is configured.
	if bearer != "" && s.token != "" && constantTimeEqual(bearer, s.token) {
		return &Actor{
			Name:        "service-token",
			IsService:   true,
			Permissions: security.AllPermissionSet(),
		}, nil
	}
	if sessionToken == "" {
		return nil, nil
	}

	row, err := s.repo.FindSession(ctx, security.HashSessionToken(sessionToken))
	if err != nil {
		return nil, apierror.Internal(err)
	}
	if row.ID == "" || row.ExpiresAt.Before(s.now()) {
		return nil, nil
	}
	u, err := s.repo.FindUserByID(ctx, row.UserID)
	if err != nil {
		return nil, apierror.Internal(err)
	}
	// A user deactivated mid-session loses access on their next request, not
	// when their session eventually expires.
	if u == nil || !u.IsActive {
		return nil, nil
	}
	_ = s.repo.TouchSession(ctx, row.ID)

	return &Actor{
		UserID:      u.ID,
		Name:        u.Name,
		Email:       u.Email,
		Permissions: security.NewPermissionSet(u.Permissions),
		SessionID:   row.ID,
	}, nil
}

// Audit records an action. Failures are logged, never surfaced: a write that
// succeeded must not be reported as failed because its audit line did not land.
func (s *Service) Audit(ctx context.Context, a *Actor, action, target string, detail map[string]any, ip string) {
	userID := ""
	if a != nil {
		userID = a.UserID
	}
	_ = s.repo.WriteAudit(ctx, userID, a.Label(), action, target, detail, ip)
}

// NormalizeEmail lower-cases and trims. Normalising BEFORE validating and
// storing is what stops "Ven@x.com" and "ven@x.com" becoming two accounts.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// ValidateEmail is deliberately loose. This is an internal tool with a handful
// of accounts created by an administrator, not a public sign-up form, so a
// strict RFC pattern would reject valid addresses for no benefit.
func ValidateEmail(s string) error {
	if len(s) < 3 || len(s) > 320 {
		return errors.New("email must be between 3 and 320 characters")
	}
	at := strings.Index(s, "@")
	if at < 1 || at == len(s)-1 || strings.Contains(s, " ") {
		return errors.New("email must look like name@example.com")
	}
	return nil
}

func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
