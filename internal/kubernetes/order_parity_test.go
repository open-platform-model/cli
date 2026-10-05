package kubernetes

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/library/opm/k8s/object"
)

// The literals below are the kind-class weight table the cli applied, deleted
// and printed by in its own pkg/resourceorder before the move to the
// library's opm/k8s/object (0012:D5:R2); while both tables existed, this test
// asserted each literal against both. They pin what the cli orders by: a
// library release that moves any of these weights fails here on the bump PR,
// so an order change is a reviewed edit in both repositories. A row the
// library adds is the library's own TestWeightTableGuard, not this test.

// retiredWeightConstants is every Weight* constant of the retired table.
var retiredWeightConstants = map[string]struct{ lib, want int }{
	"WeightCRD":                {object.WeightCRD, -100},
	"WeightNamespace":          {object.WeightNamespace, 0},
	"WeightClusterRole":        {object.WeightClusterRole, 5},
	"WeightClusterRoleBinding": {object.WeightClusterRoleBinding, 5},
	"WeightServiceAccount":     {object.WeightServiceAccount, 10},
	"WeightRole":               {object.WeightRole, 10},
	"WeightRoleBinding":        {object.WeightRoleBinding, 10},
	"WeightSecret":             {object.WeightSecret, 15},
	"WeightConfigMap":          {object.WeightConfigMap, 15},
	"WeightStorageClass":       {object.WeightStorageClass, 20},
	"WeightPersistentVolume":   {object.WeightPersistentVolume, 20},
	"WeightPVC":                {object.WeightPVC, 20},
	"WeightService":            {object.WeightService, 50},
	"WeightDeployment":         {object.WeightDeployment, 100},
	"WeightStatefulSet":        {object.WeightStatefulSet, 100},
	"WeightDaemonSet":          {object.WeightDaemonSet, 100},
	"WeightJob":                {object.WeightJob, 110},
	"WeightCronJob":            {object.WeightCronJob, 110},
	"WeightIngress":            {object.WeightIngress, 150},
	"WeightNetworkPolicy":      {object.WeightNetworkPolicy, 150},
	"WeightHPA":                {object.WeightHPA, 200},
	"WeightVPA":                {object.WeightVPA, 200},
	"WeightPDB":                {object.WeightPDB, 200},
	"WeightWebhook":            {object.WeightWebhook, 500},
	"WeightDefault":            {object.WeightDefault, 1000},
}

// retiredGVKWeights is every exact group-version-kind entry of the table.
var retiredGVKWeights = map[schema.GroupVersionKind]int{
	{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}: -100,

	{Group: "", Version: "v1", Kind: "Namespace"}:             0,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:        10,
	{Group: "", Version: "v1", Kind: "Secret"}:                15,
	{Group: "", Version: "v1", Kind: "ConfigMap"}:             15,
	{Group: "", Version: "v1", Kind: "PersistentVolume"}:      20,
	{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"}: 20,
	{Group: "", Version: "v1", Kind: "Service"}:               50,

	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}:        5,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}: 5,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}:               10,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}:        10,

	{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"}: 20,

	{Group: "apps", Version: "v1", Kind: "Deployment"}:  100,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: 100,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:   100,
	{Group: "apps", Version: "v1", Kind: "ReplicaSet"}:  100,

	{Group: "batch", Version: "v1", Kind: "Job"}:     110,
	{Group: "batch", Version: "v1", Kind: "CronJob"}: 110,

	{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}:       150,
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"}: 150,

	{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}:      200,
	{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"}:      200,
	{Group: "autoscaling.k8s.io", Version: "v1", Kind: "VerticalPodAutoscaler"}: 200,

	{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"}: 200,

	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}: 500,
	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}:   500,
}

// retiredKindWeights is every kind-only entry of the table. Each kind here
// also has an exact row above, so the test reaches the kind table through a
// group and version the exact rows do not hold.
var retiredKindWeights = map[string]int{
	"Namespace":                      0,
	"ServiceAccount":                 10,
	"Secret":                         15,
	"ConfigMap":                      15,
	"PersistentVolume":               20,
	"PersistentVolumeClaim":          20,
	"Service":                        50,
	"ClusterRole":                    5,
	"ClusterRoleBinding":             5,
	"Role":                           10,
	"RoleBinding":                    10,
	"StorageClass":                   20,
	"Deployment":                     100,
	"StatefulSet":                    100,
	"DaemonSet":                      100,
	"ReplicaSet":                     100,
	"Job":                            110,
	"CronJob":                        110,
	"Ingress":                        150,
	"NetworkPolicy":                  150,
	"HorizontalPodAutoscaler":        200,
	"VerticalPodAutoscaler":          200,
	"PodDisruptionBudget":            200,
	"ValidatingWebhookConfiguration": 500,
	"MutatingWebhookConfiguration":   500,
	"CustomResourceDefinition":       -100,
}

// retiredFallbacks are the two fallback rows: an unknown kind weighs the
// default, and an unknown version of a known kind weighs the kind.
var retiredFallbacks = map[schema.GroupVersionKind]int{
	{Group: "example.com", Version: "v1", Kind: "Foo"}:                          1000,
	{Group: "autoscaling", Version: "v2beta2", Kind: "HorizontalPodAutoscaler"}: 200,
}

func kindOnlyGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "fallback.invalid", Version: "v9", Kind: kind}
}

func TestWeightTableMatchesRetiredCopy(t *testing.T) {
	t.Run("constants", func(t *testing.T) {
		for name, c := range retiredWeightConstants {
			assert.Equal(t, c.want, c.lib, "object.%s", name)
		}
	})
	t.Run("gvk entries", func(t *testing.T) {
		for gvk, want := range retiredGVKWeights {
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
	t.Run("kind entries", func(t *testing.T) {
		for kind, want := range retiredKindWeights {
			gvk := kindOnlyGVK(kind)
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
	t.Run("fallbacks", func(t *testing.T) {
		for gvk, want := range retiredFallbacks {
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
}

// orderSet is one shuffled set covering every weight class, two ConfigMaps
// and two Deployments in a fixed input order, and one unknown kind.
func orderSet() []*unstructured.Unstructured {
	mk := func(apiVersion, kind, name string) *unstructured.Unstructured {
		u := &unstructured.Unstructured{}
		u.SetAPIVersion(apiVersion)
		u.SetKind(kind)
		u.SetName(name)
		return u
	}
	return []*unstructured.Unstructured{
		mk("apps/v1", "Deployment", "deploy-b"),
		mk("admissionregistration.k8s.io/v1", "ValidatingWebhookConfiguration", "webhook"),
		mk("v1", "ConfigMap", "cm-b"),
		mk("example.com/v1", "Foo", "unknown"),
		mk("policy/v1", "PodDisruptionBudget", "pdb"),
		mk("v1", "Namespace", "ns"),
		mk("batch/v1", "Job", "job"),
		mk("networking.k8s.io/v1", "Ingress", "ingress"),
		mk("apps/v1", "Deployment", "deploy-a"),
		mk("v1", "Service", "svc"),
		mk("storage.k8s.io/v1", "StorageClass", "sc"),
		mk("rbac.authorization.k8s.io/v1", "ClusterRole", "cr"),
		mk("v1", "ConfigMap", "cm-a"),
		mk("v1", "ServiceAccount", "sa"),
		mk("apiextensions.k8s.io/v1", "CustomResourceDefinition", "crd"),
	}
}

func objNames(objs []*unstructured.Unstructured) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.GetName()
	}
	return out
}

// retiredSortOrder is the name sequence the cli's retired sort produced for
// orderSet, per direction (recorded while both sorts existed).
var retiredSortOrder = map[object.Direction][]string{
	object.Ascending: {
		"crd", "ns", "cr", "sa", "cm-b", "cm-a", "sc", "svc",
		"deploy-b", "deploy-a", "job", "ingress", "pdb", "webhook", "unknown",
	},
	object.Descending: {
		"unknown", "webhook", "pdb", "ingress", "job", "deploy-b", "deploy-a",
		"svc", "sc", "cm-b", "cm-a", "sa", "cr", "ns", "crd",
	},
}

func TestSortMatchesRetiredCopy(t *testing.T) {
	for _, tc := range []struct {
		name string
		dir  object.Direction
	}{
		{"ascending", object.Ascending},
		{"descending", object.Descending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			objs := orderSet()
			SortObjects(objs, tc.dir)
			assert.Equal(t, retiredSortOrder[tc.dir], objNames(objs))
		})
	}
}

// TestSortObjectsStableOnLargeInput pins that SortObjects keeps equal-weight
// objects in their input order, in both directions. The input is large
// enough (96 objects) that an unstable sort reorders equal elements, which
// the 15-object orderSet does not reveal. Names count down while the input
// order counts up, so a sort that fell back to the name would also fail.
func TestSortObjectsStableOnLargeInput(t *testing.T) {
	const pairs = 48
	build := func() []*unstructured.Unstructured {
		objs := make([]*unstructured.Unstructured, 0, 2*pairs)
		for i := range pairs {
			cm := &unstructured.Unstructured{}
			cm.SetAPIVersion("v1")
			cm.SetKind("ConfigMap")
			cm.SetName(fmt.Sprintf("cm-%03d", pairs-i))
			deploy := &unstructured.Unstructured{}
			deploy.SetAPIVersion("apps/v1")
			deploy.SetKind("Deployment")
			deploy.SetName(fmt.Sprintf("deploy-%03d", pairs-i))
			objs = append(objs, cm, deploy)
		}
		return objs
	}
	inputOrder := func(kind string) []string {
		var names []string
		for _, o := range build() {
			if o.GetKind() == kind {
				names = append(names, o.GetName())
			}
		}
		return names
	}
	cms, deploys := inputOrder("ConfigMap"), inputOrder("Deployment")

	for _, tc := range []struct {
		name string
		dir  object.Direction
		want []string
	}{
		{"ascending", object.Ascending, append(append([]string{}, cms...), deploys...)},
		{"descending", object.Descending, append(append([]string{}, deploys...), cms...)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			objs := build()
			SortObjects(objs, tc.dir)
			assert.Equal(t, tc.want, objNames(objs))
		})
	}
}
