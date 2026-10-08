package apply

import (
	"context"
	"errors"
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// The tracked object is deleted and the delete of the ModuleInstance record
// then fails: DeleteRecorded returns a RecordDeleteError that names the
// record and keeps the cause reachable, so no caller can report success.
func TestDeleteRecorded_FailedRecordDeleteIsAnError(t *testing.T) {
	captureLog(t)

	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "web", "namespace": "apps", "labels": map[string]any{
			opmlabels.ManagedBy: opmlabels.ManagedByCLI,
		}},
	}}
	mi := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": inventory.APIVersionModuleInstance, "kind": inventory.KindModuleInstance,
		"metadata": map[string]any{"name": "demo", "namespace": "apps"},
	}}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{inventory.ModuleInstanceGVR: "ModuleInstanceList"},
		cm.DeepCopy(), mi)
	dyn.PrependReactor("delete", inventory.ModuleInstanceGVR.Resource, func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(moduleInstanceGR, "demo", errors.New("no delete access"))
	})

	result, err := DeleteRecorded(context.Background(), DeleteRequest{
		Client:       &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn},
		InstanceName: "demo",
		Namespace:    "apps",
		Record:       &inventory.Record{Name: "demo", Namespace: "apps", Owner: inventory.OwnerCLI},
		Live:         []*unstructured.Unstructured{cm.DeepCopy()},
		Log:          output.InstanceLogger("demo"),
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var recordErr *RecordDeleteError
	require.ErrorAs(t, err, &recordErr)
	assert.Equal(t, "apps", recordErr.Namespace)
	assert.Equal(t, "demo", recordErr.Name)
	assert.True(t, apierrors.IsForbidden(err), "the cause is reachable through unwrapping")
	assert.Contains(t, err.Error(), "apps/demo")
	assert.Contains(t, err.Error(), "the record remains")

	_, cmErr := dyn.Tracker().Get(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "apps", "web")
	assert.True(t, apierrors.IsNotFound(cmErr), "the tracked ConfigMap is deleted")
	_, miErr := dyn.Tracker().Get(inventory.ModuleInstanceGVR, "apps", "demo")
	assert.NoError(t, miErr, "the record is still there")
}
