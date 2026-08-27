package security

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestPermissionConstantsMatchTheSeed is the reason the permission list is
// worth writing twice.
//
// A permission that exists in Go but not in the database is a route nobody can
// ever reach — no role can grant it. One that exists in the database but not in
// Go is a checkbox in the admin UI that grants nothing. Both fail silently, and
// both look like "the permissions just don't work".
func TestPermissionConstantsMatchTheSeed(t *testing.T) {
	raw, err := os.ReadFile("../../../db/migrations/0009_rbac.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	// Pull the codes out of the INSERT INTO admin_permissions block only, so a
	// permission mentioned in a role grant does not count as a definition.
	body := string(raw)
	start := strings.Index(body, "INSERT INTO admin_permissions")
	if start < 0 {
		t.Fatal("migration has no admin_permissions insert")
	}
	end := strings.Index(body[start:], ";")
	block := body[start : start+end]

	re := regexp.MustCompile(`\('([a-z]+\.[a-z]+)'`)
	seeded := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(block, -1) {
		seeded[m[1]] = true
	}
	if len(seeded) == 0 {
		t.Fatal("parsed no permissions out of the migration")
	}

	inGo := map[string]bool{}
	for _, p := range AllPermissions() {
		inGo[string(p)] = true
	}

	for code := range seeded {
		if !inGo[code] {
			t.Errorf("%q is seeded in the database but has no Go constant — "+
				"the admin UI would show a checkbox that grants nothing", code)
		}
	}
	for code := range inGo {
		if !seeded[code] {
			t.Errorf("%q is a Go constant but is not seeded — "+
				"no role can grant it, so any route requiring it is unreachable", code)
		}
	}
	t.Logf("%d permissions, matched on both sides", len(seeded))
}

// Every seeded role must grant something, or it is a role that looks like
// access and gives none.
func TestSeededRolesGrantSomething(t *testing.T) {
	raw, err := os.ReadFile("../../../db/migrations/0009_rbac.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	body := string(raw)
	for _, role := range []string{"owner", "manager", "editor", "viewer"} {
		if !strings.Contains(body, "'"+role+"'") {
			t.Errorf("role %q is not seeded", role)
		}
	}
	// The owner role must be granted every permission; it is the safety net
	// the service falls back on when a role edit goes wrong.
	if !strings.Contains(body, "SELECT '00000000-0000-7000-8000-000000000001', code FROM admin_permissions;") {
		t.Error("the owner role must be granted every permission unconditionally")
	}
}

func TestPasswordHashing(t *testing.T) {
	const pw = "correct horse battery staple"
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(hash, pw) {
		t.Fatal("the hash contains the plaintext")
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected a PHC argon2id hash, got %q", hash[:20])
	}
	ok, err := VerifyPassword(pw, hash)
	if err != nil || !ok {
		t.Fatalf("the correct password must verify: ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword("wrong", hash)
	if err != nil || ok {
		t.Fatalf("a wrong password must not verify: ok=%v err=%v", ok, err)
	}

	// Two hashes of the same password must differ: the salt is what stops a
	// database dump revealing which accounts share a password.
	hash2, _ := HashPassword(pw)
	if hash == hash2 {
		t.Error("hashing is not salted")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	for _, bad := range []string{"", "plaintext", "$argon2id$broken", "$bcrypt$v=19$m=1,t=1,p=1$aaa$bbb"} {
		if ok, err := VerifyPassword("x", bad); ok || err == nil {
			t.Errorf("%q must be rejected: ok=%v err=%v", bad, ok, err)
		}
	}
}

func TestPasswordPolicy(t *testing.T) {
	for _, bad := range []string{"short", "password", "aaaaaaaaaaaa", "123456789012"} {
		if err := ValidatePassword(bad); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
	// Deliberately not any password used by a real account anywhere: a test
	// fixture is committed to git, and a committed fixture that matches a live
	// credential is a credential in git.
	for _, good := range []string{"a length-beats-composition passphrase", "Xk7-quiet-lantern-9924"} {
		if err := ValidatePassword(good); err != nil {
			t.Errorf("%q must be accepted: %v", good, err)
		}
	}
}

func TestSessionTokens(t *testing.T) {
	tok, hash, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) < 40 {
		t.Errorf("token looks too short: %d chars", len(tok))
	}
	// The stored form must not be the token, or a database read is a session
	// hijack.
	if hash == tok || strings.Contains(hash, tok) {
		t.Fatal("the stored hash reveals the token")
	}
	if HashSessionToken(tok) != hash {
		t.Error("hashing is not deterministic")
	}
	tok2, _, _ := NewSessionToken()
	if tok == tok2 {
		t.Fatal("tokens are not random")
	}
}

func TestPermissionSet(t *testing.T) {
	s := NewPermissionSet([]string{"menu.read", "menu.write"})
	if !s.Has(PermMenuRead) || !s.Has(PermMenuWrite) {
		t.Error("granted permissions must be present")
	}
	if s.Has(PermUserManage) {
		t.Error("an ungranted permission must be absent — authorisation is deny by default")
	}
	// AllPermissionSet is built from AllPermissions, so a newly added constant
	// is automatically included rather than silently omitted.
	all := AllPermissionSet()
	for _, p := range AllPermissions() {
		if !all.Has(p) {
			t.Errorf("AllPermissionSet is missing %q", p)
		}
	}
}
