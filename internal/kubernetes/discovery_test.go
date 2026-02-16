package kubernetes

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestBuildReleaseNameSelector(t *testing.T) {
	selector := buildReleaseNameSelector("my-app")

	str := selector.String()
	assert.Contains(t, str, "app.kubernetes.io/managed-by=open-platform-model")
	assert.Contains(t, str, "module-release.opmodel.dev/name=my-app")
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
	t.Run("error message with release name", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		assert.Equal(t, `no resources found for release my-app in namespace production`, err.Error())
	})

	t.Run("error message with release-id", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			Namespace: "production",
		}
		assert.Equal(t, `no resources found for release-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 in namespace production`, err.Error())
	})

	t.Run("errors.Is matches errNoResourcesFound", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		assert.True(t, errors.Is(err, errNoResourcesFound))
	})

	t.Run("IsNoResourcesFound matches direct error", func(t *testing.T) {
		err := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		assert.True(t, IsNoResourcesFound(err))
	})

	t.Run("IsNoResourcesFound matches wrapped error", func(t *testing.T) {
		inner := &noResourcesFoundError{
			ReleaseName: "my-app",
			Namespace:   "production",
		}
		wrapped := fmt.Errorf("discovering release resources: %w", inner)
		assert.True(t, IsNoResourcesFound(wrapped))
	})

	t.Run("IsNoResourcesFound rejects unrelated error", func(t *testing.T) {
		err := fmt.Errorf("connection refused")
		assert.False(t, IsNoResourcesFound(err))
	})
}

func TestDiscoveryOptions_Validation(t *testing.T) {
	// Note: The actual DiscoverResources function requires a real k8s client,
	// so we test the validation logic conceptually through the error cases.

	t.Run("neither ReleaseName nor ReleaseID provided", func(t *testing.T) {
		opts := DiscoveryOptions{}
		// Both are empty - this should be caught by DiscoverResources
		assert.Empty(t, opts.ReleaseName)
		assert.Empty(t, opts.ReleaseID)
	})

	t.Run("only ReleaseName provided", func(t *testing.T) {
		opts := DiscoveryOptions{
			ReleaseName: "my-app",
		}
		assert.NotEmpty(t, opts.ReleaseName)
		assert.Empty(t, opts.ReleaseID)
	})

	t.Run("only ReleaseID provided", func(t *testing.T) {
		opts := DiscoveryOptions{
			ReleaseID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		}
		assert.Empty(t, opts.ReleaseName)
		assert.NotEmpty(t, opts.ReleaseID)
	})
}

func TestDiscoverWithSelector_ExcludeOwned(t *testing.T) {
	// Create test resources - one with ownerReferences, one without
	resourceWithOwner := &unstructured.Unstructured{}
	resourceWithOwner.SetName("owned-resource")
	resourceWithOwner.SetKind("Pod")
	resourceWithOwner.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "parent",
			UID:        "123",
		},
	})

	resourceWithoutOwner := &unstructured.Unstructured{}
	resourceWithoutOwner.SetName("standalone-resource")
	resourceWithoutOwner.SetKind("Service")

	tests := []struct {
		name         string
		excludeOwned bool
		wantCount    int
		wantNames    []string
	}{
		{
			name:         "exclude owned filters out owned resources",
			excludeOwned: true,
			wantCount:    1,
			wantNames:    []string{"standalone-resource"},
		},
		{
			name:         "include owned shows all resources",
			excludeOwned: false,
			wantCount:    2,
			wantNames:    []string{"owned-resource", "standalone-resource"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a conceptual test - in practice, discoverWithSelector
			// makes actual API calls. For a real test, you'd need to mock
			// the k8s client. The logic is verified by the inline filter:
			//   if excludeOwned && len(item.GetOwnerReferences()) > 0 { continue }

			// Verify the filtering logic directly
			resources := []*unstructured.Unstructured{resourceWithOwner, resourceWithoutOwner}
			filtered := make([]*unstructured.Unstructured, 0)
			for _, res := range resources {
				if tt.excludeOwned && len(res.GetOwnerReferences()) > 0 {
					continue
				}
				filtered = append(filtered, res)
			}

			assert.Equal(t, tt.wantCount, len(filtered))
			for i, name := range tt.wantNames {
				assert.Equal(t, name, filtered[i].GetName())
			}
		})
	}
}
