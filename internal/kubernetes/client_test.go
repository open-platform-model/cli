package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureNamespace(t *testing.T) {
	ctx := context.Background()

	t.Run("creates namespace when missing", func(t *testing.T) {
		fakeClientset := fake.NewSimpleClientset()
		client := &Client{Clientset: fakeClientset}

		created, err := client.EnsureNamespace(ctx, "my-ns", false)
		assert.NoError(t, err)
		assert.True(t, created)

		// Verify namespace was created
		ns, err := fakeClientset.CoreV1().Namespaces().Get(ctx, "my-ns", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, "my-ns", ns.Name)
	})

	t.Run("returns false when namespace exists", func(t *testing.T) {
		fakeClientset := fake.NewSimpleClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "existing-ns"},
		})
		client := &Client{Clientset: fakeClientset}

		created, err := client.EnsureNamespace(ctx, "existing-ns", false)
		assert.NoError(t, err)
		assert.False(t, created)
	})

	t.Run("dry run does not create namespace", func(t *testing.T) {
		fakeClientset := fake.NewSimpleClientset()
		client := &Client{Clientset: fakeClientset}

		created, err := client.EnsureNamespace(ctx, "my-ns", true)
		assert.NoError(t, err)
		assert.True(t, created)

		// Verify namespace was NOT actually created
		_, err = fakeClientset.CoreV1().Namespaces().Get(ctx, "my-ns", metav1.GetOptions{})
		assert.Error(t, err, "namespace should not exist after dry run")
	})
}

func TestBuildRestConfig_InvalidPath(t *testing.T) {
	// buildRestConfig with a nonexistent kubeconfig path should return an error.
	// Values are treated as pre-resolved — no further env/precedence resolution occurs.
	_, err := buildRestConfig(ClientOptions{
		Kubeconfig: "/nonexistent/path/kubeconfig",
		Context:    "nonexistent-context",
	})
	assert.Error(t, err, "expected error for nonexistent kubeconfig path")
}
