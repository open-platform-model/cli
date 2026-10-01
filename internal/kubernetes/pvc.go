package kubernetes

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/output"
)

// pvcStoragePath is where a PersistentVolumeClaim carries its requested size.
var pvcStoragePath = []string{"spec", "resources", "requests", "storage"}

// guardPVCResize keeps an apply of an existing, Bound PersistentVolumeClaim
// from asking for a size the API server refuses. The server rejects a size
// change on a Bound claim unless its StorageClass allows volume expansion, and
// never accepts a smaller size; one such rejection fails the whole apply and
// with it the prune and the inventory write.
//
// When the rendered size differs from the live size and either the new size is
// smaller or the StorageClass does not allow expansion, the returned object
// carries the live size instead and a warning names both. The field stays in
// the applied object so server-side apply keeps the CLI's ownership of it; a
// growth on an expandable class passes through untouched. A StorageClass that
// is missing, unnamed or unreadable counts as not expandable.
//
// obj is returned as is when no guard applies, and otherwise as a copy: the
// caller's rendered object is never modified.
func guardPVCResize(ctx context.Context, client *Client, obj, live *unstructured.Unstructured) *unstructured.Unstructured {
	if obj.GetKind() != "PersistentVolumeClaim" || obj.GroupVersionKind().Group != "" {
		return obj
	}
	if phase, _, _ := unstructured.NestedString(live.Object, "status", "phase"); phase != "Bound" { //nolint:errcheck // wrong-typed phase reads as not Bound
		return obj
	}

	liveSize, _, _ := unstructured.NestedString(live.Object, pvcStoragePath...) //nolint:errcheck // wrong-typed size reads as absent
	newSize, _, _ := unstructured.NestedString(obj.Object, pvcStoragePath...)   //nolint:errcheck // wrong-typed size reads as absent
	liveQty, err := resource.ParseQuantity(liveSize)
	if err != nil {
		return obj
	}
	newQty, err := resource.ParseQuantity(newSize)
	if err != nil {
		return obj
	}
	cmp := newQty.Cmp(liveQty)
	if cmp == 0 {
		return obj
	}

	var reason string
	if cmp < 0 {
		reason = "a PersistentVolumeClaim cannot shrink"
	} else {
		className, _, _ := unstructured.NestedString(live.Object, "spec", "storageClassName") //nolint:errcheck // wrong-typed name reads as unnamed
		expandable, why := storageClassExpandable(ctx, client, className)
		if expandable {
			return obj
		}
		reason = why
	}

	output.Warn(fmt.Sprintf("PersistentVolumeClaim %s keeps its current size %s; the rendered size %s was not applied: %s",
		pvcRef(obj), liveSize, newSize, reason))

	guarded := obj.DeepCopy()
	if err := unstructured.SetNestedField(guarded.Object, liveSize, pvcStoragePath...); err != nil {
		return obj
	}
	return guarded
}

// storageClassExpandable reports whether the named StorageClass allows volume
// expansion. When it does not, or cannot be shown to, why says so.
func storageClassExpandable(ctx context.Context, client *Client, className string) (expandable bool, why string) {
	if className == "" {
		return false, "the claim names no StorageClass, so volume expansion cannot be confirmed"
	}
	if client.Clientset == nil {
		return false, fmt.Sprintf("StorageClass %q could not be read, so volume expansion cannot be confirmed", className)
	}
	sc, err := client.Clientset.StorageV1().StorageClasses().Get(ctx, className, metav1.GetOptions{})
	switch {
	case apierrors.IsNotFound(err):
		return false, fmt.Sprintf("StorageClass %q does not exist, so volume expansion cannot be confirmed", className)
	case err != nil:
		return false, fmt.Sprintf("StorageClass %q could not be read (%v), so volume expansion cannot be confirmed", className, err)
	}
	if sc.AllowVolumeExpansion == nil || !*sc.AllowVolumeExpansion {
		return false, fmt.Sprintf("StorageClass %q does not allow volume expansion", className)
	}
	return true, ""
}

func pvcRef(obj *unstructured.Unstructured) string {
	if ns := obj.GetNamespace(); ns != "" {
		return ns + "/" + obj.GetName()
	}
	return obj.GetName()
}
