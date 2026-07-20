package operator

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// waitPollInterval is how often Wait re-checks target readiness.
const waitPollInterval = 2 * time.Second

const (
	kindDeployment       = "Deployment"
	conditionStatusTrue  = "True"
	conditionEstablished = "Established"
)

// ReadyPredicate reports whether a live object (as currently observed on the
// cluster) is ready.
type ReadyPredicate func(obj *unstructured.Unstructured) bool

// DefaultPredicate dispatches to the readiness check appropriate for obj's
// kind: CRD Established=True for CustomResourceDefinitions, workload rollout
// health (via kubernetes.EvaluateHealth) for Deployments. Other kinds are
// considered ready as soon as they exist.
func DefaultPredicate(obj *unstructured.Unstructured) bool {
	switch obj.GetKind() {
	case kindCustomResourceDefinition:
		return CRDEstablishedPredicate(obj)
	case kindDeployment:
		return WorkloadReadyPredicate(obj)
	default:
		return true
	}
}

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

// WorkloadReadyPredicate reports whether a workload resource (e.g. a
// Deployment) has completed its rollout, reusing the same health evaluation
// the rest of the CLI uses for instance status.
func WorkloadReadyPredicate(obj *unstructured.Unstructured) bool {
	return kubernetes.IsHealthy(kubernetes.EvaluateHealth(obj))
}

// Wait polls the live state of each object on the cluster until predicate
// reports every one ready, or timeout elapses. Bounded and context-cancellable.
// Objects that no longer exist (e.g. a Get error) count as not yet ready.
func Wait(ctx context.Context, client *kubernetes.Client, objs []*unstructured.Unstructured, predicate ReadyPredicate, timeout time.Duration) error {
	return wait(ctx, client, objs, predicate, timeout, waitPollInterval)
}

func wait(ctx context.Context, client *kubernetes.Client, objs []*unstructured.Unstructured, predicate ReadyPredicate, timeout, pollInterval time.Duration) error {
	if len(objs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	remaining := objs
	for {
		remaining = pendingObjects(ctx, client, remaining, predicate)
		if len(remaining) == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out after %s waiting for %s to become ready", timeout, describeObjects(remaining))
		case <-ticker.C:
		}
	}
}

// pendingObjects fetches the live state of each object and returns those
// that don't yet satisfy predicate.
func pendingObjects(ctx context.Context, client *kubernetes.Client, objs []*unstructured.Unstructured, predicate ReadyPredicate) []*unstructured.Unstructured {
	var pending []*unstructured.Unstructured
	for _, obj := range objs {
		live, err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil || !predicate(live) {
			pending = append(pending, obj)
		}
	}
	return pending
}

func describeObjects(objs []*unstructured.Unstructured) string {
	return strings.Join(describeObjectList(objs), ", ")
}
