package postgres

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/stevenwilliam/thenie_v2/server/internal/platform/apierror"
)

// translate() has to reach through gorm's wrapping to the driver's error type.
// This test exists because getting that wrong is silent: the first version
// asserted *pq.Error, which never matches because gorm.io/driver/postgres is
// built on pgx/v5, so every constraint violation fell through as a 500 with a
// perfectly good message sitting unread in the log. Nothing failed; the API
// just quietly stopped explaining itself.
//
// It pins two things a dependency bump could change under us: that the concrete
// error is still reachable as *pgconn.PgError, and that translate() still maps
// it rather than passing it through.
func TestTranslateReachesTheDriverError(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Skipf("cannot reach test database: %v", err)
	}
	e := db.Exec(`INSERT INTO sys_parameters (key, value, value_type, label)
	              VALUES ('probe.dup', 'x', 'string', 'probe')`).Error
	if e != nil {
		t.Fatalf("first insert should succeed: %v", e)
	}
	defer func() { db.Exec(`DELETE FROM sys_parameters WHERE key = 'probe.dup'`) }()

	e = db.Exec(`INSERT INTO sys_parameters (key, value, value_type, label)
	             VALUES ('probe.dup', 'x', 'string', 'probe')`).Error
	if e == nil {
		t.Fatal("second insert must violate the primary key")
	}
	// gorm wraps, so the concrete type is *fmt.wrapErrors -- reaching the
	// driver error requires errors.As, never a type assertion.
	t.Logf("concrete type: %T", e)

	var pg *pgconn.PgError
	if !errors.As(e, &pg) {
		t.Fatalf("expected *pgconn.PgError under the wrapper, got %T (%v)", e, e)
	}
	if pg.Code != "23505" {
		t.Errorf("expected SQLSTATE 23505 unique_violation, got %s", pg.Code)
	}

	mapped := translate(e)
	if mapped == e {
		t.Fatal("translate() passed the driver error straight through")
	}
	var ae *apierror.Error
	if !errors.As(mapped, &ae) {
		t.Fatalf("translate() must produce an *apierror.Error, got %T", mapped)
	}
	if ae.Code != apierror.CodeConflict {
		t.Errorf("got code %s, want %s", ae.Code, apierror.CodeConflict)
	}
	if ae.Status != 409 {
		t.Errorf("got status %d, want 409", ae.Status)
	}
	// The driver message must not reach the client.
	if strings.Contains(ae.Message, "SQLSTATE") || strings.Contains(ae.Message, "sys_parameters_pkey") {
		t.Errorf("driver detail leaked into the client message: %q", ae.Message)
	}
}

// The exclusion constraint is the one this service leans on hardest: it is how
// the schema refuses two published cycles claiming the same delivery date.
func TestTranslateMapsCycleOverlap(t *testing.T) {
	err := translate(fmt.Errorf("wrapped: %w", &pgconn.PgError{
		Code:           "23P01",
		ConstraintName: "menu_cycles_no_overlap",
	}))
	var ae *apierror.Error
	if !errors.As(err, &ae) {
		t.Fatalf("expected *apierror.Error, got %T", err)
	}
	if ae.Code != apierror.CodeCycleOverlap {
		t.Errorf("got %s, want %s", ae.Code, apierror.CodeCycleOverlap)
	}
	if ae.Status != 409 {
		t.Errorf("got status %d, want 409", ae.Status)
	}
}

func TestTranslateLeavesUnknownErrorsAlone(t *testing.T) {
	plain := errors.New("something else entirely")
	if got := translate(plain); got != plain {
		t.Errorf("an unrecognised error must pass through unchanged, got %v", got)
	}
}
