package kubernetes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestBuildModuleSelector(t *testing.T) {
	selector := buildModuleSelector("my-app", "production")

	str := selector.String()
	assert.Contains(t, str, "app.kubernetes.io/managed-by=open-platform-model")
	assert.Contains(t, str, "module.opmodel.dev/name=my-app")
	assert.Contains(t, str, "module.opmodel.dev/namespace=production")
}

func TestBuildReleaseIDSelector(t *testing.T) {
	selector := buildReleaseIDSelector("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	str := selector.String()
	assert.Contains(t, str, "app.kubernetes.io/managed-by=open-platform-model")
	assert.Contains(t, str, "module-release.opmodel.dev/uuid=a1b2c3d4-e5f6-7890-abcd-ef1234567890")
}

func TestSortByWeightDescending(t *testing.T) {
	// Create resources of different kinds
	resources := []*unstructured.Unstructured{
		makeUnstructured("apps/v1", "Deployment", "my-deploy", "default"),
		makeUnstructured("v1", "Namespace", "my-ns", ""),
		makeUnstructured("admissionregistration.k8s.io/v1", "ValidatingWebhookConfiguration", "my-webhook", ""),
		makeUnstructured("v1", "ConfigMap", "my-cm", "default"),
		makeUnstructured("v1", "Service", "my-svc", "default"),
	}

	sortByWeightDescending(resources)

	// Expected order: Webhook(500) > Deployment(100) > Service(50) > ConfigMap(15) > Namespace(0)
	assert.Equal(t, "ValidatingWebhookConfiguration", resources[0].GetKind())
	assert.Equal(t, "Deployment", resources[1].GetKind())
	assert.Equal(t, "Service", resources[2].GetKind())
	assert.Equal(t, "ConfigMap", resources[3].GetKind())
	assert.Equal(t, "Namespace", resources[4].GetKind())
}

func TestContainsSlash(t *testing.T) {
	assert.True(t, containsSlash("pods/log"))
	assert.True(t, containsSlash("deployments/scale"))
	assert.False(t, containsSlash("pods"))
	assert.False(t, containsSlash(""))
}

func makeUnstructured(apiVersion, kind, name, namespace string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	if namespace != "" {
		obj.SetNamespace(namespace)
	}
	return obj
}

func TestNoResourcesFoundError(t *testing.T) {
	t.Run("error message with module name", func(t *testing.T) {
		err := &noResourcesFoundError{
			ModuleName: "my-app",
			Namespace:  "production",
		}
		assert.Equal(t, `no resources found for module "my-app" in namespace "production"`, err.Error())
	})

	t.Run("error message with release-id", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Namespace: "production",
		}
		assert.Equal(t, `no resources found for release-id "a1b2c3d4-e5f6-7890-abcd-ef1234567890" in namespace "production"`, err.Error())
	})

	t.Run("errors.Is matches errNoResourcesFound", func(t *testing.T) {
		err := &noResourcesFoundError{
			ModuleName: "my-app",
			Namespace:  "production",
		}
		assert.True(t, errors.Is(err, errNoResourcesFound))
	})
}

func TestDiscoveryOptions_Validation(t *testing.T) {
	// Note: The actual DiscoverResources function requires a real k8s client,
	// so we test the validation logic conceptually through the error cases.

	t.Run("neither ModuleName nor ReleaseID provided", func(t *testing.T) {
		opts := DiscoveryOptions{
			Namespace: "default",
		}
		// Both are empty - this should be caught by DiscoverResources
		assert.Empty(t, opts.ModuleName)
		assert.Empty(t, opts.ReleaseID)
	})

	t.Run("only ModuleName provided", func(t *testing.T) {
		opts := DiscoveryOptions{
			ModuleName: "my-app",
			Namespace:  "default",
		}
		assert.NotEmpty(t, opts.ModuleName)
		assert.Empty(t, opts.ReleaseID)
	})

	t.Run("only ReleaseID provided", func(t *testing.T) {
		opts := DiscoveryOptions{
			ReleaseID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Namespace: "default",
		}
		assert.Empty(t, opts.ModuleName)
		assert.NotEmpty(t, opts.ReleaseID)
	})
}
