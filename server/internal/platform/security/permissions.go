package security

import "sort"

// Permission is a single capability a handler may require.
//
// Authorisation is DENY BY DEFAULT: a route that declares no permission serves
// nobody. These constants must match the codes seeded in
// db/migrations/0009_rbac.up.sql — TestPermissionConstantsMatchTheSeed fails if
// they drift, because a permission that exists in Go but not in the database is
// a route nobody can ever reach, and one that exists in the database but not in
// Go is a checkbox in the UI that grants nothing.
type Permission string

const (
	PermMenuRead    Permission = "menu.read"
	PermMenuWrite   Permission = "menu.write"
	PermMenuPublish Permission = "menu.publish"

	PermPriceRead  Permission = "price.read"
	PermPriceWrite Permission = "price.write"
	PermRulesRead  Permission = "rules.read"
	PermRulesWrite Permission = "rules.write"

	PermContentRead  Permission = "content.read"
	PermContentWrite Permission = "content.write"

	PermUserManage Permission = "user.manage"
	PermAuditRead  Permission = "audit.read"
)

// AllPermissions is every permission the code knows about, sorted, used to
// assert the Go constants and the seeded rows have not drifted apart.
func AllPermissions() []Permission {
	all := []Permission{
		PermMenuRead, PermMenuWrite, PermMenuPublish,
		PermPriceRead, PermPriceWrite, PermRulesRead, PermRulesWrite,
		PermContentRead, PermContentWrite,
		PermUserManage, PermAuditRead,
	}
	sort.Slice(all, func(i, j int) bool { return all[i] < all[j] })
	return all
}

// PermissionSet is the set of permissions an actor holds.
type PermissionSet map[Permission]bool

// NewPermissionSet builds a set from codes read out of the database.
func NewPermissionSet(codes []string) PermissionSet {
	s := make(PermissionSet, len(codes))
	for _, c := range codes {
		s[Permission(c)] = true
	}
	return s
}

// Has reports whether the actor holds a permission.
func (s PermissionSet) Has(p Permission) bool { return s[p] }

// Codes returns the held permissions as sorted strings, for the UI and for
// audit lines.
func (s PermissionSet) Codes() []string {
	out := make([]string, 0, len(s))
	for p, ok := range s {
		if ok {
			out = append(out, string(p))
		}
	}
	sort.Strings(out)
	return out
}

// AllPermissionSet is what the service token holds: everything. It is built
// from AllPermissions so a newly added permission is automatically included —
// forgetting to add one here would silently lock automation out of a route.
func AllPermissionSet() PermissionSet {
	s := PermissionSet{}
	for _, p := range AllPermissions() {
		s[p] = true
	}
	return s
}
