// Package kubetest holds test support for code that takes a
// kubernetes.Client. It is imported by tests only.
package kubetest

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// FakeResources is the resource resolver for a client backed by client-go's
// fake dynamic client. That fake files every object it is seeded with under
// meta.UnsafeGuessKindToResource, so a resolver for it has to give the same
// answer; a real cluster is never asked this way.
type FakeResources struct{}

// Resources returns the resolver to set as kubernetes.Client.Resources on a
// client built over a fake dynamic client.
func Resources() FakeResources {
	return FakeResources{}
}

// ResourceFor returns the resource the fake dynamic client stores gvk under.
func (FakeResources) ResourceFor(_ context.Context, gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	return gvr, nil
}
