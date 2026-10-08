package inventory

import (
	"context"
	"fmt"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/object"
)

// PreApplyExistenceCheck verifies that resources do not conflict with existing
// cluster state on a first-time apply (no previous inventory).
//
// For each rendered resource entry, a GET is performed:
//   - If the resource exists with a deletionTimestamp → error (terminating)
//   - If the resource exists without OPM managed-by label → error (untracked),
//     unless admit holds it
//   - If the resource does not exist → OK
//   - If the read fails with anything but NotFound → error (unreadable), with
//     the read error in the chain
//
// admit passes the untracked test only, never the terminating one and never
// the unreadable one. Only
// `opm operator install` passes a non-empty set: the objects it proved came
// from an earlier operator release manifest, or that carry the operator
// instance's identity (0012:D8:R6). Every other caller passes nil.
//
// This check should be skipped entirely when a previous inventory exists.
func PreApplyExistenceCheck(ctx context.Context, client *kubernetes.Client, entries []k8sinventory.Entry, admit AdmitSet) error {
	for _, entry := range entries {
		gvr := schema.GroupVersionResource{
			Group:    entry.Group,
			Version:  entry.Version,
			Resource: kubernetes.KindToResource(entry.Kind),
		}

		var obj interface{ GetDeletionTimestamp() *metav1.Time }
		unstrObj, err := client.ResourceClient(gvr, entry.Namespace).Get(ctx, entry.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue // Resource doesn't exist — OK for first install
			}
			// Any other answer leaves the question open, and the forced apply
			// that follows would take over whatever holds the name.
			return fmt.Errorf("cannot check whether %s/%s in namespace %q already exists: %w\n"+
				"apply stopped before any rendered resource was applied. Check that you can read that resource, then run the command again",
				entry.Kind, entry.Name, entry.Namespace, err)
		}

		obj = unstrObj

		// Check for terminating resources
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("resource %s/%s in namespace %q is terminating (deletionTimestamp set) — wait for deletion to complete before applying",
				entry.Kind, entry.Name, entry.Namespace)
		}

		// Check for untracked resources (not managed by OPM).
		// Accepts any known OPM actor value (opm-cli, opm-controller, or
		// legacy open-platform-model) for backward compatibility.
		labels := unstrObj.GetLabels()
		if !opmlabels.IsOPMManagedBy(labels[opmlabels.ManagedBy]) && !admit.Has(entry) {
			return fmt.Errorf("resource %s/%s in namespace %q already exists and is not managed by OPM — remove or rename it, or change the module to render a different name",
				entry.Kind, entry.Name, entry.Namespace)
		}
	}
	return nil
}

// SplitProtected partitions a stale set into the entries prune may delete and
// the entries it always leaves behind (kubernetes.IsProtectedKind: core
// Namespaces and CRDs), keeping the input order in both halves.
func SplitProtected(stale []k8sinventory.Entry) (prunable, protected []k8sinventory.Entry) {
	for _, e := range stale {
		if kubernetes.IsProtectedKind(e.Group, e.Kind) {
			protected = append(protected, e)
			continue
		}
		prunable = append(prunable, e)
	}
	return prunable, protected
}

// PruneError reports the stale resources a prune could not delete. The
// objects are still in the cluster, so a caller that records an inventory
// after the prune must keep Failed in it.
type PruneError struct {
	// Failed are the entries whose delete failed, in delete order.
	Failed []k8sinventory.Entry
	// Errs holds the delete error of each entry in Failed, by index.
	Errs []error
}

func (e *PruneError) Error() string {
	if len(e.Errs) == 0 {
		return "pruning stale resources failed"
	}
	return fmt.Sprintf("pruning stale resources: %d error(s): %v", len(e.Errs), e.Errs[0])
}

// Unwrap exposes every delete error to errors.Is and errors.As.
func (e *PruneError) Unwrap() []error {
	return e.Errs
}

// PruneStaleResources deletes the stale resources from the cluster.
// Resources are deleted in reverse weight order (highest weight first).
// A core Namespace or a CRD (kubernetes.IsProtectedKind) is never deleted,
// even when the caller passes one; callers that report what was left behind
// split the set first with SplitProtected.
// 404 (not found) errors are treated as success (idempotent).
//
// A delete that fails does not stop the loop. When any failed, the error is a
// *PruneError naming the entries that are still in the cluster.
func PruneStaleResources(ctx context.Context, client *kubernetes.Client, stale []k8sinventory.Entry) error {
	if len(stale) == 0 {
		return nil
	}

	// Sort in reverse weight order (highest weight deleted first)
	sorted := make([]k8sinventory.Entry, len(stale))
	copy(sorted, stale)
	object.Sort(sorted, func(e k8sinventory.Entry) schema.GroupVersionKind {
		return schema.GroupVersionKind{Group: e.Group, Version: e.Version, Kind: e.Kind}
	}, object.Descending)

	var failed PruneError
	for _, entry := range sorted {
		if kubernetes.IsProtectedKind(entry.Group, entry.Kind) {
			output.Debug("leaving protected resource behind", "kind", entry.Kind, "name", entry.Name)
			continue
		}

		gvr := schema.GroupVersionResource{
			Group:    entry.Group,
			Version:  entry.Version,
			Resource: kubernetes.KindToResource(entry.Kind),
		}

		propagation := metav1.DeletePropagationForeground
		err := client.ResourceClient(gvr, entry.Namespace).Delete(ctx, entry.Name, metav1.DeleteOptions{
			PropagationPolicy: &propagation,
		})

		if err != nil && !apierrors.IsNotFound(err) {
			failed.Failed = append(failed.Failed, entry)
			failed.Errs = append(failed.Errs, fmt.Errorf("deleting %s/%s: %w", entry.Kind, entry.Name, err))
			continue
		}

		output.Debug("pruned stale resource", "kind", entry.Kind, "namespace", entry.Namespace, "name", entry.Name)
	}

	if len(failed.Failed) > 0 {
		return &failed
	}
	return nil
}
