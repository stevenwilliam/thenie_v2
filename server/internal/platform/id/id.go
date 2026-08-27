// Package id generates identifiers.
//
// Primary keys are UUIDv7 — time-ordered, so a b-tree insert lands at the right
// edge instead of scattering, without being a sequence that leaks row counts.
//
// This service only ever mints primary keys. The Crockford order-code and
// transfer-code generators in healthy_catering's copy are deliberately not
// carried over: nothing here takes an order.
package id

import (
	"fmt"

	"github.com/google/uuid"
)

// New returns a UUIDv7.
func New() uuid.UUID {
	v, err := uuid.NewV7()
	if err != nil {
		// NewV7 only fails if the system CSPRNG fails, in which case nothing
		// this service does is safe to continue.
		panic(fmt.Sprintf("uuidv7: %v", err))
	}
	return v
}

// NewString returns a UUIDv7 as a string.
func NewString() string { return New().String() }
