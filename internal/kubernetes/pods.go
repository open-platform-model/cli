package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
)

// listWorkloadPods lists pods for a workload resource using its label selector.
// Only supported for workload kinds (Deployment, StatefulSet, DaemonSet).
// Returns an empty slice for non-workload kinds.
// Never call this on MissingResource entries — they have no live object.
func listWorkloadPods(ctx context.Context, client *Client, resource *unstructured.Unstructured) ([]podInfo, error) {
	if resource == nil {
		return nil, nil
	}
	kind := resource.GetKind()
	if !workloadKinds[kind] {
		return nil, nil
	}

	// Extract .spec.selector.matchLabels from the workload
	matchLabels, found, err := unstructured.NestedStringMap(resource.Object, "spec", "selector", "matchLabels")
	if err != nil {
		return nil, err
	}
	if !found || len(matchLabels) == 0 {
		return nil, nil
	}

	selector := labels.SelectorFromSet(matchLabels).String()
	namespace := resource.GetNamespace()

	podList, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return nil, err
	}

	result := make([]podInfo, 0, len(podList.Items))
	for i := range podList.Items {
		result = append(result, extractPodInfoFromPod(&podList.Items[i]))
	}

	return result, nil
}

// extractPodInfoFromPod extracts a podInfo from a corev1.Pod.
//
// Phase display rules (matching spec):
//   - When a container has a waiting reason, that reason overrides the pod phase
//     display (e.g. CrashLoopBackOff → "CrashLoop", ImagePullBackOff → "ImagePullBackOff").
//   - Otherwise the pod phase is used as-is (Running, Pending, Failed, …).
//
// Reason display rules:
//   - The last-terminated reason (OOMKilled, Error, …) is shown as detail text.
//   - "Completed" is excluded — it means the container exited with code 0 (normal
//     exit), which is not a useful diagnostic for an unhealthy workload.
func extractPodInfoFromPod(pod *corev1.Pod) podInfo {
	info := podInfo{
		Name:  pod.Name,
		Phase: string(pod.Status.Phase),
	}

	// Determine ready from conditions
	for _, cond := range pod.Status.Conditions {
		if string(cond.Type) == string(HealthReady) {
			info.Ready = cond.Status == conditionStatusTrue
			break
		}
	}

	// Extract phase override, termination reason, and restart count from container statuses.
	// We scan all containers: the first waiting reason wins for the phase override; the
	// first non-"Completed" termination reason wins for the detail text.
	var totalRestarts int32
	var terminatedReason string
	for i := range pod.Status.ContainerStatuses {
		cs := &pod.Status.ContainerStatuses[i]
		totalRestarts += cs.RestartCount

		// Waiting reason overrides the pod phase display (spec: "display the
		// container's waiting reason instead of the pod phase").
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			// Only override once (first container with a waiting reason wins).
			if info.Phase == string(pod.Status.Phase) {
				info.Phase = mapWaitingReason(cs.State.Waiting.Reason)
			}
		}

		// Last-terminated reason provides additional context (e.g. OOMKilled).
		// Skip "Completed" — normal exit (code 0) is not a diagnostic reason.
		if terminatedReason == "" &&
			cs.LastTerminationState.Terminated != nil &&
			cs.LastTerminationState.Terminated.Reason != "" &&
			cs.LastTerminationState.Terminated.Reason != "Completed" {
			terminatedReason = cs.LastTerminationState.Terminated.Reason
		}
	}

	info.Restarts = int(totalRestarts)
	info.Reason = terminatedReason

	return info
}

// mapWaitingReason converts a Kubernetes container waiting reason to a
// display-friendly string. CrashLoopBackOff is shortened to CrashLoop;
// all other reasons are returned unchanged.
func mapWaitingReason(reason string) string {
	if reason == "CrashLoopBackOff" {
		return "CrashLoop"
	}
	return reason
}
