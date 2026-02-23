package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/opmodel/cli/internal/output"
)

// DiscoverChildren finds Kubernetes-owned child resources of the given parents.
// It walks ownerReferences downward using knowledge of standard workload hierarchies:
//
//   - Deployment → ReplicaSet → Pod
//   - StatefulSet → Pod
//   - DaemonSet → Pod
//   - Job → Pod
//   - CronJob → Job → Pod
//
// Non-workload parents (ConfigMap, Secret, Service, etc.) are skipped.
// API errors during child listing are non-fatal: a warning is logged and
// traversal continues with the remaining parents.
//
// This function follows the same traversal patterns as walkOwnership in tree.go,
// but returns []*unstructured.Unstructured (for UID extraction) rather than
// []ResourceNode (for tree display).
func DiscoverChildren(
	ctx context.Context,
	client *Client,
	parents []*unstructured.Unstructured,
	namespace string,
) ([]*unstructured.Unstructured, error) {
	var children []*unstructured.Unstructured

	for _, parent := range parents {
		switch parent.GetKind() {
		case kindDeployment:
			kids := discoverDeploymentChildren(ctx, client, parent, namespace)
			children = append(children, kids...)
		case kindStatefulSet:
			kids := discoverPodsOwnedBy(ctx, client, namespace, parent.GetUID(), "statefulset", parent.GetName())
			children = append(children, kids...)
		case kindDaemonSet:
			kids := discoverPodsOwnedBy(ctx, client, namespace, parent.GetUID(), "daemonset", parent.GetName())
			children = append(children, kids...)
		case kindJob:
			kids := discoverPodsOwnedBy(ctx, client, namespace, parent.GetUID(), "job", parent.GetName())
			children = append(children, kids...)
		case "CronJob":
			kids := discoverCronJobChildren(ctx, client, parent, namespace)
			children = append(children, kids...)
		}
		// Non-workload kinds: no children to discover.
	}

	return children, nil
}

// discoverDeploymentChildren finds ReplicaSets owned by a Deployment, then
// Pods owned by those ReplicaSets.
func discoverDeploymentChildren(ctx context.Context, client *Client, deploy *unstructured.Unstructured, namespace string) []*unstructured.Unstructured {
	uid := deploy.GetUID()

	rsList, err := client.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		output.Debug("failed to list replicasets for children discovery",
			"deployment", deploy.GetName(), "namespace", namespace, "error", err)
		return nil
	}

	var children []*unstructured.Unstructured
	for i := range rsList.Items {
		rs := &rsList.Items[i]
		if !hasOwnerWithUID(rs.OwnerReferences, uid) {
			continue
		}

		// Add the ReplicaSet itself as a child.
		rsUnstructured := toUnstructured(rs.Kind, rs.Name, rs.Namespace, rs.UID)
		if rsUnstructured == nil {
			// If Kind is empty (typed objects), set it explicitly.
			rsUnstructured = &unstructured.Unstructured{}
			rsUnstructured.SetKind(kindReplicaSet)
			rsUnstructured.SetName(rs.Name)
			rsUnstructured.SetNamespace(rs.Namespace)
			rsUnstructured.SetUID(rs.UID)
		}
		children = append(children, rsUnstructured)

		// Find Pods owned by this ReplicaSet.
		pods := discoverPodsOwnedBy(ctx, client, namespace, rs.UID, "replicaset", rs.Name)
		children = append(children, pods...)
	}

	return children
}

// discoverCronJobChildren finds Jobs owned by a CronJob, then Pods owned by
// those Jobs.
func discoverCronJobChildren(ctx context.Context, client *Client, cronJob *unstructured.Unstructured, namespace string) []*unstructured.Unstructured {
	uid := cronJob.GetUID()

	jobList, err := client.Clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		output.Debug("failed to list jobs for children discovery",
			"cronjob", cronJob.GetName(), "namespace", namespace, "error", err)
		return nil
	}

	var children []*unstructured.Unstructured
	for i := range jobList.Items {
		job := &jobList.Items[i]
		if !hasOwnerWithUID(job.OwnerReferences, uid) {
			continue
		}

		// Add the Job itself as a child.
		jobU := &unstructured.Unstructured{}
		jobU.SetKind(kindJob)
		jobU.SetName(job.Name)
		jobU.SetNamespace(job.Namespace)
		jobU.SetUID(job.UID)
		children = append(children, jobU)

		// Find Pods owned by this Job.
		pods := discoverPodsOwnedBy(ctx, client, namespace, job.UID, "job", job.Name)
		children = append(children, pods...)
	}

	return children
}

// discoverPodsOwnedBy lists all Pods in the namespace and returns those owned
// by the given UID as unstructured objects.
func discoverPodsOwnedBy(ctx context.Context, client *Client, namespace string, uid types.UID, parentKind, parentName string) []*unstructured.Unstructured {
	podList, err := client.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		output.Debug("failed to list pods for children discovery",
			parentKind, parentName, "namespace", namespace, "error", err)
		return nil
	}

	var children []*unstructured.Unstructured
	for i := range podList.Items {
		pod := &podList.Items[i]
		if !hasOwnerWithUID(pod.OwnerReferences, uid) {
			continue
		}
		podU := &unstructured.Unstructured{}
		podU.SetKind("Pod")
		podU.SetName(pod.Name)
		podU.SetNamespace(pod.Namespace)
		podU.SetUID(pod.UID)
		children = append(children, podU)
	}

	return children
}

// toUnstructured creates a minimal unstructured object with the given metadata.
// Returns nil if kind is empty.
func toUnstructured(kind, name, namespace string, uid types.UID) *unstructured.Unstructured {
	if kind == "" {
		return nil
	}
	u := &unstructured.Unstructured{}
	u.SetKind(kind)
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetUID(uid)
	return u
}
