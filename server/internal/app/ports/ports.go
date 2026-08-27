// Package ports declares what the application layer needs from the outside
// world. Adapters implement these; the app depends on the interface, never on
// gorm or gin.
package ports

import (
	"context"

	"github.com/stevenwilliam/thenie_v2/server/internal/app/siteconfig"
)

// ConfigRepository reads the whole public configuration surface.
type ConfigRepository interface {
	// Revision returns the current content revision. It is a single-row read,
	// so the HTTP layer can answer a conditional request without assembling
	// the full document.
	Revision(ctx context.Context) (int64, error)

	// Load assembles the document from the database.
	Load(ctx context.Context) (*siteconfig.Document, error)
}

// MenuRepository is the write side of the menu rotation -- the part an admin
// actually edits week to week.
type MenuRepository interface {
	UpsertCycle(ctx context.Context, in UpsertCycleInput) (string, error)
	PublishCycle(ctx context.Context, isoYear, isoWeek int, published bool) error
	DeleteCycle(ctx context.Context, isoYear, isoWeek int) error
}

// UpsertCycleInput is one week of menus for any number of plans, written as a
// unit: either the whole week lands or none of it does.
type UpsertCycleInput struct {
	ISOYear  int
	ISOWeek  int
	StartsOn string // YYYY-MM-DD
	EndsOn   string
	Label    string
	Publish  bool
	// Days is plan slug -> that plan's days.
	Days map[string][]DayInput
}

// DayInput is one plan's menu for one date.
type DayInput struct {
	Date      string // YYYY-MM-DD
	IsMeatDay bool
	Kcal      int
	Items     []ComponentInput
}

// ComponentInput is one item on the plate.
type ComponentInput struct {
	Name  string
	Grams int
}

// ParamRepository is CRUD over sys_parameters.
type ParamRepository interface {
	List(ctx context.Context) ([]Param, error)
	Set(ctx context.Context, key, value string) error
}

// Param is one configurable value.
type Param struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Group       string `json:"group"`
	SortOrder   int    `json:"sort_order"`
}

// RateRepository is the write side of the price catalogue.
type RateRepository interface {
	SetPlanRates(ctx context.Context, planSlug string, r siteconfig.Rates) error
	SetTierPrices(ctx context.Context, productSlug, packageName string, bands []siteconfig.Band) error
	SetKantorRates(ctx context.Context, grade, period string, bands []siteconfig.Band) error
}
