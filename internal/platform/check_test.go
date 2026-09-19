package platform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	libplatform "github.com/open-platform-model/library/opm/platform"
)

const (
	catOPM  = "opmodel.dev/catalogs/opm@v4"
	catK8up = "opmodel.dev/catalogs/k8up@v2"

	container = "opmodel.dev/catalogs/opm/resources/container@v1beta1"
	backup    = "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"

	deployment = "opmodel.dev/catalogs/opm/transformers/deployment@1.0.0"
	schedule   = "opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0"
	mirror     = "opmodel.dev/catalogs/velero/transformers/mirror@1.4.0"
)

func localRes() Resolution {
	return Resolution{Source: SourceLocalDefault, Location: "/home/u/.opm/platform", Dir: "/home/u/.opm/platform"}
}

// cleanInv is one contract, defined and implemented exactly once.
func cleanInv() *libplatform.ContractInventory {
	return &libplatform.ContractInventory{
		DefinedBy:     map[string]string{container: catOPM},
		RequiredBy:    map[string][]string{container: {deployment}},
		Fulfilled:     true,
		Routable:      true,
		Discriminated: true,
	}
}

func TestReportRender(t *testing.T) {
	tests := []struct {
		name    string
		inv     *libplatform.ContractInventory
		present []string
		absent  []string
	}{
		{
			name: "clean",
			inv:  cleanInv(),
			present: []string{
				"platform: /home/u/.opm/platform (local default)",
				"defined contracts: 1",
				"defined by      " + catOPM,
				"implemented by  " + deployment,
				"fulfilled: yes",
				"routable:  yes",
				"discriminated: yes",
			},
			absent: []string{"unfulfilled contracts", "over-subscribed contracts", "comparable transformer pairs", "vacuously"},
		},
		{
			name: "unfulfilled only",
			inv: &libplatform.ContractInventory{
				DefinedBy:     map[string]string{container: catOPM, backup: catOPM},
				RequiredBy:    map[string][]string{container: {deployment}, backup: {}},
				Unfulfilled:   []string{backup},
				Fulfilled:     false,
				Routable:      true,
				Discriminated: true,
			},
			present: []string{
				"defined contracts: 2",
				"implemented by  nothing on this platform",
				"unfulfilled contracts: 1",
				backup + " (defined by " + catOPM + ")",
				"not fail this check",
				"fulfilled: no — 1 contract is unfulfilled",
				"routable:  yes",
				"discriminated: yes",
			},
			absent: []string{"over-subscribed contracts", "comparable transformer pairs"},
		},
		{
			name: "over-subscribed only",
			inv: &libplatform.ContractInventory{
				DefinedBy:      map[string]string{backup: catK8up},
				RequiredBy:     map[string][]string{backup: {schedule, deployment}},
				OverSubscribed: []string{backup},
				Fulfilled:      true,
				Routable:       false,
				Discriminated:  true,
			},
			present: []string{
				"over-subscribed contracts: 1",
				backup + " (defined by " + catK8up + ")",
				// Both competing catalogs are named, through the
				// transformers that require the contract.
				"required by  " + schedule + ", " + deployment,
				"fulfilled: yes",
				"routable:  no — 1 contract is over-subscribed",
				"discriminated: yes",
			},
			absent: []string{"unfulfilled contracts", "comparable transformer pairs"},
		},
		{
			// A comparable pair is a separate refusal from
			// over-subscription (0015:D5): this platform's
			// contracts each have one supplier, so it is routable, and
			// the two transformers are still never told apart.
			name: "comparable only",
			inv: &libplatform.ContractInventory{
				DefinedBy:  map[string]string{container: catOPM},
				RequiredBy: map[string][]string{container: {mirror, schedule}},
				Comparable: []libplatform.ComparablePredicates{
					{Broader: mirror, Narrower: schedule, Contracts: []string{container}},
				},
				Fulfilled:     true,
				Routable:      true,
				Discriminated: false,
			},
			present: []string{
				"comparable transformer pairs: 1",
				"so both would render",
				mirror + " (broader)",
				"and  " + schedule + " (narrower)",
				"over  " + container,
				"fulfilled: yes",
				"routable:  yes",
				"discriminated: no — 1 pair is comparable",
			},
			absent: []string{"unfulfilled contracts", "over-subscribed contracts"},
		},
		{
			// Both refusals at once, each under its own heading, each
			// with its own verdict line.
			name: "over-subscribed and comparable",
			inv: &libplatform.ContractInventory{
				DefinedBy:      map[string]string{container: catOPM, backup: catK8up},
				RequiredBy:     map[string][]string{container: {mirror, schedule}, backup: {schedule, deployment}},
				OverSubscribed: []string{backup},
				Comparable: []libplatform.ComparablePredicates{
					{Broader: mirror, Narrower: schedule, Contracts: []string{backup, container}},
					{Broader: deployment, Narrower: schedule, Contracts: []string{container}},
				},
				Fulfilled:     true,
				Routable:      false,
				Discriminated: false,
			},
			present: []string{
				"over-subscribed contracts: 1",
				"comparable transformer pairs: 2",
				// Shared contracts sort within a row: the inventory
				// listed backup first, and `resources/` sorts before
				// `traits/`.
				"over  " + container + ", " + backup,
				"routable:  no — 1 contract is over-subscribed",
				"discriminated: no — 2 pairs are comparable",
			},
			absent: []string{"unfulfilled contracts"},
		},
		{
			name: "both",
			inv: &libplatform.ContractInventory{
				DefinedBy:      map[string]string{container: catOPM, backup: catK8up},
				RequiredBy:     map[string][]string{container: {schedule, deployment}, backup: {}},
				Unfulfilled:    []string{backup},
				OverSubscribed: []string{container},
				Fulfilled:      false,
				Routable:       false,
				Discriminated:  true,
			},
			present: []string{
				"unfulfilled contracts: 1",
				"over-subscribed contracts: 1",
				"fulfilled: no — 1 contract is unfulfilled",
				"routable:  no — 1 contract is over-subscribed",
			},
		},
		{
			name: "empty inventory",
			inv: &libplatform.ContractInventory{
				DefinedBy:     map[string]string{},
				RequiredBy:    map[string][]string{},
				Fulfilled:     true,
				Routable:      true,
				Discriminated: true,
			},
			present: []string{
				"the enabled catalogs define no contracts",
				"nothing was verified here",
				"fulfilled: yes (vacuously — no contract is defined)",
				"routable:  yes (vacuously — no contract is defined)",
				"discriminated: yes (vacuously — no contract is defined)",
			},
			// An empty inventory must not read as a verified platform.
			absent: []string{"defined contracts:", "unfulfilled contracts", "over-subscribed contracts", "comparable transformer pairs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewReport(localRes(), tt.inv).Render()
			for _, want := range tt.present {
				assert.Contains(t, got, want)
			}
			for _, unwanted := range tt.absent {
				assert.NotContains(t, got, unwanted)
			}
			assert.True(t, strings.HasPrefix(got, "platform: "), "the provenance line comes first")
			assert.False(t, strings.HasSuffix(got, "\n"), "render is not newline-terminated")
		})
	}
}

// TestReportRenderIsDeterministic guards the map iteration order: the report
// is read by people diffing two runs.
func TestReportRenderIsDeterministic(t *testing.T) {
	inv := &libplatform.ContractInventory{
		DefinedBy:     map[string]string{container: catOPM, backup: catK8up},
		RequiredBy:    map[string][]string{container: {schedule, deployment}, backup: {}},
		Fulfilled:     true,
		Routable:      true,
		Discriminated: true,
	}
	first := NewReport(localRes(), inv).Render()
	for range 20 {
		require.Equal(t, first, NewReport(localRes(), inv).Render())
	}
}

// TestComparableRowsRenderInAStableOrder pins that the comparable section does
// not inherit the build's comprehension order: the inventory hands the rows
// over unsorted, so two platforms differing only in that order must render
// byte-identically.
func TestComparableRowsRenderInAStableOrder(t *testing.T) {
	rows := func(rs ...libplatform.ComparablePredicates) *libplatform.ContractInventory {
		return &libplatform.ContractInventory{
			DefinedBy:     map[string]string{container: catOPM},
			RequiredBy:    map[string][]string{container: {deployment, mirror, schedule}},
			Comparable:    rs,
			Fulfilled:     true,
			Routable:      true,
			Discriminated: false,
		}
	}
	a := libplatform.ComparablePredicates{Broader: mirror, Narrower: schedule, Contracts: []string{container, backup}}
	b := libplatform.ComparablePredicates{Broader: deployment, Narrower: schedule, Contracts: []string{backup, container}}

	require.Equal(t,
		NewReport(localRes(), rows(a, b)).Render(),
		NewReport(localRes(), rows(b, a)).Render(),
		"rows sort by broader then narrower, and shared contracts sort within a row")
}

// TestRenderDoesNotReorderTheInventory pins that sorting for the report never
// reaches back into the value the library handed over: the inventory is the
// caller's, and a second reader must see the order the build produced.
func TestRenderDoesNotReorderTheInventory(t *testing.T) {
	inv := &libplatform.ContractInventory{
		DefinedBy:  map[string]string{container: catOPM},
		RequiredBy: map[string][]string{container: {mirror, schedule}},
		Comparable: []libplatform.ComparablePredicates{
			{Broader: mirror, Narrower: schedule, Contracts: []string{container, backup}},
			{Broader: deployment, Narrower: schedule, Contracts: []string{container}},
		},
		Fulfilled:     true,
		Routable:      true,
		Discriminated: false,
	}
	_ = NewReport(localRes(), inv).Render()

	assert.Equal(t, mirror, inv.Comparable[0].Broader, "the row order the build produced is untouched")
	assert.Equal(t, []string{container, backup}, inv.Comparable[0].Contracts, "a row's contract order is untouched")
}

// TestRoutableIsAGateAndFulfilledIsNot pins the asymmetry 0015:D18
// requires: `unfulfilled` is a report and `overSubscribed` is the gate, so an
// unfulfilled-only platform is routable and an over-subscribed one is not.
// Do not "fix" this into a single healthy/unhealthy verdict: a pre-flight that
// failed on unfulfilled would refuse platforms the operator generates from
// happily, and one that passed on over-subscription would bless a platform
// platform-package generation refuses.
func TestRoutableIsAGateAndFulfilledIsNot(t *testing.T) {
	unfulfilled := NewReport(localRes(), &libplatform.ContractInventory{
		DefinedBy:     map[string]string{backup: catOPM},
		RequiredBy:    map[string][]string{backup: {}},
		Unfulfilled:   []string{backup},
		Fulfilled:     false,
		Routable:      true,
		Discriminated: true,
	})
	assert.True(t, unfulfilled.Routable(), "an unfulfilled contract is reported, never a gate (0015:D18)")

	overSubscribed := NewReport(localRes(), &libplatform.ContractInventory{
		DefinedBy:      map[string]string{backup: catK8up},
		RequiredBy:     map[string][]string{backup: {schedule, deployment}},
		OverSubscribed: []string{backup},
		Fulfilled:      true,
		Routable:       false,
		Discriminated:  true,
	})
	assert.False(t, overSubscribed.Routable(), "over-subscription is what platform-package generation refuses on")

	assert.True(t, NewReport(localRes(), cleanInv()).Routable())
}

// TestDiscriminatedIsTheOtherGate pins that the report carries two independent
// gates, not one. Enhancement 0015:D5 makes a comparable pair a condition
// platform-package generation refuses on, exactly as 0010:D37 does over-subscription
// — and 0015:D18 keeps `fulfilled` out of both. The three are orthogonal: a platform
// can fail either gate while passing the other, so the command reads both
// accessors rather than a single combined verdict that would hide which fired.
func TestDiscriminatedIsTheOtherGate(t *testing.T) {
	undiscriminated := NewReport(localRes(), &libplatform.ContractInventory{
		DefinedBy:  map[string]string{container: catOPM},
		RequiredBy: map[string][]string{container: {mirror, schedule}},
		Comparable: []libplatform.ComparablePredicates{
			{Broader: mirror, Narrower: schedule, Contracts: []string{container}},
		},
		Fulfilled:     true,
		Routable:      true,
		Discriminated: false,
	})
	assert.True(t, undiscriminated.Routable(), "a comparable pair is not over-subscription")
	assert.False(t, undiscriminated.Discriminated(), "0015:D5: generation refuses a comparable pair")

	overSubscribed := NewReport(localRes(), &libplatform.ContractInventory{
		DefinedBy:      map[string]string{backup: catK8up},
		RequiredBy:     map[string][]string{backup: {schedule, deployment}},
		OverSubscribed: []string{backup},
		Fulfilled:      true,
		Routable:       false,
		Discriminated:  true,
	})
	assert.False(t, overSubscribed.Routable())
	assert.True(t, overSubscribed.Discriminated(), "over-subscription says nothing about predicates")

	// `fulfilled` decides neither gate (0015:D18).
	unfulfilled := NewReport(localRes(), &libplatform.ContractInventory{
		DefinedBy:     map[string]string{backup: catOPM},
		RequiredBy:    map[string][]string{backup: {}},
		Unfulfilled:   []string{backup},
		Fulfilled:     false,
		Routable:      true,
		Discriminated: true,
	})
	assert.True(t, unfulfilled.Routable())
	assert.True(t, unfulfilled.Discriminated())

	clean := NewReport(localRes(), cleanInv())
	assert.True(t, clean.Routable())
	assert.True(t, clean.Discriminated())
}
