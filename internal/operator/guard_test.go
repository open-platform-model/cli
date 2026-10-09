package operator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	"github.com/open-platform-model/cli/internal/inventory"
)

// otherInstanceUUID is the identity of an instance that is not the operator's.
const otherInstanceUUID = "99999999-9999-9999-9999-999999999999"

// liveOf is a rendered fixture object as it stands in the cluster.
func liveOf(objs []*unstructured.Unstructured, kind, name string) *unstructured.Unstructured {
	return findObj(objs, kind, name).DeepCopy()
}

// ownedBy relabels a live object as another instance's.
func ownedBy(obj *unstructured.Unstructured, uuid string) *unstructured.Unstructured {
	labels := obj.GetLabels()
	labels[opmlabels.ModuleInstanceUUID] = uuid
	obj.SetLabels(labels)
	return obj
}

// adoptedElsewhere sets the adopt annotation of a live object to another
// instance's identity.
func adoptedElsewhere(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: otherInstanceUUID})
	return obj
}

// clientSideApplied gives a live object fields of a client-side kubectl
// apply.
func clientSideApplied(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj.SetManagedFields([]metav1.ManagedFieldsEntry{{Manager: "kubectl-client-side-apply", Operation: metav1.ManagedFieldsOperationUpdate}})
	return obj
}

// recordListing is the operator instance's record with the given objects in
// its inventory.
func recordListing(objs ...*unstructured.Unstructured) *unstructured.Unstructured {
	rec := operatorRecord("cli", nil)
	entries := make([]any, 0, len(objs))
	for _, o := range objs {
		gvk := o.GroupVersionKind()
		entries = append(entries, map[string]any{
			"group": gvk.Group, "kind": gvk.Kind, "v": gvk.Version, "namespace": o.GetNamespace(), "name": o.GetName(),
		})
	}
	_ = unstructured.SetNestedSlice(rec.Object, entries, "status", "inventory", "entries")
	_ = unstructured.SetNestedField(rec.Object, int64(len(entries)), "status", "inventory", "count")
	return rec
}

// The apply guard runs on every install, with or without a record of the
// operator's instance, and refuses in the check phase, before any write, for
// an object another instance owns or adopts. Install needs every object it
// renders, so an adoption by another instance refuses too.
func TestPlanInstall_GuardRefusesWhatAnotherInstanceOwns(t *testing.T) {
	rendered := moduleObjects(renderOpts{extraRole: "team-a-role"})
	crd := CRDNames()[0]
	foreignRole := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole", "metadata": map[string]any{"name": "team-a-role"},
	}}
	deployment := func() *unstructured.Unstructured { return liveOf(rendered, kindDeployment, ControllerDeploymentName) }
	namespace := func() *unstructured.Unstructured { return liveOf(rendered, kindNamespace, OperatorNamespace) }

	cases := []struct {
		name    string
		cluster []*unstructured.Unstructured
		want    []string
	}{
		{
			name:    "no record: an object of another instance",
			cluster: []*unstructured.Unstructured{ownedBy(liveOf(rendered, "ClusterRole", "team-a-role"), otherInstanceUUID)},
			want:    []string{"ClusterRole/team-a-role belongs to module instance " + otherInstanceUUID},
		},
		{
			name:    "record: a foreign object new to the inventory",
			cluster: []*unstructured.Unstructured{recordListing(), foreignRole},
			want:    []string{"ClusterRole/team-a-role exists and is not managed by OPM", opmlabels.AnnotationAdopt + "=" + testInstanceUUID},
		},
		{
			name:    "record: another instance's object new to the inventory",
			cluster: []*unstructured.Unstructured{recordListing(), ownedBy(liveOf(rendered, "ClusterRole", "team-a-role"), otherInstanceUUID)},
			want:    []string{"ClusterRole/team-a-role belongs to module instance " + otherInstanceUUID},
		},
		{
			name:    "no record: the controller Deployment is adopted by another instance",
			cluster: []*unstructured.Unstructured{adoptedElsewhere(clientSideApplied(deployment()))},
			want:    []string{"Deployment/" + OperatorNamespace + "/" + ControllerDeploymentName, "adopted by module instance " + otherInstanceUUID},
		},
		{
			name: "record: the recorded controller Deployment is adopted by another instance",
			cluster: []*unstructured.Unstructured{
				recordListing(deployment()),
				adoptedElsewhere(clientSideApplied(deployment())),
			},
			want: []string{"Deployment/" + OperatorNamespace + "/" + ControllerDeploymentName, "was adopted by module instance " + otherInstanceUUID},
		},
		{
			name:    "record: the recorded Namespace is adopted by another instance",
			cluster: []*unstructured.Unstructured{recordListing(namespace()), adoptedElsewhere(namespace())},
			want:    []string{"Namespace/" + OperatorNamespace + " was adopted by module instance " + otherInstanceUUID},
		},
		{
			name: "record: a recorded CRD is adopted by another instance",
			cluster: []*unstructured.Unstructured{
				recordListing(liveOf(rendered, kindCustomResourceDefinition, crd)),
				adoptedElsewhere(liveOf(rendered, kindCustomResourceDefinition, crd)),
			},
			want: []string{"CustomResourceDefinition/" + crd + " was adopted by module instance " + otherInstanceUUID},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fastPolling(t)
			fc := newFakeCluster(t, c.cluster...)

			plan, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: rendered}),
				testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})

			var ge *GuardError
			require.ErrorAs(t, err, &ge, "the command exits 2 on every GuardError")
			var refusal *inventory.GuardRefusalError
			require.ErrorAs(t, err, &refusal)
			for _, w := range c.want {
				assert.Contains(t, err.Error(), w)
			}
			assert.Contains(t, err.Error(), "nothing was changed")
			assert.Nil(t, plan, "no plan, so no write of the install can follow")
			assert.Empty(t, fc.Writes(), "no apply, no delete and no managedFields patch was sent")
			for _, a := range fc.fake.Actions() {
				assert.Contains(t, []string{"get", "list"}, a.GetVerb(), "the check phase only reads")
			}
		})
	}
}

// With a record, a read the guard fails refuses the install as a guard
// refusal, like the guard's other refusals, and nothing is written.
func TestPlanInstall_GuardReadFailureWithARecord(t *testing.T) {
	fc := newFakeCluster(t, recordListing())
	denyRoleReads(fc, 2) // the terminating wait reads it first
	r := &fakeRender{objs: moduleObjects(renderOpts{})}

	_, err := PlanInstall(context.Background(), newEnv(fc, r), testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})

	var ge *GuardError
	require.ErrorAs(t, err, &ge)
	assert.Contains(t, err.Error(), "cannot check whether ClusterRole/"+managerRole)
	assert.Empty(t, fc.Writes())
}

// A CRD that carries another instance's identity is refused by the guard
// in the check phase, as any other object: the refusal exits 2, names the
// other instance and the adopt annotation, and writes nothing.
func TestPlanInstall_CRDOfAnotherInstanceRefusesBeforeAnyWrite(t *testing.T) {
	fastPolling(t)
	rendered := moduleObjects(renderOpts{})
	crd := CRDNames()[0]
	fc := newFakeCluster(t, ownedBy(liveOf(rendered, kindCustomResourceDefinition, crd), otherInstanceUUID))

	plan, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: rendered}),
		testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})

	var ge *GuardError
	require.ErrorAs(t, err, &ge, "the command exits 2 on a guard refusal")
	assert.Contains(t, err.Error(), "CustomResourceDefinition/"+crd+" belongs to module instance "+otherInstanceUUID)
	assert.Contains(t, err.Error(), opmlabels.AnnotationAdopt+"="+testInstanceUUID)
	assert.Nil(t, plan)
	assert.Empty(t, fc.Writes())
}

// A reinstall over the instance's own recorded objects passes the guard. A
// recorded object passes whatever its identity label says.
func TestPlanInstall_ReinstallOverRecordedObjectsPasses(t *testing.T) {
	fastPolling(t)
	rendered := moduleObjects(renderOpts{extraRole: "team-a-role"})
	cluster := make([]*unstructured.Unstructured, 0, 1+len(rendered))
	cluster = append(cluster, recordListing(rendered...))
	for _, obj := range rendered {
		live := obj.DeepCopy()
		if live.GetName() == "team-a-role" {
			ownedBy(live, "uuid-before-the-module-moved")
		}
		cluster = append(cluster, live)
	}
	fc := newFakeCluster(t, cluster...)

	_, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: rendered}),
		testResolution("v0.1.0"), defaultTarget, PlanOptions{Timeout: time.Second})

	require.NoError(t, err)
	assert.Empty(t, fc.Writes())
}
