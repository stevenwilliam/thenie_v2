package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/admin"
	"github.com/stevenwilliam/thenie_v2/server/internal/platform/security"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Skipf("cannot reach test database: %v", err)
	}
	return db
}

// cleanupUser removes an account and everything hanging off it.
func cleanupUser(t *testing.T, db *gorm.DB, email string) {
	t.Helper()
	t.Cleanup(func() { db.Exec(`DELETE FROM admin_users WHERE email = ?`, email) })
	db.Exec(`DELETE FROM admin_users WHERE email = ?`, email)
}

// The union of permissions across roles is the whole point of having roles:
// holding two grants the sum, never the intersection.
func TestRolesGrantTheUnionOfTheirPermissions(t *testing.T) {
	db := testDB(t)
	repo := NewAdminRepo(db)
	ctx := context.Background()

	const email = "union-test@example.com"
	cleanupUser(t, db, email)
	hash, _ := security.HashPassword("a perfectly fine password")

	// viewer grants only reads; editor adds menu and content writes.
	id, err := repo.CreateUser(ctx, email, "Union Test", hash, []string{"viewer", "editor"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := repo.FindUserByID(ctx, id)
	if err != nil || u == nil {
		t.Fatalf("read back: %v", err)
	}
	set := security.NewPermissionSet(u.Permissions)
	for _, want := range []security.Permission{
		security.PermMenuRead, security.PermMenuWrite, security.PermContentWrite, security.PermAuditRead,
	} {
		if !set.Has(want) {
			t.Errorf("holding viewer+editor must grant %q; got %v", want, u.Permissions)
		}
	}
	// Neither role grants these, so the union must not either.
	for _, notWant := range []security.Permission{security.PermUserManage, security.PermPriceWrite} {
		if set.Has(notWant) {
			t.Errorf("neither viewer nor editor grants %q, but the union does", notWant)
		}
	}
}

func TestUnknownRoleIsRefused(t *testing.T) {
	db := testDB(t)
	repo := NewAdminRepo(db)
	const email = "badrole-test@example.com"
	cleanupUser(t, db, email)
	hash, _ := security.HashPassword("a perfectly fine password")

	// Silently creating the account with fewer permissions than asked for would
	// look like the roles system not working.
	if _, err := repo.CreateUser(context.Background(), email, "Bad Role", hash, []string{"wizard"}); err == nil {
		t.Fatal("an unknown role must be refused, not ignored")
	}
	var n int64
	db.Raw(`SELECT count(*) FROM admin_users WHERE email = ?`, email).Scan(&n)
	if n != 0 {
		t.Error("the transaction must roll the half-created account back")
	}
}

// CountUsersWithPermission is what stops the last administrator being deleted.
func TestCountUsersWithPermissionExcludesInactive(t *testing.T) {
	db := testDB(t)
	repo := NewAdminRepo(db)
	ctx := context.Background()
	const email = "lastadmin-test@example.com"
	cleanupUser(t, db, email)
	hash, _ := security.HashPassword("a perfectly fine password")

	before, err := repo.CountUsersWithPermission(ctx, string(security.PermUserManage), "")
	if err != nil {
		t.Fatal(err)
	}
	id, err := repo.CreateUser(ctx, email, "Last Admin", hash, []string{"owner"})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := repo.CountUsersWithPermission(ctx, string(security.PermUserManage), "")
	if after != before+1 {
		t.Fatalf("count = %d, want %d after adding an owner", after, before+1)
	}
	// Excluding the new account must bring the count back.
	excl, _ := repo.CountUsersWithPermission(ctx, string(security.PermUserManage), id)
	if excl != before {
		t.Errorf("excluding the new owner gave %d, want %d", excl, before)
	}
	// A deactivated owner does not count: they cannot log in to fix anything.
	if err := repo.UpdateUser(ctx, id, "Last Admin", false); err != nil {
		t.Fatal(err)
	}
	deact, _ := repo.CountUsersWithPermission(ctx, string(security.PermUserManage), "")
	if deact != before {
		t.Errorf("a deactivated owner still counted: %d, want %d", deact, before)
	}
}

// Sessions must be revocable and must expire.
func TestSessionLifecycle(t *testing.T) {
	db := testDB(t)
	repo := NewAdminRepo(db)
	ctx := context.Background()
	const email = "session-test@example.com"
	cleanupUser(t, db, email)
	hash, _ := security.HashPassword("a perfectly fine password")
	id, err := repo.CreateUser(ctx, email, "Session Test", hash, []string{"viewer"})
	if err != nil {
		t.Fatal(err)
	}

	token, tokenHash, _ := security.NewSessionToken()
	if _, err := repo.CreateSession(ctx, id, tokenHash, time.Now().Add(time.Hour), "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	// The raw token must not be findable in the table — only its hash is stored.
	var n int64
	db.Raw(`SELECT count(*) FROM admin_sessions WHERE token_hash = ?`, token).Scan(&n)
	if n != 0 {
		t.Error("the raw token is stored; a database read would be a session hijack")
	}

	row, err := repo.FindSession(ctx, tokenHash)
	if err != nil || row.UserID != id {
		t.Fatalf("session lookup failed: %+v %v", row, err)
	}

	// Deactivating a user must be able to kill their sessions immediately.
	if err := repo.DeleteUserSessions(ctx, id); err != nil {
		t.Fatal(err)
	}
	row, _ = repo.FindSession(ctx, tokenHash)
	if row.ID != "" {
		t.Error("sessions survived DeleteUserSessions")
	}

	// An expired session is purged. The schema refuses to INSERT a row whose
	// expiry precedes its creation — correctly — so age a valid one instead of
	// weakening the constraint to suit the test.
	_, expiredHash, _ := security.NewSessionToken()
	if _, err := repo.CreateSession(ctx, id, expiredHash, time.Now().Add(time.Hour), "", ""); err != nil {
		t.Fatal(err)
	}
	db.Exec(`UPDATE admin_sessions SET created_at = now() - interval '2 hours',
	                                   expires_at = now() - interval '1 hour'
	          WHERE token_hash = ?`, expiredHash)
	purged, err := repo.PurgeExpiredSessions(ctx)
	if err != nil || purged < 1 {
		t.Errorf("purge removed %d sessions, want at least 1 (err=%v)", purged, err)
	}
}

// The lockout counter and the audit trail are what make a brute-force attempt
// both slow and visible.
func TestLoginLockoutAndAudit(t *testing.T) {
	db := testDB(t)
	repo := NewAdminRepo(db)
	svc := admin.NewService(repo, "")
	ctx := context.Background()
	const email = "lockout-test@example.com"
	const pw = "a perfectly fine password"
	cleanupUser(t, db, email)
	hash, _ := security.HashPassword(pw)
	if _, err := repo.CreateUser(ctx, email, "Lockout Test", hash, []string{"viewer"}); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < admin.MaxFailedAttempts; i++ {
		if _, _, err := svc.Login(ctx, email, "wrong", "ua", "ip"); err == nil {
			t.Fatal("a wrong password must fail")
		}
	}
	// The correct password is now refused too — that is the lockout working.
	if _, _, err := svc.Login(ctx, email, pw, "ua", "ip"); err == nil {
		t.Fatal("the account must be locked after repeated failures")
	}

	// Every failure is recorded.
	var failures int64
	db.Raw(`SELECT count(*) FROM admin_audit_log WHERE actor = ? AND action = 'auth.login.failed'`, email).
		Scan(&failures)
	if failures < int64(admin.MaxFailedAttempts) {
		t.Errorf("only %d failed logins audited, want at least %d", failures, admin.MaxFailedAttempts)
	}
	t.Cleanup(func() { db.Exec(`DELETE FROM admin_audit_log WHERE actor = ?`, email) })

	// Clearing the lock lets the real password back in.
	db.Exec(`UPDATE admin_users SET locked_until = NULL, failed_attempts = 0 WHERE email = ?`, email)
	if _, u, err := svc.Login(ctx, email, pw, "ua", "ip"); err != nil || u == nil {
		t.Fatalf("login should succeed once unlocked: %v", err)
	}
}

// The service token is a machine credential: full permissions, no person.
func TestServiceTokenResolvesToAnAttributableActor(t *testing.T) {
	db := testDB(t)
	svc := admin.NewService(NewAdminRepo(db), "a-long-enough-service-token-value")

	actor, err := svc.Resolve(context.Background(), "", "a-long-enough-service-token-value")
	if err != nil || actor == nil {
		t.Fatalf("the service token must resolve: %v", err)
	}
	if !actor.IsService || actor.Label() != "service-token" {
		t.Errorf("got label %q is_service=%v", actor.Label(), actor.IsService)
	}
	for _, p := range security.AllPermissions() {
		if !actor.Can(p) {
			t.Errorf("the service token must hold %q", p)
		}
	}
	// A wrong token grants nothing.
	if a, _ := svc.Resolve(context.Background(), "", "wrong-token-of-the-same-length!!"); a != nil {
		t.Error("a wrong bearer token must not resolve to an actor")
	}
	// And no credentials at all resolve to nobody, not to an error.
	if a, err := svc.Resolve(context.Background(), "", ""); a != nil || err != nil {
		t.Errorf("no credentials must give (nil, nil), got (%v, %v)", a, err)
	}
}
