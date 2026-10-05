package apply

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// namespaceDryRun records one run: the resources patched, whether a
// Namespace was created, and the log.
type namespaceDryRun struct {
	mu        sync.Mutex
	patched   []string
	nsCreated bool
	log       bytes.Buffer
}

// runNamespaceDryRun dry-runs one ConfigMap in the instance namespace
// "default", with --create-namespace set to createNS and the namespace present
// on the cluster when nsExists.
func runNamespaceDryRun(t *testing.T, createNS, nsExists bool) (*namespaceDryRun, error) {
	t.Helper()
	run := &namespaceDryRun{}
	output.SetLogWriter(&run.log)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })

	fake := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	fake.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		run.mu.Lock()
		defer run.mu.Unlock()
		run.patched = append(run.patched, action.GetResource().Resource)
		return true, renderedConfigMap("cfg"), nil
	})

	var objs []runtime.Object
	if nsExists {
		objs = append(objs, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	}
	cs := k8sfake.NewClientset(objs...)
	cs.PrependReactor("create", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
		run.mu.Lock()
		defer run.mu.Unlock()
		run.nsCreated = true
		return false, nil, nil
	})

	err := Execute(context.Background(), Request{
		Result: &workflowrender.Result{
			Resources: []*unstructured.Unstructured{renderedConfigMap("cfg")},
			Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default"},
		},
		K8sClient: &kubernetes.Client{Dynamic: fake, Clientset: cs},
		Log:       output.InstanceLogger("demo"),
		Options:   Options{DryRun: true, CreateNS: createNS},
	})
	return run, err
}

// A dry run with --create-namespace and no such namespace does not send the
// objects in it: the namespace is not created, so the server would refuse
// them although the real apply would not.
func TestExecute_DryRunSkipsObjectsOfNamespaceCreateNamespaceWouldCreate(t *testing.T) {
	run, err := runNamespaceDryRun(t, true, false)
	require.NoError(t, err)

	run.mu.Lock()
	defer run.mu.Unlock()
	assert.Empty(t, run.patched, "the ConfigMap in the new namespace is never sent")
	assert.False(t, run.nsCreated, "a dry run creates no namespace")
	assert.Contains(t, run.log.String(), `namespace "default" would be created`)
	assert.Contains(t, run.log.String(), "skipping ConfigMap/cfg in default: namespace default is created by this apply, so a dry run cannot validate it")
	assert.Contains(t, run.log.String(), "0 resources would be applied, 1 skipped")
}

// Without --create-namespace a missing namespace is not skipped: the real
// apply would fail on it too, so the dry run sends the object.
func TestExecute_DryRunWithoutCreateNamespaceSendsObjectsOfMissingNamespace(t *testing.T) {
	run, err := runNamespaceDryRun(t, false, false)
	require.NoError(t, err)

	run.mu.Lock()
	defer run.mu.Unlock()
	assert.Equal(t, []string{"configmaps"}, run.patched)
	assert.NotContains(t, run.log.String(), "skipped")
}

// With --create-namespace and the namespace already present nothing is
// skipped.
func TestExecute_DryRunCreateNamespaceExistingSendsEverything(t *testing.T) {
	run, err := runNamespaceDryRun(t, true, true)
	require.NoError(t, err)

	run.mu.Lock()
	defer run.mu.Unlock()
	assert.Equal(t, []string{"configmaps"}, run.patched)
	assert.Contains(t, run.log.String(), "1 resources would be applied")
	assert.NotContains(t, run.log.String(), "skipped")
}
