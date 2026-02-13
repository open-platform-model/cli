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
	executor := NewExecutor()

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
	executor := NewExecutor()

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

func TestMapToVolumeMountsArray(t *testing.T) {
	volumeMounts := map[string]any{
		"config": map[string]any{
			"name":      "config",
			"mountPath": "/config",
		},
		"data": map[string]any{
			"name":      "data",
			"mountPath": "/data",
		},
	}

	result := mapToVolumeMountsArray(volumeMounts)

	assert.Len(t, result, 2)

	// Sorted by key: config, data
	vm0 := result[0].(map[string]any)
	assert.Equal(t, "config", vm0["name"])
	assert.Equal(t, "/config", vm0["mountPath"])

	vm1 := result[1].(map[string]any)
	assert.Equal(t, "data", vm1["name"])
	assert.Equal(t, "/data", vm1["mountPath"])
}

func TestMapToVolumeMountsArray_SetsNameFromKey(t *testing.T) {
	// If the volume mount doesn't have a "name" field, the map key should be used
	volumeMounts := map[string]any{
		"myvolume": map[string]any{
			"mountPath": "/mnt/data",
			"readOnly":  true,
		},
	}

	result := mapToVolumeMountsArray(volumeMounts)
	require.Len(t, result, 1)

	vm := result[0].(map[string]any)
	assert.Equal(t, "myvolume", vm["name"])
	assert.Equal(t, "/mnt/data", vm["mountPath"])
	assert.Equal(t, true, vm["readOnly"])
}

func TestNormalizeContainers_VolumeMounts(t *testing.T) {
	containers := []any{
		map[string]any{
			"name":  "app",
			"image": "nginx",
			"volumeMounts": map[string]any{
				"config": map[string]any{
					"name":      "config",
					"mountPath": "/config",
				},
			},
		},
	}

	normalizeContainers(containers)

	c := containers[0].(map[string]any)
	vms, ok := c["volumeMounts"].([]any)
	require.True(t, ok, "volumeMounts should be an array after normalization")
	require.Len(t, vms, 1)
	assert.Equal(t, "/config", vms[0].(map[string]any)["mountPath"])
}

func TestMapToVolumesArray(t *testing.T) {
	volumes := map[string]any{
		"config": map[string]any{
			"name": "config",
			"persistentVolumeClaim": map[string]any{
				"claimName": "config",
			},
		},
		"data": map[string]any{
			"name": "data",
			"persistentVolumeClaim": map[string]any{
				"claimName": "data",
			},
		},
	}

	result := mapToVolumesArray(volumes)

	assert.Len(t, result, 2)

	// Sorted by key: config, data
	vol0 := result[0].(map[string]any)
	assert.Equal(t, "config", vol0["name"])
	pvc0 := vol0["persistentVolumeClaim"].(map[string]any)
	assert.Equal(t, "config", pvc0["claimName"])

	vol1 := result[1].(map[string]any)
	assert.Equal(t, "data", vol1["name"])
	pvc1 := vol1["persistentVolumeClaim"].(map[string]any)
	assert.Equal(t, "data", pvc1["claimName"])
}

func TestMapToVolumesArray_SetsNameFromKey(t *testing.T) {
	// If the volume doesn't have a "name" field, the map key should be used
	volumes := map[string]any{
		"myvolume": map[string]any{
			"persistentVolumeClaim": map[string]any{
				"claimName": "myvolume",
			},
		},
	}

	result := mapToVolumesArray(volumes)
	require.Len(t, result, 1)

	vol := result[0].(map[string]any)
	assert.Equal(t, "myvolume", vol["name"])
}

func TestNormalizeK8sResource_Volumes(t *testing.T) {
	// StatefulSet-like resource with volumes as a map (OPM-style)
	obj := map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "StatefulSet",
		"metadata": map[string]any{
			"name": "jellyfin",
		},
		"spec": map[string]any{
			"template": map[string]any{
				"spec": map[string]any{
					"containers": []any{
						map[string]any{
							"name":  "jellyfin",
							"image": "jellyfin:latest",
							"volumeMounts": map[string]any{
								"config": map[string]any{
									"name":      "config",
									"mountPath": "/config",
								},
								"movies": map[string]any{
									"name":      "movies",
									"mountPath": "/data/movies",
								},
							},
						},
					},
					"volumes": map[string]any{
						"config": map[string]any{
							"name": "config",
							"persistentVolumeClaim": map[string]any{
								"claimName": "config",
							},
						},
						"movies": map[string]any{
							"name": "movies",
							"persistentVolumeClaim": map[string]any{
								"claimName": "movies",
							},
						},
					},
				},
			},
		},
	}

	normalizeK8sResource(obj)

	// Verify volumes were converted from map to array
	spec := obj["spec"].(map[string]any)
	template := spec["template"].(map[string]any)
	templateSpec := template["spec"].(map[string]any)

	volumes, ok := templateSpec["volumes"].([]any)
	require.True(t, ok, "volumes should be an array after normalization")
	assert.Len(t, volumes, 2)

	// Sorted: config, movies
	vol0 := volumes[0].(map[string]any)
	assert.Equal(t, "config", vol0["name"])
	vol1 := volumes[1].(map[string]any)
	assert.Equal(t, "movies", vol1["name"])

	// Verify volumeMounts were also normalized
	containers := templateSpec["containers"].([]any)
	container := containers[0].(map[string]any)
	vms, ok := container["volumeMounts"].([]any)
	require.True(t, ok, "volumeMounts should be an array after normalization")
	assert.Len(t, vms, 2)
}

func TestNormalizeK8sResource_CronJobVolumes(t *testing.T) {
	// CronJob with volumes in the nested jobTemplate
	obj := map[string]any{
		"apiVersion": "batch/v1",
		"kind":       "CronJob",
		"metadata": map[string]any{
			"name": "backup",
		},
		"spec": map[string]any{
			"jobTemplate": map[string]any{
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name":  "backup",
									"image": "backup:latest",
								},
							},
							"volumes": map[string]any{
								"data": map[string]any{
									"name": "data",
									"persistentVolumeClaim": map[string]any{
										"claimName": "data",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	normalizeK8sResource(obj)

	spec := obj["spec"].(map[string]any)
	jobTemplate := spec["jobTemplate"].(map[string]any)
	jobSpec := jobTemplate["spec"].(map[string]any)
	template := jobSpec["template"].(map[string]any)
	templateSpec := template["spec"].(map[string]any)

	volumes, ok := templateSpec["volumes"].([]any)
	require.True(t, ok, "volumes should be an array after normalization")
	assert.Len(t, volumes, 1)
	assert.Equal(t, "data", volumes[0].(map[string]any)["name"])
}
