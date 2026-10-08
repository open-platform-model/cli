package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
)

// DeletedObject is an object whose delete the API server accepted.
type DeletedObject struct {
	Entry k8sinventory.Entry

	// UID is the UID of the object that was deleted; empty when the delete
	// carried no UID precondition.
	UID types.UID
}

// DeletedObjects lists the objects whose delete a deletion run sent and the
// API server accepted, in plan order. An object the plan skipped, one that
// was already gone and one whose read or delete failed are not among them. A
// dry run lists the deletes it would send, with no UID.
func DeletedObjects(run DeletionRun) []DeletedObject {
	var deleted []DeletedObject
	for i := range run.Steps {
		if step := &run.Steps[i]; step.Outcome.Result == lifecycle.ResultDeleted {
			deleted = append(deleted, DeletedObject{Entry: step.Entry, UID: step.UID})
		}
	}
	return deleted
}

// TerminatingObject is a deleted object that still exists.
type TerminatingObject struct {
	Entry k8sinventory.Entry

	// Finalizers are the finalizers of the object as it was last read. An
	// object stays until each one is removed.
	Finalizers []string

	// Err is the error of the last read when that read failed with an error
	// other than NotFound; the object then counts as still there.
	Err error
}

// TerminatingError reports that the budget of WaitDeleted ran out while
// deleted objects still existed.
type TerminatingError struct {
	// Elapsed is the time the wait took.
	Elapsed time.Duration
	// Objects are the objects that still exist, in the order given.
	Objects []TerminatingObject
}

func (e *TerminatingError) Error() string {
	return fmt.Sprintf("timed out after %s waiting for %d deleted resource(s) to be gone", e.Elapsed, len(e.Objects))
}

// WaitDeleted polls each deleted object until every one is gone, or ctx is
// done. An object is gone when its read returns NotFound, or when the live
// object has another UID than the one that was deleted: the name then belongs
// to a new object, and that one was not deleted. Any other read error, a kind
// that cannot be resolved included, keeps the object pending; it is never
// read as gone.
//
// It carries no timeout of its own: the deadline of ctx is the budget, and
// since marks when that budget started. When the deadline passes it returns a
// *TerminatingError naming what is left; when ctx is canceled, the context's
// error. The first poll is immediate, and the rest follow at
// WaitPollInterval.
func WaitDeleted(ctx context.Context, client *Client, objs []DeletedObject, since time.Time) error {
	pending := make([]pendingDelete, len(objs))
	for i, obj := range objs {
		pending[i] = pendingDelete{object: obj}
	}

	ticker := time.NewTicker(WaitPollInterval)
	defer ticker.Stop()

	for {
		pending = pollDeleted(ctx, client, pending)
		if len(pending) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				// Canceled by the caller (Ctrl-C), not by the budget.
				return ctx.Err()
			}
			left := make([]TerminatingObject, len(pending))
			for i := range pending {
				left[i] = pending[i].seen
				left[i].Entry = pending[i].object.Entry
			}
			return &TerminatingError{Elapsed: time.Since(since).Round(time.Second), Objects: left}
		case <-ticker.C:
		}
	}
}

// pendingDelete is a deleted object the wait has not yet seen gone, with
// what its last completed read showed.
type pendingDelete struct {
	object DeletedObject
	seen   TerminatingObject
}

// pollDeleted reads each pending object once and returns the ones that are
// still there. A read that ctx cut short changes nothing: what the read
// before it showed is kept for the report.
func pollDeleted(ctx context.Context, client *Client, pending []pendingDelete) []pendingDelete {
	left := make([]pendingDelete, 0, len(pending))
	for i := range pending {
		p := pending[i]
		entry := p.object.Entry
		finalizers, err := readDeleted(ctx, client, p.object)
		switch {
		case err == nil && finalizers == nil:
			continue
		case ctx.Err() != nil:
		case err != nil:
			p.seen.Err = fmt.Errorf("reading %s/%s: %w", entry.Kind, entry.Name, err)
		default:
			p.seen = TerminatingObject{Finalizers: finalizers}
		}
		left = append(left, p)
	}
	return left
}

// readDeleted reads a deleted object. It returns nil finalizers and no error
// when the object is gone, the finalizers (never nil) of the object while it
// is still there, or the error of a read that did not answer.
func readDeleted(ctx context.Context, client *Client, obj DeletedObject) ([]string, error) {
	resource, err := client.ResourceClientFor(ctx, entryGVK(obj.Entry), obj.Entry.Namespace)
	if err != nil {
		return nil, err
	}
	live, err := resource.Get(ctx, obj.Entry.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if obj.UID != "" && live.GetUID() != obj.UID {
		return nil, nil
	}
	return append([]string{}, live.GetFinalizers()...), nil
}
