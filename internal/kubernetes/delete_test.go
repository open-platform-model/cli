package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/open-platform-model/cli/pkg/resourceorder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

func TestSortObjects_Descending(t *testing.T) {
	// Create resources of different kinds
	resources := []*unstructured.Unstructured{
		makeUnstructured("apps/v1", "Deployment", "my-deploy", "default"),
		makeUnstructured("v1", "Namespace", "my-ns", ""),
		makeUnstructured("admissionregistration.k8s.io/v1", "ValidatingWebhookConfiguration", "my-webhook", ""),
		makeUnstructured("v1", "ConfigMap", "my-cm", "default"),
		makeUnstructured("v1", "Service", "my-svc", "default"),
	}

	SortObjects(resources, resourceorder.Descending)

	// Expected order: Webhook(500) > Deployment(100) > Service(50) > ConfigMap(15) > Namespace(0)
	assert.Equal(t, "ValidatingWebhookConfiguration", resources[0].GetKind())
	assert.Equal(t, "Deployment", resources[1].GetKind())
	assert.Equal(t, "Service", resources[2].GetKind())
	assert.Equal(t, "ConfigMap", resources[3].GetKind())
	assert.Equal(t, "Namespace", resources[4].GetKind())
}

func TestDelete_DeletesOnlyTrackedInventoryResources(t *testing.T) {
	ctx := context.Background()
	namespace := "default"

	tracked := makeUnstructured("v1", "ConfigMap", "tracked", namespace)
	setOwnership(tracked, pkgcore.LabelManagedByValue, testInstanceUUID)
	untracked := makeUnstructured("v1", "ConfigMap", "untracked", namespace)

	scheme := runtime.NewScheme()
	client := &Client{
		Clientset: fake.NewClientset(tracked.DeepCopy(), untracked.DeepCopy()),
		Dynamic:   dynamicfake.NewSimpleDynamicClient(scheme, tracked.DeepCopy(), untracked.DeepCopy()),
	}

	// The ModuleInstance CR is deleted last by the caller, not by Delete; here
	// we only assert Delete removes the tracked workload and leaves untracked
	// resources alone.
	result, err := Delete(ctx, client, DeleteOptions{
		InstanceName:          "demo",
		Namespace:             namespace,
		InstanceUUID:          testInstanceUUID,
		InventoryLive:         []*unstructured.Unstructured{tracked.DeepCopy()},
		InventoryRecordExists: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Deleted)

	_, err = client.ResourceClient(GVRFromUnstructured(tracked), namespace).Get(ctx, tracked.GetName(), metav1.GetOptions{})
	assert.Error(t, err)

	remaining, err := client.ResourceClient(GVRFromUnstructured(untracked), namespace).Get(ctx, untracked.GetName(), metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, untracked.GetName(), remaining.GetName())
}

const testInstanceUUID = "uuid-demo"

// setOwnership stamps the managed-by and instance UUID labels; an empty value
// leaves that label off.
func setOwnership(obj *unstructured.Unstructured, managedBy, uuid string) {
	labels := map[string]string{}
	if managedBy != "" {
		labels[pkgcore.LabelManagedBy] = managedBy
	}
	if uuid != "" {
		labels[pkgcore.LabelModuleInstanceUUID] = uuid
	}
	obj.SetLabels(labels)
}

func owned(apiVersion, kind, name, namespace, managedBy, uuid string) *unstructured.Unstructured {
	obj := makeUnstructured(apiVersion, kind, name, namespace)
	setOwnership(obj, managedBy, uuid)
	return obj
}

// TestDelete_LeavesBehind covers the per-object checks Delete makes before
// each delete: protected kinds, the live re-read, the managed-by and instance
// UUID match, and the NotFound and read-error outcomes. Each case runs once
// for real and once as a dry run, which must report the same left-behind set
// and delete nothing.
func TestDelete_LeavesBehind(t *testing.T) {
	opmCM := func(name string) *unstructured.Unstructured {
		return owned("v1", "ConfigMap", name, "default", pkgcore.LabelManagedByValue, testInstanceUUID)
	}
	notFound := func(resource, name string) error {
		return apierrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
	}

	tests := []struct {
		name         string
		inventory    *unstructured.Unstructured // what discovery read
		live         *unstructured.Unstructured // what the cluster holds now; nil = absent
		instanceUUID string
		reactor      func(dyn *dynamicfake.FakeDynamicClient)
		wantDeleted  int
		dryDeleted   int    // a dry run issues no delete call, so a delete-time NotFound cannot happen
		wantReason   string // non-empty: left behind with this reason
		wantErr      bool
		wantPresent  bool // the live object still exists afterwards
	}{
		{
			name:         "Namespace is left behind",
			inventory:    owned("v1", "Namespace", "apps", "", pkgcore.LabelManagedByValue, testInstanceUUID),
			live:         owned("v1", "Namespace", "apps", "", pkgcore.LabelManagedByValue, testInstanceUUID),
			instanceUUID: testInstanceUUID,
			wantReason:   ProtectedKindReason,
			wantPresent:  true,
		},
		{
			name:         "CRD is left behind",
			inventory:    owned("apiextensions.k8s.io/v1", "CustomResourceDefinition", "widgets.example.io", "", pkgcore.LabelManagedByValue, testInstanceUUID),
			live:         owned("apiextensions.k8s.io/v1", "CustomResourceDefinition", "widgets.example.io", "", pkgcore.LabelManagedByValue, testInstanceUUID),
			instanceUUID: testInstanceUUID,
			wantReason:   ProtectedKindReason,
			wantPresent:  true,
		},
		{
			name:         "managed-by removed after discovery",
			inventory:    opmCM("cm"),
			live:         owned("v1", "ConfigMap", "cm", "default", "", testInstanceUUID),
			instanceUUID: testInstanceUUID,
			wantReason:   reasonNotManaged,
			wantPresent:  true,
		},
		{
			name:         "UUID of another instance",
			inventory:    opmCM("cm"),
			live:         owned("v1", "ConfigMap", "cm", "default", pkgcore.LabelManagedByValue, "uuid-other"),
			instanceUUID: testInstanceUUID,
			wantReason:   reasonOtherInstance,
			wantPresent:  true,
		},
		{
			name:         "no UUID label deletes on managed-by alone",
			inventory:    opmCM("cm"),
			live:         owned("v1", "ConfigMap", "cm", "default", pkgcore.LabelManagedByControllerValue, ""),
			instanceUUID: testInstanceUUID,
			wantDeleted:  1,
		},
		{
			name:        "no recorded instance UUID deletes on managed-by alone",
			inventory:   opmCM("cm"),
			live:        owned("v1", "ConfigMap", "cm", "default", pkgcore.LabelManagedByValue, "uuid-other"),
			wantDeleted: 1,
		},
		{
			name:         "already gone at the re-read",
			inventory:    opmCM("cm"),
			instanceUUID: testInstanceUUID,
		},
		{
			name:         "gone between the re-read and the delete",
			inventory:    opmCM("cm"),
			live:         opmCM("cm"),
			instanceUUID: testInstanceUUID,
			reactor: func(dyn *dynamicfake.FakeDynamicClient) {
				dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, notFound("configmaps", "cm")
				})
			},
			dryDeleted:  1,
			wantPresent: true, // the reactor answered NotFound without touching the tracker
		},
		{
			name:         "read error is a failure",
			inventory:    opmCM("cm"),
			live:         opmCM("cm"),
			instanceUUID: testInstanceUUID,
			reactor: func(dyn *dynamicfake.FakeDynamicClient) {
				dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
					return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "cm", errors.New("denied"))
				})
			},
			wantErr:     true,
			wantPresent: true,
		},
	}

	for _, tc := range tests {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dryRun=%v", tc.name, dryRun), func(t *testing.T) {
				ctx := context.Background()
				var objs []runtime.Object
				if tc.live != nil {
					objs = append(objs, tc.live.DeepCopy())
				}
				dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
				if tc.reactor != nil {
					tc.reactor(dyn)
				}
				client := &Client{Dynamic: dyn}

				result, err := Delete(ctx, client, DeleteOptions{
					InstanceName:          "demo",
					Namespace:             "default",
					InstanceUUID:          tc.instanceUUID,
					DryRun:                dryRun,
					InventoryLive:         []*unstructured.Unstructured{tc.inventory.DeepCopy()},
					InventoryRecordExists: true,
				})
				require.NoError(t, err)

				wantDeleted := tc.wantDeleted
				if dryRun && tc.dryDeleted > 0 {
					wantDeleted = tc.dryDeleted
				}
				assert.Equal(t, wantDeleted, result.Deleted)
				if tc.wantErr {
					assert.Len(t, result.Errors, 1)
				} else {
					assert.Empty(t, result.Errors)
				}
				if tc.wantReason != "" {
					require.Len(t, result.LeftBehind, 1)
					lb := result.LeftBehind[0]
					assert.Equal(t, tc.inventory.GetKind(), lb.Kind)
					assert.Equal(t, tc.inventory.GetName(), lb.Name)
					assert.Equal(t, tc.inventory.GetNamespace(), lb.Namespace)
					assert.Equal(t, tc.wantReason, lb.Reason)
				} else {
					assert.Empty(t, result.LeftBehind)
				}

				if tc.live == nil {
					return
				}
				_, getErr := dyn.Tracker().Get(GVRFromUnstructured(tc.live), tc.live.GetNamespace(), tc.live.GetName())
				if tc.wantPresent || dryRun {
					assert.NoError(t, getErr, "the object is still on the cluster")
				} else {
					assert.True(t, apierrors.IsNotFound(getErr), "the object was deleted")
				}
			})
		}
	}
}

func makeUnstructured(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return obj
}

// TestDelete_Unreadable covers tracked resources the caller's discovery could
// not read. Each is a per-resource error, except a protected kind, which is
// left behind as it would be if read; the readable resources are still
// processed. Each case runs for real and as a dry run.
func TestDelete_Unreadable(t *testing.T) {
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "settings", errors.New("denied"))
	cmUnreadable := UnreadableResource{Kind: "ConfigMap", Namespace: "default", Name: "settings", Err: forbidden}
	nsUnreadable := UnreadableResource{Kind: "Namespace", Name: "apps", Err: forbidden}
	crdUnreadable := UnreadableResource{Group: "apiextensions.k8s.io", Kind: "CustomResourceDefinition", Name: "widgets.example.io", Err: forbidden}

	tests := []struct {
		name          string
		unreadable    []UnreadableResource
		withLive      bool
		recordExists  bool
		wantErrors    int
		wantLeftKinds []string
	}{
		{name: "unreadable ConfigMap fails beside a deleted Deployment", unreadable: []UnreadableResource{cmUnreadable}, withLive: true, recordExists: true, wantErrors: 1},
		{name: "unreadable Namespace is left behind", unreadable: []UnreadableResource{nsUnreadable}, withLive: true, recordExists: true, wantLeftKinds: []string{"Namespace"}},
		{name: "unreadable CRD is left behind", unreadable: []UnreadableResource{crdUnreadable}, withLive: true, recordExists: true, wantLeftKinds: []string{"CustomResourceDefinition"}},
		{name: "unreadable alone is not no-resources-found", unreadable: []UnreadableResource{cmUnreadable}, wantErrors: 1},
	}

	for _, tc := range tests {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dryRun=%v", tc.name, dryRun), func(t *testing.T) {
				ctx := context.Background()
				deploy := owned("apps/v1", "Deployment", "web", "default", pkgcore.LabelManagedByValue, testInstanceUUID)
				var objs []runtime.Object
				var live []*unstructured.Unstructured
				if tc.withLive {
					objs = append(objs, deploy.DeepCopy())
					live = append(live, deploy.DeepCopy())
				}
				dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), objs...)
				client := &Client{Dynamic: dyn}

				result, err := Delete(ctx, client, DeleteOptions{
					InstanceName:          "demo",
					Namespace:             "default",
					InstanceUUID:          testInstanceUUID,
					DryRun:                dryRun,
					InventoryLive:         live,
					InventoryRecordExists: tc.recordExists,
					Unreadable:            tc.unreadable,
				})
				require.NoError(t, err)

				require.Len(t, result.Errors, tc.wantErrors)
				if tc.wantErrors > 0 {
					assert.Equal(t, "ConfigMap", result.Errors[0].Kind)
					assert.Equal(t, "settings", result.Errors[0].Name)
					assert.True(t, apierrors.IsForbidden(result.Errors[0].Err))
				}
				var leftKinds []string
				for _, lb := range result.LeftBehind {
					assert.Equal(t, ProtectedKindReason, lb.Reason)
					leftKinds = append(leftKinds, lb.Kind)
				}
				assert.Equal(t, tc.wantLeftKinds, leftKinds)

				if !tc.withLive {
					assert.Equal(t, 0, result.Deleted)
					return
				}
				assert.Equal(t, 1, result.Deleted, "the readable Deployment is still processed")
				_, getErr := dyn.Tracker().Get(GVRFromUnstructured(deploy), "default", "web")
				if dryRun {
					assert.NoError(t, getErr, "a dry run deletes nothing")
				} else {
					assert.True(t, apierrors.IsNotFound(getErr), "the Deployment is deleted")
				}
			})
		}
	}
}
