package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// HealthStatus represents the health state of a resource.
type HealthStatus string

const (
	// HealthReady means the resource is ready and healthy.
	HealthReady HealthStatus = "Ready"
	// HealthNotReady means the resource exists but is not yet ready.
	HealthNotReady HealthStatus = "NotReady"
	// HealthComplete means the resource has completed (e.g., a Job).
	HealthComplete HealthStatus = "Complete"
	// HealthUnknown means the health state could not be determined.
	HealthUnknown HealthStatus = "Unknown"
	// HealthMissing means the resource is tracked in the inventory but no longer
	// exists on the cluster (deleted outside of OPM).
	HealthMissing HealthStatus = "Missing"
	// HealthApplied means the resource exists on the cluster and has no
	// readiness concept (RBAC, identity, config, networking objects, custom
	// resources that report no Ready condition). It says the object was
	// applied, nothing more, and counts as healthy.
	HealthApplied HealthStatus = "Applied"
	// HealthBound means a PersistentVolumeClaim is bound to a PersistentVolume.
	HealthBound HealthStatus = "Bound"
)

// conditionStatusTrue is the Kubernetes condition status value representing "true".
const conditionStatusTrue = "True"

// conditionTypeReady is the Kubernetes condition type for readiness.
// Distinct from HealthStatus "Ready" — this is the .conditions[].type field value.
const conditionTypeReady = "Ready"

// workloadKinds are resources evaluated by rollout state (generation, replica
// counts and, for Deployments, the Progressing condition) rather than by an
// Available/Ready condition alone.
var workloadKinds = map[string]bool{
	kindDeployment: true,
}

// passiveKinds are resources with no readiness concept: they are reported as
// HealthApplied as soon as they exist.
// Note: PersistentVolumeClaim is intentionally excluded — it has a lifecycle
// phase (Pending → Bound → Lost) evaluated by evaluatePVCHealth.
// Note: DaemonSet is intentionally excluded — it is a workload, evaluated by
// evaluateDaemonSetHealth.
var passiveKinds = map[string]bool{
	"ConfigMap":           true,
	"Secret":              true,
	"Service":             true,
	"ServiceAccount":      true,
	"Namespace":           true,
	"ClusterRole":         true,
	"ClusterRoleBinding":  true,
	"Role":                true,
	"RoleBinding":         true,
	"Ingress":             true,
	"NetworkPolicy":       true,
	"PodDisruptionBudget": true,
	"ResourceQuota":       true,
	"LimitRange":          true,
	"StorageClass":        true,
	"PriorityClass":       true,
}

// EvaluateHealth determines the health status of a Kubernetes resource
// based on its kind and status conditions.
func EvaluateHealth(resource *unstructured.Unstructured) HealthStatus {
	kind := resource.GetKind()

	// Deployment: rollout state, not just the Available condition
	if workloadKinds[kind] {
		return evaluateWorkloadHealth(resource)
	}

	// StatefulSet: does not emit status conditions; check rollout counters
	if kind == kindStatefulSet {
		return evaluateStatefulSetHealth(resource)
	}

	// DaemonSet: does not reliably emit conditions; check rollout counters
	if kind == kindDaemonSet {
		return evaluateDaemonSetHealth(resource)
	}

	// Jobs: check Complete condition
	if kind == kindJob {
		return evaluateJobHealth(resource)
	}

	// CronJobs: scheduled, no readiness phase
	if kind == "CronJob" {
		return HealthApplied
	}

	// PersistentVolumeClaim: has a lifecycle phase (Pending → Bound → Lost).
	if kind == kindPersistentVolumeClaim {
		return evaluatePVCHealth(resource)
	}

	// Passive resources: applied on creation, no readiness phase
	if passiveKinds[kind] {
		return HealthApplied
	}

	// Custom resources: check for Ready condition, fallback to passive
	return evaluateCustomHealth(resource)
}

// IsHealthy returns true if the given health status represents a healthy state.
// Healthy statuses are: HealthReady, HealthApplied, HealthComplete, HealthBound.
// Every caller that decides "is this resource fine" goes through here.
func IsHealthy(status HealthStatus) bool {
	return status == HealthReady || status == HealthApplied ||
		status == HealthComplete || status == HealthBound
}

// QuickInstanceHealth evaluates aggregate health from pre-fetched resources.
// It calls EvaluateHealth on each live resource and counts healthy vs total.
// unhealthyCount is the number of inventory-tracked resources that are missing
// from the cluster or could not be read; each counts toward the total and not
// toward ready. Returns the aggregate status, ready count, and total count.
func QuickInstanceHealth(resources []*unstructured.Unstructured, unhealthyCount int) (status HealthStatus, readyCount, totalCount int) {
	total := len(resources) + unhealthyCount
	if total == 0 {
		return HealthUnknown, 0, 0
	}

	ready := 0
	for _, res := range resources {
		if IsHealthy(EvaluateHealth(res)) {
			ready++
		}
	}

	if ready == total {
		return HealthReady, ready, total
	}
	return HealthNotReady, ready, total
}

// evaluatePVCHealth reads the PVC lifecycle phase from status.phase.
// Bound → HealthBound (green), Pending/Lost → their raw phase (yellow).
// Falls back to HealthReady for PVCs with no status yet (e.g. just created).
func evaluatePVCHealth(resource *unstructured.Unstructured) HealthStatus {
	phase, _, _ := unstructured.NestedString(resource.Object, "status", "phase") //nolint:errcheck // best-effort PVC phase display
	if phase != "" {
		return HealthStatus(phase)
	}
	return HealthReady // fallback: PVC created but not yet provisioned
}

// conditionReasonProgressDeadline is the Progressing condition reason a
// Deployment carries once its rollout has stalled past progressDeadlineSeconds.
const conditionReasonProgressDeadline = "ProgressDeadlineExceeded"

// evaluateWorkloadHealth reports whether a Deployment has finished rolling out.
// The Available condition alone is not enough: it stays True while the old
// ReplicaSet serves traffic during a stuck upgrade. A Deployment is healthy
// only when the controller has observed the current generation, every replica
// is on the new template and available, and no old replicas linger.
func evaluateWorkloadHealth(resource *unstructured.Unstructured) HealthStatus {
	if !generationObserved(resource) {
		return HealthNotReady
	}
	for _, c := range getConditions(resource) {
		if c.Type == "Progressing" && c.Reason == conditionReasonProgressDeadline {
			return HealthNotReady
		}
	}

	desired := specReplicas(resource)
	updated := statusInt(resource, "updatedReplicas")
	available := statusInt(resource, "availableReplicas")
	total := statusInt(resource, "replicas")

	if updated != desired || available != desired || total > updated {
		return HealthNotReady
	}
	return HealthReady
}

// evaluateStatefulSetHealth reports whether a StatefulSet has finished
// rolling out. StatefulSets do not emit Available/Ready status conditions;
// readiness is signaled by the controller observing the current generation,
// the update revision becoming the current revision, and the replica counters
// reaching spec.replicas. With a RollingUpdate partition (or the OnDelete
// strategy) the revisions legitimately differ, so only the counters the
// strategy guarantees are required.
func evaluateStatefulSetHealth(resource *unstructured.Unstructured) HealthStatus {
	if !generationObserved(resource) {
		return HealthNotReady
	}

	desired := specReplicas(resource)
	if statusInt(resource, "readyReplicas") != desired {
		return HealthNotReady
	}

	strategy, _, _ := unstructured.NestedString(resource.Object, "spec", "updateStrategy", "type") //nolint:errcheck // best-effort strategy read
	switch strategy {
	case "OnDelete":
		return HealthReady
	default: // RollingUpdate is the default strategy
		partition, _, _ := unstructured.NestedInt64(resource.Object, "spec", "updateStrategy", "rollingUpdate", "partition") //nolint:errcheck // best-effort partition read
		if partition > 0 {
			if statusInt(resource, "updatedReplicas") < desired-partition {
				return HealthNotReady
			}
			return HealthReady
		}
	}

	if statusInt(resource, "updatedReplicas") != desired {
		return HealthNotReady
	}
	updateRev, _, _ := unstructured.NestedString(resource.Object, "status", "updateRevision")   //nolint:errcheck // best-effort revision read
	currentRev, _, _ := unstructured.NestedString(resource.Object, "status", "currentRevision") //nolint:errcheck // best-effort revision read
	if updateRev != currentRev {
		return HealthNotReady
	}
	return HealthReady
}

// evaluateDaemonSetHealth reports whether a DaemonSet has finished rolling
// out: the current generation is observed and every node that should run a
// pod runs an updated, available one.
func evaluateDaemonSetHealth(resource *unstructured.Unstructured) HealthStatus {
	if !generationObserved(resource) {
		return HealthNotReady
	}
	desired := statusInt(resource, "desiredNumberScheduled")
	if statusInt(resource, "updatedNumberScheduled") != desired || statusInt(resource, "numberAvailable") != desired {
		return HealthNotReady
	}
	return HealthReady
}

// generationObserved reports whether status.observedGeneration has caught up
// with metadata.generation, i.e. the controller has seen the latest spec.
func generationObserved(resource *unstructured.Unstructured) bool {
	observed, _, _ := unstructured.NestedInt64(resource.Object, "status", "observedGeneration") //nolint:errcheck // best-effort generation read
	return observed >= resource.GetGeneration()
}

// specReplicas returns spec.replicas, which defaults to 1 when omitted.
func specReplicas(resource *unstructured.Unstructured) int64 {
	desired, found, _ := unstructured.NestedInt64(resource.Object, "spec", "replicas") //nolint:errcheck // best-effort replica count
	if !found {
		return 1
	}
	return desired
}

// statusInt returns an integer status field, 0 when absent.
func statusInt(resource *unstructured.Unstructured, field string) int64 {
	v, _, _ := unstructured.NestedInt64(resource.Object, "status", field) //nolint:errcheck // best-effort status counter
	return v
}

// evaluateJobHealth checks the Complete condition on Job resources.
func evaluateJobHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Complete" {
			if c.Status == conditionStatusTrue {
				return HealthComplete
			}
		}
		if c.Type == "Failed" {
			if c.Status == conditionStatusTrue {
				return HealthNotReady
			}
		}
	}
	return HealthNotReady
}

// evaluateCustomHealth checks for a Ready condition on custom resources.
// If no Ready condition exists, treats the resource as passive (HealthApplied).
func evaluateCustomHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == conditionTypeReady {
			if c.Status == conditionStatusTrue {
				return HealthReady
			}
			return HealthNotReady
		}
	}
	// No Ready condition — treat as passive
	return HealthApplied
}

// condition represents a Kubernetes status condition.
type condition struct {
	Type   string
	Status string
	Reason string
}

// getConditions extracts status conditions from an unstructured resource.
func getConditions(resource *unstructured.Unstructured) []condition {
	status, found, err := unstructured.NestedMap(resource.Object, "status")
	if err != nil || !found {
		return nil
	}

	rawConditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found {
		return nil
	}

	var conditions []condition
	for _, raw := range rawConditions {
		c, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(c, "type")     //nolint:errcheck // best-effort condition parsing
		condStatus, _, _ := unstructured.NestedString(c, "status") //nolint:errcheck // best-effort condition parsing

		condReason, _, _ := unstructured.NestedString(c, "reason") //nolint:errcheck // best-effort condition parsing

		if condType != "" {
			conditions = append(conditions, condition{
				Type:   condType,
				Status: condStatus,
				Reason: condReason,
			})
		}
	}

	return conditions
}
