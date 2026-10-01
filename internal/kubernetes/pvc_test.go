package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/output"
)

func pvc(size, phase, class string) *unstructured.Unstructured {
	spec := map[string]any{
		"resources": map[string]any{"requests": map[string]any{"storage": size}},
	}
	if class != "" {
		spec["storageClassName"] = class
	}
	obj := map[string]any{
		"apiVersion": "v1",
		"kind":       "PersistentVolumeClaim",
		"metadata":   map[string]any{"name": "data", "namespace": "default"},
		"spec":       spec,
	}
	if phase != "" {
		obj["status"] = map[string]any{"phase": phase}
	}
	return &unstructured.Unstructured{Object: obj}
}

func storageClass(name string, allowExpansion *bool) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		ObjectMeta:           metav1.ObjectMeta{Name: name},
		AllowVolumeExpansion: allowExpansion,
	}
}

func boolPtr(b bool) *bool { return &b }

func captureWarnings(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

func TestGuardPVCResize(t *testing.T) {
	tests := []struct {
		name         string
		rendered     *unstructured.Unstructured
		live         *unstructured.Unstructured
		clientset    func() *Client
		wantSize     string
		wantWarning  []string
		wantNoWarn   bool
		wantSameObj  bool
		wantSCLookup bool
	}{
		{
			name:        "same size passes through",
			rendered:    pvc("10Gi", "", "standard"),
			live:        pvc("10Gi", "Bound", "standard"),
			clientset:   func() *Client { return &Client{Clientset: fake.NewClientset()} },
			wantSize:    "10Gi",
			wantNoWarn:  true,
			wantSameObj: true,
		},
		{
			name:        "equal quantity in another notation passes through",
			rendered:    pvc("10240Mi", "", "standard"),
			live:        pvc("10Gi", "Bound", "standard"),
			clientset:   func() *Client { return &Client{Clientset: fake.NewClientset()} },
			wantSize:    "10240Mi",
			wantNoWarn:  true,
			wantSameObj: true,
		},
		{
			name:     "growth on a non-expandable class keeps the live size",
			rendered: pvc("20Gi", "", "standard"),
			live:     pvc("10Gi", "Bound", "standard"),
			clientset: func() *Client {
				return &Client{Clientset: fake.NewClientset(storageClass("standard", nil))}
			},
			wantSize:    "10Gi",
			wantWarning: []string{"default/data", "10Gi", "20Gi", `StorageClass "standard" does not allow volume expansion`},
		},
		{
			name:     "growth on a class with expansion explicitly off keeps the live size",
			rendered: pvc("20Gi", "", "standard"),
			live:     pvc("10Gi", "Bound", "standard"),
			clientset: func() *Client {
				return &Client{Clientset: fake.NewClientset(storageClass("standard", boolPtr(false)))}
			},
			wantSize:    "10Gi",
			wantWarning: []string{"does not allow volume expansion"},
		},
		{
			name:     "growth on an expandable class passes through",
			rendered: pvc("20Gi", "", "fast"),
			live:     pvc("10Gi", "Bound", "fast"),
			clientset: func() *Client {
				return &Client{Clientset: fake.NewClientset(storageClass("fast", boolPtr(true)))}
			},
			wantSize:    "20Gi",
			wantNoWarn:  true,
			wantSameObj: true,
		},
		{
			name:     "shrink on an expandable class keeps the live size",
			rendered: pvc("5Gi", "", "fast"),
			live:     pvc("10Gi", "Bound", "fast"),
			clientset: func() *Client {
				return &Client{Clientset: fake.NewClientset(storageClass("fast", boolPtr(true)))}
			},
			wantSize:    "10Gi",
			wantWarning: []string{"default/data", "10Gi", "5Gi", "cannot shrink"},
		},
		{
			name:        "missing StorageClass is treated as non-expandable",
			rendered:    pvc("20Gi", "", "gone"),
			live:        pvc("10Gi", "Bound", "gone"),
			clientset:   func() *Client { return &Client{Clientset: fake.NewClientset()} },
			wantSize:    "10Gi",
			wantWarning: []string{`StorageClass "gone" does not exist`},
		},
		{
			name:     "unreadable StorageClass is treated as non-expandable",
			rendered: pvc("20Gi", "", "standard"),
			live:     pvc("10Gi", "Bound", "standard"),
			clientset: func() *Client {
				cs := fake.NewClientset()
				cs.PrependReactor("get", "storageclasses", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, errors.New("forbidden")
				})
				return &Client{Clientset: cs}
			},
			wantSize:    "10Gi",
			wantWarning: []string{`StorageClass "standard" could not be read`, "forbidden"},
		},
		{
			name:        "claim without a StorageClass name is treated as non-expandable",
			rendered:    pvc("20Gi", "", ""),
			live:        pvc("10Gi", "Bound", ""),
			clientset:   func() *Client { return &Client{Clientset: fake.NewClientset()} },
			wantSize:    "10Gi",
			wantWarning: []string{"names no StorageClass"},
		},
		{
			name:        "no clientset is treated as unreadable",
			rendered:    pvc("20Gi", "", "standard"),
			live:        pvc("10Gi", "Bound", "standard"),
			clientset:   func() *Client { return &Client{} },
			wantSize:    "10Gi",
			wantWarning: []string{`StorageClass "standard" could not be read`},
		},
		{
			name:     "a claim that is not Bound is left to the API server",
			rendered: pvc("20Gi", "", "standard"),
			live:     pvc("10Gi", "Pending", "standard"),
			clientset: func() *Client {
				return &Client{Clientset: fake.NewClientset(storageClass("standard", nil))}
			},
			wantSize:    "20Gi",
			wantNoWarn:  true,
			wantSameObj: true,
		},
		{
			name:        "an unparseable size is left to the API server",
			rendered:    pvc("lots", "", "standard"),
			live:        pvc("10Gi", "Bound", "standard"),
			clientset:   func() *Client { return &Client{Clientset: fake.NewClientset()} },
			wantSize:    "lots",
			wantNoWarn:  true,
			wantSameObj: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureWarnings(t)
			rendered := tt.rendered
			before := rendered.DeepCopy()

			got := guardPVCResize(context.Background(), tt.clientset(), rendered, tt.live)

			size, found, err := unstructured.NestedString(got.Object, pvcStoragePath...)
			require.NoError(t, err)
			require.True(t, found, "the storage field stays in the applied object so SSA keeps owning it")
			assert.Equal(t, tt.wantSize, size)
			assert.Equal(t, before.Object, rendered.Object, "the caller's rendered object is never modified")
			if tt.wantSameObj {
				assert.Same(t, rendered, got)
			}
			if tt.wantNoWarn {
				assert.Empty(t, logs.String())
			}
			for _, want := range tt.wantWarning {
				assert.Contains(t, logs.String(), want)
			}
			if len(tt.wantWarning) > 0 {
				assert.Contains(t, logs.String(), "WARN")
			}
		})
	}
}

func TestGuardPVCResize_IgnoresOtherKinds(t *testing.T) {
	logs := captureWarnings(t)
	cm := configMap("a", false)
	live := configMap("b", true)
	live.Object["status"] = map[string]any{"phase": "Bound"}

	got := guardPVCResize(context.Background(), &Client{}, cm, live)

	assert.Same(t, cm, got)
	assert.Empty(t, logs.String())
}

// ApplyOne must send the live size, not the rendered one, for a clamped claim,
// and must still send the field.
func TestApplyOne_PVCResizeSendsLiveSize(t *testing.T) {
	logs := captureWarnings(t)

	live := pvc("10Gi", "Bound", "standard")
	live.SetResourceVersion("5") // an object served by a real API server always has one
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), live)
	var sent []byte
	dyn.PrependReactor("patch", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		sent = action.(k8stesting.PatchAction).GetPatch()
		return true, live.DeepCopy(), nil
	})
	client := &Client{
		Dynamic:   dyn,
		Clientset: fake.NewClientset(storageClass("standard", nil)),
	}

	rendered := pvc("20Gi", "", "standard")
	status, err := ApplyOne(context.Background(), client, rendered, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, output.StatusUnchanged, status)

	var patch struct {
		Spec struct {
			Resources struct {
				Requests struct {
					Storage string `json:"storage"`
				} `json:"requests"`
			} `json:"resources"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal(sent, &patch))
	assert.Equal(t, "10Gi", patch.Spec.Resources.Requests.Storage)
	assert.Contains(t, logs.String(), "default/data")

	got, _, _ := unstructured.NestedString(rendered.Object, pvcStoragePath...)
	assert.Equal(t, "20Gi", got, "the caller's object is untouched")
}
