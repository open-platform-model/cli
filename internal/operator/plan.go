package operator

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
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
	kubernetes.SortObjects(plan, resourceorder.Ascending)
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
	kubernetes.SortObjects(plan, resourceorder.Ascending)
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
	kubernetes.SortObjects(plan, resourceorder.Descending)
	return plan
}
