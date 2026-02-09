package build

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteJob_SingleResource(t *testing.T) {
	ctx := cuecontext.New()
	executor := NewExecutor(1)

	// A minimal transformer that produces a single Deployment
	transformerCUE := ctx.CompileString(`{
		#transform: {
			#component: _
			#context: {
				name: string
				namespace: string
				...
			}
			output: {
				apiVersion: "apps/v1"
				kind: "Deployment"
				metadata: {
					name: "test-deploy"
					namespace: "default"
				}
				spec: {}
			}
		}
	}`)
	require.NoError(t, transformerCUE.Err())

	componentCUE := ctx.CompileString(`{
		metadata: { name: "web" }
		spec: {}
	}`)
	require.NoError(t, componentCUE.Err())

	job := Job{
		Transformer: &LoadedTransformer{
			Name:  "DeploymentTransformer",
			FQN:   "test/transformers#DeploymentTransformer",
			Value: transformerCUE,
		},
		Component: &LoadedComponent{
			Name:        "web",
			Labels:      map[string]string{},
			Annotations: map[string]string{},
			Resources:   map[string]cue.Value{},
			Traits:      map[string]cue.Value{},
			Value:       componentCUE,
		},
		Release: &BuiltRelease{
			Metadata: ReleaseMetadata{
				Name:      "test",
				Namespace: "default",
				Version:   "1.0.0",
				Labels:    map[string]string{},
			},
		},
	}

	result := executor.executeJob(job)

	require.NoError(t, result.Error)
	assert.Len(t, result.Resources, 1, "single resource output should produce exactly 1 resource")
	assert.Equal(t, "Deployment", result.Resources[0].GetKind())
	assert.Equal(t, "test-deploy", result.Resources[0].GetName())
}

func TestExecuteJob_MapOutput(t *testing.T) {
	ctx := cuecontext.New()
	executor := NewExecutor(1)

	// A transformer that produces a map of resources (like PVC/ConfigMap/Secret transformers)
	transformerCUE := ctx.CompileString(`{
		#transform: {
			#component: _
			#context: {
				name: string
				namespace: string
				...
			}
			output: {
				"config": {
					apiVersion: "v1"
					kind: "PersistentVolumeClaim"
					metadata: {
						name: "config"
						namespace: "default"
					}
					spec: {
						accessModes: ["ReadWriteOnce"]
					}
				}
				"data": {
					apiVersion: "v1"
					kind: "PersistentVolumeClaim"
					metadata: {
						name: "data"
						namespace: "default"
					}
					spec: {
						accessModes: ["ReadWriteOnce"]
					}
				}
			}
		}
	}`)
	require.NoError(t, transformerCUE.Err())

	componentCUE := ctx.CompileString(`{
		metadata: { name: "storage" }
		spec: {}
	}`)
	require.NoError(t, componentCUE.Err())

	job := Job{
		Transformer: &LoadedTransformer{
			Name:  "PVCTransformer",
			FQN:   "test/transformers#PVCTransformer",
			Value: transformerCUE,
		},
		Component: &LoadedComponent{
			Name:        "storage",
			Labels:      map[string]string{},
			Annotations: map[string]string{"transformer.opmodel.dev/list-output": "true"},
			Resources:   map[string]cue.Value{},
			Traits:      map[string]cue.Value{},
			Value:       componentCUE,
		},
		Release: &BuiltRelease{
			Metadata: ReleaseMetadata{
				Name:      "test",
				Namespace: "default",
				Version:   "1.0.0",
				Labels:    map[string]string{},
			},
		},
	}

	result := executor.executeJob(job)

	require.NoError(t, result.Error)
	assert.Len(t, result.Resources, 2, "map output with 2 entries should produce 2 resources")

	// Verify all resources are PVCs
	for _, res := range result.Resources {
		assert.Equal(t, "PersistentVolumeClaim", res.GetKind())
	}
}
