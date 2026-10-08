package kubernetes

import (
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

func TestResourceClient(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeDynamic := fakedynamic.NewSimpleDynamicClient(scheme)
	client := &Client{Resources: kubetest.Resources(), Dynamic: fakeDynamic}

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

	t.Run("namespace-scoped", func(t *testing.T) {
		rc := client.ResourceClient(gvr, "production")
		// Verify we get a non-nil interface back
		assert.NotNil(t, rc)
	})

	t.Run("cluster-scoped", func(t *testing.T) {
		rc := client.ResourceClient(gvr, "")
		// Verify we get a non-nil interface back
		assert.NotNil(t, rc)
	})
}
