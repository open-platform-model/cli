package render

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/pkg/bundle"
	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/provider"
)

func TestProcessModuleRelease_Success(t *testing.T) {
	ctx := cuecontext.New()
	raw := ctx.CompileString(`{
		metadata: {
			name: "demo"
			namespace: "apps"
		}
		#module: {
			metadata: {
				name: "demo"
				modulePath: "example.com/modules"
				version: "v1"
				fqn: "example.com/modules/demo@v1"
			}
			#config: {
				replicas: int
			}
		}
		values: {
			replicas: 2
		}
		components: {
			api: {
				metadata: labels: {
					"core.opmodel.dev/workload-type": "stateless"
				}
				#resources: {
					"res/container@v1": {}
				}
				spec: {
					replicas: values.replicas
				}
			}
		}
	}`)
	pv := ctx.CompileString(`{
		#transformers: {
			"tf/deployment@v1": {
				requiredLabels: {
					"core.opmodel.dev/workload-type": "stateless"
				}
				requiredResources: {
					"res/container@v1": {}
				}
				requiredTraits: {}
				optionalTraits: {}
				#transform: {
					#component: _
					#context: _
					output: {
						apiVersion: "apps/v1"
						kind: "Deployment"
						metadata: {
							name: #context.#componentMetadata.name
							namespace: #context.#moduleReleaseMetadata.namespace
						}
						spec: replicas: #component.spec.replicas
					}
				}
			}
		}
	}`)

	// Construct a fully prepared release (as ParseModuleRelease would produce).
	rel := &module.Release{
		Metadata: &module.ReleaseMetadata{Name: "demo", Namespace: "apps"},
		Module:   module.Module{Metadata: &module.ModuleMetadata{FQN: "example.com/modules/demo@v1", Version: "v1"}},
		Spec:     raw,
		Values:   ctx.CompileString(`{replicas: 2}`),
	}

	result, err := ProcessModuleRelease(context.Background(), rel, &provider.Provider{Data: pv})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Resources, 1)
	assert.True(t, result.MatchPlan.Matches["api"]["tf/deployment@v1"].Matched)
}

func TestProcessBundleRelease_StubAfterValidation(t *testing.T) {
	ctx := cuecontext.New()
	raw := ctx.CompileString(`{
		metadata: name: "stack"
		#bundle: {
			#config: {
				replicas: int
			}
		}
	}`)
	br := &bundle.Release{
		Metadata: &bundle.ReleaseMetadata{Name: "stack"},
		Spec:     raw,
		Config:   raw.LookupPath(cue.ParsePath("#bundle.#config")),
		Releases: map[string]*module.Release{},
	}

	_, err := ProcessBundleRelease(context.Background(), br, []cue.Value{ctx.CompileString(`{replicas: 1}`)}, &provider.Provider{Data: ctx.CompileString(`{}`)})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.True(t, br.Values.Exists())
}
