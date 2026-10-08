package kubernetes

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/ownership"
)

// DeletionOptions configures one run of a deletion plan.
type DeletionOptions struct {
	// DryRun performs the reads and sends no delete. A delete the plan
	// names is recorded as it would be after a successful delete.
	DryRun bool

	// StopOnDiscoveryFailure answers every read after the first failed API
	// discovery request (IsDiscoveryFailure) with that failure, without a
	// request: the cluster cannot say where the remaining objects live.
	StopOnDiscoveryFailure bool

	// Unreadable holds the read error of objects the caller already failed
	// to read. The plan's read of such an object is answered with that
	// error and no second request is sent.
	Unreadable map[ownership.Object]error

	// OnStep, when set, is called for each step as it finishes, in plan
	// order, so a caller prints its lines while the run goes on.
	OnStep func(StepResult)
}

// StepResult is what happened to one step of a deletion plan.
type StepResult struct {
	// Entry is the step's object.
	Entry k8sinventory.Entry

	// Outcome is the plan's record of the step: deleted, skipped (with the
	// reason and the library's message) or failed.
	Outcome lifecycle.Outcome

	// Failed is the action that failed, lifecycle.ActionRead or
	// lifecycle.ActionDelete; empty when the step did not fail.
	Failed lifecycle.ActionKind

	// Err is the error of the failed action, for callers that tell errors
	// apart by type (IsDiscoveryFailure, IsKindNotServed, ErrReplaced). A
	// delete refused on its UID precondition wraps ErrReplaced. Nil when
	// the step did not fail.
	Err error

	// UID is the UID the accepted delete of the step was sent with, the one
	// of the object that was read and judged. Empty when no delete was
	// accepted (a dry run, a skip, a failure) or the plan named no UID
	// precondition. WaitDeleted uses it to tell the deleted object from a
	// new one under the same name.
	UID types.UID
}

// DeletionRun is a deletion plan driven as far as it went: the plan, its
// last state and one result per finished step, in plan order.
type DeletionRun struct {
	Plan  lifecycle.DeletionPlan
	State lifecycle.State
	Steps []StepResult
}

// RunDeletion drives plan to its end with the CLI's client. It is the one
// way the CLI deletes an object of an instance: the library's deletion
// transition (lifecycle.Advance) names every read, delete and skip, judging
// each object with the delete verdict, and RunDeletion performs the reads
// and deletes it names, each read under the resource the cluster serves the
// kind as, each delete with the propagation policy and the UID precondition
// the transition gives. It adds no delete of its own.
//
// A failed read or delete is a failed step and the plan goes on. The
// returned error is non-nil only when the transition refuses the state,
// which a run from the zero state cannot cause; the run so far is returned
// with it.
func RunDeletion(ctx context.Context, client *Client, plan lifecycle.DeletionPlan, opts DeletionOptions) (DeletionRun, error) {
	run := DeletionRun{Plan: plan}
	p := performer{client: client, opts: opts}
	steps := plan.Steps()

	var (
		ev      lifecycle.Event
		last    lifecycle.ActionKind // the action ev answers
		lastErr error                // its error as the caller sees it
		lastUID types.UID            // the UID a sent delete carried
	)
	for {
		next, act, err := lifecycle.Advance(plan, run.State, ev)
		if err != nil {
			return run, fmt.Errorf("advancing the deletion plan: %w", err)
		}
		for _, o := range next.Outcomes[len(run.State.Outcomes):] {
			res := stepResult(steps[o.Step].Entry, o, last, lastErr, lastUID)
			run.Steps = append(run.Steps, res)
			if opts.OnStep != nil {
				opts.OnStep(res)
			}
		}
		run.State = next
		ev, last, lastErr, lastUID = lifecycle.Event{}, act.Kind, nil, ""

		switch act.Kind {
		case lifecycle.ActionDone:
			return run, nil
		case lifecycle.ActionRead:
			ev.Live, ev.Err = p.read(ctx, act.Entry)
			lastErr = ev.Err
		case lifecycle.ActionDelete:
			if opts.DryRun {
				continue
			}
			ev.Err = p.sendDelete(ctx, act)
			lastErr = ev.Err
			lastUID = preconditionUID(act)
			if ev.Err != nil && apierrors.IsConflict(ev.Err) && act.Preconditions != nil {
				lastErr = fmt.Errorf("%w: %w", ErrReplaced, ev.Err)
			}
		case lifecycle.ActionSkip:
			// Recorded by the transition; nothing to perform.
		}
	}
}

// stepResult is the result of a step the transition just finished with
// outcome o. last, lastErr and lastUID describe the action the transition was
// answering: its kind, its error as the caller sees it, and the UID a sent
// delete carried.
func stepResult(entry k8sinventory.Entry, o lifecycle.Outcome, last lifecycle.ActionKind, lastErr error, lastUID types.UID) StepResult {
	res := StepResult{Entry: entry, Outcome: o}
	switch o.Result {
	case lifecycle.ResultDeleted:
		// Recorded only on the answer to the step's own delete.
		res.UID = lastUID
	case lifecycle.ResultFailed:
		// A step fails only on the answer to its own read or delete.
		res.Failed = last
		res.Err = lastErr
		if res.Err == nil {
			// The transition refused what the read returned.
			res.Err = errors.New(o.Message)
		}
	case lifecycle.ResultSkipped:
	}
	return res
}

// preconditionUID is the UID precondition of a delete the plan named; empty
// when it named none.
func preconditionUID(act lifecycle.Action) types.UID {
	if act.Preconditions == nil || act.Preconditions.UID == nil {
		return ""
	}
	return *act.Preconditions.UID
}

// performer performs the reads and deletes a deletion plan names.
type performer struct {
	client *Client
	opts   DeletionOptions
	// discoveryErr is the first failed discovery request, kept when
	// opts.StopOnDiscoveryFailure is set.
	discoveryErr error
}

// read returns the live object of entry; a NotFound error when it is gone.
func (p *performer) read(ctx context.Context, entry k8sinventory.Entry) (*unstructured.Unstructured, error) {
	if err, ok := p.opts.Unreadable[entryObject(entry)]; ok && err != nil {
		return nil, err
	}
	if p.discoveryErr != nil {
		return nil, p.discoveryErr
	}
	resource, err := p.client.ResourceClientFor(ctx, entryGVK(entry), entry.Namespace)
	if err != nil {
		if p.opts.StopOnDiscoveryFailure && IsDiscoveryFailure(err) {
			p.discoveryErr = err
		}
		return nil, err
	}
	return resource.Get(ctx, entry.Name, metav1.GetOptions{})
}

// sendDelete sends the DELETE the plan named, exactly as it named it.
func (p *performer) sendDelete(ctx context.Context, act lifecycle.Action) error {
	resource, err := p.client.ResourceClientFor(ctx, entryGVK(act.Entry), act.Entry.Namespace)
	if err != nil {
		return err
	}
	propagation := act.Propagation
	return resource.Delete(ctx, act.Entry.Name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
		Preconditions:     act.Preconditions,
	})
}

// entryObject is the ownership identity of an inventory entry.
func entryObject(e k8sinventory.Entry) ownership.Object {
	return ownership.Object{Group: e.Group, Kind: e.Kind, Namespace: e.Namespace, Name: e.Name}
}

// entryGVK is the group, version and kind an inventory entry records.
func entryGVK(e k8sinventory.Entry) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: e.Group, Version: e.Version, Kind: e.Kind}
}
