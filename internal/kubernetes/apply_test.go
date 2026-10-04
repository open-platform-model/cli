package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// stagingCluster is a fake API server for the staging tests. It answers every
// server-side apply by echoing the patch body and recording "Kind/name" in
// patch order, and every GET with NotFound except for the CustomResourceDefinition,
// which exists once applied (or from the start, with crdExists) and reports
// Established=True only when established is set, and the objects named in
// existing.
type stagingCluster struct {
	mu          sync.Mutex
	patched     []string
	crdExists   bool
	established bool
	// existing names objects, as "<resource>/<name>" (namespaces/demo), that
	// every read finds.
	existing map[string]bool
	// crdForbidden makes every read of the CustomResourceDefinition fail
	// with Forbidden, as for a user who may patch it but not get it.
	crdForbidden bool
}

func (c *stagingCluster) client(t *testing.T) *Client {
	t.Helper()
	fake := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	fake.PrependReactor("get", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		get := action.(k8stesting.GetAction)
		if c.crdForbidden && action.GetResource().Resource == "customresourcedefinitions" {
			return true, nil, apierrors.NewForbidden(action.GetResource().GroupResource(), get.GetName(), nil)
		}
		if c.existing[action.GetResource().Resource+"/"+get.GetName()] {
			found := &unstructured.Unstructured{Object: map[string]any{}}
			found.SetName(get.GetName())
			found.SetResourceVersion("1")
			return true, found, nil
		}
		if action.GetResource().Resource != "customresourcedefinitions" || !c.crdExists {
			return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), get.GetName())
		}
		crd := stagingCRD()
		crd.SetResourceVersion("1")
		if c.established {
			_ = unstructured.SetNestedSlice(crd.Object, []any{
				map[string]any{"type": "Established", "status": "True"},
			}, "status", "conditions")
		}
		return true, crd, nil
	})
	fake.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		patch := action.(k8stesting.PatchActionImpl)
		obj := &unstructured.Unstructured{}
		require.NoError(t, json.Unmarshal(patch.GetPatch(), &obj.Object))
		c.patched = append(c.patched, obj.GetKind()+"/"+obj.GetName())
		if obj.GetKind() == "CustomResourceDefinition" && len(patch.PatchOptions.DryRun) == 0 {
			c.crdExists = true
		}
		obj.SetResourceVersion("2")
		return true, obj, nil
	})
	return &Client{Dynamic: fake}
}

func (c *stagingCluster) patchOrder() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.patched...)
}

func stagingObject(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersion,
		"kind":       kind,
		"metadata":   map[string]any{"name": name},
	}}
	if namespace != "" {
		u.SetNamespace(namespace)
	}
	return u
}

func stagingCRD() *unstructured.Unstructured {
	crd := stagingObject("apiextensions.k8s.io/v1", "CustomResourceDefinition", "foos.example.com", "")
	_ = unstructured.SetNestedField(crd.Object, "example.com", "spec", "group")
	_ = unstructured.SetNestedField(crd.Object, "Foo", "spec", "names", "kind")
	return crd
}

// stagingInput is a module's resources in the worst build order: everything
// before the definitions it depends on.
func stagingInput() []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		stagingObject("apps/v1", "Deployment", "web", "demo"),
		stagingObject("v1", "Service", "web", "demo"),
		stagingObject("example.com/v1", "Foo", "my-foo", "demo"),
		stagingObject("v1", "ConfigMap", "cfg", "demo"),
		stagingObject("v1", "Namespace", "demo", ""),
		stagingCRD(),
	}
}

func kindsAndNames(objs []*unstructured.Unstructured) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.GetKind() + "/" + o.GetName()
	}
	return out
}

func shortWaitPoll(t *testing.T) {
	t.Helper()
	prev := WaitPollInterval
	WaitPollInterval = 5 * time.Millisecond
	t.Cleanup(func() { WaitPollInterval = prev })
}

func TestApply_StagesDefinitionsThenWeight(t *testing.T) {
	shortWaitPoll(t)
	cluster := &stagingCluster{established: true}
	input := stagingInput()
	before := kindsAndNames(input)

	result, err := Apply(context.Background(), cluster.client(t), input, "test", ApplyOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 6, result.Applied)
	assert.Equal(t, []string{
		"CustomResourceDefinition/foos.example.com",
		"Namespace/demo",
		"ConfigMap/cfg",
		"Service/web",
		"Deployment/web",
		"Foo/my-foo",
	}, cluster.patchOrder())
	assert.Equal(t, before, kindsAndNames(input), "the caller's slice must not be reordered")
}

func TestApply_CRDNotEstablishedStopsBeforeSecondStage(t *testing.T) {
	shortWaitPoll(t)
	cluster := &stagingCluster{established: false}

	_, err := Apply(context.Background(), cluster.client(t), stagingInput(), "test", ApplyOptions{
		EstablishDeadline: time.Now().Add(200 * time.Millisecond),
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "CustomResourceDefinition/foos.example.com")
	assert.ErrorContains(t, err, "established")
	assert.Equal(t, []string{
		"CustomResourceDefinition/foos.example.com",
		"Namespace/demo",
	}, cluster.patchOrder(), "nothing of the second stage may be applied")
}

func TestApply_CRDTimeoutReportsTimeSinceBudgetStart(t *testing.T) {
	shortWaitPoll(t)
	cluster := &stagingCluster{established: false}

	// The budget started an hour before the wait: the apply spent it. The
	// timeout must report the hour, not the wait's own few milliseconds.
	_, err := Apply(context.Background(), cluster.client(t), stagingInput(), "test", ApplyOptions{
		EstablishDeadline: time.Now().Add(100 * time.Millisecond),
		BudgetStart:       time.Now().Add(-time.Hour),
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out after 1h0m0s")
}

func TestApply_DryRunSkipsCustomResourceOfNewCRD(t *testing.T) {
	// A wait would time out: the CRD never exists and never reports
	// Established. The short deadline makes an accidental wait fail fast.
	shortWaitPoll(t)
	cluster := &stagingCluster{existing: demoNamespaceExists()}

	result, err := Apply(context.Background(), cluster.client(t), stagingInput(), "test", ApplyOptions{
		DryRun:            true,
		EstablishDeadline: time.Now().Add(100 * time.Millisecond),
	})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 5, result.Applied)
	assert.NotContains(t, cluster.patchOrder(), "Foo/my-foo")
}

func TestApply_DryRunSendsCustomResourceOfExistingCRD(t *testing.T) {
	cluster := &stagingCluster{crdExists: true, established: true, existing: demoNamespaceExists()}

	result, err := Apply(context.Background(), cluster.client(t), stagingInput(), "test", ApplyOptions{DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 0, result.Skipped)
	assert.Contains(t, cluster.patchOrder(), "Foo/my-foo")
}

func TestApply_DryRunSendsCustomResourceWhenCRDReadIsForbidden(t *testing.T) {
	// A refused read is not proof the CRD is new, so the custom resource is
	// sent, not skipped.
	cluster := &stagingCluster{crdForbidden: true, existing: demoNamespaceExists()}

	result, err := Apply(context.Background(), cluster.client(t), stagingInput(), "test", ApplyOptions{DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Skipped)
	assert.Contains(t, cluster.patchOrder(), "Foo/my-foo")
}

// demoNamespaceExists marks the staging namespace as existing, so a dry-run
// test of the CustomResourceDefinition skip checks that reason alone.
func demoNamespaceExists() map[string]bool {
	return map[string]bool{"namespaces/demo": true}
}

// namespacedInput is stagingInput without the CustomResourceDefinition and
// its custom resource, plus a cluster-scoped ClusterRole: a Namespace and
// three objects in it.
func namespacedInput() []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		stagingObject("apps/v1", "Deployment", "web", "demo"),
		stagingObject("v1", "Service", "web", "demo"),
		stagingObject("rbac.authorization.k8s.io/v1", "ClusterRole", "reader", ""),
		stagingObject("v1", "ConfigMap", "cfg", "demo"),
		stagingObject("v1", "Namespace", "demo", ""),
	}
}

var objectsInDemo = []string{"Deployment/web", "Service/web", "ConfigMap/cfg"}

func TestApply_DryRunSkipsObjectsOfNewNamespace(t *testing.T) {
	cluster := &stagingCluster{}

	result, err := Apply(context.Background(), cluster.client(t), namespacedInput(), "test", ApplyOptions{DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 3, result.Skipped)
	assert.Equal(t, 2, result.Applied)
	assert.Equal(t, []string{"Namespace/demo", "ClusterRole/reader"}, cluster.patchOrder(),
		"the Namespace and the cluster-scoped object are sent, the objects in demo are not")
}

func TestApply_DryRunSkipsObjectsOfNewNamespacesOption(t *testing.T) {
	cluster := &stagingCluster{}
	input := namespacedInput()[:4] // no Namespace object

	result, err := Apply(context.Background(), cluster.client(t), input, "test", ApplyOptions{
		DryRun:        true,
		NewNamespaces: []string{"demo"},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 3, result.Skipped)
	assert.Equal(t, []string{"ClusterRole/reader"}, cluster.patchOrder())
}

func TestApply_DryRunSendsObjectsOfExistingNamespace(t *testing.T) {
	cluster := &stagingCluster{existing: demoNamespaceExists()}

	result, err := Apply(context.Background(), cluster.client(t), namespacedInput(), "test", ApplyOptions{DryRun: true})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 0, result.Skipped)
	assert.Subset(t, cluster.patchOrder(), objectsInDemo)
}

func TestApply_RealApplyIgnoresNewNamespaces(t *testing.T) {
	cluster := &stagingCluster{}

	result, err := Apply(context.Background(), cluster.client(t), namespacedInput(), "test", ApplyOptions{
		NewNamespaces: []string{"demo"},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 0, result.Skipped)
	assert.Equal(t, 5, result.Applied)
	assert.Subset(t, cluster.patchOrder(), objectsInDemo)
}

func TestApply_DryRunCustomResourceOfNewCRDInNewNamespaceSkippedOnce(t *testing.T) {
	// The Foo has both reasons to be skipped; it counts once and is warned
	// about for its CustomResourceDefinition, not for its namespace.
	shortWaitPoll(t)
	cluster := &stagingCluster{}
	var logBuf bytes.Buffer
	output.SetLogWriter(&logBuf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	result, err := Apply(context.Background(), cluster.client(t), stagingInput(), "test", ApplyOptions{
		DryRun:            true,
		EstablishDeadline: time.Now().Add(100 * time.Millisecond),
	})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 4, result.Skipped)
	assert.Equal(t, 2, result.Applied)
	assert.NotContains(t, cluster.patchOrder(), "Foo/my-foo")
	logged := logBuf.String()
	assert.Contains(t, logged, "its CustomResourceDefinition foos.example.com is created by this apply")
	assert.NotContains(t, logged, "skipping Foo/my-foo in demo: namespace")
}
