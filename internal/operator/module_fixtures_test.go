package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/modref"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// testInstanceUUID is the instance UUID the fixture render stamps.
const testInstanceUUID = "015ffc0e-04da-5dfe-9b9e-e278cd603c03"

var crdGVR = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}

// renderOpts shapes a fixture render of the operator module.
type renderOpts struct {
	imageTag     string // controller image tag; "" is "v1.0.0-beta.7"
	noCRDFloor   bool   // the ModuleInstance CRD lacks spec.owner
	extraRole    string // an extra ClusterRole the render carries
	moduleVer    string // module.opmodel.dev/version; "" is "0.1.0"
	noDeployment bool
}

// labeled stamps the identity labels the kernel puts on every object.
func labeled(obj *unstructured.Unstructured, component, moduleVersion string) *unstructured.Unstructured {
	obj.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by":     "opm-cli",
		"app.kubernetes.io/name":           component,
		"component.opmodel.dev/name":       component,
		"module-instance.opmodel.dev/name": OperatorInstanceName,
		"module-instance.opmodel.dev/uuid": testInstanceUUID,
		"module.opmodel.dev/name":          "opm_operator",
		"module.opmodel.dev/version":       moduleVersion,
	})
	return obj
}

func renderedCRDObject(name string, floor bool) *unstructured.Unstructured {
	specProps := map[string]any{"module": map[string]any{"type": "object"}}
	if floor {
		specProps["owner"] = map[string]any{"type": "string"}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]any{"name": name},
		"spec": map[string]any{
			"group": "opmodel.dev",
			"versions": []any{map[string]any{
				"name": "v1alpha1", "served": true, "storage": true,
				"schema": map[string]any{"openAPIV3Schema": map[string]any{"properties": map[string]any{
					"spec":   map[string]any{"properties": specProps},
					"status": map[string]any{"properties": map[string]any{"inventory": map[string]any{"type": "object"}}},
				}}},
			}},
		},
	}}
}

// moduleObjects is a fixture render of the operator module: the four CRDs,
// the Namespace, a bound ClusterRole, the ServiceAccount and the controller
// Deployment, each carrying the instance identity.
func moduleObjects(o renderOpts) []*unstructured.Unstructured {
	tag := o.imageTag
	if tag == "" {
		tag = "v1.0.0-beta.7"
	}
	mv := o.moduleVer
	if mv == "" {
		mv = "0.1.0"
	}
	var objs []*unstructured.Unstructured
	for i, name := range CRDNames() {
		objs = append(objs, labeled(renderedCRDObject(name, i != 0 || !o.noCRDFloor), "crds", mv))
	}
	objs = append(objs,
		labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "Namespace", "metadata": map[string]any{"name": OperatorNamespace},
		}}, "namespace", mv),
		labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole",
			"metadata": map[string]any{"name": "opm-operator-manager-role"},
			"rules":    []any{},
		}}, "manager-rbac", mv),
		labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "ServiceAccount",
			"metadata": map[string]any{"name": ControllerDeploymentName, "namespace": OperatorNamespace},
		}}, "controller-manager", mv),
	)
	if o.extraRole != "" {
		objs = append(objs, labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole",
			"metadata": map[string]any{"name": o.extraRole},
			"rules":    []any{},
		}}, "admin-roles", mv))
	}
	if !o.noDeployment {
		objs = append(objs, labeled(&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]any{"name": ControllerDeploymentName, "namespace": OperatorNamespace},
			"spec": map[string]any{"template": map[string]any{"spec": map[string]any{"containers": []any{
				map[string]any{"name": "manager", "image": "ghcr.io/open-platform-model/opm-operator:" + tag + "@sha256:0000"},
			}}}},
		}}, "controller-manager", mv))
	}
	return objs
}

// renderResult wraps fixture objects as the render workflow returns them.
func renderResult(objs []*unstructured.Unstructured, values map[string]any, moduleVersion string) *workflowrender.Result {
	return &workflowrender.Result{
		Resources: objs,
		Instance:  module.InstanceMetadata{Name: OperatorInstanceName, Namespace: OperatorNamespace, UUID: testInstanceUUID},
		Module: module.ModuleMetadata{
			Name: "opm_operator", ModulePath: OperatorModulePath + "@v0", Version: moduleVersion,
		},
		Values:       values,
		RenderDigest: "sha256:render",
	}
}

// fakeRender records the values source it was handed and returns a render
// of objs carrying those values, or err.
type fakeRender struct {
	objs   []*unstructured.Unstructured
	err    error
	calls  int
	values kernel.Source
}

func (f *fakeRender) render(_ context.Context, res *modref.Resolution, values kernel.Source) (*workflowrender.Result, error) {
	f.calls++
	f.values = values
	if f.err != nil {
		return nil, f.err
	}
	var wrapper struct {
		Values map[string]any `json:"values"`
	}
	if err := json.Unmarshal(values.Data, &wrapper); err != nil {
		return nil, err
	}
	version := "0.1.0"
	if res != nil {
		version = res.Version[1:]
	}
	objs := make([]*unstructured.Unstructured, len(f.objs))
	for i, o := range f.objs {
		objs[i] = o.DeepCopy()
	}
	return renderResult(objs, wrapper.Values, version), nil
}

// testResolution is the resolved module version a plan installs.
func testResolution(version string) *modref.Resolution {
	return &modref.Resolution{Path: OperatorModulePath, Major: "v0", Version: version, Strategy: modref.Exact}
}

// fakeCluster is a dynamic fake that serves server-side apply the way the
// API server does for the operator's objects: it stores the applied object,
// keeps uid and resourceVersion when nothing changed, marks CRDs Established
// and Deployments rolled out, and applies a status patch to the stored
// object's status only. Every write verb is logged.
type fakeCluster struct {
	t      *testing.T
	fake   *fakedynamic.FakeDynamicClient
	client *kubernetes.Client

	mu     sync.Mutex
	writes []string
	rv     int
	// notReady keeps Deployments from rolling out.
	notReady bool
	// denyStatusRBAC makes the status-subresource access review deny.
	denyStatusRBAC bool
}

func newFakeCluster(t *testing.T, objs ...*unstructured.Unstructured) *fakeCluster {
	t.Helper()
	listKinds := map[schema.GroupVersionResource]string{
		moduleInstanceGVR:     "ModuleInstanceList",
		inventory.PlatformGVR: "PlatformList",
		crdGVR:                "CustomResourceDefinitionList",
	}
	runtimeObjs := make([]runtime.Object, len(objs))
	for i, o := range objs {
		runtimeObjs[i] = o
	}
	fc := &fakeCluster{t: t}
	fc.fake = fakedynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), listKinds, runtimeObjs...)
	fc.fake.PrependReactor("patch", "*", fc.applyPatch)
	fc.fake.PrependReactor("*", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		switch action.GetVerb() {
		case "create", "update", "delete", "delete-collection":
			fc.mu.Lock()
			fc.writes = append(fc.writes, action.GetVerb()+" "+action.GetResource().Resource)
			fc.mu.Unlock()
		}
		return false, nil, nil
	})
	cs := k8sfake.NewClientset()
	// The module renders its Namespace, so install must never create one
	// outside the instance apply (the --create-namespace path writes through
	// the typed client): log it and fail the call.
	cs.PrependReactor("create", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		fc.mu.Lock()
		fc.writes = append(fc.writes, "create namespaces")
		fc.mu.Unlock()
		return true, nil, fmt.Errorf("fake cluster: install created Namespace outside the render")
	})
	cs.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authorizationv1.SelfSubjectAccessReview{
			Status: authorizationv1.SubjectAccessReviewStatus{Allowed: !fc.denyStatusRBAC, Reason: "test"},
		}, nil
	})
	fc.client = &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fc.fake, Clientset: cs}
	return fc
}

// Writes returns the logged writes.
func (fc *fakeCluster) Writes() []string {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return append([]string(nil), fc.writes...)
}

func (fc *fakeCluster) applyPatch(action k8stesting.Action) (bool, runtime.Object, error) {
	patch, ok := action.(k8stesting.PatchAction)
	if !ok {
		return false, nil, nil
	}
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.writes = append(fc.writes, "patch "+action.GetResource().Resource)
	if patch.GetPatchType() != types.ApplyPatchType {
		return false, nil, nil // JSON patches (finalizers) go to the tracker
	}

	applied := &unstructured.Unstructured{}
	if err := json.Unmarshal(patch.GetPatch(), applied); err != nil {
		return true, nil, err
	}
	gvr := action.GetResource()
	ns := action.GetNamespace()
	tracker := fc.fake.Tracker()
	existingObj, getErr := tracker.Get(gvr, ns, patch.GetName())
	var existing *unstructured.Unstructured
	if getErr == nil {
		existing = existingObj.(*unstructured.Unstructured)
	} else if !apierrors.IsNotFound(getErr) {
		return true, nil, getErr
	}

	if patch.GetSubresource() == "status" {
		if existing == nil {
			return true, nil, apierrors.NewNotFound(gvr.GroupResource(), patch.GetName())
		}
		updated := existing.DeepCopy()
		updated.Object["status"] = applied.Object["status"]
		fc.rv++
		updated.SetResourceVersion(strconv.Itoa(fc.rv))
		return true, updated, tracker.Update(gvr, updated, ns)
	}

	next := applied.DeepCopy()
	switch next.GetKind() {
	case kindCustomResourceDefinition:
		_ = unstructured.SetNestedSlice(next.Object, []any{map[string]any{"type": "Established", "status": "True"}}, "status", "conditions")
	case kindDeployment:
		if !fc.notReady {
			_ = unstructured.SetNestedMap(next.Object, map[string]any{
				"observedGeneration": int64(0), "replicas": int64(1), "updatedReplicas": int64(1), "availableReplicas": int64(1),
			}, "status")
		}
	}
	if existing == nil {
		fc.rv++
		next.SetUID(types.UID(fmt.Sprintf("uid-%d", fc.rv)))
		next.SetResourceVersion(strconv.Itoa(fc.rv))
		return true, next, tracker.Add(next)
	}
	if status, ok := existing.Object["status"]; ok && next.Object["status"] == nil {
		next.Object["status"] = status
	}
	next.SetUID(existing.GetUID())
	next.SetResourceVersion(existing.GetResourceVersion())
	next.SetFinalizers(existing.GetFinalizers())
	if reflect.DeepEqual(contentOf(existing), contentOf(next)) {
		return true, existing, nil
	}
	fc.rv++
	next.SetResourceVersion(strconv.Itoa(fc.rv))
	return true, next, tracker.Update(gvr, next, ns)
}

// contentOf is an object without the fields the server manages.
func contentOf(obj *unstructured.Unstructured) map[string]any {
	c := obj.DeepCopy().Object
	unstructured.RemoveNestedField(c, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(c, "metadata", "managedFields")
	unstructured.RemoveNestedField(c, "status")
	return c
}
