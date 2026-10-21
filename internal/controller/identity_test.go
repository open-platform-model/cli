package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	"github.com/open-platform-model/cli/internal/controller/crds"
	"github.com/open-platform-model/cli/internal/inventory"
)

// adoptedHere sets the adopt annotation of a live object to the controller
// instance's identity.
func adoptedHere(obj *unstructured.Unstructured) *unstructured.Unstructured {
	obj.SetAnnotations(map[string]string{opmlabels.AnnotationAdopt: testInstanceUUID})
	return obj
}

// manifestApplied turns a rendered fixture object into what an opm-controller
// release manifest left in a cluster: no label at all on a CRD, kustomize's
// labels on the others, no OPM label.
func manifestApplied(obj *unstructured.Unstructured) *unstructured.Unstructured {
	live := obj.DeepCopy()
	if live.GetKind() == kindCustomResourceDefinition {
		live.SetLabels(nil)
		return live
	}
	live.SetLabels(map[string]string{
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/name":       "opm-controller",
	})
	return live
}

// liveCopies is every rendered object as it stands in the cluster, after
// change.
func liveCopies(rendered []*unstructured.Unstructured, change func(*unstructured.Unstructured) *unstructured.Unstructured) []*unstructured.Unstructured {
	out := make([]*unstructured.Unstructured, 0, len(rendered))
	for _, obj := range rendered {
		out = append(out, change(obj.DeepCopy()))
	}
	return out
}

// planOver plans an install of the fixture render over a cluster.
func planOver(t *testing.T, cluster ...*unstructured.Unstructured) (*fakeCluster, error) {
	t.Helper()
	fastPolling(t)
	fc := newFakeCluster(t, cluster...)
	_, err := PlanInstall(context.Background(), newEnv(fc, &fakeRender{objs: moduleObjects(renderOpts{})}), testResolution("v0.1.0"), defaultTarget,
		PlanOptions{Timeout: time.Second})
	return fc, err
}

// An install over objects that carry another instance's identity (the
// instance's module path, name or namespace changed) is decided by the
// ownership guard alone: the record lets them pass, the adopt annotation lets
// them pass, and with neither the install refuses and names the annotation.
func TestInstall_ObjectsOfAnotherIdentity(t *testing.T) {
	relabel := func(o *unstructured.Unstructured) *unstructured.Unstructured { return ownedBy(o, otherInstanceUUID) }

	t.Run("the record lists them: applied", func(t *testing.T) {
		releasedCLI(t)
		fastPolling(t)
		rendered := moduleObjects(renderOpts{})
		cluster := append([]*unstructured.Unstructured{recordListing(rendered...)}, liveCopies(rendered, relabel)...)
		fc := newFakeCluster(t, cluster...)

		result, err := install(t, fc, &fakeRender{objs: rendered}, PlanOptions{})

		require.NoError(t, err)
		assert.True(t, result.Recorded)
		assert.Len(t, entryNames(fc.record()), len(rendered))
		for _, crd := range CRDNames() {
			assert.Equal(t, testInstanceUUID, fc.mustGet(crdGVR, "", crd).GetLabels()[opmlabels.ModuleInstanceUUID], crd)
		}
	})

	t.Run("each carries the adopt annotation, no record: applied and recorded", func(t *testing.T) {
		releasedCLI(t)
		fastPolling(t)
		rendered := moduleObjects(renderOpts{})
		fc := newFakeCluster(t, liveCopies(rendered, func(o *unstructured.Unstructured) *unstructured.Unstructured {
			return adoptedHere(relabel(o))
		})...)

		result, err := install(t, fc, &fakeRender{objs: rendered}, PlanOptions{})

		require.NoError(t, err)
		assert.True(t, result.Recorded)
		assert.Len(t, entryNames(fc.record()), len(rendered))
	})

	t.Run("neither: refused, naming the annotation", func(t *testing.T) {
		rendered := moduleObjects(renderOpts{})
		fc, err := planOver(t, liveCopies(rendered, relabel)...)

		var ge *GuardError
		require.ErrorAs(t, err, &ge)
		msg := err.Error()
		assert.Contains(t, msg, fmt.Sprintf("refusing to install: %d object(s) cannot be applied by this instance:", len(rendered)))
		for _, crd := range CRDNames() {
			assert.Contains(t, msg, "CustomResourceDefinition/"+crd+" belongs to module instance "+otherInstanceUUID)
		}
		assert.Equal(t, len(rendered), strings.Count(msg, "annotate it "+opmlabels.AnnotationAdopt+"="+testInstanceUUID))
		assert.NotContains(t, msg, "controller migration refused")
		assert.NotContains(t, msg, "remove or rename")
		assert.Empty(t, fc.Writes())
	})
}

// A controller applied from an earlier release manifest is not taken
// over: its objects carry no OPM managed-by label, so the guard refuses each
// of them as any other object OPM does not manage, and install adopts none
// by itself. The CRDs-only form refuses the CRDs the same way.
func TestPlanInstall_ManifestInstallIsRefused(t *testing.T) {
	rendered := moduleObjects(renderOpts{})
	var cluster []*unstructured.Unstructured
	for _, obj := range rendered {
		switch obj.GetKind() {
		case kindCustomResourceDefinition, kindNamespace, kindDeployment:
			cluster = append(cluster, manifestApplied(obj))
		}
	}
	remedy := " exists and is not managed by OPM; to let this instance take it over, annotate it " + opmlabels.AnnotationAdopt + "=" + testInstanceUUID

	t.Run("full install", func(t *testing.T) {
		fc, err := planOver(t, cluster...)

		var ge *GuardError
		require.ErrorAs(t, err, &ge)
		msg := err.Error()
		assert.Contains(t, msg, "refusing to install: 6 object(s) cannot be applied by this instance:")
		for _, crd := range CRDNames() {
			assert.Contains(t, msg, "CustomResourceDefinition/"+crd+remedy)
		}
		assert.Contains(t, msg, "Namespace/"+ControllerNamespace+remedy)
		assert.Contains(t, msg, "Deployment/"+ControllerNamespace+"/"+ControllerDeploymentName+remedy)
		assert.True(t, strings.HasSuffix(msg, "nothing was changed"))
		assert.NotContains(t, msg, "migration")
		assert.Empty(t, fc.Writes())
	})

	// The CRDs-only form plans from the embedded CRDs and asks the same guard
	// as the instance those CRDs were rendered for.
	t.Run("CRDs only", func(t *testing.T) {
		fastPolling(t)
		fc := newFakeCluster(t, cluster...)
		embedded, err := crds.Objects()
		require.NoError(t, err)
		crdRemedy := " exists and is not managed by OPM; to let this instance take it over, annotate it " + opmlabels.AnnotationAdopt + "=" + embeddedInstanceUUID(embedded)

		_, err = PlanCRDsOnly(context.Background(), newEnv(fc, &fakeRender{}), PlanOptions{Timeout: time.Second})

		var ge *GuardError
		require.ErrorAs(t, err, &ge)
		msg := err.Error()
		assert.Contains(t, msg, "refusing to install: 4 object(s) cannot be applied by this instance:")
		for _, crd := range CRDNames() {
			assert.Contains(t, msg, "CustomResourceDefinition/"+crd+crdRemedy)
		}
		assert.NotContains(t, msg, "Namespace/")
		assert.True(t, strings.HasSuffix(msg, "nothing was changed"))
		assert.Empty(t, fc.Writes())
	})
}

// The instance's own objects that no record lists (a run that stopped before
// it wrote the record, a CRDs-only install, a kubectl apply of the module's
// render) pass the guard by the labels OPM stamps. The UUID label alone is
// not enough: without OPM's managed-by label the object is refused as not
// managed by OPM, and the refusal names the adopt annotation.
func TestPlanInstall_OwnUnrecordedObjects(t *testing.T) {
	keep := func(o *unstructured.Unstructured) *unstructured.Unstructured { return o }
	withManagedBy := func(value string) func(*unstructured.Unstructured) *unstructured.Unstructured {
		return func(o *unstructured.Unstructured) *unstructured.Unstructured {
			labels := o.GetLabels()
			if value == "" {
				delete(labels, opmlabels.ManagedBy)
			} else {
				labels[opmlabels.ManagedBy] = value
			}
			o.SetLabels(labels)
			return o
		}
	}
	rendered := moduleObjects(renderOpts{})

	t.Run("with OPM's labels: pass", func(t *testing.T) {
		fc, err := planOver(t, liveCopies(rendered, keep)...)
		require.NoError(t, err)
		assert.Empty(t, fc.Writes())
	})

	// A first install stamps every object; with its record gone, the cluster
	// is what a run that stopped before writing the record leaves behind.
	t.Run("with OPM's labels: a run that lost its record completes and records them", func(t *testing.T) {
		releasedCLI(t)
		fastPolling(t)
		fc := newFakeCluster(t)
		_, err := install(t, fc, &fakeRender{objs: rendered}, PlanOptions{})
		require.NoError(t, err)
		uid := fc.mustGet(deploymentGVR, ControllerNamespace, ControllerDeploymentName).GetUID()
		require.NoError(t, fc.client.ResourceClient(inventory.ModuleInstanceGVR, ControllerNamespace).Delete(context.Background(), ControllerInstanceName, metav1.DeleteOptions{}))
		require.Nil(t, fc.record())

		result, err := install(t, fc, &fakeRender{objs: rendered}, PlanOptions{})

		require.NoError(t, err)
		assert.True(t, result.Recorded)
		assert.Len(t, entryNames(fc.record()), len(rendered))
		assert.Equal(t, uid, fc.mustGet(deploymentGVR, ControllerNamespace, ControllerDeploymentName).GetUID(), "nothing is recreated")
	})

	// The guard's rule for every apply: OPM manages the object and no other
	// instance's UUID is on it. The migration proof used to refuse this for
	// the objects on its list.
	t.Run("OPM's managed-by label and no UUID label: pass", func(t *testing.T) {
		fc, err := planOver(t, liveCopies(rendered, func(o *unstructured.Unstructured) *unstructured.Unstructured {
			labels := o.GetLabels()
			delete(labels, opmlabels.ModuleInstanceUUID)
			o.SetLabels(labels)
			return o
		})...)
		require.NoError(t, err)
		assert.Empty(t, fc.Writes())
	})

	for name, value := range map[string]string{"managed-by label removed": "", "managed-by names another tool": "Helm"} {
		t.Run(name+": refused", func(t *testing.T) {
			fc, err := planOver(t, liveCopies(rendered, withManagedBy(value))...)

			var ge *GuardError
			require.ErrorAs(t, err, &ge)
			msg := err.Error()
			assert.Equal(t, len(rendered), strings.Count(msg, "exists and is not managed by OPM; to let this instance take it over, annotate it "+opmlabels.AnnotationAdopt+"="+testInstanceUUID))
			assert.Empty(t, fc.Writes())
		})
	}

	t.Run("managed-by label removed, the record lists them: pass", func(t *testing.T) {
		cluster := append([]*unstructured.Unstructured{recordListing(rendered...)}, liveCopies(rendered, withManagedBy(""))...)
		fc, err := planOver(t, cluster...)
		require.NoError(t, err)
		assert.Empty(t, fc.Writes())
	})
}
