package render

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/pkg/core"
	"github.com/open-platform-model/cli/pkg/module"
	"github.com/open-platform-model/cli/pkg/provider"
)

func TestProcessModuleInstance_Success(t *testing.T) {
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
							namespace: #context.#moduleInstanceMetadata.namespace
							labels: "app.kubernetes.io/managed-by": #context.#runtimeName
						}
						spec: replicas: #component.spec.replicas
					}
				}
			}
		}
	}`)

	// Construct a fully prepared instance (as ParseModuleInstance would produce).
	rel := &module.Instance{
		Metadata: &module.InstanceMetadata{Name: "demo", Namespace: "apps"},
		Module:   module.Module{Metadata: &module.ModuleMetadata{FQN: "example.com/modules/demo@v1", Version: "v1"}},
		Spec:     raw,
		Values:   ctx.CompileString(`{replicas: 2}`),
	}

	result, err := ProcessModuleInstance(context.Background(), rel, &provider.Provider{Data: pv}, core.LabelManagedByValue)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Resources, 1)
	assert.True(t, result.MatchPlan.Matches["api"]["tf/deployment@v1"].Matched)

	labelsVal := result.Resources[0].Value.LookupPath(cue.ParsePath("metadata.labels"))
	var labels map[string]string
	require.NoError(t, labelsVal.Decode(&labels))
	assert.Equal(t, core.LabelManagedByValue, labels[core.LabelManagedBy])
}
