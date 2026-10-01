package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/output"
)

// configMap builds a v1 ConfigMap with one data key, plus the server-managed
// fields a live object carries.
func configMap(value string, withServerFields bool) *unstructured.Unstructured {
	obj := map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      "cm",
			"namespace": "default",
		},
		"data": map[string]any{"key": value},
	}
	u := &unstructured.Unstructured{Object: obj}
	if withServerFields {
		meta := obj["metadata"].(map[string]any)
		meta["resourceVersion"] = "100"
		meta["generation"] = int64(1)
		meta["managedFields"] = []any{map[string]any{"manager": "opm", "time": "2026-01-01T00:00:00Z"}}
	}
	return u
}

// dryRunClient serves existing from the tracker and answers every apply patch
// like a server-side dry-run: the response is the object the apply would
// produce (projected) and its resourceVersion is never bumped.
func dryRunClient(t *testing.T, existing *unstructured.Unstructured, projected func() *unstructured.Unstructured) *Client {
	t.Helper()
	var objs []runtime.Object
	if existing != nil {
		objs = append(objs, existing)
	}
	fake := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
	fake.PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, projected(), nil
	})
	return &Client{Dynamic: fake}
}

func TestApplyOne_DryRunStatus(t *testing.T) {
	tests := []struct {
		name      string
		existing  *unstructured.Unstructured
		projected func() *unstructured.Unstructured
		want      string
	}{
		{
			name:     "absent object is created",
			existing: nil,
			projected: func() *unstructured.Unstructured {
				return configMap("a", false)
			},
			want: output.StatusCreated,
		},
		{
			name:     "same content with different volatile fields is unchanged",
			existing: configMap("a", true),
			projected: func() *unstructured.Unstructured {
				u := configMap("a", true)
				// A dry-run response carries refreshed managedFields timestamps.
				u.Object["metadata"].(map[string]any)["managedFields"] = []any{
					map[string]any{"manager": "opm", "time": "2026-02-02T00:00:00Z"},
				}
				u.Object["status"] = map[string]any{"observed": "x"}
				return u
			},
			want: output.StatusUnchanged,
		},
		{
			name:     "changed data is configured despite identical resourceVersion",
			existing: configMap("a", true),
			projected: func() *unstructured.Unstructured {
				return configMap("b", true)
			},
			want: output.StatusConfigured,
		},
		{
			name:     "added label is configured",
			existing: configMap("a", true),
			projected: func() *unstructured.Unstructured {
				u := configMap("a", true)
				u.SetLabels(map[string]string{"app": "x"})
				return u
			},
			want: output.StatusConfigured,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := dryRunClient(t, tt.existing, tt.projected)
			got, err := ApplyOne(context.Background(), client, configMap("b", false), ApplyOptions{DryRun: true})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// A real apply still decides by resourceVersion: the server bumps it only when
// the object changed.
func TestApplyOne_ResourceVersionComparisonKeptOnRealApply(t *testing.T) {
	tests := []struct {
		name      string
		newRV     string
		wantState string
	}{
		{"resourceVersion unchanged", "100", output.StatusUnchanged},
		{"resourceVersion bumped", "101", output.StatusConfigured},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := dryRunClient(t, configMap("a", true), func() *unstructured.Unstructured {
				u := configMap("a", true) // identical content: only the version differs
				u.SetResourceVersion(tt.newRV)
				return u
			})
			got, err := ApplyOne(context.Background(), client, configMap("a", false), ApplyOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantState, got)
		})
	}
}

func TestNormalizedContent_DoesNotMutateInput(t *testing.T) {
	obj := configMap("a", true)
	obj.Object["status"] = map[string]any{"x": "y"}

	got := normalizedContent(obj)

	meta := got["metadata"].(map[string]any)
	assert.NotContains(t, meta, "managedFields")
	assert.NotContains(t, meta, "resourceVersion")
	assert.NotContains(t, meta, "generation")
	assert.NotContains(t, got, "status")
	assert.Equal(t, "100", obj.GetResourceVersion(), "input keeps its volatile fields")
	assert.Contains(t, obj.Object, "status")
}
