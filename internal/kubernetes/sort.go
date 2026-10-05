package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/object"
)

// SortObjects is the library's object.Sort over unstructured objects: a
// stable sort in place by the library weight table, ascending (apply) or
// descending (delete).
func SortObjects(objs []*unstructured.Unstructured, dir object.Direction) {
	object.Sort(objs, (*unstructured.Unstructured).GroupVersionKind, dir)
}
