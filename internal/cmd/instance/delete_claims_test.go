package instance

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
)

var (
	claimGVR     = schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
	configMapGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
)

// claimScenario is a CLI-owned instance "demo" in "apps" whose inventory
// tracks a ConfigMap and a PersistentVolumeClaim, both live and both carrying
// the instance's identity.
type claimScenario struct {
	fake   *fakedynamic.FakeDynamicClient
	client *kubernetes.Client
	inv    *inventory.Record
	live   []*unstructured.Unstructured
}

func newClaimScenario() *claimScenario {
	const uuid = "uuid-demo"
	labels := map[string]any{opmlabels.ManagedBy: opmlabels.ManagedByCLI, opmlabels.ModuleInstanceUUID: uuid}
	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "web", "namespace": "apps", "labels": labels},
	}}
	claim := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "PersistentVolumeClaim",
		"metadata": map[string]any{"name": "data", "namespace": "apps", "labels": labels},
	}}
	mi := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": "demo", "namespace": "apps"},
		"spec":     map[string]any{"owner": inventory.OwnerCLI},
		"status": map[string]any{"instanceUUID": uuid, "inventory": map[string]any{"revision": int64(1), "entries": []any{
			map[string]any{"group": "", "kind": "ConfigMap", "namespace": "apps", "name": "web", "v": "v1", "component": "app"},
			map[string]any{"group": "", "kind": "PersistentVolumeClaim", "namespace": "apps", "name": "data", "v": "v1", "component": "app"},
		}}},
	}}
	fake := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"},
		cm.DeepCopy(), claim.DeepCopy(), mi.DeepCopy())
	return &claimScenario{
		fake:   fake,
		client: &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fake},
		inv:    &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI, InstanceUUID: uuid},
		live:   []*unstructured.Unstructured{cm, claim},
	}
}

func (s *claimScenario) run(t *testing.T, dryRun, deleteData bool) (string, error) {
	t.Helper()
	var runErr error
	out := captureOutput(t, func() {
		runErr = executeInstanceDelete(context.Background(), s.client, &cmdutil.InstanceSelectorFlags{InstanceName: "demo"}, "apps",
			s.inv, s.live, nil, dryRun, deleteData, output.InstanceLogger("demo"))
	})
	return out, runErr
}

func (s *claimScenario) exists(gvr schema.GroupVersionResource, name string) bool {
	_, err := s.fake.Tracker().Get(gvr, "apps", name)
	return !apierrors.IsNotFound(err)
}

// lineWith returns the first output line that holds every part.
func lineWith(out string, parts ...string) string {
	for _, line := range strings.Split(out, "\n") {
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(line, p)
		}
		if ok {
			return line
		}
	}
	return ""
}

// By default delete keeps the claim: the ConfigMap and the ModuleInstance go,
// the claim stays, the command passes, and the output lists the claim as kept
// at the informational level and prints the command that deletes it.
func TestExecuteInstanceDelete_KeepsClaimsByDefault(t *testing.T) {
	s := newClaimScenario()
	out, err := s.run(t, false, false)
	require.NoError(t, err, out)

	assert.True(t, s.exists(claimGVR, "data"), "the claim is kept")
	assert.False(t, s.exists(configMapGVR, "web"), "the ConfigMap is deleted")
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"), "the ModuleInstance is deleted: the kept claim is left untracked")

	keptLine := lineWith(out, "PersistentVolumeClaim/apps/data", output.StatusKept)
	require.NotEmpty(t, keptLine, "the claim is listed as kept:\n%s", out)
	assert.Contains(t, keptLine, "INFO", "a kept claim is an informational line")
	assert.NotContains(t, out, "WARN", "keeping a claim prints no warning")
	assert.NotContains(t, out, "ERRO", "keeping a claim prints no error")

	assert.Contains(t, out, "Instance deleted")
	assert.NotContains(t, out, "all resources have been deleted")
	assert.Contains(t, out, "Kept 1 PersistentVolumeClaim(s)")
	assert.Contains(t, out, "kubectl delete pvc data -n apps")
	assert.Contains(t, out, "--delete-data")
}

// With --delete-data the claim is deleted like every other tracked resource.
func TestExecuteInstanceDelete_DeleteDataDeletesClaims(t *testing.T) {
	s := newClaimScenario()
	out, err := s.run(t, false, true)
	require.NoError(t, err, out)

	assert.False(t, s.exists(claimGVR, "data"), "the claim is deleted")
	assert.False(t, s.exists(configMapGVR, "web"))
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
	assert.NotEmpty(t, lineWith(out, "PersistentVolumeClaim/apps/data", output.StatusDeleted), out)
	assert.Empty(t, lineWith(out, "PersistentVolumeClaim/apps/data", output.StatusKept), out)
	assert.NotContains(t, out, "kubectl delete pvc")
	assert.Contains(t, out, "all resources have been deleted")
}

// A dry run lists the claim as kept and says how many a real run would keep;
// with --delete-data it counts the claim among the resources to delete.
func TestExecuteInstanceDelete_DryRunReportsKeptClaims(t *testing.T) {
	for _, deleteData := range []bool{false, true} {
		t.Run(fmt.Sprintf("deleteData=%v", deleteData), func(t *testing.T) {
			s := newClaimScenario()
			out, err := s.run(t, true, deleteData)
			require.NoError(t, err, out)

			assert.True(t, s.exists(claimGVR, "data"), "a dry run deletes nothing")
			assert.True(t, s.exists(configMapGVR, "web"), "a dry run deletes nothing")
			assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"), "a dry run keeps the ModuleInstance")
			assert.NotContains(t, out, "WARN")

			if deleteData {
				assert.Contains(t, out, "dry run complete: 2 resources would be deleted")
				assert.NotContains(t, out, "would be kept")
				return
			}
			assert.NotEmpty(t, lineWith(out, "PersistentVolumeClaim/apps/data", output.StatusKept), out)
			assert.Contains(t, out, "dry run complete: 1 resources would be deleted")
			assert.Contains(t, out, "1 PersistentVolumeClaim(s) would be kept")
			assert.Contains(t, out, "--delete-data")
		})
	}
}

// The prompt names every claim --delete-data deletes and says the data goes;
// without the flag it says claims are kept and names none.
func TestDeletePrompt(t *testing.T) {
	with := deletePrompt("jellyfin", "", "media", []string{"media/config", "media/library"})
	assert.Contains(t, with, "media/config")
	assert.Contains(t, with, "media/library")
	assert.Contains(t, with, "the data on them will be deleted")
	assert.Contains(t, with, `instance "jellyfin" in namespace "media"`)
	assert.True(t, strings.HasSuffix(with, "[y/N]: "), "the question comes last")
	assert.NotContains(t, with, "are kept")

	without := deletePrompt("jellyfin", "", "media", nil)
	assert.Contains(t, without, "PersistentVolumeClaims are kept")
	assert.NotContains(t, without, "will be deleted")
	assert.Equal(t, 0, strings.Count(without, "\n"), "one line when no claim is deleted")

	byID := deletePrompt("", "abc-123", "media", nil)
	assert.Contains(t, byID, `instance-id "abc-123"`)
}

// claimsToDelete feeds the prompt: the live tracked claims of a CLI-owned
// instance with --delete-data, and nothing in every other case.
func TestClaimsToDelete(t *testing.T) {
	s := newClaimScenario()
	otherGroup := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "example.io/v1", "kind": "PersistentVolumeClaim",
		"metadata": map[string]any{"name": "lookalike", "namespace": "apps"},
	}}
	live := append([]*unstructured.Unstructured{otherGroup}, s.live...)

	assert.Equal(t, []string{"apps/data"}, claimsToDelete(s.inv, live, true))
	assert.Empty(t, claimsToDelete(s.inv, live, false), "without the flag no claim is deleted")
	assert.Empty(t, claimsToDelete(operatorOwnedRecord(), live, true), "the operator decides for an operator-managed instance")
}

func TestReadConfirmation(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"y\n", true}, {"YES\n", true}, {" yes \n", true},
		{"n\n", false}, {"\n", false}, {"", false}, {"yep\n", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, readConfirmation(strings.NewReader(tt.in)), "input %q", tt.in)
	}
}

// --delete-data reaches the delete, is off by default, and is not what --yes
// or the deprecated --force set.
func TestInstanceDeleteCmd_DeleteDataFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"default keeps data", []string{"jellyfin"}, false},
		{"--yes keeps data", []string{"jellyfin", "--yes"}, false},
		{"deprecated --force keeps data", []string{"jellyfin", "--force"}, false},
		{"--delete-data", []string{"jellyfin", "--delete-data"}, true},
		{"--delete-data --yes", []string{"jellyfin", "--delete-data", "--yes"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got deleteFlags
			orig := runDelete
			t.Cleanup(func() { runDelete = orig })
			runDelete = func(_ context.Context, _ string, _ *config.GlobalConfig, _ *cmdutil.K8sFlags, _ string, flags deleteFlags) error {
				got = flags
				return nil
			}
			cmd := NewInstanceDeleteCmd(&config.GlobalConfig{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(tt.args)
			require.NoError(t, cmd.Execute())
			assert.Equal(t, tt.want, got.DeleteData)
		})
	}

	flag := NewInstanceDeleteCmd(&config.GlobalConfig{}).Flags().Lookup("delete-data")
	require.NotNil(t, flag)
	assert.Equal(t, "false", flag.DefValue)
	assert.Empty(t, flag.Shorthand)
	assert.Empty(t, flag.Deprecated)
}

// On an operator-managed instance --delete-data changes nothing, and the
// command says so before it goes on.
func TestDeleteResolvedInstance_DeleteDataOnOperatorManagedWarns(t *testing.T) {
	rec := operatorOwnedRecord()
	run := func(deleteData bool) string {
		return captureOutput(t, func() {
			// The delete itself stops at the operator-readiness guard on this
			// empty cluster; the note is printed before it.
			_ = deleteResolvedInstance(context.Background(), emptyClusterClient(), &cmdutil.InstanceSelectorFlags{InstanceName: rec.Name}, rec.Namespace,
				rec, nil, nil, 0, false, deleteData, output.InstanceLogger("demo"))
		})
	}
	assert.Contains(t, run(true), workflowapply.DeleteDataOperatorManagedNote)
	assert.NotContains(t, run(false), "--delete-data")
}

// confirm runs the whole read, prompt and delete step against the scenario's
// cluster, with answer as standard input.
func (s *claimScenario) confirm(t *testing.T, instance string, flags deleteFlags, answer string) (string, error) {
	t.Helper()
	var runErr error
	out := captureOutput(t, func() {
		runErr = confirmAndDelete(context.Background(), s.client, &cmdutil.InstanceSelectorFlags{InstanceName: instance}, "apps",
			flags, strings.NewReader(answer), output.InstanceLogger(instance))
	})
	return out, runErr
}

// With --delete-data the prompt names the claims read from the instance's
// record before anything is deleted, and a "no" deletes nothing.
func TestConfirmAndDelete_PromptNamesTheClaimsDeleteDataDeletes(t *testing.T) {
	s := newClaimScenario()
	out, err := s.confirm(t, "demo", deleteFlags{DeleteData: true}, "n\n")
	require.NoError(t, err, out)

	assert.Contains(t, out, "these PersistentVolumeClaims and the data on them will be deleted")
	assert.Contains(t, out, "  apps/data\n")
	assert.Contains(t, out, "deletion canceled")
	assert.True(t, s.exists(claimGVR, "data"), "a declined prompt deletes nothing")
	assert.True(t, s.exists(configMapGVR, "web"), "a declined prompt deletes nothing")
	assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"))

	out, err = s.confirm(t, "demo", deleteFlags{DeleteData: true}, "y\n")
	require.NoError(t, err, out)
	assert.False(t, s.exists(claimGVR, "data"), "a confirmed --delete-data deletes the claim")
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
}

// Without --delete-data the prompt says claims are kept, and a "yes" keeps
// them. --yes skips the prompt and keeps them too.
func TestConfirmAndDelete_WithoutDeleteDataKeepsClaims(t *testing.T) {
	for name, tc := range map[string]struct {
		flags      deleteFlags
		wantPrompt bool
	}{
		"confirmed at the prompt": {deleteFlags{}, true},
		"--yes":                   {deleteFlags{SkipConfirm: true}, false},
	} {
		t.Run(name, func(t *testing.T) {
			s := newClaimScenario()
			out, err := s.confirm(t, "demo", tc.flags, "y\n")
			require.NoError(t, err, out)

			assert.Equal(t, tc.wantPrompt, strings.Contains(out, "[y/N]"), out)
			if tc.wantPrompt {
				assert.Contains(t, out, "PersistentVolumeClaims are kept")
			}
			assert.NotContains(t, out, "will be deleted")
			assert.True(t, s.exists(claimGVR, "data"), "the claim is kept")
			assert.False(t, s.exists(configMapGVR, "web"))
			assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
		})
	}
}

// An instance with no record is reported as not found (exit 5) before any
// prompt: the user is not asked to confirm a delete of nothing.
func TestConfirmAndDelete_MissingInstanceIsReportedBeforeThePrompt(t *testing.T) {
	s := newClaimScenario()
	out, err := s.confirm(t, "nosuch", deleteFlags{}, "y\n")
	requireExitCode(t, err, opmexit.ExitNotFound)
	assert.NotContains(t, out, "[y/N]", "no prompt for an instance that does not exist")
}
