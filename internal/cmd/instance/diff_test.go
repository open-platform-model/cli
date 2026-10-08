package instance

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

const noDifferences = "No differences found"

func renderedConfigMap(name, value string) *unstructured.Unstructured {
	cm := trackedConfigMap(name)
	cm.Object["data"] = map[string]any{"key": value}
	return cm
}

// failGets makes the GET of each named object of a resource fail with its error.
func failGets(fake interface {
	PrependReactor(verb, resource string, reaction k8stesting.ReactionFunc)
}, resource string, failures map[string]error) {
	fake.PrependReactor("get", resource, func(a k8stesting.Action) (bool, runtime.Object, error) {
		if err, ok := failures[a.(k8stesting.GetAction).GetName()]; ok {
			return true, nil, err
		}
		return false, nil, nil
	})
}

func runDiff(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	var runErr error
	out := captureOutput(t, func() { runErr = fn() })
	return out, runErr
}

// Every rendered resource is unreadable: one error per resource, no "No
// differences found", and the exit code of the cause.
func TestExecuteInstanceDiff_AllReadsForbiddenFails(t *testing.T) {
	client, fake := fakeClusterClient(renderedConfigMap("a", "1"), renderedConfigMap("b", "1"))
	failGets(fake, "configmaps", map[string]error{
		"a": forbiddenRead("configmaps", "a"),
		"b": forbiddenRead("configmaps", "b"),
	})

	out, err := runDiff(t, func() error {
		return executeInstanceDiff(context.Background(), client,
			[]*unstructured.Unstructured{renderedConfigMap("a", "1"), renderedConfigMap("b", "1")},
			"demo", "apps", "", output.InstanceLogger("demo"))
	})

	requireExitCode(t, err, opmexit.ExitPermissionDenied)
	assert.NotContains(t, out, noDifferences)
	assert.Contains(t, out, "name=a")
	assert.Contains(t, out, "name=b")
	assert.Contains(t, out, "diff is incomplete: 2 object(s) could not be read or compared")
}

// One resource differs and another cannot be read: the difference is printed,
// the unreadable resource is an error, and the command fails.
func TestExecuteInstanceDiff_PrintsDifferencesAndFails(t *testing.T) {
	client, fake := fakeClusterClient(renderedConfigMap("changed", "old"), renderedConfigMap("broken", "1"))
	failGets(fake, "configmaps", map[string]error{"broken": apierrors.NewInternalError(errors.New("etcd is down"))})

	out, err := runDiff(t, func() error {
		return executeInstanceDiff(context.Background(), client,
			[]*unstructured.Unstructured{renderedConfigMap("changed", "new"), renderedConfigMap("broken", "1")},
			"demo", "apps", "", output.InstanceLogger("demo"))
	})

	requireExitCode(t, err, opmexit.ExitGeneralError)
	assert.Contains(t, out, "ConfigMap/changed (apps) [modified]")
	assert.Contains(t, out, "name=broken")
	assert.NotContains(t, out, noDifferences)
}

// The instance record cannot be read: orphan detection did not run, so the
// diff is incomplete even though every rendered resource is unchanged.
func TestExecuteInstanceDiff_UnreadableRecordFails(t *testing.T) {
	client, fake := fakeClusterClient(renderedConfigMap("web", "1"))
	failGets(fake, inventory.ModuleInstanceGVR.Resource, map[string]error{
		"demo": apierrors.NewForbidden(schema.GroupResource{Group: inventory.GroupOpmodel, Resource: "moduleinstances"}, "demo", errors.New("denied")),
	})

	out, err := runDiff(t, func() error {
		return executeInstanceDiff(context.Background(), client,
			[]*unstructured.Unstructured{renderedConfigMap("web", "1")},
			"demo", "apps", "uuid-demo", output.InstanceLogger("demo"))
	})

	requireExitCode(t, err, opmexit.ExitPermissionDenied)
	assert.NotContains(t, out, noDifferences)
	assert.Contains(t, out, "apps/demo")
	assert.Contains(t, out, "orphan detection did not run")
}

// A tracked resource that is not rendered cannot be read: it may be an orphan,
// so the diff names it, says orphan detection could not check it, and fails.
// A tracked resource that is rendered and unreadable counts once.
func TestExecuteInstanceDiff_UnreadableTrackedResourceFails(t *testing.T) {
	mi := moduleInstanceObj("apps", "demo")
	require.NoError(t, unstructured.SetNestedSlice(mi.Object, []any{
		map[string]any{"kind": "ConfigMap", "namespace": "apps", "name": "web", "v": "v1"},
		map[string]any{"kind": "ConfigMap", "namespace": "apps", "name": "settings", "v": "v1"},
		map[string]any{"kind": "ConfigMap", "namespace": "apps", "name": "both", "v": "v1"},
	}, "status", "inventory", "entries"))
	client, fake := fakeClusterClient(mi, renderedConfigMap("web", "1"), renderedConfigMap("settings", "1"), renderedConfigMap("both", "1"))
	failGets(fake, "configmaps", map[string]error{
		"settings": forbiddenRead("configmaps", "settings"),
		"both":     forbiddenRead("configmaps", "both"),
	})

	out, err := runDiff(t, func() error {
		return executeInstanceDiff(context.Background(), client,
			[]*unstructured.Unstructured{renderedConfigMap("web", "1"), renderedConfigMap("both", "1")},
			"demo", "apps", "uuid-demo", output.InstanceLogger("demo"))
	})

	requireExitCode(t, err, opmexit.ExitPermissionDenied)
	assert.NotContains(t, out, noDifferences)
	assert.Contains(t, out, "could not read tracked resource")
	assert.Contains(t, out, "name=settings")
	assert.Contains(t, out, "orphan detection could not check 1 tracked resource(s)")
	assert.Contains(t, out, "diff is incomplete: 2 object(s) could not be read or compared")
}

// Failures of two classes have no single cause to report: exit 1.
func TestExecuteInstanceDiff_MixedCausesExitGeneral(t *testing.T) {
	client, fake := fakeClusterClient(renderedConfigMap("a", "1"), renderedConfigMap("b", "1"))
	failGets(fake, "configmaps", map[string]error{
		"a": forbiddenRead("configmaps", "a"),
		"b": apierrors.NewInternalError(errors.New("etcd is down")),
	})

	_, err := runDiff(t, func() error {
		return executeInstanceDiff(context.Background(), client,
			[]*unstructured.Unstructured{renderedConfigMap("a", "1"), renderedConfigMap("b", "1")},
			"demo", "apps", "", output.InstanceLogger("demo"))
	})

	requireExitCode(t, err, opmexit.ExitGeneralError)
}

// Nothing failed and nothing differs: the one case that prints "No
// differences found". An instance with no record yet is not a failure.
func TestExecuteInstanceDiff_CleanDiff(t *testing.T) {
	client, _ := fakeClusterClient(renderedConfigMap("web", "1"))

	out, err := runDiff(t, func() error {
		return executeInstanceDiff(context.Background(), client,
			[]*unstructured.Unstructured{renderedConfigMap("web", "1")},
			"demo", "apps", "uuid-demo", output.InstanceLogger("demo"))
	})

	require.NoError(t, err, out)
	assert.Contains(t, out, noDifferences)
}
