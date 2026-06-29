package render

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/pkg/core"
	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/provider"
)

// TestRender_RuntimeName_StampsManagedByLabel verifies that the CLI render
// pipeline fills #TransformerContext.#runtimeName with the runtimeName argument
// passed to ProcessModuleInstance, and that the value reaches rendered resources
// via #context.#runtimeName.
//
// The test uses a minimal inline provider that reads #context.#runtimeName and
// writes it to metadata.labels[app.kubernetes.io/managed-by] — the same wiring
// the real catalog's #TransformerContext.controllerLabels performs. It also
// confirms instance-level labels (module-instance.opmodel.dev/uuid) continue to
// flow through #moduleInstanceMetadata.labels onto the output resource.
func TestRender_RuntimeName_StampsManagedByLabel(t *testing.T) {
	cueCtx := cuecontext.New()
	instanceUUID := "a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab"

	raw := cueCtx.CompileString(`{
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
			#config: {}
		}
		values: {}
		components: {
			cfg: {
				#resources: { "res/cfg@v1": {} }
				spec: { data: { foo: "bar" } }
			}
		}
	}`)

	pv := cueCtx.CompileString(`{
		#transformers: {
			"tf/configmap@v1": {
				requiredLabels: {}
				requiredResources: { "res/cfg@v1": {} }
				requiredTraits: {}
				optionalTraits: {}
				#transform: {
					#component: _
					#context: _
					output: {
						apiVersion: "v1"
						kind: "ConfigMap"
						metadata: {
							name: #context.#componentMetadata.name
							namespace: #context.#moduleInstanceMetadata.namespace
							labels: {
								"app.kubernetes.io/managed-by": #context.#runtimeName
								if #context.#moduleInstanceMetadata.labels != _|_ {
									for k, v in #context.#moduleInstanceMetadata.labels {
										(k): "\(v)"
									}
								}
							}
						}
						data: #component.spec.data
					}
				}
			}
		}
	}`)

	rel := &module.Instance{
		Metadata: &module.InstanceMetadata{
			Name:      "demo",
			Namespace: "apps",
			UUID:      instanceUUID,
			Labels: map[string]string{
				core.LabelModuleInstanceUUID: instanceUUID,
			},
		},
		Module: module.Module{Metadata: &module.ModuleMetadata{FQN: "example.com/modules/demo@v1", Version: "v1"}},
		Spec:   raw,
		Values: cueCtx.CompileString(`{}`),
	}

	result, err := ProcessModuleInstance(context.Background(), rel, &provider.Provider{Data: pv}, core.LabelManagedByValue)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.Resources, 1)

	labelsVal := result.Resources[0].Value.LookupPath(cue.ParsePath("metadata.labels"))
	require.True(t, labelsVal.Exists(), "metadata.labels missing from rendered resource")

	var labels map[string]string
	require.NoError(t, labelsVal.Decode(&labels))

	assert.Equal(t, core.LabelManagedByValue, labels[core.LabelManagedBy],
		"managed-by must equal the runtimeName the CLI supplies")
	assert.NotEmpty(t, labels[core.LabelModuleInstanceUUID],
		"module-instance.opmodel.dev/uuid must flow through from instance metadata labels")
}

// TestRender_RuntimeName_EmptyFailsAtBoundary verifies that ProcessModuleInstance
// rejects an empty runtimeName at the public API boundary, before any CUE
// evaluation happens. This is the primary guard against a caller forgetting to
// supply the runtime identity.
func TestRender_RuntimeName_EmptyFailsAtBoundary(t *testing.T) {
	cueCtx := cuecontext.New()
	raw := cueCtx.CompileString(`{
		metadata: { name: "demo", namespace: "apps" }
		components: {}
	}`)
	rel := &module.Instance{
		Metadata: &module.InstanceMetadata{Name: "demo", Namespace: "apps"},
		Module:   module.Module{Metadata: &module.ModuleMetadata{FQN: "example.com/modules/demo@v1", Version: "v1"}},
		Spec:     raw,
	}
	p := &provider.Provider{Data: cueCtx.CompileString(`{#transformers:{}}`)}

	_, err := ProcessModuleInstance(context.Background(), rel, p, "")
	require.Error(t, err, "empty runtimeName must be rejected at the boundary")
	assert.Contains(t, err.Error(), "runtimeName")
}

// TestRender_RuntimeName_MissingFailsCUEEvaluation verifies that the catalog's
// schema-level contract (#runtimeName declared with "!") surfaces a CUE evaluation
// error when a value is missing. This is a defense-in-depth check against a
// future code path that bypasses ProcessModuleInstance entirely.
//
// The test exercises the mandatory-field constraint directly with an inline schema
// that mirrors the catalog's #runtimeName!: string declaration — pkg/render tests
// do not have the real catalog on the CUE module path.
func TestRender_RuntimeName_MissingFailsCUEEvaluation(t *testing.T) {
	cueCtx := cuecontext.New()

	val := cueCtx.CompileString(`
		#Ctx: {
			#moduleInstanceMetadata: {
				name!:      string
				namespace!: string
				fqn:        string
				version:    string
				uuid:       string
			}
			#componentMetadata: {
				name!: string
			}
			#runtimeName!: string
		}

		ctx: #Ctx & {
			#moduleInstanceMetadata: {
				name:      "demo"
				namespace: "apps"
				fqn:       "example.com/modules/demo@v1"
				version:   "v1"
				uuid:      "a3b8f2e1-7c4d-5a9e-b6f0-1234567890ab"
			}
			#componentMetadata: {
				name: "cfg"
			}
		}
	`)
	require.NoError(t, val.Err())

	ctxVal := val.LookupPath(cue.ParsePath("ctx"))
	err := ctxVal.Validate(cue.Concrete(true))
	require.Error(t, err, "missing #runtimeName must surface as a CUE evaluation error")
	assert.Contains(t, err.Error(), "runtimeName",
		"error must mention the missing required field")
}
