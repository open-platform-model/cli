package operator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newObj(kind, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": apiVersionForKind(kind),
		"kind":       kind,
		"metadata": map[string]any{
			"name": name,
		},
	}}
}

// apiVersionForKind returns a plausible apiVersion for the given test fixture kind,
// matching the group resourceorder.GetWeight looks up by GVK.
func apiVersionForKind(kind string) string {
	switch kind {
	case "CustomResourceDefinition":
		return "apiextensions.k8s.io/v1"
	case "ClusterRole", "ClusterRoleBinding":
		return "rbac.authorization.k8s.io/v1"
	case "Deployment":
		return "apps/v1"
	default:
		return "v1"
	}
}

func fixtureManifest() []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		newObj("Deployment", "controller-manager"),
		newObj("Namespace", "opm-operator-system"),
		newObj("CustomResourceDefinition", "moduleinstances.opmodel.dev"),
		newObj("ServiceAccount", "controller-manager"),
		newObj("Service", "controller-manager-metrics"),
		newObj("CustomResourceDefinition", "platforms.opmodel.dev"),
		newObj("ClusterRole", "manager-role"),
		newObj("ClusterRoleBinding", "manager-rolebinding"),
	}
}

func TestInstallPlan_OrdersAscendingByWeight(t *testing.T) {
	plan := InstallPlan(fixtureManifest())
	require.Len(t, plan, 8)

	kinds := kindsOf(plan)
	assert.Equal(t, []string{
		"CustomResourceDefinition",
		"CustomResourceDefinition",
		"Namespace",
		"ClusterRole",
		"ClusterRoleBinding",
		"ServiceAccount",
		"Service",
		"Deployment",
	}, kinds)
}

func TestInstallPlan_DoesNotMutateInput(t *testing.T) {
	input := fixtureManifest()
	original := kindsOf(input)

	InstallPlan(input)

	assert.Equal(t, original, kindsOf(input))
}

func TestCRDsOnlyPlan_ReturnsOnlyCRDs(t *testing.T) {
	plan := CRDsOnlyPlan(fixtureManifest())
	require.Len(t, plan, 2)
	for _, obj := range plan {
		assert.Equal(t, "CustomResourceDefinition", obj.GetKind())
	}
}

func TestCRDsOnlyPlan_EmptyWhenNoCRDs(t *testing.T) {
	objs := []*unstructured.Unstructured{newObj("Deployment", "d"), newObj("Service", "s")}
	plan := CRDsOnlyPlan(objs)
	assert.Empty(t, plan)
}

func TestUninstallPlan_OrdersDescendingByWeight(t *testing.T) {
	plan := UninstallPlan(fixtureManifest())

	kinds := kindsOf(plan)
	assert.Equal(t, []string{
		"Deployment",
		"Service",
		"ServiceAccount",
		"ClusterRole",
		"ClusterRoleBinding",
	}, kinds)
}

// TestUninstallPlan_NeverIncludesCRDsOrNamespace is the property the design
// relies on: uninstall must be structurally unable to remove CRDs or the
// Namespace, across arbitrary manifest compositions.
func TestUninstallPlan_NeverIncludesCRDsOrNamespace(t *testing.T) {
	tests := []struct {
		name string
		objs []*unstructured.Unstructured
	}{
		{"full fixture manifest", fixtureManifest()},
		{"CRDs and namespace only", []*unstructured.Unstructured{
			newObj("CustomResourceDefinition", "a.opmodel.dev"),
			newObj("CustomResourceDefinition", "b.opmodel.dev"),
			newObj("Namespace", "opm-operator-system"),
		}},
		{"empty manifest", nil},
		{"no CRDs or namespace", []*unstructured.Unstructured{
			newObj("Deployment", "d"),
			newObj("ClusterRole", "r"),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := UninstallPlan(tt.objs)
			for _, obj := range plan {
				assert.NotEqual(t, "CustomResourceDefinition", obj.GetKind())
				assert.NotEqual(t, "Namespace", obj.GetKind())
			}
		})
	}
}

func TestUninstallPlan_DoesNotMutateInput(t *testing.T) {
	input := fixtureManifest()
	original := kindsOf(input)

	UninstallPlan(input)

	assert.Equal(t, original, kindsOf(input))
}

func kindsOf(objs []*unstructured.Unstructured) []string {
	kinds := make([]string, len(objs))
	for i, obj := range objs {
		kinds[i] = obj.GetKind()
	}
	return kinds
}
