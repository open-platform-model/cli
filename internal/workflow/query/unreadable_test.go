package query

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/open-platform-model/library/opm/k8s/health"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/cmdutil"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

// captureLog redirects the package logger to a buffer for the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

func forbiddenEntry(kind, namespace, name string) inventory.UnreadableEntry {
	return inventory.UnreadableEntry{
		Entry: k8sinventory.Entry{Kind: kind, Namespace: namespace, Name: name, Version: "v1"},
		Err:   apierrors.NewForbidden(schema.GroupResource{Resource: strings.ToLower(kind) + "s"}, name, nil),
	}
}

func TestWarnUnreadable_OneLinePerEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf)
	WarnUnreadable(logger, []inventory.UnreadableEntry{
		forbiddenEntry("ConfigMap", "apps", "settings"),
		forbiddenEntry("Secret", "apps", "creds"),
	})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "could not read tracked resource")
	assert.Contains(t, lines[0], "kind=ConfigMap")
	assert.Contains(t, lines[0], "namespace=apps")
	assert.Contains(t, lines[0], "name=settings")
	assert.Contains(t, lines[0], "forbidden")
	assert.Contains(t, lines[1], "name=creds")

	buf.Reset()
	WarnUnreadable(logger, nil)
	assert.Empty(t, buf.String())
}

// A status with an unreadable resource prints the Unknown row and exits 2.
func TestPrintInstanceStatus_UnreadableExitsNotReady(t *testing.T) {
	inv := &inventory.Record{Name: "demo", Namespace: "apps"}
	opts := BuildStatusOptions("apps", &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}, output.FormatTable, false, inv, nil, nil,
		[]inventory.UnreadableEntry{forbiddenEntry("ConfigMap", "apps", "settings")})
	require.Len(t, opts.UnreadableResources, 1)
	assert.Equal(t, "settings", opts.UnreadableResources[0].Name)

	err := PrintInstanceStatus(context.Background(), nil, opts, "demo")
	var exitErr *opmexit.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
}

// list counts an unreadable resource toward the total, not toward ready, and
// prints one warning for the instance.
func TestEvaluateInstanceHealth_UnreadableCountsNotReady(t *testing.T) {
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "settings", "namespace": "apps"},
	}}
	sa := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ServiceAccount",
		"metadata": map[string]any{"name": "runner", "namespace": "apps"},
	}}
	client := makeCRClient(cm, sa)
	forbidConfigMapGets(t, client)
	inv := &inventory.Record{Name: "demo", Namespace: "apps", Inventory: inventory.Inventory{Entries: []k8sinventory.Entry{
		{Kind: "ConfigMap", Namespace: "apps", Name: "settings", Version: "v1"},
		{Kind: "ServiceAccount", Namespace: "apps", Name: "runner", Version: "v1"},
	}}}
	buf := captureLog(t)

	summaries := EvaluateInstanceHealth(context.Background(), client, []*inventory.Record{inv}, 1, false)
	require.Len(t, summaries, 1)
	assert.Equal(t, string(health.NotReady), summaries[0].Status)
	assert.Equal(t, 1, summaries[0].ReadyCount)
	assert.Equal(t, 2, summaries[0].TotalCount)

	logged := strings.TrimSpace(buf.String())
	assert.Equal(t, 1, strings.Count(logged, "\n")+1, "one warning line: %s", logged)
	assert.Contains(t, logged, `instance "demo" in "apps"`)
	assert.Contains(t, logged, "could not read 1 tracked resource(s)")
	assert.Contains(t, logged, "opm instance status")
}
