package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/pkg/resourceorder"
)

// SortObjects is resourceorder.Sort over unstructured objects: a stable sort
// in place by resource weight, ascending (apply) or descending (delete).
func SortObjects(objs []*unstructured.Unstructured, dir resourceorder.Direction) {
	resourceorder.Sort(objs, (*unstructured.Unstructured).GroupVersionKind, dir)
}
