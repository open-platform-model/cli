package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
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

// DiscoveryError reports that the API discovery request for a group and
// version failed, so the cluster gave no answer about the kinds it serves
// there. It wraps the request error, so apierrors.IsForbidden and the like
// see it. A caller that holds one stops: every further object of that group
// and version would fail the same way.
type DiscoveryError struct {
	GroupVersion schema.GroupVersion
	Err          error
}

func (e *DiscoveryError) Error() string {
	return fmt.Sprintf("discovering the resources of %s: %v", e.GroupVersion, e.Err)
}

func (e *DiscoveryError) Unwrap() error { return e.Err }

// IsDiscoveryFailure reports whether err is, or wraps, a *DiscoveryError.
func IsDiscoveryFailure(err error) bool {
	var failed *DiscoveryError
	return errors.As(err, &failed)
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
// (/api/v1, /apis/<group>/<version>) through client-go's discovery client and
// keeps it in memory. It reads single documents instead of building a
// RESTMapper over every group, because that mapper drops a group whose
// discovery failed, and a failed request would then look like a kind that is
// not served.
//
// Only a document that holds the asked kind answers from memory. A kind that
// is missing is asked for again on every call, because a
// CustomResourceDefinition can start to serve it during the command; that
// costs one request, the same as the read it stands in front of.
type discoveryResolver struct {
	discovery discovery.ServerResourcesInterfaceWithContext

	mu     sync.Mutex
	served map[schema.GroupVersion]map[string]string // kind -> resource
}

// NewDiscoveryResolver returns a ResourceResolver that asks the API server
// behind client.
func NewDiscoveryResolver(client discovery.DiscoveryInterface) ResourceResolver {
	return &discoveryResolver{
		discovery: discovery.ToDiscoveryInterfaceWithContext(client),
		served:    map[schema.GroupVersion]map[string]string{},
	}
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
		return schema.GroupVersionResource{}, &DiscoveryError{GroupVersion: gv, Err: err}
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
	list, err := r.discovery.ServerResourcesForGroupVersionWithContext(ctx, gv.String())
	if err != nil {
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
