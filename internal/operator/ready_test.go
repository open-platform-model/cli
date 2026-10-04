package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// namedCRDFixture is an Established CRD of the given name.
func namedCRDFixture(name string) *unstructured.Unstructured {
	obj := crdFixture(true)
	obj.SetName(name)
	return obj
}

// readyOperatorObjects is a running operator at its fixed names, minus any
// object named in skip.
func readyOperatorObjects(skip ...string) []*unstructured.Unstructured {
	skipped := map[string]bool{}
	for _, s := range skip {
		skipped[s] = true
	}
	var objs []*unstructured.Unstructured
	for _, name := range CRDNames() {
		if !skipped[name] {
			objs = append(objs, namedCRDFixture(name))
		}
	}
	if !skipped[ControllerDeploymentName] {
		objs = append(objs, deploymentFixture(true))
	}
	return objs
}

func TestCheckReady_FixedNamesReadyOperatorPasses(t *testing.T) {
	client := fakeClientWith(readyOperatorObjects()...)

	assert.NoError(t, CheckReady(context.Background(), client))
}

func TestCheckReady_MissingDeploymentIsNotReady(t *testing.T) {
	client := fakeClientWith(readyOperatorObjects(ControllerDeploymentName)...)

	err := CheckReady(context.Background(), client)
	var notReady *NotReadyError
	require.ErrorAs(t, err, &notReady)
	require.Len(t, notReady.Pending, 1)
	assert.Contains(t, notReady.Pending[0], ControllerDeploymentName)
	assert.Contains(t, notReady.Pending[0], OperatorNamespace)
}

// An operator older than v1.0.0-alpha.18 serves no TransformerRegistration
// CRD; it is reported as not ready, which points the user at install.
func TestCheckReady_MissingFourthCRDIsNotReady(t *testing.T) {
	client := fakeClientWith(readyOperatorObjects("transformerregistrations.opmodel.dev")...)

	err := CheckReady(context.Background(), client)
	var notReady *NotReadyError
	require.ErrorAs(t, err, &notReady)
	require.Len(t, notReady.Pending, 1)
	assert.Contains(t, notReady.Pending[0], "transformerregistrations.opmodel.dev")
	assert.Contains(t, err.Error(), "opm operator install")
}

// recordReads records every call CheckReady makes against client, as
// "<verb> <resource> <namespace>/<name>" for gets.
func recordReads(t *testing.T, client *kubernetes.Client) *[]string {
	t.Helper()
	fake, ok := client.Dynamic.(interface {
		PrependReactor(verb, resource string, reaction k8stesting.ReactionFunc)
	})
	require.True(t, ok)

	reads := &[]string{}
	fake.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		target := action.GetVerb() + " " + action.GetResource().Resource
		if get, isGet := action.(k8stesting.GetAction); isGet {
			target += " " + get.GetNamespace() + "/" + get.GetName()
		}
		*reads = append(*reads, target)
		return false, nil, nil
	})
	return reads
}

// fixedNameReads is what CheckReady reads: the four CRDs and the controller
// Deployment, nothing else.
func fixedNameReads() []string {
	want := make([]string, 0, len(CRDNames())+1)
	for _, name := range CRDNames() {
		want = append(want, "get customresourcedefinitions /"+name)
	}
	return append(want, "get deployments "+OperatorNamespace+"/"+ControllerDeploymentName)
}

// The check reads exactly the four CRDs and the controller Deployment: no
// instance record, no Namespace.
func TestCheckReady_ReadsOnlyTheFixedNames(t *testing.T) {
	client := fakeClientWith(readyOperatorObjects()...)
	reads := recordReads(t, client)

	require.NoError(t, CheckReady(context.Background(), client))
	assert.ElementsMatch(t, fixedNameReads(), *reads)
}

// An operator installed as the opm-operator module instance keeps the same
// fixed names, so the check finds it the same way and never reads the
// ModuleInstance that deployed it.
func TestCheckReady_ModuleInstalledOperatorFoundByTheSameNames(t *testing.T) {
	objs := append(readyOperatorObjects(), moduleInstanceFixture(OperatorNamespace, OperatorInstanceName))
	client := fakeClientWith(objs...)
	reads := recordReads(t, client)

	require.NoError(t, CheckReady(context.Background(), client))
	assert.ElementsMatch(t, fixedNameReads(), *reads)
}

func TestCheckReady_AbsentOperatorIsNotReady(t *testing.T) {
	// An empty cluster: none of the fixed names exists.
	client := fakeClientWith()

	err := CheckReady(context.Background(), client)
	require.Error(t, err)

	var notReady *NotReadyError
	require.ErrorAs(t, err, &notReady)
	assert.Len(t, notReady.Pending, len(CRDNames())+1)
	assert.Contains(t, err.Error(), "opm operator install")
}

func TestNotReadyError_IncludesTheCallerHint(t *testing.T) {
	err := &NotReadyError{
		Pending: []string{"Deployment/opm-operator-controller-manager in opm-operator-system"},
		Hint:    "deleting now would wedge the instance",
	}

	assert.Contains(t, err.Error(), "not ready")
	assert.Contains(t, err.Error(), "opm-operator-controller-manager")
	assert.Contains(t, err.Error(), "deleting now would wedge the instance")
	assert.Contains(t, err.Error(), "opm operator install")
}
