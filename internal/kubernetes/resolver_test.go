package kubernetes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// discoveryServer is an API server that answers discovery documents only.
// docs maps a request path to the resources it lists; a path in status
// answers that HTTP status instead; any other path answers 404. Both maps can
// be changed between requests through set.
type discoveryServer struct {
	mu     sync.Mutex
	docs   map[string][]metav1.APIResource
	status map[string]int
	hits   map[string]int
}

func (s *discoveryServer) set(change func(s *discoveryServer)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	change(s)
}

func (s *discoveryServer) requests(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[path]
}

func (s *discoveryServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits[r.URL.Path]++

	w.Header().Set("Content-Type", "application/json")
	if code, ok := s.status[r.URL.Path]; ok {
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(metav1.Status{
			TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
			Status:   metav1.StatusFailure, Code: int32(code),
			Reason: map[int]metav1.StatusReason{
				http.StatusForbidden:          metav1.StatusReasonForbidden,
				http.StatusServiceUnavailable: metav1.StatusReasonServiceUnavailable,
			}[code],
		})
		return
	}
	resources, ok := s.docs[r.URL.Path]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(metav1.Status{
			TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
			Status:   metav1.StatusFailure, Code: http.StatusNotFound, Reason: metav1.StatusReasonNotFound,
		})
		return
	}
	_ = json.NewEncoder(w).Encode(metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{Kind: "APIResourceList", APIVersion: "v1"},
		APIResources: resources,
	})
}

// newDiscoveryResolver starts a discoveryServer and returns the resolver
// NewClient would build against it.
func newDiscoveryResolver(t *testing.T, docs map[string][]metav1.APIResource) (ResourceResolver, *discoveryServer) {
	t.Helper()
	server := &discoveryServer{docs: docs, status: map[string]int{}, hits: map[string]int{}}
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	cs, err := clientset.NewForConfig(&rest.Config{Host: httpServer.URL})
	require.NoError(t, err)
	return NewDiscoveryResolver(cs.Discovery()), server
}

var (
	prometheusGVK = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "Prometheus"}
	widgetGVK     = schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Widget"}
)

const (
	monitoringPath = "/apis/monitoring.coreos.com/v1"
	examplePath    = "/apis/example.io/v1"
	appsPath       = "/apis/apps/v1"
)

func TestDiscoveryResolver_TakesTheNameTheClusterGives(t *testing.T) {
	resolver, _ := newDiscoveryResolver(t, map[string][]metav1.APIResource{
		// The guessed plural of Prometheus was "prometheus".
		monitoringPath: {{Name: "prometheuses", Kind: "Prometheus"}},
		"/api/v1":      {{Name: "endpoints", Kind: "Endpoints"}, {Name: "configmaps", Kind: "ConfigMap"}},
	})

	gvr, err := resolver.ResourceFor(context.Background(), prometheusGVK)
	require.NoError(t, err)
	assert.Equal(t, schema.GroupVersionResource{Group: "monitoring.coreos.com", Version: "v1", Resource: "prometheuses"}, gvr)

	gvr, err = resolver.ResourceFor(context.Background(), schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"})
	require.NoError(t, err, "the core group is read from /api/v1")
	assert.Equal(t, schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, gvr)
}

func TestDiscoveryResolver_OneRequestPerGroupVersion(t *testing.T) {
	resolver, server := newDiscoveryResolver(t, map[string][]metav1.APIResource{
		appsPath: {
			{Name: "deployments", Kind: "Deployment"},
			{Name: "deployments/status", Kind: "Deployment"},
			{Name: "deployments/scale", Kind: "Scale"},
			{Name: "statefulsets", Kind: "StatefulSet"},
		},
	})

	for _, kind := range []string{"Deployment", "StatefulSet", "Deployment"} {
		gvr, err := resolver.ResourceFor(context.Background(), schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: kind})
		require.NoError(t, err)
		assert.NotContains(t, gvr.Resource, "/", "a subresource never serves a kind")
	}
	assert.Equal(t, 1, server.requests(appsPath))

	_, err := resolver.ResourceFor(context.Background(), schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Scale"})
	assert.True(t, IsKindNotServed(err), "a kind only a subresource has is not served: %v", err)
}

func TestDiscoveryResolver_KindNotServed(t *testing.T) {
	resolver, _ := newDiscoveryResolver(t, map[string][]metav1.APIResource{
		examplePath: {{Name: "gadgets", Kind: "Gadget"}},
	})

	tests := map[string]schema.GroupVersionKind{
		"the group and version are not served": prometheusGVK,
		"the kind is not in the group version": widgetGVK,
	}
	for name, gvk := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := resolver.ResourceFor(context.Background(), gvk)
			require.Error(t, err)
			assert.True(t, IsKindNotServed(err))
			assert.False(t, apierrors.IsNotFound(err), "an unserved kind must never read as an object that is gone")
			assert.Contains(t, err.Error(), gvk.Kind)
			assert.Contains(t, err.Error(), gvk.GroupVersion().String())
		})
	}
}

func TestDiscoveryResolver_AsksAgainForAKindThatWasMissing(t *testing.T) {
	resolver, server := newDiscoveryResolver(t, map[string][]metav1.APIResource{
		examplePath: {{Name: "gadgets", Kind: "Gadget"}},
	})
	gadget := schema.GroupVersionKind{Group: "example.io", Version: "v1", Kind: "Gadget"}

	_, err := resolver.ResourceFor(context.Background(), gadget)
	require.NoError(t, err)
	_, err = resolver.ResourceFor(context.Background(), widgetGVK)
	require.True(t, IsKindNotServed(err))

	// A CustomResourceDefinition starts to serve Widget during the command.
	server.set(func(s *discoveryServer) {
		s.docs[examplePath] = append(s.docs[examplePath], metav1.APIResource{Name: "widgets", Kind: "Widget"})
	})

	gvr, err := resolver.ResourceFor(context.Background(), widgetGVK)
	require.NoError(t, err)
	assert.Equal(t, "widgets", gvr.Resource)

	before := server.requests(examplePath)
	_, err = resolver.ResourceFor(context.Background(), widgetGVK)
	require.NoError(t, err)
	_, err = resolver.ResourceFor(context.Background(), gadget)
	require.NoError(t, err)
	assert.Equal(t, before, server.requests(examplePath), "a served kind answers from memory")
}

func TestDiscoveryResolver_FailedRequestIsNotAnUnservedKind(t *testing.T) {
	tests := map[string]struct {
		code  int
		check func(error) bool
	}{
		"forbidden":   {code: http.StatusForbidden, check: apierrors.IsForbidden},
		"unavailable": {code: http.StatusServiceUnavailable, check: apierrors.IsServiceUnavailable},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			resolver, server := newDiscoveryResolver(t, nil)
			// The core group too: it is read from another path.
			server.set(func(s *discoveryServer) {
				s.status[examplePath] = tt.code
				s.status["/api/v1"] = tt.code
			})

			for _, gvk := range []schema.GroupVersionKind{widgetGVK, {Version: "v1", Kind: "ConfigMap"}} {
				_, err := resolver.ResourceFor(context.Background(), gvk)
				require.Error(t, err)
				assert.False(t, IsKindNotServed(err), "a failed request is not an answer: %v", err)
				assert.True(t, IsDiscoveryFailure(err))
				assert.False(t, apierrors.IsNotFound(err))
				assert.True(t, tt.check(err), "the API error stays in the chain: %v", err)
				assert.Contains(t, err.Error(), gvk.GroupVersion().String())
			}
		})
	}
}

func TestDiscoveryResolver_CanceledContext(t *testing.T) {
	resolver, _ := newDiscoveryResolver(t, map[string][]metav1.APIResource{
		examplePath: {{Name: "widgets", Kind: "Widget"}},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := resolver.ResourceFor(ctx, widgetGVK)
	require.Error(t, err)
	assert.False(t, IsKindNotServed(err))
	assert.ErrorIs(t, err, context.Canceled)
}

// NewClient must give the client the resolver that asks the cluster: every
// other test sets a fake one.
func TestNewClient_ResolvesThroughTheClustersDiscovery(t *testing.T) {
	server := &discoveryServer{
		docs:   map[string][]metav1.APIResource{monitoringPath: {{Name: "prometheuses", Kind: "Prometheus"}}},
		status: map[string]int{}, hits: map[string]int{},
	}
	httpServer := httptest.NewServer(server)
	t.Cleanup(httpServer.Close)

	ResetClient()
	t.Cleanup(ResetClient)
	client, err := NewClient(ClientOptions{Kubeconfig: writeKubeconfig(t, httpServer.URL)})
	require.NoError(t, err)

	gvr, err := client.ResourceFor(context.Background(), prometheusGVK)
	require.NoError(t, err)
	assert.Equal(t, "prometheuses", gvr.Resource)
	assert.Equal(t, 1, server.requests(monitoringPath))
}

func TestClient_ResourceForWithoutResolver(t *testing.T) {
	_, err := (&Client{}).ResourceFor(context.Background(), widgetGVK)
	require.Error(t, err)
	assert.False(t, IsKindNotServed(err))

	_, err = (&Client{}).ResourceClientFor(context.Background(), widgetGVK, "default")
	require.Error(t, err)
}
