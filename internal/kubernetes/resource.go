package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// ResourceResolver resolves the resource that serves a kind. The name of a
// resource is the cluster's to give: it is never derived from the spelling of
// the kind.
type ResourceResolver interface {
	// ResourceFor returns the resource that serves gvk. The error is a
	// *KindNotServedError when the cluster answered and does not serve the
	// kind, and any other error when the cluster could not be asked.
	ResourceFor(ctx context.Context, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error)
}

// KindNotServedError reports that API discovery answered and the cluster does
// not serve the kind at that group and version. It carries no API status
// error, so apierrors.IsNotFound is false for it: an object of a kind that is
// not served is not an object that is already gone.
type KindNotServedError struct {
	GVK schema.GroupVersionKind
}

func (e *KindNotServedError) Error() string {
	return fmt.Sprintf("kind %q of %s is not served by the cluster", e.GVK.Kind, e.GVK.GroupVersion())
}

// IsKindNotServed reports whether err is, or wraps, a *KindNotServedError.
func IsKindNotServed(err error) bool {
	var notServed *KindNotServedError
	return errors.As(err, &notServed)
}

// errNoResolver is returned by a Client built without a ResourceResolver.
var errNoResolver = errors.New("kubernetes client has no resource resolver")

// ResourceFor returns the resource that serves gvk, as c.Resources resolves
// it.
func (c *Client) ResourceFor(ctx context.Context, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	if c.Resources == nil {
		return schema.GroupVersionResource{}, errNoResolver
	}
	return c.Resources.ResourceFor(ctx, gvk)
}

// ResourceClientFor returns the dynamic resource client for the resource that
// serves gvk, scoped to ns (cluster-scoped when ns is empty).
func (c *Client) ResourceClientFor(ctx context.Context, gvk schema.GroupVersionKind, ns string) (dynamic.ResourceInterface, error) {
	gvr, err := c.ResourceFor(ctx, gvk)
	if err != nil {
		return nil, err
	}
	return c.ResourceClient(gvr, ns), nil
}

// ResourceClient returns the appropriate dynamic resource client for the given
// GVR and namespace. If namespace is empty, returns a cluster-scoped client.
func (c *Client) ResourceClient(gvr schema.GroupVersionResource, ns string) dynamic.ResourceInterface {
	if ns != "" {
		return c.Dynamic.Resource(gvr).Namespace(ns)
	}
	return c.Dynamic.Resource(gvr)
}

// discoveryResolver is the ResourceResolver backed by the cluster's API
// discovery. It reads one discovery document per group and version
// (/api/v1, /apis/<group>/<version>) and keeps it in memory.
//
// Only a document that holds the asked kind answers from memory. A kind that
// is missing is asked for again on every call, because a
// CustomResourceDefinition can start to serve it during the command; that
// costs one request, the same as the read it stands in front of.
type discoveryResolver struct {
	rest rest.Interface

	mu     sync.Mutex
	served map[schema.GroupVersion]map[string]string // kind -> resource
}

// NewDiscoveryResolver returns a ResourceResolver that asks the API server
// behind restClient, the REST client of a discovery client.
func NewDiscoveryResolver(restClient rest.Interface) ResourceResolver {
	return &discoveryResolver{rest: restClient, served: map[schema.GroupVersion]map[string]string{}}
}

func (r *discoveryResolver) ResourceFor(ctx context.Context, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	gv := gvk.GroupVersion()

	// One lock for the whole lookup: concurrent callers that need the same
	// document wait for one request instead of each sending their own.
	r.mu.Lock()
	defer r.mu.Unlock()

	if resource, ok := r.served[gv][gvk.Kind]; ok {
		return gv.WithResource(resource), nil
	}

	kinds, err := r.fetch(ctx, gv)
	if err != nil {
		if apierrors.IsNotFound(err) {
			delete(r.served, gv)
			return schema.GroupVersionResource{}, &KindNotServedError{GVK: gvk}
		}
		return schema.GroupVersionResource{}, fmt.Errorf("discovering the resources of %s: %w", gv, err)
	}
	r.served[gv] = kinds

	resource, ok := kinds[gvk.Kind]
	if !ok {
		return schema.GroupVersionResource{}, &KindNotServedError{GVK: gvk}
	}
	return gv.WithResource(resource), nil
}

// fetch reads the discovery document of gv and returns its kinds with the
// resource that serves each. Subresources are left out; when several
// resources serve one kind, the first listed wins.
func (r *discoveryResolver) fetch(ctx context.Context, gv schema.GroupVersion) (map[string]string, error) {
	path := "/apis/" + gv.Group + "/" + gv.Version
	if gv.Group == "" {
		path = "/api/" + gv.Version
	}

	list := &metav1.APIResourceList{}
	if err := r.rest.Get().AbsPath(path).Do(ctx).Into(list); err != nil {
		return nil, err
	}

	kinds := make(map[string]string, len(list.APIResources))
	for i := range list.APIResources {
		res := &list.APIResources[i]
		if strings.Contains(res.Name, "/") {
			continue
		}
		if _, seen := kinds[res.Kind]; !seen {
			kinds[res.Kind] = res.Name
		}
	}
	return kinds, nil
}

// GVRFromUnstructured derives GroupVersionResource from an unstructured object.
func GVRFromUnstructured(obj *unstructured.Unstructured) schema.GroupVersionResource {
	gvk := obj.GroupVersionKind()
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: KindToResource(gvk.Kind),
	}
}

// knownKindResources maps Kind to its plural resource name for well-known types.
// This avoids incorrect heuristic pluralization (e.g., Endpoints -> endpointses).
var knownKindResources = map[string]string{
	"Namespace":                        "namespaces",
	"ServiceAccount":                   "serviceaccounts",
	"Secret":                           "secrets",
	"ConfigMap":                        "configmaps",
	"PersistentVolume":                 "persistentvolumes",
	"PersistentVolumeClaim":            "persistentvolumeclaims",
	"Service":                          "services",
	"Endpoints":                        "endpoints",
	"EndpointSlice":                    "endpointslices",
	"ClusterRole":                      "clusterroles",
	"ClusterRoleBinding":               "clusterrolebindings",
	"Role":                             "roles",
	"RoleBinding":                      "rolebindings",
	"StorageClass":                     "storageclasses",
	"Deployment":                       "deployments",
	"StatefulSet":                      "statefulsets",
	"DaemonSet":                        "daemonsets",
	"ReplicaSet":                       "replicasets",
	"Job":                              "jobs",
	"CronJob":                          "cronjobs",
	"Ingress":                          "ingresses",
	"IngressClass":                     "ingressclasses",
	"NetworkPolicy":                    "networkpolicies",
	"HorizontalPodAutoscaler":          "horizontalpodautoscalers",
	"VerticalPodAutoscaler":            "verticalpodautoscalers",
	"PodDisruptionBudget":              "poddisruptionbudgets",
	"ValidatingWebhookConfiguration":   "validatingwebhookconfigurations",
	"MutatingWebhookConfiguration":     "mutatingwebhookconfigurations",
	kindCustomResourceDefinition:       "customresourcedefinitions",
	"ResourceQuota":                    "resourcequotas",
	"LimitRange":                       "limitranges",
	"Pod":                              "pods",
	"Node":                             "nodes",
	"Event":                            "events",
	"PriorityClass":                    "priorityclasses",
	"ValidatingAdmissionPolicy":        "validatingadmissionpolicies",
	"ValidatingAdmissionPolicyBinding": "validatingadmissionpolicybindings",
}

// KindToResource converts a Kind to its plural resource name.
// Uses a known lookup table for common types, falls back to heuristic.
func KindToResource(kind string) string {
	if resource, ok := knownKindResources[kind]; ok {
		return resource
	}
	return HeuristicPluralize(kind)
}

// HeuristicPluralize applies simple English pluralization rules.
func HeuristicPluralize(kind string) string {
	lower := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(lower, "ss") || strings.HasSuffix(lower, "sh") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x"):
		return lower + "es"
	case strings.HasSuffix(lower, "s"):
		// Already plural (e.g., Endpoints)
		return lower
	case strings.HasSuffix(lower, "y") && !isVowel(lower[len(lower)-2]):
		return lower[:len(lower)-1] + "ies"
	default:
		return lower + "s"
	}
}

func isVowel(b byte) bool {
	return b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u'
}
