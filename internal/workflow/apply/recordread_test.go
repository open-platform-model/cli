package apply

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/library/opm/module"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// applyCluster is a fake cluster for Execute: the dynamic client holds the
// ModuleInstance CRD and the seeded objects, answers every server-side apply
// with the object named in the patch, and the clientset allows the status
// write the RBAC gate asks about.
type applyCluster struct {
	dyn    *dynamicfake.FakeDynamicClient
	client *kubernetes.Client
}

func newApplyCluster(objs ...*unstructured.Unstructured) *applyCluster {
	runtimeObjs := make([]runtime.Object, 0, 1+len(objs))
	runtimeObjs = append(runtimeObjs, makeModuleInstanceCRD(true, true))
	for _, o := range objs {
		runtimeObjs = append(runtimeObjs, o)
	}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{
			inventory.ModuleInstanceGVR: "ModuleInstanceList",
			inventory.PlatformGVR:       "PlatformList",
			crdGVR:                      "CustomResourceDefinitionList",
			configMapGVR:                "ConfigMapList",
		}, runtimeObjs...)
	// The plain tracker cannot serve server-side apply.
	dyn.PrependReactor("patch", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, renderedConfigMap(action.(k8stesting.PatchAction).GetName()), nil
	})
	dyn.PrependReactor("patch", "moduleinstances", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, cliOwnedInstance(action.(k8stesting.PatchAction).GetName(), action.GetNamespace()), nil
	})

	clientset := k8sfake.NewClientset()
	clientset.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{Status: authorizationv1.SubjectAccessReviewStatus{Allowed: true}}, nil
	})
	return &applyCluster{dyn: dyn, client: &kubernetes.Client{Dynamic: dyn, Clientset: clientset}}
}

// request is an apply of the instance "demo" in "default" rendering the named
// ConfigMaps.
func (c *applyCluster) request(opts Options, configMaps ...string) Request {
	resources := make([]*unstructured.Unstructured, 0, len(configMaps))
	for _, name := range configMaps {
		resources = append(resources, renderedConfigMap(name))
	}
	opts.SuccessAppliedMessage = "applied"
	opts.SuccessUpToDateMessage = "up to date"
	return Request{
		Result: &workflowrender.Result{
			Resources: resources,
			Instance:  module.InstanceMetadata{Name: "demo", Namespace: "default", UUID: "uuid-1"},
		},
		K8sClient: c.client,
		Log:       output.InstanceLogger("demo"),
		Options:   opts,
	}
}

// writes lists every mutating call the dynamic client received, as
// "<verb> <resource>[/<subresource>] <name>".
func (c *applyCluster) writes() []string {
	var out []string
	for _, a := range c.dyn.Actions() {
		res := a.GetResource().Resource
		if a.GetSubresource() != "" {
			res += "/" + a.GetSubresource()
		}
		switch a.GetVerb() {
		case "patch":
			out = append(out, "patch "+res+" "+a.(k8stesting.PatchAction).GetName())
		case "delete":
			out = append(out, "delete "+res+" "+a.(k8stesting.DeleteAction).GetName())
		case "create", "update", "delete-collection":
			out = append(out, a.GetVerb()+" "+res)
		}
	}
	return out
}

// writtenInventory is the names of the inventory entries of the last status
// write, and whether a status write was issued at all.
func (c *applyCluster) writtenInventory(t *testing.T) (names []string, written bool) {
	t.Helper()
	for _, a := range c.dyn.Actions() {
		patch, ok := a.(k8stesting.PatchAction)
		if !ok || a.GetResource().Resource != "moduleinstances" || a.GetSubresource() != "status" {
			continue
		}
		var body struct {
			Status struct {
				Inventory struct {
					Count   int `json:"count"`
					Entries []struct {
						Name string `json:"name"`
					} `json:"entries"`
				} `json:"inventory"`
			} `json:"status"`
		}
		require.NoError(t, json.Unmarshal(patch.GetPatch(), &body))
		names = names[:0]
		for _, e := range body.Status.Inventory.Entries {
			names = append(names, e.Name)
		}
		assert.Equal(t, len(names), body.Status.Inventory.Count, "the recorded count matches the entries")
		written = true
	}
	return names, written
}

// captureLog routes the log stream into a buffer for the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

// requireExitCode asserts err is an ExitError with the given code.
func requireExitCode(t *testing.T, err error, code int) {
	t.Helper()
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr), "the error carries an exit code: %v", err)
	assert.Equal(t, code, exitErr.Code)
}

var moduleInstanceGR = schema.GroupResource{Group: inventory.GroupOpmodel, Resource: "moduleinstances"}

// A ModuleInstance read that fails with anything but NotFound stops the
// apply, real or dry run: nothing is applied, nothing is written, and the
// exit code follows the cause. The cluster holds an operator-owned record,
// the case where going on as a first install would take the instance over.
func TestExecute_UnreadableRecordStopsTheApply(t *testing.T) {
	causes := []struct {
		name string
		err  error
		code int
	}{
		{"internal error", apierrors.NewInternalError(errors.New("etcd leader changed")), opmexit.ExitGeneralError},
		{"forbidden", apierrors.NewForbidden(moduleInstanceGR, "demo", errors.New("no access")), opmexit.ExitPermissionDenied},
		{"unavailable", apierrors.NewServiceUnavailable("apiserver is shutting down"), opmexit.ExitConnectivityError},
	}
	for _, cause := range causes {
		for _, dryRun := range []bool{false, true} {
			name := cause.name
			if dryRun {
				name += " dry run"
			}
			t.Run(name, func(t *testing.T) {
				withReleasedCLIVersion(t)
				logBuf := captureLog(t)

				operatorOwned := cliOwnedInstance("demo", "default", "app")
				require.NoError(t, unstructured.SetNestedField(operatorOwned.Object, inventory.OwnerOperator, "spec", "owner"))
				cluster := newApplyCluster(operatorOwned)
				cluster.dyn.PrependReactor("get", "moduleinstances", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, cause.err
				})

				err := Execute(context.Background(), cluster.request(Options{DryRun: dryRun}, "app"))

				requireExitCode(t, err, cause.code)
				assert.Contains(t, err.Error(), `"demo"`, "the error names the instance")
				assert.Contains(t, err.Error(), `"default"`, "the error names the namespace")
				assert.ErrorIs(t, err, cause.err, "the cause stays in the chain")
				assert.Empty(t, cluster.writes(), "nothing is applied and no record is written")
				assert.NotContains(t, logBuf.String(), "dry run complete", "no preview is printed without the record")
				assert.NotContains(t, logBuf.String(), "applying", "the apply never starts")
			})
		}
	}
}

// A NotFound answer still means no record: the apply runs as a first install
// and writes the record at revision 1.
func TestExecute_MissingRecordIsAFirstInstall(t *testing.T) {
	withReleasedCLIVersion(t)
	captureLog(t)
	cluster := newApplyCluster()

	require.NoError(t, Execute(context.Background(), cluster.request(Options{}, "app")))

	names, written := cluster.writtenInventory(t)
	require.True(t, written, "the record is written")
	assert.Equal(t, []string{"app"}, names)
}
