package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/library/opm/k8s/health"
)

// WaitPollInterval is how often the wait loops re-check the cluster. A
// variable so tests can poll faster; only tests may change it.
var WaitPollInterval = 2 * time.Second

// conditionEstablished is the CustomResourceDefinition condition type that
// reports the API server serves the new kind.
const conditionEstablished = "Established"

// conditionStatusTrue is the Kubernetes condition status value representing "true".
const conditionStatusTrue = "True"

// ReadyPredicate reports whether a live object (as currently observed on the
// cluster) is ready.
type ReadyPredicate func(obj *unstructured.Unstructured) bool

// CRDEstablishedPredicate reports whether a CustomResourceDefinition has
// reached the Established=True condition.
func CRDEstablishedPredicate(obj *unstructured.Unstructured) bool {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return false
	}
	for _, raw := range conditions {
		c, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		condType, _, _ := unstructured.NestedString(c, "type")     //nolint:errcheck // best-effort condition parsing
		condStatus, _, _ := unstructured.NestedString(c, "status") //nolint:errcheck // best-effort condition parsing
		if condType == conditionEstablished && condStatus == conditionStatusTrue {
			return true
		}
	}
	return false
}

// HealthyPredicate reports whether a live object is healthy by the same rule
// `opm instance status` applies: health.Evaluate judged by health.IsHealthy,
// from the library's opm/k8s/health.
// Workloads must have finished their rollout, Jobs completed and PVCs bound;
// kinds with no readiness concept are healthy once they exist.
func HealthyPredicate(obj *unstructured.Unstructured) bool {
	return health.IsHealthy(health.Evaluate(obj))
}

// AbsentPredicate is the readiness predicate of absence mode: no live object
// ever satisfies it, so the wait ends only when every object reads NotFound.
func AbsentPredicate(*unstructured.Unstructured) bool { return false }

// waitMode selects how the poll loop treats a NotFound read.
type waitMode int

const (
	// modeReady waits for live objects to satisfy the predicate. A NotFound
	// read is a definitive failure: the caller applied the object and it has
	// since disappeared, so polling until the deadline would only hide it.
	modeReady waitMode = iota

	// modeAbsent waits for objects to disappear. A NotFound read satisfies
	// the predicate; a live object is checked against it (AbsentPredicate
	// keeps it pending).
	modeAbsent
)

// Wait polls the live state of each object on the cluster until predicate
// reports every one ready, or ctx is done. It carries no timeout of its own:
// the caller's ctx deadline is the budget, and since marks when that budget
// started so the timeout error reports the time actually consumed. An object
// that reads NotFound fails the wait at once, naming it: the caller applied it
// and it has since disappeared. Any other Get error counts as not yet ready.
func Wait(ctx context.Context, client *Client, objs []*unstructured.Unstructured, predicate ReadyPredicate, since time.Time) error {
	return waitUntil(ctx, client, objs, predicate, modeReady, since, WaitPollInterval)
}

// WaitAbsent polls each object until every one reads NotFound, or ctx is
// done. It carries no timeout of its own: the caller's ctx deadline is the
// budget, and since marks when that budget started so the timeout error
// reports the time actually consumed. Any Get error other than NotFound
// counts as still present.
func WaitAbsent(ctx context.Context, client *Client, objs []*unstructured.Unstructured, since time.Time) error {
	return waitUntil(ctx, client, objs, AbsentPredicate, modeAbsent, since, WaitPollInterval)
}

// waitUntil is the shared poll loop behind Wait and WaitAbsent. since is when
// the budget in force (ctx's deadline) started; the timeout error reports the
// elapsed time from it rather than any nominal --timeout value, so a deadline
// shared across several waits is reported honestly.
func waitUntil(ctx context.Context, client *Client, objs []*unstructured.Unstructured, predicate ReadyPredicate, mode waitMode, since time.Time, pollInterval time.Duration) error {
	if len(objs) == 0 {
		return nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	remaining := objs
	for {
		var err error
		remaining, err = pollObjects(ctx, client, remaining, predicate, mode)
		if err != nil {
			return err
		}
		if len(remaining) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				// Canceled by the caller (e.g. Ctrl-C), not by the budget.
				return ctx.Err()
			}
			elapsed := time.Since(since).Round(time.Second)
			if mode == modeAbsent {
				return fmt.Errorf("timed out after %s waiting for %s to finish terminating", elapsed, describeObjects(remaining))
			}
			return fmt.Errorf("timed out after %s waiting for %s to become ready", elapsed, describeObjects(remaining))
		case <-ticker.C:
		}
	}
}

// pollObjects fetches the live state of each object and returns those that
// don't yet satisfy predicate. A NotFound read satisfies absence mode and
// fails readiness mode with an error naming the object; any other Get error
// leaves the object pending in both modes. A kind that cannot be resolved
// (not served, or a failed discovery request) is returned as an error in both
// modes: it is never read as an object that is gone.
func pollObjects(ctx context.Context, client *Client, objs []*unstructured.Unstructured, predicate ReadyPredicate, mode waitMode) ([]*unstructured.Unstructured, error) {
	var pending []*unstructured.Unstructured
	for _, obj := range objs {
		resource, err := client.ResourceClientFor(ctx, obj.GroupVersionKind(), obj.GetNamespace())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", describeObjects([]*unstructured.Unstructured{obj}), err)
		}
		live, err := resource.Get(ctx, obj.GetName(), metav1.GetOptions{})
		switch {
		case apierrors.IsNotFound(err):
			if mode == modeReady {
				return nil, fmt.Errorf("%s was applied and has since disappeared", describeObjects([]*unstructured.Unstructured{obj}))
			}
		case err != nil || !predicate(live):
			pending = append(pending, obj)
		}
	}
	return pending, nil
}

func describeObjects(objs []*unstructured.Unstructured) string {
	return strings.Join(DescribeObjectList(objs), ", ")
}

// DescribeObjectList names each object as Kind/name, adding "in <namespace>"
// for a namespaced one.
func DescribeObjectList(objs []*unstructured.Unstructured) []string {
	names := make([]string, len(objs))
	for i, obj := range objs {
		if ns := obj.GetNamespace(); ns != "" {
			names[i] = fmt.Sprintf("%s/%s in %s", obj.GetKind(), obj.GetName(), ns)
		} else {
			names[i] = fmt.Sprintf("%s/%s", obj.GetKind(), obj.GetName())
		}
	}
	return names
}
