// Package kubetest holds test support for code that takes a
// kubernetes.Client. It is imported by tests only.
package kubetest

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Outcome is the answer a FakeResources gives for one kind: the resource
// name, or the error.
type Outcome struct {
	Resource string
	Err      error
}

// FakeResources is the resource resolver for a client backed by client-go's
// fake dynamic client. That fake files every object it is seeded with under
// meta.UnsafeGuessKindToResource, so a resolver for it has to give the same
// answer unless a test says otherwise; a real cluster is never asked this
// way.
type FakeResources struct {
	mu       sync.Mutex
	outcomes map[schema.GroupVersionKind]Outcome
	calls    map[schema.GroupVersionKind]int
}

// Resources returns the resolver to set as kubernetes.Client.Resources on a
// client built over a fake dynamic client.
func Resources() *FakeResources {
	return ResourcesWith(nil)
}

// ResourcesWith is Resources with a fixed outcome for the kinds in outcomes:
// a resource name the fake tracker holds the objects under, or the error a
// resolver would return.
func ResourcesWith(outcomes map[schema.GroupVersionKind]Outcome) *FakeResources {
	f := &FakeResources{outcomes: map[schema.GroupVersionKind]Outcome{}, calls: map[schema.GroupVersionKind]int{}}
	for gvk, o := range outcomes {
		f.outcomes[gvk] = o
	}
	return f
}

// Set changes the outcome for gvk from now on.
func (f *FakeResources) Set(gvk schema.GroupVersionKind, o Outcome) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outcomes[gvk] = o
}

// Calls returns how many times gvk was resolved.
func (f *FakeResources) Calls(gvk schema.GroupVersionKind) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[gvk]
}

// ResourceFor returns the fixed outcome for gvk, or else the resource the
// fake dynamic client stores gvk under.
func (f *FakeResources) ResourceFor(_ context.Context, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[gvk]++
	if o, ok := f.outcomes[gvk]; ok {
		if o.Err != nil {
			return schema.GroupVersionResource{}, o.Err
		}
		return gvk.GroupVersion().WithResource(o.Resource), nil
	}
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	return gvr, nil
}

// GVR returns the resource the fake dynamic client stores obj under, for a
// test that reads or changes the fake's tracker directly.
func GVR(obj *unstructured.Unstructured) schema.GroupVersionResource {
	gvr, _ := meta.UnsafeGuessKindToResource(obj.GroupVersionKind())
	return gvr
}
