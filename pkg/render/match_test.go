package render

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/pkg/provider"
)

func TestMatch_BuildsPlan(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{
		api: {
			metadata: {
				labels: {
					"core.opmodel.dev/workload-type": "stateless"
				}
			}
			#resources: {
				"res/container@v1": {}
			}
			#traits: {
				"trait/expose@v1": {}
			}
		}
	}`)
	require.NoError(t, components.Err())

	providerVal := ctx.CompileString(`{
		#transformers: {
			"tf/deployment@v1": {
				requiredLabels: {
					"core.opmodel.dev/workload-type": "stateless"
				}
				requiredResources: {
					"res/container@v1": {}
				}
				requiredTraits: {}
				optionalTraits: {
					"trait/expose@v1": {}
				}
			}
			"tf/service@v1": {
				requiredResources: {
					"res/container@v1": {}
				}
				requiredTraits: {
					"trait/expose@v1": {}
				}
				optionalTraits: {}
			}
			"tf/stateful@v1": {
				requiredLabels: {
					"core.opmodel.dev/workload-type": "stateful"
				}
				requiredResources: {
					"res/container@v1": {}
				}
				requiredTraits: {}
				optionalTraits: {}
			}
		}
	}`)
	require.NoError(t, providerVal.Err())

	plan, err := Match(components, &provider.Provider{Data: providerVal})
	require.NoError(t, err)
	assert.Empty(t, plan.Unmatched)
	assert.Empty(t, plan.UnhandledTraits["api"])
	assert.True(t, plan.Matches["api"]["tf/deployment@v1"].Matched)
	assert.True(t, plan.Matches["api"]["tf/service@v1"].Matched)
	assert.False(t, plan.Matches["api"]["tf/stateful@v1"].Matched)
	assert.Equal(t, []string{"core.opmodel.dev/workload-type=stateful"}, plan.Matches["api"]["tf/stateful@v1"].MissingLabels)
}

func TestMatch_UnmatchedAndWarningsDeterministic(t *testing.T) {
	ctx := cuecontext.New()
	components := ctx.CompileString(`{
		api: {
			#resources: {}
			#traits: {
				"trait/expose@v1": {}
				"trait/ingress@v1": {}
			}
		}
	}`)
	providerVal := ctx.CompileString(`{
		#transformers: {
			"tf/service@v1": {
				requiredResources: {}
				requiredTraits: {
					"trait/expose@v1": {}
				}
				optionalTraits: {}
			}
		}
	}`)

	plan, err := Match(components, &provider.Provider{Data: providerVal})
	require.NoError(t, err)
	assert.Empty(t, plan.Unmatched)
	assert.Equal(t, []string{"trait/ingress@v1"}, plan.UnhandledTraits["api"])
	assert.Equal(t, []string{`component "api": trait "trait/ingress@v1" is not handled by any matched transformer (values will be ignored)`}, plan.Warnings())
}

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
