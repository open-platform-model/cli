package core

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resource represents a single rendered platform resource.
type Resource struct {
	Object      *unstructured.Unstructured
	Component   string
	Transformer string
}

// GVK returns the GroupVersionKind of the resource.
func (r *Resource) GVK() schema.GroupVersionKind {
	return r.Object.GroupVersionKind()
}

// Kind returns the resource kind (e.g., "Deployment").
func (r *Resource) Kind() string {
	return r.Object.GetKind()
}

// Name returns the resource name from metadata.
func (r *Resource) Name() string {
	return r.Object.GetName()
}

// Namespace returns the resource namespace from metadata.
// Empty string for cluster-scoped resources.
func (r *Resource) Namespace() string {
	return r.Object.GetNamespace()
}

// Labels returns the resource labels.
func (r *Resource) Labels() map[string]string {
	return r.Object.GetLabels()
}

// GetObject returns the underlying unstructured object.
func (r *Resource) GetObject() *unstructured.Unstructured {
	return r.Object
}

// GetGVK returns the GroupVersionKind.
func (r *Resource) GetGVK() schema.GroupVersionKind {
	return r.GVK()
}

// GetKind returns the resource kind.
func (r *Resource) GetKind() string {
	return r.Kind()
}

// GetName returns the resource name.
func (r *Resource) GetName() string {
	return r.Name()
}

// GetNamespace returns the resource namespace.
func (r *Resource) GetNamespace() string {
	return r.Namespace()
}

// GetComponent returns the source component name.
func (r *Resource) GetComponent() string {
	return r.Component
}

// GetTransformer returns the transformer FQN.
func (r *Resource) GetTransformer() string {
	return r.Transformer
}
