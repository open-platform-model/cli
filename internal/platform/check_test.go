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
	catVel  = "opmodel.dev/catalogs/velero@v1"

	container = "opmodel.dev/catalogs/opm/resources/container@v1beta1"
	backup    = "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"

	catBaseV1 = "testing.opmodel.dev/catalogs/base@v1"
	catBaseV2 = "testing.opmodel.dev/catalogs/base@v2"

	collisionNote = "(a colliding contract is left out of the defined, required, unfulfilled and comparable sections; keep one of its defining entries enabled)"

	deployment = "opmodel.dev/catalogs/opm/transformers/deployment@1.0.0"
	schedule   = "opmodel.dev/catalogs/k8up/transformers/schedule@2.0.0"
	mirror     = "opmodel.dev/catalogs/velero/transformers/mirror@1.4.0"
)

func flagRes() Resolution {
	return Resolution{Source: SourceFlagDir, Location: "/home/u/platforms/staging", Dir: "/home/u/platforms/staging"}
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
		// ordered, when set, are substrings that must appear in this order.
		ordered []string
	}{
		{
			name: "clean",
			inv:  cleanInv(),
			present: []string{
				"platform: /home/u/platforms/staging (--platform)",
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
				ProvidedBy:     map[string][]string{backup: {catVel, catK8up}},
				OverSubscribed: []string{backup},
				Fulfilled:      true,
				Routable:       false,
				Discriminated:  true,
			},
			present: []string{
				"over-subscribed contracts: 1",
				backup + " (defined by " + catK8up + ")",
				"enabled registry entry. Two majors of one catalog are two entries.",
				// Both competing registry entries are named, sorted.
				"provided by  " + catK8up + ", " + catVel,
				"fulfilled: yes",
				"routable:  no — 1 contract is over-subscribed",
				"discriminated: yes",
			},
			absent: []string{"unfulfilled contracts", "comparable transformer pairs", "    required by  ", "defined contracts: 0"},
		},
		{
			// The defining catalog is disabled or absent: nothing is
			// defined, and two registry entries still provide the
			// contract. The inventory's verdict is read, never worded
			// as vacuous.
			name: "definer-less over-subscription",
			inv: &libplatform.ContractInventory{
				DefinedBy:      map[string]string{},
				RequiredBy:     map[string][]string{},
				ProvidedBy:     map[string][]string{backup: {catK8up, catVel}},
				OverSubscribed: []string{backup},
				Fulfilled:      true,
				Routable:       false,
				Discriminated:  true,
			},
			present: []string{
				"defined contracts: 0",
				"The enabled catalogs define no contracts. The provider counts below come",
				"over-subscribed contracts: 1",
				backup + " (defined by no enabled catalog)",
				"provided by  " + catK8up + ", " + catVel,
				"fulfilled: yes",
				"routable:  no — 1 contract is over-subscribed",
				"discriminated: yes",
			},
			absent: []string{"vacuously", "nothing was verified here", "unfulfilled contracts"},
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
				ProvidedBy:     map[string][]string{backup: {catK8up, catVel}},
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
				"provided by  " + catK8up + ", " + catVel,
				// Shared contracts sort within a row: the inventory
				// listed backup first, and `resources/` sorts before
				// `traits/`.
				"over  " + container + ", " + backup,
				"routable:  no — 1 contract is over-subscribed",
				"discriminated: no — 2 pairs are comparable",
			},
			absent: []string{"unfulfilled contracts", "    required by  "},
		},
		{
			name: "both",
			inv: &libplatform.ContractInventory{
				DefinedBy:      map[string]string{container: catOPM, backup: catK8up},
				RequiredBy:     map[string][]string{container: {schedule, deployment}, backup: {}},
				ProvidedBy:     map[string][]string{container: {catK8up, catOPM}},
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
			// Two majors of one catalog list the same keys: nothing is
			// defined or provided, and the platform is still not a
			// vacuous one. fulfilled and discriminated read yes (core
			// computes them without the colliding keys); routable does not.
			name: "collision only",
			inv: &libplatform.ContractInventory{
				DefinedBy:  map[string]string{},
				RequiredBy: map[string][]string{},
				ProvidedBy: map[string][]string{},
				Collisions: []string{container, backup},
				CollidingEntries: map[string][]string{
					container: {catBaseV1, catBaseV2},
					backup:    {catBaseV1, catBaseV2},
				},
				Fulfilled:     true,
				Routable:      false,
				Discriminated: true,
			},
			present: []string{
				"defined contracts: 0",
				"colliding contracts: 2",
				"  " + container + "\n    defined by  " + catBaseV1 + ", " + catBaseV2,
				"  " + backup + "\n    defined by  " + catBaseV1 + ", " + catBaseV2,
				collisionNote,
				"fulfilled: yes",
				"routable:  no — 2 contracts collide, 0 over-subscribed",
				"discriminated: yes",
			},
			absent: []string{"vacuously", "define no contracts", "0 contracts are over-subscribed", "over-subscribed contracts:"},
		},
		{
			// A collision on one key and an over-subscription on another,
			// each under its own heading, the colliding one first.
			name: "collision and over-subscription",
			inv: &libplatform.ContractInventory{
				DefinedBy:        map[string]string{backup: catBaseV1},
				RequiredBy:       map[string][]string{backup: {schedule, mirror}},
				ProvidedBy:       map[string][]string{backup: {catVel, catK8up}},
				OverSubscribed:   []string{backup},
				Collisions:       []string{container},
				CollidingEntries: map[string][]string{container: {catBaseV1, catBaseV2}},
				Fulfilled:        true,
				Routable:         false,
				Discriminated:    true,
			},
			present: []string{
				"colliding contracts: 1",
				"over-subscribed contracts: 1",
				"routable:  no — 1 contract collides, 1 over-subscribed",
			},
			absent: []string{"1 contract is over-subscribed", "vacuously"},
			ordered: []string{
				"colliding contracts: 1",
				container,
				collisionNote,
				"over-subscribed contracts: 1",
				"provided by  " + catK8up + ", " + catVel,
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
			got := NewReport(flagRes(), tt.inv).Render()
			for _, want := range tt.present {
				assert.Contains(t, got, want)
			}
			for _, unwanted := range tt.absent {
				assert.NotContains(t, got, unwanted)
			}
			rest := got
			for _, want := range tt.ordered {
				i := strings.Index(rest, want)
				if !assert.NotEqual(t, -1, i, "%q out of order in:\n%s", want, got) {
					break
				}
				rest = rest[i+len(want):]
			}
			assert.True(t, strings.HasPrefix(got, "platform: "), "the provenance line comes first")
			assert.False(t, strings.HasSuffix(got, "\n"), "render is not newline-terminated")
		})
	}
}

// TestEmptyInventoryRenderIsUnchanged pins the vacuous report byte for byte:
// it is reserved for a platform where nothing is defined AND nothing is
// provided, and its wording did not move when the provider count did.
func TestEmptyInventoryRenderIsUnchanged(t *testing.T) {
	got := NewReport(flagRes(), &libplatform.ContractInventory{
		DefinedBy:     map[string]string{},
		RequiredBy:    map[string][]string{},
		ProvidedBy:    map[string][]string{},
		Fulfilled:     true,
		Routable:      true,
		Discriminated: true,
	}).Render()

	// The library decodes an absent collision report as empty; an empty
	// one reads exactly as none.
	withEmptyCollisions := NewReport(flagRes(), &libplatform.ContractInventory{
		DefinedBy:        map[string]string{},
		RequiredBy:       map[string][]string{},
		ProvidedBy:       map[string][]string{},
		Collisions:       []string{},
		CollidingEntries: map[string][]string{},
		Fulfilled:        true,
		Routable:         true,
		Discriminated:    true,
	}).Render()
	assert.Equal(t, got, withEmptyCollisions)

	want := "platform: /home/u/platforms/staging (--platform)\n" +
		"\nthe enabled catalogs define no contracts — nothing was verified here.\n" +
		"A platform whose catalogs populate no contract maps produces an empty\n" +
		"inventory, which is not the same answer as a platform whose contracts all\n" +
		"check out.\n" +
		"\nfulfilled: yes (vacuously — no contract is defined)\n" +
		"routable:  yes (vacuously — no contract is defined)\n" +
		"discriminated: yes (vacuously — no contract is defined)"
	assert.Equal(t, want, got)
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
	first := NewReport(flagRes(), inv).Render()
	for range 20 {
		require.Equal(t, first, NewReport(flagRes(), inv).Render())
	}

	// The providing registry keys print sorted whatever order the
	// inventory hands them over in.
	provided := func(keys ...string) *libplatform.ContractInventory {
		return &libplatform.ContractInventory{
			DefinedBy:      map[string]string{backup: catOPM},
			RequiredBy:     map[string][]string{backup: {schedule, mirror}},
			ProvidedBy:     map[string][]string{backup: keys},
			OverSubscribed: []string{backup},
			Fulfilled:      true,
			Routable:       false,
			Discriminated:  true,
		}
	}
	require.Equal(t,
		NewReport(flagRes(), provided(catK8up, catVel)).Render(),
		NewReport(flagRes(), provided(catVel, catK8up)).Render())

	// The colliding keys, and the entries defining each, print sorted
	// whatever order the inventory hands them over in.
	colliding := func(keys, entries []string) *libplatform.ContractInventory {
		return &libplatform.ContractInventory{
			Collisions: keys,
			CollidingEntries: map[string][]string{
				container: entries,
				backup:    {catBaseV1, catBaseV2},
			},
			Fulfilled:     true,
			Routable:      false,
			Discriminated: true,
		}
	}
	require.Equal(t,
		NewReport(flagRes(), colliding([]string{container, backup}, []string{catBaseV1, catBaseV2})).Render(),
		NewReport(flagRes(), colliding([]string{backup, container}, []string{catBaseV2, catBaseV1})).Render())
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
		NewReport(flagRes(), rows(a, b)).Render(),
		NewReport(flagRes(), rows(b, a)).Render(),
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
	_ = NewReport(flagRes(), inv).Render()

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
	unfulfilled := NewReport(flagRes(), &libplatform.ContractInventory{
		DefinedBy:     map[string]string{backup: catOPM},
		RequiredBy:    map[string][]string{backup: {}},
		Unfulfilled:   []string{backup},
		Fulfilled:     false,
		Routable:      true,
		Discriminated: true,
	})
	assert.True(t, unfulfilled.Routable(), "an unfulfilled contract is reported, never a gate (0015:D18)")

	overSubscribed := NewReport(flagRes(), &libplatform.ContractInventory{
		DefinedBy:      map[string]string{backup: catK8up},
		RequiredBy:     map[string][]string{backup: {schedule, deployment}},
		OverSubscribed: []string{backup},
		Fulfilled:      true,
		Routable:       false,
		Discriminated:  true,
	})
	assert.False(t, overSubscribed.Routable(), "over-subscription is what platform-package generation refuses on")

	// A colliding key leaves the defined contracts, so the fulfilled
	// verdict can read yes while the platform is not routable: routable is
	// the gate the collision fails.
	colliding := NewReport(flagRes(), &libplatform.ContractInventory{
		Collisions:       []string{container},
		CollidingEntries: map[string][]string{container: {catBaseV1, catBaseV2}},
		Fulfilled:        true,
		Routable:         false,
		Discriminated:    true,
	})
	assert.False(t, colliding.Routable(), "a colliding contract key fails the routable gate")
	assert.Contains(t, colliding.Render(), "fulfilled: yes", "the fulfilled verdict is printed as the inventory carries it")

	assert.True(t, NewReport(flagRes(), cleanInv()).Routable())
}

// TestDiscriminatedIsTheOtherGate pins that the report carries two independent
// gates, not one. Enhancement 0015:D5 makes a comparable pair a condition
// platform-package generation refuses on, exactly as 0010:D37 does over-subscription
// — and 0015:D18 keeps `fulfilled` out of both. The three are orthogonal: a platform
// can fail either gate while passing the other, so the command reads both
// accessors rather than a single combined verdict that would hide which fired.
func TestDiscriminatedIsTheOtherGate(t *testing.T) {
	undiscriminated := NewReport(flagRes(), &libplatform.ContractInventory{
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

	overSubscribed := NewReport(flagRes(), &libplatform.ContractInventory{
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
	unfulfilled := NewReport(flagRes(), &libplatform.ContractInventory{
		DefinedBy:     map[string]string{backup: catOPM},
		RequiredBy:    map[string][]string{backup: {}},
		Unfulfilled:   []string{backup},
		Fulfilled:     false,
		Routable:      true,
		Discriminated: true,
	})
	assert.True(t, unfulfilled.Routable())
	assert.True(t, unfulfilled.Discriminated())

	clean := NewReport(flagRes(), cleanInv())
	assert.True(t, clean.Routable())
	assert.True(t, clean.Discriminated())
}
