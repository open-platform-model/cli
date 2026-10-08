package inventory

import (
	"context"
	"fmt"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

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

// SplitDataClaims partitions a stale set into the entries prune may delete
// and the PersistentVolumeClaims (kubernetes.IsDataClaim) it keeps unless the
// user passes --delete-data, keeping the input order in both halves. Unlike a
// protected entry, a kept claim stays in the record the caller writes, so a
// later apply with the flag finds it stale and prunes it.
func SplitDataClaims(stale []k8sinventory.Entry) (prunable, claims []k8sinventory.Entry) {
	for _, e := range stale {
		if kubernetes.IsDataClaim(e.Group, e.Kind) {
			claims = append(claims, e)
			continue
		}
		prunable = append(prunable, e)
	}
	return prunable, claims
}

// ClaimsInCluster returns the claims prune keeps that are still, or may still
// be, in the cluster, in input order. A claim whose read answers NotFound is
// dropped: nothing is left to keep, so the caller neither reports it nor
// records it, as a prune treats an object that is already gone. Any other
// read error keeps the claim, since it may still hold data.
func ClaimsInCluster(ctx context.Context, client *kubernetes.Client, claims []k8sinventory.Entry) []k8sinventory.Entry {
	var present []k8sinventory.Entry
	for _, e := range claims {
		if _, err := getEntry(ctx, client, e); apierrors.IsNotFound(err) {
			output.Debug("stale claim already gone", "namespace", e.Namespace, "name", e.Name)
			continue
		}
		present = append(present, e)
	}
	return present
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

// LeftBehind is a stale entry the prune did not delete because the library's
// delete verdict skipped it: its live object is not OPM-managed, belongs to
// another instance, or is adopted by another instance. Reason is the
// library's message. The object is not the instance's to track, so a caller
// that records an inventory after the prune leaves the entry out.
type LeftBehind struct {
	Entry  k8sinventory.Entry
	Reason string
}

// PruneStaleResources deletes the stale resources the instance still owns,
// as the library's deletion plan orders and judges them
// (kubernetes.RunDeletion): highest weight first, each live object read and
// the library's delete verdict asked with instanceUUID, which is the
// identity stored in the instance's record, the one that applied the stale
// objects, and never the identity of the current render: after a module
// moved to a new path the two differ, and the stale objects carry the
// recorded one. An entry the verdict skips is returned in leftBehind and is
// not deleted. A delete carries a precondition on the UID that was read.
// A prune holds no record, so it asks no hold verdict: the caller writes the
// record afterwards and keeps the failed entries in it.
//
// A core Namespace or a CRD (kubernetes.IsProtectedKind) is never deleted,
// even when the caller passes one; callers that report what was left behind
// split the set first with SplitProtected.
// A PersistentVolumeClaim is judged like any other entry here: the caller
// decides on --delete-data and removes the claims it keeps from stale first
// (SplitDataClaims).
// An object that is already gone is a success (idempotent). An entry whose
// kind is not served at the recorded version is a failed delete, never a
// success: the object may still be in the cluster. So is an entry whose live
// read fails, and one whose object was replaced between the read and the
// delete (kubernetes.ErrReplaced).
//
// A delete that fails does not stop the prune. A failed API discovery request
// ends its requests: that entry and every entry not yet tried are reported
// as failed, with the discovery error. When any failed, the error is a
// *PruneError naming the entries that are still in the cluster.
func PruneStaleResources(ctx context.Context, client *kubernetes.Client, stale []k8sinventory.Entry, instanceUUID string) (leftBehind []LeftBehind, err error) {
	if len(stale) == 0 {
		return nil, nil
	}

	var failed PruneError
	plan := lifecycle.NewDeletionPlan(stale, lifecycle.Policy{Prune: true}, instanceUUID)
	_, err = kubernetes.RunDeletion(ctx, client, plan, kubernetes.DeletionOptions{
		StopOnDiscoveryFailure: true,
		OnStep: func(step kubernetes.StepResult) {
			entry := step.Entry
			switch {
			case step.Outcome.Result == lifecycle.ResultFailed:
				failed.Failed = append(failed.Failed, entry)
				failed.Errs = append(failed.Errs, fmt.Errorf("deleting %s/%s: %w", entry.Kind, entry.Name, step.Err))
			case step.Outcome.Skip == ownership.SkipAlreadyAbsent:
				output.Debug("stale resource already gone", "kind", entry.Kind, "namespace", entry.Namespace, "name", entry.Name)
			case step.Outcome.Skip == ownership.SkipSafetyExcluded:
				output.Debug("leaving protected resource behind", "kind", entry.Kind, "name", entry.Name)
			case step.Outcome.Skip != "":
				leftBehind = append(leftBehind, LeftBehind{Entry: entry, Reason: step.Outcome.Message})
			default:
				output.Debug("pruned stale resource", "kind", entry.Kind, "namespace", entry.Namespace, "name", entry.Name)
			}
		},
	})
	if err != nil {
		return leftBehind, err
	}

	if len(failed.Failed) > 0 {
		return leftBehind, &failed
	}
	return leftBehind, nil
}

// entryObject is the ownership identity of an inventory entry.
func entryObject(e k8sinventory.Entry) ownership.Object {
	return ownership.Object{Group: e.Group, Kind: e.Kind, Namespace: e.Namespace, Name: e.Name}
}

// entryGVK is the group, version and kind an inventory entry records.
func entryGVK(entry k8sinventory.Entry) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: entry.Group, Version: entry.Version, Kind: entry.Kind}
}

// getEntry reads the live object of an inventory entry, under the resource
// the cluster serves its kind as.
func getEntry(ctx context.Context, client *kubernetes.Client, entry k8sinventory.Entry) (*unstructured.Unstructured, error) {
	resource, err := client.ResourceClientFor(ctx, entryGVK(entry), entry.Namespace)
	if err != nil {
		return nil, err
	}
	return resource.Get(ctx, entry.Name, metav1.GetOptions{})
}
