package instance

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	"github.com/open-platform-model/cli/internal/operator"
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

// operatorManaged turns the scenario's instance into an operator-managed one
// with the given spec.prune and spec.dataPolicy ("" leaves the field absent).
func (s *claimScenario) operatorManaged(t *testing.T, prune bool, dataPolicy string) {
	t.Helper()
	mi, err := s.fake.Tracker().Get(inventory.ModuleInstanceGVR, "apps", "demo")
	require.NoError(t, err)
	obj := mi.(*unstructured.Unstructured)
	require.NoError(t, unstructured.SetNestedField(obj.Object, inventory.OwnerOperator, "spec", "owner"))
	require.NoError(t, unstructured.SetNestedField(obj.Object, prune, "spec", "prune"))
	if dataPolicy != "" {
		require.NoError(t, unstructured.SetNestedField(obj.Object, dataPolicy, "spec", "dataPolicy"))
	}
	require.NoError(t, s.fake.Tracker().Update(inventory.ModuleInstanceGVR, obj, "apps"))
}

// moduleInstanceCRD is the ModuleInstance CRD of an operator release: with
// spec.dataPolicy in its schema, or as every release before the field.
func moduleInstanceCRD(withDataPolicy bool) *unstructured.Unstructured {
	props := map[string]any{"prune": map[string]any{"type": "boolean"}}
	if withDataPolicy {
		props["dataPolicy"] = map[string]any{"type": "string"}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1", "kind": "CustomResourceDefinition",
		"metadata": map[string]any{"name": inventory.CRDNameModuleInstances},
		"spec": map[string]any{"versions": []any{map[string]any{
			"name": "v1alpha1", "served": true, "storage": true,
			"schema": map[string]any{"openAPIV3Schema": map[string]any{
				"properties": map[string]any{"spec": map[string]any{"properties": props}},
			}},
		}}},
		"status": map[string]any{"conditions": []any{map[string]any{"type": "Established", "status": "True"}}},
	}}
}

// readyOperator is a running operator whose ModuleInstance CRD is crd; nil
// leaves the bare CRD of runningOperatorObjects, which has no schema.
func readyOperator(crd *unstructured.Unstructured) []runtime.Object {
	var objs []runtime.Object
	for _, o := range runningOperatorObjects() {
		if u := o.(*unstructured.Unstructured); crd == nil || u.GetName() != inventory.CRDNameModuleInstances {
			objs = append(objs, o)
		}
	}
	if crd != nil {
		objs = append(objs, crd)
	}
	return objs
}

// installOperator puts a ready operator into the scenario's cluster.
func (s *claimScenario) installOperator(t *testing.T, crd *unstructured.Unstructured) {
	t.Helper()
	for _, o := range readyOperator(crd) {
		require.NoError(t, s.fake.Tracker().Add(o))
	}
}

// The three operators a delete can meet, by what the ModuleInstance CRD says.
const (
	crdWithField    = "CRD has spec.dataPolicy"
	crdWithoutField = "CRD has no spec.dataPolicy"
	crdNoSchema     = "CRD has no readable schema"
)

func crdFor(kind string) *unstructured.Unstructured {
	switch kind {
	case crdWithField:
		return moduleInstanceCRD(true)
	case crdWithoutField:
		return moduleInstanceCRD(false)
	default:
		return nil
	}
}

const (
	promptOperatorDeletesClaims = "so the operator deletes its tracked resources, PersistentVolumeClaims and the data on them included."
	promptOperatorKeepsClaims   = "so the operator deletes its tracked resources and keeps PersistentVolumeClaims and the data on them."
	promptOperatorOrphans       = "spec.prune is not set, so the operator leaves its tracked resources running."
	promptOldOperator           = "spec.prune is set and the operator in this cluster has no spec.dataPolicy, " + promptOperatorDeletesClaims
)

// On an operator-managed instance the operator deletes, not opm, so the prompt
// says what the operator does, read from the cluster. It says that claims are
// kept only when the installed CRD has spec.dataPolicy and the instance's
// value is not Delete, and then with what an operator older than its CRDs
// does. An operator whose CRD has no such field deletes claims, and the prompt
// says so whatever the instance carries. The prompt is the same with and
// without --delete-data, and the note about the flag comes before it.
func TestConfirmAndDelete_OperatorManagedPromptSaysWhatTheOperatorDoes(t *testing.T) {
	tests := []struct {
		name       string
		crd        string
		prune      bool
		dataPolicy string
		want       []string
		notWant    []string
	}{
		{"no prune", crdWithField, false, "", []string{promptOperatorOrphans}, []string{"PersistentVolumeClaims"}},
		{"no prune, Delete", crdWithField, false, "Delete", []string{promptOperatorOrphans}, []string{"PersistentVolumeClaims"}},
		{"prune, Delete", crdWithField, true, "Delete",
			[]string{"spec.prune is set and spec.dataPolicy is Delete, " + promptOperatorDeletesClaims}, []string{keptClaimsHedge, "keeps"}},
		{"prune, Keep", crdWithField, true, "Keep",
			[]string{"spec.prune is set and spec.dataPolicy is Keep, " + promptOperatorKeepsClaims, keptClaimsHedge}, []string{promptOperatorDeletesClaims}},
		{"prune, absent", crdWithField, true, "",
			[]string{"spec.prune is set and spec.dataPolicy is not set, " + promptOperatorKeepsClaims, keptClaimsHedge}, []string{promptOperatorDeletesClaims}},
		{"prune, unknown value", crdWithField, true, "Retain",
			[]string{`spec.prune is set and spec.dataPolicy is "Retain", not a value opm knows, read as Keep, ` + promptOperatorKeepsClaims, keptClaimsHedge},
			[]string{promptOperatorDeletesClaims}},
		{"prune, lower-case delete is not Delete", crdWithField, true, "delete",
			[]string{`spec.dataPolicy is "delete"`, promptOperatorKeepsClaims}, []string{promptOperatorDeletesClaims}},
		{"operator without the field", crdWithoutField, true, "", []string{promptOldOperator}, []string{"keeps", "kept", keptClaimsHedge}},
		{"operator without the field, no prune", crdWithoutField, false, "", []string{promptOperatorOrphans}, []string{"PersistentVolumeClaims"}},
		{"CRD without a schema", crdNoSchema, true, "",
			[]string{"spec.prune is set and spec.dataPolicy is not set, so the operator deletes its tracked resources.\n", undecidedClaimsNote},
			[]string{promptOperatorKeepsClaims, promptOperatorDeletesClaims}},
		{"CRD without a schema, Delete", crdNoSchema, true, "Delete",
			[]string{"spec.prune is set and spec.dataPolicy is Delete, " + promptOperatorDeletesClaims}, []string{undecidedClaimsNote}},
	}
	for _, tt := range tests {
		for _, deleteData := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/deleteData=%v", tt.name, deleteData), func(t *testing.T) {
				s := newClaimScenario()
				s.operatorManaged(t, tt.prune, tt.dataPolicy)
				s.installOperator(t, crdFor(tt.crd))
				out, err := s.confirm(t, "demo", deleteFlags{DeleteData: deleteData}, "n\n")
				require.NoError(t, err, out)

				question := strings.Index(out, "[y/N]")
				require.GreaterOrEqual(t, question, 0, out)
				start := strings.Index(out, "This instance is operator-managed")
				require.GreaterOrEqual(t, start, 0, out)
				prompt := out[start:]
				for _, w := range tt.want {
					assert.Contains(t, prompt, w)
				}
				for _, w := range tt.notWant {
					assert.NotContains(t, prompt, w)
				}
				assert.NotContains(t, out, "will be deleted:", "the claim list is for CLI-owned instances")
				assert.NotContains(t, out, "PersistentVolumeClaims are kept)", "the CLI-owned prompt is not used")

				note := strings.Index(out, workflowapply.DeleteDataOperatorManagedNote)
				if deleteData {
					require.GreaterOrEqual(t, note, 0, out)
					assert.Less(t, note, question, "the note comes before the question")
					assert.Contains(t, lineWith(out, workflowapply.DeleteDataOperatorManagedNote), "WARN")
				} else {
					assert.Equal(t, -1, note, out)
				}

				assert.Contains(t, out, "deletion canceled")
				assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"), "a declined prompt deletes nothing")
				assert.True(t, s.exists(claimGVR, "data"))
			})
		}
	}
}

// An instance that tracks no claim gets no sentence about claims.
func TestOperatorManagedDeletePrompt_NoTrackedClaimSaysNothingAboutClaims(t *testing.T) {
	p := operatorManagedDeletePrompt("demo", "", "apps", true, operatorClaims{dataPolicy: "Keep", field: operator.FieldPresent})
	assert.Contains(t, p, "This instance is operator-managed: spec.prune is set, so the operator deletes its tracked resources.\n")
	assert.NotContains(t, p, "PersistentVolumeClaims")
	assert.NotContains(t, p, "spec.dataPolicy")
}

// A delete that the readiness gate refuses asks nothing: the refusal comes
// before the question, so nobody answers a question about their data for a
// delete that is not attempted. The answer waiting on standard input is not
// read, and nothing is deleted.
func TestConfirmAndDelete_OperatorManagedRefusalComesBeforeTheQuestion(t *testing.T) {
	notReady := readyOperator(moduleInstanceCRD(true))[1:] // one operator CRD is missing
	for name, objs := range map[string][]runtime.Object{"no operator": nil, "operator not ready": notReady} {
		t.Run(name, func(t *testing.T) {
			s := newClaimScenario()
			s.operatorManaged(t, true, "")
			for _, o := range objs {
				require.NoError(t, s.fake.Tracker().Add(o))
			}
			out, err := s.confirm(t, "demo", deleteFlags{}, "y\n")
			requireExitCode(t, err, opmexit.ExitValidationError)
			assert.Contains(t, err.Error(), "not ready")
			assert.NotContains(t, out, "[y/N]", "no question before a refusal")
			assert.NotContains(t, out, "This instance is operator-managed")
			assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
			assert.True(t, s.exists(claimGVR, "data"))
		})
	}
}

// The question comes after the gate and before the delete: a "yes" deletes the
// ModuleInstance, and the ModuleInstance CRD is read once for both the gate
// and the question.
func TestConfirmAndDelete_OperatorManagedAsksAfterTheGateAndReadsTheCRDOnce(t *testing.T) {
	s := newClaimScenario()
	s.operatorManaged(t, true, "")
	s.installOperator(t, moduleInstanceCRD(true))

	out, err := s.confirm(t, "demo", deleteFlags{Timeout: 5 * time.Second}, "y\n")
	require.NoError(t, err, out)
	assert.Contains(t, out, promptOperatorKeepsClaims)
	assert.False(t, s.exists(inventory.ModuleInstanceGVR, "demo"), "a yes deletes the ModuleInstance")

	reads, question := 0, -1
	for i, a := range s.fake.Actions() {
		get, ok := a.(k8stesting.GetAction)
		if ok && a.GetResource().Resource == "customresourcedefinitions" && get.GetName() == inventory.CRDNameModuleInstances {
			reads++
			question = i
		}
		if a.GetVerb() == "delete" {
			assert.Greater(t, i, question, "the gate reads before anything is deleted")
		}
	}
	assert.Equal(t, 1, reads, "one read of the ModuleInstance CRD")
}

// With --yes there is no prompt; the note about --delete-data still prints.
func TestConfirmAndDelete_OperatorManagedDeleteDataWarnsWithYes(t *testing.T) {
	s := newClaimScenario()
	s.operatorManaged(t, true, "")
	// The delete then stops at the operator-readiness guard of this cluster.
	out, err := s.confirm(t, "demo", deleteFlags{DeleteData: true, SkipConfirm: true}, "")
	require.Error(t, err)
	assert.Contains(t, out, workflowapply.DeleteDataOperatorManagedNote)
	assert.NotContains(t, out, "[y/N]")
	assert.True(t, s.exists(inventory.ModuleInstanceGVR, "demo"))
}

func trackedDataClaim() []k8sinventory.Entry {
	return []k8sinventory.Entry{{Kind: "PersistentVolumeClaim", Namespace: "apps", Name: "data"}}
}

func TestOperatorManagedDeletePrompt(t *testing.T) {
	p := operatorManagedDeletePrompt("", "abc-123", "media", true,
		operatorClaims{tracked: trackedDataClaim(), dataPolicy: "Delete", field: operator.FieldPresent})
	assert.Contains(t, p, `instance-id "abc-123" in namespace "media"`)
	assert.True(t, strings.HasSuffix(p, "[y/N]: "))
	assert.NotContains(t, p, "keeps PersistentVolumeClaims")
}

// The value comes from the cluster. One opm does not know is shown quoted, so
// a control character in it cannot rewrite the terminal line of the prompt.
func TestOperatorManagedDeletePrompt_UnknownPolicyIsQuoted(t *testing.T) {
	p := operatorManagedDeletePrompt("demo", "", "apps", true,
		operatorClaims{tracked: trackedDataClaim(), dataPolicy: "Delete\x1b[2K\rKeep", field: operator.FieldPresent})
	assert.NotContains(t, p, "\x1b")
	assert.NotContains(t, p, "\r")
	assert.Contains(t, p, `"Delete\x1b[2K\rKeep"`)
	assert.Contains(t, p, promptOperatorKeepsClaims)
}

// operatorClaimRecord is an operator-owned record with spec.prune set that
// tracks a Deployment and the claim apps/data.
func operatorClaimRecord(dataPolicy string) *inventory.Record {
	return &inventory.Record{
		Name: "demo", Namespace: "apps", Owner: inventory.OwnerOperator, Prune: true, DataPolicy: dataPolicy,
		Inventory: inventory.Inventory{Entries: []k8sinventory.Entry{
			{Group: "apps", Kind: "Deployment", Namespace: "apps", Name: "web"},
			{Kind: "PersistentVolumeClaim", Namespace: "apps", Name: "data"},
		}},
	}
}

// runOperatorOwnedDelete deletes rec, with the question answered yes, against
// a cluster with a ready operator whose ModuleInstance CRD has
// spec.dataPolicy or not.
func runOperatorOwnedDelete(t *testing.T, rec *inventory.Record, withDataPolicy, dryRun bool) string {
	t.Helper()
	client, _ := fakeClusterClient(append(readyOperator(moduleInstanceCRD(withDataPolicy)), moduleInstanceObj(rec.Namespace, rec.Name))...)
	var runErr error
	out := captureOutput(t, func() {
		runErr = deleteOperatorOwned(context.Background(), client, rec, confirmYes, 5*time.Second, dryRun, output.InstanceLogger(rec.Name))
	})
	require.NoError(t, runErr, out)
	return out
}

// The ModuleInstance being gone does not prove that claims are gone or left.
// When the operator may have kept a claim, the closing output says neither
// that everything was pruned nor that the claim is still there: it names the
// claim, the policy and what an operator older than its CRDs did, and prints
// the commands that show and delete the claim.
func TestDeleteOperatorOwned_KeptClaimsAreNamedAtTheEnd(t *testing.T) {
	for _, dataPolicy := range []string{"", "Keep", "Retain"} {
		t.Run("dataPolicy="+dataPolicy, func(t *testing.T) {
			out := runOperatorOwnedDelete(t, operatorClaimRecord(dataPolicy), true, false)

			assert.Contains(t, out, "PersistentVolumeClaims and the data on them kept ("+describeDataPolicy(dataPolicy)+") unless the operator is older than its CRDs")
			assert.Contains(t, out, "Instance deleted: the operator finished its cleanup")
			assert.NotContains(t, out, "operator pruned 2 resources")
			assert.NotContains(t, out, "all resources have been deleted")
			assert.NotContains(t, out, "keeps PersistentVolumeClaims", "nothing read proves that the claims are still there")
			assert.Contains(t, out, "The instance tracked 1 PersistentVolumeClaim(s). An operator that has spec.dataPolicy keeps them")
			assert.Contains(t, out, describeDataPolicy(dataPolicy))
			assert.Contains(t, out, "An operator older than its CRDs deleted them")
			assert.Contains(t, out, "kubectl get pvc -n apps")
			assert.Contains(t, out, "kubectl delete pvc data -n apps")
			assert.NotContains(t, out, "kubectl delete pvc web")
		})
	}
}

// The operator prunes everything the instance tracks when the policy is
// Delete, when its CRD has no spec.dataPolicy, and when no claim is tracked.
// The closing line then says so, as it did before the field existed.
func TestDeleteOperatorOwned_FullPruneIsReportedAsBefore(t *testing.T) {
	noClaim := operatorClaimRecord("")
	noClaim.Inventory.Entries = noClaim.Inventory.Entries[:1]
	tests := []struct {
		name           string
		rec            *inventory.Record
		withDataPolicy bool
		want           []string
	}{
		{"Delete", operatorClaimRecord("Delete"), true, []string{
			"operator pruned 2 resources", "PersistentVolumeClaims and the data on them included (spec.dataPolicy is Delete)",
		}},
		{"operator without the field", operatorClaimRecord(""), false, []string{
			"operator pruned 2 resources", "PersistentVolumeClaims and the data on them included (the operator in this cluster has no spec.dataPolicy)",
		}},
		{"no claim tracked", noClaim, true, []string{"operator pruned 1 resources"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runOperatorOwnedDelete(t, tt.rec, tt.withDataPolicy, false)
			for _, w := range tt.want {
				assert.Contains(t, out, w)
			}
			assert.NotContains(t, out, "kubectl delete pvc")
			assert.NotContains(t, out, "kept")
		})
	}
	assert.NotContains(t, runOperatorOwnedDelete(t, noClaim, true, false), "PersistentVolumeClaims")
}

// The dry run states the same outcome for claims as the prompt does.
func TestDeleteOperatorOwned_DryRunStatesTheClaimOutcome(t *testing.T) {
	kept := runOperatorOwnedDelete(t, operatorClaimRecord(""), true, true)
	assert.Contains(t, kept, "would prune its 2 tracked resource(s), PersistentVolumeClaims and the data on them kept (spec.dataPolicy is not set) unless the operator is older than its CRDs")

	deleted := runOperatorOwnedDelete(t, operatorClaimRecord("Delete"), true, true)
	assert.Contains(t, deleted, "would prune its 2 tracked resource(s), PersistentVolumeClaims and the data on them included (spec.dataPolicy is Delete)")

	old := runOperatorOwnedDelete(t, operatorClaimRecord("Keep"), false, true)
	assert.Contains(t, old, "would prune its 2 tracked resource(s), PersistentVolumeClaims and the data on them included (the operator in this cluster has no spec.dataPolicy)")
	assert.NotContains(t, old, "kept")

	orphaned := operatorClaimRecord("Delete")
	orphaned.Prune = false
	out := runOperatorOwnedDelete(t, orphaned, true, true)
	assert.Contains(t, out, "would be left running (spec.prune is not set)")
	assert.NotContains(t, out, "PersistentVolumeClaims")
}

// A CRD whose schema cannot be read decides nothing. The closing output then
// names both operators that deleted the claims and says why opm cannot tell.
func TestReportOperatorPrune_UndecidedSaysWhy(t *testing.T) {
	rec := operatorClaimRecord("")
	claims := operatorClaimsOf(rec, nil)
	require.Equal(t, claimsUndecided, claims.fate())
	out := captureOutput(t, func() { reportOperatorPrune(rec, claims, output.InstanceLogger(rec.Name)) })
	assert.Contains(t, out, "An operator without spec.dataPolicy, or older than its CRDs, deleted them")
	assert.Contains(t, out, "could not read")
	assert.Contains(t, out, "kubectl delete pvc data -n apps")
	assert.NotContains(t, out, "operator pruned 2 resources")
}

// Without spec.prune the operator leaves everything, so nothing is said
// about claims whatever the CRD and the policy say.
func TestOperatorClaimsOf_NoPruneTracksNothing(t *testing.T) {
	rec := operatorClaimRecord("Delete")
	rec.Prune = false
	assert.Equal(t, claimsNone, operatorClaimsOf(rec, moduleInstanceCRD(true)).fate())
	assert.Equal(t, claimsKept, operatorClaimsOf(operatorClaimRecord(""), moduleInstanceCRD(true)).fate())
	assert.Equal(t, claimsDeleted, operatorClaimsOf(operatorClaimRecord("Keep"), moduleInstanceCRD(false)).fate())
}

// A core-group claim only: a kind of the same name in another API group is
// not data the operator's policy covers.
func TestTrackedClaims_CoreGroupOnly(t *testing.T) {
	rec := operatorClaimRecord("")
	rec.Inventory.Entries = append(rec.Inventory.Entries,
		k8sinventory.Entry{Group: "example.com", Kind: "PersistentVolumeClaim", Namespace: "apps", Name: "other"})
	claims := trackedClaims(rec)
	require.Len(t, claims, 1)
	assert.Equal(t, "data", claims[0].Name)
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
