package instance

import (
	"context"
	"testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

func trackedConfigMap(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": name, "namespace": "apps"},
	}}
}

// Tree warns about each tracked resource it could not read and shows the ones
// it could.
func TestShowInstanceTree_WarnsAboutUnreadable(t *testing.T) {
	web := trackedConfigMap("web")
	client, _ := fakeClusterClient(web.DeepCopy())
	inv := &inventory.Record{Name: "demo", Namespace: "apps", Inventory: inventory.Inventory{Entries: []k8sinventory.Entry{
		{Kind: "ConfigMap", Namespace: "apps", Name: "web", Version: "v1"},
		{Kind: "ConfigMap", Namespace: "apps", Name: "settings", Version: "v1"},
	}}}
	unreadable := []inventory.UnreadableEntry{{Entry: inv.Inventory.Entries[1], Err: forbiddenRead("configmaps", "settings")}}

	var runErr error
	out := captureOutput(t, func() {
		runErr = showInstanceTree(context.Background(), client, inv, []*unstructured.Unstructured{web}, unreadable, 0, output.FormatTable, output.InstanceLogger("demo"))
	})
	require.NoError(t, runErr, out)
	assert.Contains(t, out, "could not read tracked resource")
	assert.Contains(t, out, "name=settings")
	assert.Contains(t, out, "forbidden")
	assert.Contains(t, out, "1 resource", "the tree shows the readable resource")
}

// With every tracked resource unreadable, tree prints the warnings and exits 5
// (no resources found), as for a record that tracks no live resources.
func TestShowInstanceTree_AllUnreadableExitsNotFound(t *testing.T) {
	client, _ := fakeClusterClient()
	inv := &inventory.Record{Name: "demo", Namespace: "apps", Inventory: inventory.Inventory{Entries: []k8sinventory.Entry{
		{Kind: "ConfigMap", Namespace: "apps", Name: "settings", Version: "v1"},
	}}}
	unreadable := []inventory.UnreadableEntry{{Entry: inv.Inventory.Entries[0], Err: forbiddenRead("configmaps", "settings")}}

	var runErr error
	out := captureOutput(t, func() {
		runErr = showInstanceTree(context.Background(), client, inv, nil, unreadable, 0, output.FormatTable, output.InstanceLogger("demo"))
	})
	requireExitCode(t, runErr, opmexit.ExitNotFound)
	assert.Contains(t, out, "could not read tracked resource")
	assert.Contains(t, out, "name=settings")
	assert.Contains(t, out, "no resources found")
}

// Status warns about each tracked resource it could not read, lists it as an
// Unknown row, and exits 2 because the instance is not ready.
func TestShowInstanceStatus_UnreadableIsUnknownAndExitsNotReady(t *testing.T) {
	web := trackedConfigMap("web")
	client, _ := fakeClusterClient(web.DeepCopy())
	inv := &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI, Inventory: inventory.Inventory{Entries: []k8sinventory.Entry{
		{Kind: "ConfigMap", Namespace: "apps", Name: "web", Version: "v1"},
		{Kind: "ConfigMap", Namespace: "apps", Name: "settings", Version: "v1"},
	}}}
	unreadable := []inventory.UnreadableEntry{{Entry: inv.Inventory.Entries[1], Err: forbiddenRead("configmaps", "settings")}}
	rsf := &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}

	var runErr error
	out := captureOutput(t, func() {
		runErr = showInstanceStatus(context.Background(), client, "apps", rsf, output.FormatTable, false,
			inv, []*unstructured.Unstructured{web}, nil, unreadable, "demo")
	})
	requireExitCode(t, runErr, opmexit.ExitValidationError)
	assert.Contains(t, out, "could not read tracked resource")
	assert.Contains(t, out, "name=settings")
	assert.Contains(t, out, "forbidden")
	assert.Regexp(t, `settings\s.*Unknown|Unknown.*\ssettings`, out, "the unreadable resource is an Unknown row")
}
