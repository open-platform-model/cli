package inventory

import (
	"context"
	"fmt"
	"strings"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/k8s/ownership"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// GuardInput is what the apply guard judges.
type GuardInput struct {
	// Entries are the rendered objects to judge, in render order.
	Entries []k8sinventory.Entry
	// Previous is the inventory of the instance's record; nil on a first
	// apply. An entry that is the same object as one of these is judged as
	// inventoried.
	Previous []k8sinventory.Entry
	// InstanceUUID is the identity of the render.
	InstanceUUID string
	// RefuseLetGo makes an object adopted by another instance a refusal.
	// Only `opm operator install` sets it: it needs every object it renders.
	RefuseLetGo bool
}

// LetGo is a rendered object whose adopt annotation names another instance:
// this instance does not apply it and does not record it. Message is the
// library's text.
type LetGo struct {
	Entry   k8sinventory.Entry
	Message string
}

// Refused is a rendered object the ownership verdict refuses, with the
// library's reason and message.
type Refused struct {
	Entry   k8sinventory.Entry
	Reason  ownership.ApplyRefusal
	Message string
}

// GuardResult is what the guard found among the objects it did not refuse.
type GuardResult struct {
	// Managed are the allowed entries whose object exists under OPM
	// management and is not in Previous, in entry order: the apply takes
	// them into the inventory although no record listed them.
	Managed []k8sinventory.Entry
	// LetGo are the entries refused as adopted by another instance, when
	// RefuseLetGo is not set.
	LetGo []LetGo
}

// GuardRefusalError is the guard's refusal: every refused object, in entry
// order.
type GuardRefusalError struct {
	Refused []Refused
}

func (e *GuardRefusalError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d object(s) cannot be applied by this instance:", len(e.Refused))
	for i := range e.Refused {
		b.WriteString("\n  " + e.Refused[i].Message)
	}
	return b.String()
}

// Has reports whether any object was refused for reason.
func (e *GuardRefusalError) Has(reason ownership.ApplyRefusal) bool {
	for i := range e.Refused {
		if e.Refused[i].Reason == reason {
			return true
		}
	}
	return false
}

// Guard is the apply guard: it reads every entry's live object and asks the
// library's apply verdict (ownership.CanApply) whether this instance may
// apply over it (0012:D4:R2, 0012:D8:R1). It writes nothing.
//
// An object that does not exist, or whose kind the cluster does not serve
// (typically the module renders its CustomResourceDefinition in the same
// apply), passes. A read or discovery failure with any other answer is
// returned at once with the error in the chain: the question stays open, and
// the forced apply that follows would take over whatever holds the name.
//
// When the verdict refuses any object, the error is a *GuardRefusalError holding
// all of them. The result is filled in that case too, for a caller that only
// looks (a dry run).
func Guard(ctx context.Context, client *kubernetes.Client, in GuardInput) (GuardResult, error) {
	recorded := make(map[ownership.Object]struct{}, len(in.Previous))
	for _, e := range in.Previous {
		recorded[entryObject(e)] = struct{}{}
	}

	var (
		result  GuardResult
		refusal GuardRefusalError
	)
	for _, entry := range in.Entries {
		live, err := getEntry(ctx, client, entry)
		if err != nil {
			if apierrors.IsNotFound(err) || kubernetes.IsKindNotServed(err) {
				continue
			}
			return GuardResult{}, fmt.Errorf("cannot check whether %s/%s in namespace %q already exists: %w\n"+
				"Check that you can read that resource, then run the command again",
				entry.Kind, entry.Name, entry.Namespace, err)
		}

		obj := entryObject(entry)
		_, inInventory := recorded[obj]
		verdict := ownership.CanApply(ownership.ApplyInput{
			Object:       obj,
			Live:         live,
			InInventory:  inInventory,
			InstanceUUID: in.InstanceUUID,
		})
		switch {
		case verdict.Allowed():
			if !inInventory && opmlabels.IsOPMManagedBy(live.GetLabels()[opmlabels.ManagedBy]) {
				result.Managed = append(result.Managed, entry)
			}
		case verdict.Refuse == ownership.RefuseAdoptedElsewhere && !in.RefuseLetGo:
			result.LetGo = append(result.LetGo, LetGo{Entry: entry, Message: verdict.Message})
		default:
			refusal.Refused = append(refusal.Refused, Refused{Entry: entry, Reason: verdict.Refuse, Message: verdict.Message})
		}
	}

	if len(refusal.Refused) > 0 {
		return result, &refusal
	}
	return result, nil
}
