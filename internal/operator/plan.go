package operator

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/pkg/resourceorder"
)

const (
	kindCustomResourceDefinition = "CustomResourceDefinition"
	kindNamespace                = "Namespace"
)

// InstallPlan returns every manifest document ordered ascending by resource
// weight (CRDs first, workloads last) — the order the objects must be applied in.
func InstallPlan(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	plan := append([]*unstructured.Unstructured(nil), objs...)
	sortByWeightAscending(plan)
	return plan
}

// CRDsOnlyPlan returns only the CustomResourceDefinition documents from objs,
// ordered ascending by resource weight.
func CRDsOnlyPlan(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	var plan []*unstructured.Unstructured
	for _, obj := range objs {
		if obj.GetKind() == kindCustomResourceDefinition {
			plan = append(plan, obj)
		}
	}
	sortByWeightAscending(plan)
	return plan
}

// UninstallPlan returns every manifest document except CustomResourceDefinitions
// and the Namespace, ordered descending by resource weight (matching delete.go's
// teardown convention). CRDs and the Namespace are deliberately excluded here —
// uninstall must never remove them.
func UninstallPlan(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	var plan []*unstructured.Unstructured
	for _, obj := range objs {
		if kind := obj.GetKind(); kind == kindCustomResourceDefinition || kind == kindNamespace {
			continue
		}
		plan = append(plan, obj)
	}
	sortByWeightDescending(plan)
	return plan
}

func sortByWeightAscending(objs []*unstructured.Unstructured) {
	sort.SliceStable(objs, func(i, j int) bool {
		return resourceorder.GetWeight(objs[i].GroupVersionKind()) < resourceorder.GetWeight(objs[j].GroupVersionKind())
	})
}

func sortByWeightDescending(objs []*unstructured.Unstructured) {
	sort.SliceStable(objs, func(i, j int) bool {
		return resourceorder.GetWeight(objs[i].GroupVersionKind()) > resourceorder.GetWeight(objs[j].GroupVersionKind())
	})
}
