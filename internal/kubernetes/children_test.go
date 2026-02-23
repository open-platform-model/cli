package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func makeParent(kind, name, namespace string, uid types.UID) *unstructured.Unstructured { //nolint:unparam // test helper, namespace is always "default" for now
	u := &unstructured.Unstructured{}
	u.SetKind(kind)
	u.SetName(name)
	u.SetNamespace(namespace)
	u.SetUID(uid)
	return u
}

// TestDiscoverChildren_APIErrorNonFatal verifies that DiscoverChildren returns
// nil error even when the underlying client has no API resources registered.
// The function logs warnings via output.Debug but never propagates errors.
func TestDiscoverChildren_APIErrorNonFatal(t *testing.T) {
	clientset := fake.NewSimpleClientset() //nolint:staticcheck // matches existing test patterns

	// A Deployment parent with no corresponding ReplicaSets or Pods — the
	// function should return empty children, not an error.
	parents := []*unstructured.Unstructured{
		makeParent("Deployment", "web", "default", "uid-1"),
		makeParent("StatefulSet", "db", "default", "uid-2"),
		makeParent("DaemonSet", "log", "default", "uid-3"),
	}

	client := &Client{Clientset: clientset}
	children, err := DiscoverChildren(context.Background(), client, parents, "default")
	require.NoError(t, err, "DiscoverChildren must not return an error even with empty results")
	assert.Empty(t, children)
}

func TestDiscoverChildren(t *testing.T) {
	tests := []struct {
		name          string
		parents       []*unstructured.Unstructured
		replicaSets   []appsv1.ReplicaSet
		pods          []corev1.Pod
		expectedKinds []string // kinds of returned children
	}{
		{
			name: "Deployment children: RS + Pods",
			parents: []*unstructured.Unstructured{
				makeParent("Deployment", "web", "default", "deploy-uid-1"),
			},
			replicaSets: []appsv1.ReplicaSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "web-abc12", Namespace: "default", UID: "rs-uid-1",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "deploy-uid-1"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "other-rs", Namespace: "default", UID: "rs-uid-2",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "other-deploy-uid"},
						},
					},
				},
			},
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "web-abc12-x1", Namespace: "default", UID: "pod-uid-1",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "rs-uid-1"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "unrelated-pod", Namespace: "default", UID: "pod-uid-2",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "rs-uid-2"},
						},
					},
				},
			},
			expectedKinds: []string{"ReplicaSet", "Pod"},
		},
		{
			name: "StatefulSet children: Pods only",
			parents: []*unstructured.Unstructured{
				makeParent("StatefulSet", "db", "default", "sts-uid-1"),
			},
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "db-0", Namespace: "default", UID: "pod-uid-3",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "sts-uid-1"},
						},
					},
				},
			},
			expectedKinds: []string{"Pod"},
		},
		{
			name: "DaemonSet children: Pods only",
			parents: []*unstructured.Unstructured{
				makeParent("DaemonSet", "logger", "default", "ds-uid-1"),
			},
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "logger-node1", Namespace: "default", UID: "pod-uid-ds-1",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "ds-uid-1"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "logger-node2", Namespace: "default", UID: "pod-uid-ds-2",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "ds-uid-1"},
						},
					},
				},
			},
			expectedKinds: []string{"Pod", "Pod"},
		},
		{
			name: "Job children: Pods only",
			parents: []*unstructured.Unstructured{
				makeParent("Job", "migrate", "default", "job-uid-1"),
			},
			pods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "migrate-xyz", Namespace: "default", UID: "pod-uid-job-1",
						OwnerReferences: []metav1.OwnerReference{
							{UID: "job-uid-1"},
						},
					},
				},
			},
			expectedKinds: []string{"Pod"},
		},
		{
			name: "Non-workload parents skipped",
			parents: []*unstructured.Unstructured{
				makeParent("ConfigMap", "cfg", "default", "cm-uid-1"),
				makeParent("Service", "svc", "default", "svc-uid-1"),
				makeParent("Secret", "sec", "default", "sec-uid-1"),
			},
			expectedKinds: nil,
		},
		{
			name: "No children exist",
			parents: []*unstructured.Unstructured{
				makeParent("Deployment", "empty", "default", "deploy-uid-empty"),
			},
			expectedKinds: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset() //nolint:staticcheck // matches existing test patterns

			// Seed ReplicaSets.
			for i := range tt.replicaSets {
				_, err := clientset.AppsV1().ReplicaSets(tt.replicaSets[i].Namespace).Create(
					context.Background(), &tt.replicaSets[i], metav1.CreateOptions{})
				require.NoError(t, err)
			}

			// Seed Pods.
			for i := range tt.pods {
				_, err := clientset.CoreV1().Pods(tt.pods[i].Namespace).Create(
					context.Background(), &tt.pods[i], metav1.CreateOptions{})
				require.NoError(t, err)
			}

			client := &Client{Clientset: clientset}
			children, err := DiscoverChildren(context.Background(), client, tt.parents, "default")
			require.NoError(t, err)

			if tt.expectedKinds == nil {
				assert.Empty(t, children)
				return
			}

			var gotKinds []string
			for _, c := range children {
				gotKinds = append(gotKinds, c.GetKind())
			}
			assert.Equal(t, tt.expectedKinds, gotKinds)
		})
	}
}
