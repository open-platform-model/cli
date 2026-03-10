package match

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
