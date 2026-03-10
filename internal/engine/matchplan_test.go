package engine

import (
	"context"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/runtime/bundlerelease"
	"github.com/opmodel/cli/internal/runtime/modulerelease"
	"github.com/opmodel/cli/pkg/provider"
)

func TestMatchedPairs_Sorted(t *testing.T) {
	plan := &MatchPlan{
		Matches: map[string]map[string]MatchResult{
			"zebra": {
				"tf/b": {Matched: true},
				"tf/a": {Matched: true},
			},
			"alpha": {
				"tf/z": {Matched: true},
				"tf/a": {Matched: false}, // not matched — excluded
			},
			"middle": {
				"tf/x": {Matched: true},
			},
		},
	}

	pairs := plan.MatchedPairs()

	// Expected sort order: alpha/tf/z, middle/tf/x, zebra/tf/a, zebra/tf/b
	expected := []MatchedPair{
		{ComponentName: "alpha", TransformerFQN: "tf/z"},
		{ComponentName: "middle", TransformerFQN: "tf/x"},
		{ComponentName: "zebra", TransformerFQN: "tf/a"},
		{ComponentName: "zebra", TransformerFQN: "tf/b"},
	}

	assert.Equal(t, expected, pairs)
}

func TestMatchedPairs_Empty(t *testing.T) {
	plan := &MatchPlan{
		Matches: map[string]map[string]MatchResult{
			"comp": {
				"tf/a": {Matched: false},
			},
		},
	}

	pairs := plan.MatchedPairs()
	assert.Empty(t, pairs)
}

func TestMatchedPairs_NilMatches(t *testing.T) {
	plan := &MatchPlan{}
	pairs := plan.MatchedPairs()
	assert.Empty(t, pairs)
}

func TestWarnings_Deterministic(t *testing.T) {
	// Populate map with multiple components and multiple traits per component.
	// Go map iteration is non-deterministic, so the test must verify that output
	// is always the same regardless of iteration order.
	plan := &MatchPlan{
		UnhandledTraits: map[string][]string{
			"comp-z": {"trait/b", "trait/a"}, // traits out of order
			"comp-a": {"trait/x"},
			"comp-m": {"trait/c", "trait/b", "trait/a"}, // traits out of order
		},
	}

	// Run multiple times to catch non-determinism.
	var first []string
	for i := 0; i < 20; i++ {
		got := plan.Warnings()
		if i == 0 {
			first = got
		} else {
			assert.Equal(t, first, got, "Warnings() output changed between run %d and run 0", i)
		}
	}

	// Verify the exact expected order: comp-a, comp-m, comp-z; traits sorted within each.
	expected := []string{
		`component "comp-a": trait "trait/x" is not handled by any matched transformer (values will be ignored)`,
		`component "comp-m": trait "trait/a" is not handled by any matched transformer (values will be ignored)`,
		`component "comp-m": trait "trait/b" is not handled by any matched transformer (values will be ignored)`,
		`component "comp-m": trait "trait/c" is not handled by any matched transformer (values will be ignored)`,
		`component "comp-z": trait "trait/a" is not handled by any matched transformer (values will be ignored)`,
		`component "comp-z": trait "trait/b" is not handled by any matched transformer (values will be ignored)`,
	}
	assert.Equal(t, expected, first)
}

func TestWarnings_Empty(t *testing.T) {
	plan := &MatchPlan{
		UnhandledTraits: map[string][]string{},
	}
	assert.Nil(t, plan.Warnings())
}

func TestWarnings_NilUnhandledTraits(t *testing.T) {
	plan := &MatchPlan{}
	assert.Nil(t, plan.Warnings())
}

func TestSortMatchedPairs(t *testing.T) {
	pairs := []MatchedPair{
		{ComponentName: "z", TransformerFQN: "b"},
		{ComponentName: "a", TransformerFQN: "z"},
		{ComponentName: "z", TransformerFQN: "a"},
		{ComponentName: "m", TransformerFQN: "m"},
	}

	sortMatchedPairs(pairs)

	expected := []MatchedPair{
		{ComponentName: "a", TransformerFQN: "z"},
		{ComponentName: "m", TransformerFQN: "m"},
		{ComponentName: "z", TransformerFQN: "a"},
		{ComponentName: "z", TransformerFQN: "b"},
	}
	assert.Equal(t, expected, pairs)
}

func TestModuleRenderer_RenderReturnsNonNilEmptySlices(t *testing.T) {
	ctx := cuecontext.New()
	providerVal := ctx.CompileString(`{#transformers:{}}`)
	raw := ctx.CompileString(`{components:{}}`)
	data := ctx.CompileString(`{}`)

	renderer := NewModuleRenderer(&provider.Provider{Data: providerVal})
	result, err := renderer.Render(context.Background(), &modulerelease.ModuleRelease{
		Metadata:       &modulerelease.ReleaseMetadata{Name: "demo"},
		RawCUE:         raw,
		DataComponents: data,
	}, &MatchPlan{Matches: map[string]map[string]MatchResult{}, UnhandledTraits: map[string][]string{}})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Resources)
	assert.NotNil(t, result.Components)
	assert.NotNil(t, result.Warnings)
	assert.Empty(t, result.Resources)
	assert.Empty(t, result.Components)
	assert.Empty(t, result.Warnings)
}

func TestBundleRenderer_RenderReturnsNonNilEmptySlices(t *testing.T) {
	ctx := cuecontext.New()
	providerVal := ctx.CompileString(`{#transformers:{}}`)
	renderer := NewBundleRenderer(&provider.Provider{Data: providerVal})
	result, err := renderer.Render(context.Background(), &bundlerelease.BundleRelease{Releases: map[string]*modulerelease.ModuleRelease{}})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Resources)
	assert.NotNil(t, result.Warnings)
	assert.NotNil(t, result.ReleaseOrder)
	assert.Empty(t, result.Resources)
	assert.Empty(t, result.Warnings)
	assert.Empty(t, result.ReleaseOrder)
}
