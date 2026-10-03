package operator

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

const kindDeployment = "Deployment"

// DefaultPredicate dispatches to the readiness check appropriate for obj's
// kind: CRD Established=True for CustomResourceDefinitions, workload rollout
// health (via kubernetes.EvaluateHealth) for Deployments. Other kinds are
// considered ready as soon as they exist.
func DefaultPredicate(obj *unstructured.Unstructured) bool {
	switch obj.GetKind() {
	case kindCustomResourceDefinition:
		return kubernetes.CRDEstablishedPredicate(obj)
	case kindDeployment:
		return WorkloadReadyPredicate(obj)
	default:
		return true
	}
}

// WorkloadReadyPredicate reports whether a workload resource (e.g. a
// Deployment) has completed its rollout, reusing the same health evaluation
// the rest of the CLI uses for instance status.
func WorkloadReadyPredicate(obj *unstructured.Unstructured) bool {
	return kubernetes.HealthyPredicate(obj)
}

// pendingObjects fetches the live state of each object and returns those
// that don't yet satisfy predicate. Single-shot semantics: an object that
// reads NotFound (or any other Get error) is simply pending. CheckReady
// relies on this; the post-apply wait uses kubernetes.Wait instead.
func pendingObjects(ctx context.Context, client *kubernetes.Client, objs []*unstructured.Unstructured, predicate kubernetes.ReadyPredicate) []*unstructured.Unstructured {
	var pending []*unstructured.Unstructured
	for _, obj := range objs {
		live, err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil || !predicate(live) {
			pending = append(pending, obj)
		}
	}
	return pending
}
