package kubernetes

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/library/opm/k8s/object"
)

// The literals below are the cli's reviewed copy of the library's kind-class
// weight table (opm/k8s/object), which the cli applies, prunes, deletes and
// prints by. They pin that order: a library release that moves any of these
// weights fails here on the bump PR, so an order change is an edit a cli
// reviewer reads, never a side effect of a pin bump. The values are the
// library's since v1.0.0-beta.7, which follow Flux's staged apply order
// (0012:D5:R1). The table the cli carried in its own pkg/resourceorder
// before the move to the library is history and is no longer compared here.
// A row the library adds is the library's own TestWeightTableGuard, not this
// test.

// pinnedWeightConstants is every Weight* constant of the library's table.
var pinnedWeightConstants = map[string]struct{ lib, want int }{
	"WeightCRD":                {object.WeightCRD, -100},
	"WeightNamespace":          {object.WeightNamespace, 0},
	"WeightClusterRole":        {object.WeightClusterRole, 5},
	"WeightClass":              {object.WeightClass, 6},
	"WeightStorageClass":       {object.WeightStorageClass, 6},
	"WeightClusterRoleBinding": {object.WeightClusterRoleBinding, 7},
	"WeightResourceQuota":      {object.WeightResourceQuota, 8},
	"WeightServiceAccount":     {object.WeightServiceAccount, 10},
	"WeightRole":               {object.WeightRole, 10},
	"WeightRoleBinding":        {object.WeightRoleBinding, 10},
	"WeightSecret":             {object.WeightSecret, 15},
	"WeightConfigMap":          {object.WeightConfigMap, 15},
	"WeightService":            {object.WeightService, 50},
	"WeightLimitRange":         {object.WeightLimitRange, 60},
	"WeightDeployment":         {object.WeightDeployment, 100},
	"WeightStatefulSet":        {object.WeightStatefulSet, 100},
	"WeightCronJob":            {object.WeightCronJob, 105},
	"WeightPDB":                {object.WeightPDB, 108},
	"WeightPersistentVolume":   {object.WeightPersistentVolume, 1000},
	"WeightPVC":                {object.WeightPVC, 1000},
	"WeightDaemonSet":          {object.WeightDaemonSet, 1000},
	"WeightJob":                {object.WeightJob, 1000},
	"WeightIngress":            {object.WeightIngress, 1000},
	"WeightNetworkPolicy":      {object.WeightNetworkPolicy, 1000},
	"WeightHPA":                {object.WeightHPA, 1000},
	"WeightVPA":                {object.WeightVPA, 1000},
	"WeightDefault":            {object.WeightDefault, 1000},
	"WeightWebhook":            {object.WeightWebhook, 2000},
}

// pinnedGVKWeights is every exact group-version-kind entry of the table.
var pinnedGVKWeights = map[schema.GroupVersionKind]int{
	{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}: -100,

	{Group: "", Version: "v1", Kind: "Namespace"}:             0,
	{Group: "", Version: "v1", Kind: "ResourceQuota"}:         8,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:        10,
	{Group: "", Version: "v1", Kind: "Secret"}:                15,
	{Group: "", Version: "v1", Kind: "ConfigMap"}:             15,
	{Group: "", Version: "v1", Kind: "Service"}:               50,
	{Group: "", Version: "v1", Kind: "LimitRange"}:            60,
	{Group: "", Version: "v1", Kind: "PersistentVolume"}:      1000,
	{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"}: 1000,

	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}:        5,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}: 7,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}:               10,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}:        10,

	{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"}:     6,
	{Group: "scheduling.k8s.io", Version: "v1", Kind: "PriorityClass"}: 6,
	{Group: "node.k8s.io", Version: "v1", Kind: "RuntimeClass"}:        6,
	{Group: "networking.k8s.io", Version: "v1", Kind: "IngressClass"}:  6,

	{Group: "apps", Version: "v1", Kind: "Deployment"}:  100,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: 100,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:   1000,
	{Group: "apps", Version: "v1", Kind: "ReplicaSet"}:  1000,

	{Group: "batch", Version: "v1", Kind: "Job"}:     1000,
	{Group: "batch", Version: "v1", Kind: "CronJob"}: 105,

	{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}:       1000,
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"}: 1000,

	{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}:      1000,
	{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"}:      1000,
	{Group: "autoscaling.k8s.io", Version: "v1", Kind: "VerticalPodAutoscaler"}: 1000,

	{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"}: 108,

	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}: 2000,
	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}:   2000,
}

// pinnedDefinitionWeights is the three cluster definitions, which keep their
// weight in any version of their own group. Each is read here through a
// version the exact rows do not hold.
var pinnedDefinitionWeights = map[schema.GroupVersionKind]int{
	{Group: "apiextensions.k8s.io", Version: "v9", Kind: "CustomResourceDefinition"}: -100,
	{Group: "", Version: "v9", Kind: "Namespace"}:                                    0,
	{Group: "rbac.authorization.k8s.io", Version: "v9", Kind: "ClusterRole"}:         5,
}

// pinnedKindWeights is every kind-only entry of the table, read through a
// group and version no other row holds. A kind with a cluster definition's
// name in another group weighs as a class kind.
var pinnedKindWeights = map[string]int{
	"CustomResourceDefinition":       6,
	"Namespace":                      6,
	"ClusterRole":                    6,
	"ClusterClass":                   6,
	"RuntimeClass":                   6,
	"PriorityClass":                  6,
	"StorageClass":                   6,
	"VolumeSnapshotClass":            6,
	"IngressClass":                   6,
	"GatewayClass":                   6,
	"ClusterRoleBinding":             7,
	"ResourceQuota":                  8,
	"ServiceAccount":                 10,
	"Role":                           10,
	"RoleBinding":                    10,
	"Secret":                         15,
	"ConfigMap":                      15,
	"Service":                        50,
	"LimitRange":                     60,
	"Deployment":                     100,
	"StatefulSet":                    100,
	"CronJob":                        105,
	"PodDisruptionBudget":            108,
	"PersistentVolume":               1000,
	"PersistentVolumeClaim":          1000,
	"DaemonSet":                      1000,
	"ReplicaSet":                     1000,
	"Job":                            1000,
	"Ingress":                        1000,
	"NetworkPolicy":                  1000,
	"HorizontalPodAutoscaler":        1000,
	"VerticalPodAutoscaler":          1000,
	"ValidatingWebhookConfiguration": 2000,
	"MutatingWebhookConfiguration":   2000,
}

// pinnedFallbacks are the fallback rows: an unknown kind weighs the default,
// an unknown version of a known kind weighs the kind, and a kind no row
// holds whose name ends in "Class" (case-sensitive) weighs as a class kind.
var pinnedFallbacks = map[schema.GroupVersionKind]int{
	{Group: "example.com", Version: "v1", Kind: "Foo"}:                          1000,
	{Group: "autoscaling", Version: "v2beta2", Kind: "HorizontalPodAutoscaler"}: 1000,
	{Group: "policy", Version: "v1beta1", Kind: "PodDisruptionBudget"}:          108,
	{Group: "example.com", Version: "v1", Kind: "WidgetClass"}:                  6,
	{Group: "example.com", Version: "v1", Kind: "Widgetclass"}:                  1000,
}

func kindOnlyGVK(kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "fallback.invalid", Version: "v9", Kind: kind}
}

func TestWeightTableMatchesPinnedCopy(t *testing.T) {
	t.Run("constants", func(t *testing.T) {
		for name, c := range pinnedWeightConstants {
			assert.Equal(t, c.want, c.lib, "object.%s", name)
		}
	})
	t.Run("gvk entries", func(t *testing.T) {
		for gvk, want := range pinnedGVKWeights {
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
	t.Run("cluster definitions in another version", func(t *testing.T) {
		for gvk, want := range pinnedDefinitionWeights {
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
	t.Run("kind entries", func(t *testing.T) {
		for kind, want := range pinnedKindWeights {
			gvk := kindOnlyGVK(kind)
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
	t.Run("fallbacks", func(t *testing.T) {
		for gvk, want := range pinnedFallbacks {
			assert.Equal(t, want, object.Weight(gvk), "object.Weight(%s)", gvk)
		}
	})
}

// orderSet is one shuffled set covering every weight class, two ConfigMaps
// and two Deployments in a fixed input order, and four kinds of the default
// weight (an unknown kind, a Job, an Ingress and a PersistentVolumeClaim),
// which keep their input order among themselves.
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
		mk("rbac.authorization.k8s.io/v1", "ClusterRoleBinding", "crb"),
		mk("v1", "ResourceQuota", "quota"),
		mk("v1", "LimitRange", "limits"),
		mk("batch/v1", "CronJob", "cronjob"),
		mk("v1", "PersistentVolumeClaim", "pvc"),
	}
}

func objNames(objs []*unstructured.Unstructured) []string {
	out := make([]string, len(objs))
	for i, o := range objs {
		out[i] = o.GetName()
	}
	return out
}

// pinnedSortOrder is the name sequence the sort gives for orderSet, per
// direction: the order the cli applies in (ascending) and the order it
// deletes and prunes in (descending).
var pinnedSortOrder = map[object.Direction][]string{
	object.Ascending: {
		"crd", "ns", "cr", "sc", "crb", "quota", "sa", "cm-b", "cm-a", "svc",
		"limits", "deploy-b", "deploy-a", "cronjob", "pdb",
		"unknown", "job", "ingress", "pvc", "webhook",
	},
	object.Descending: {
		"webhook", "unknown", "job", "ingress", "pvc",
		"pdb", "cronjob", "deploy-b", "deploy-a", "limits",
		"svc", "cm-b", "cm-a", "sa", "quota", "crb", "sc", "cr", "ns", "crd",
	},
}

func TestSortMatchesPinnedOrder(t *testing.T) {
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
			assert.Equal(t, pinnedSortOrder[tc.dir], objNames(objs))
		})
	}
}

// TestSortObjectsStableOnLargeInput pins that SortObjects keeps equal-weight
// objects in their input order, in both directions. The input is large
// enough (96 objects) that an unstable sort reorders equal elements, which
// the small orderSet does not reveal. Names count down while the input
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
