package cue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// ErrNoManifests is returned when no manifests are found in the module.
	ErrNoManifests = errors.New("no manifests found")
	// ErrRenderFailed is returned when manifest rendering fails.
	ErrRenderFailed = errors.New("render failed")
)

// Renderer renders modules to Kubernetes manifests.
type Renderer struct{}

// NewRenderer creates a new Renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// RenderModule generates manifests from a module.
func (r *Renderer) RenderModule(ctx context.Context, module *Module) (*ManifestSet, error) {
	manifestSet := NewManifestSet(module.Metadata)

	// Look for manifests in common locations
	locations := []string{
		"manifests",     // manifests: [...]
		"resources",     // resources: [...]
		"objects",       // objects: [...]
		"kubernetes",    // kubernetes: [...]
		"out.manifests", // out: manifests: [...]
	}

	var foundManifests bool
	for _, loc := range locations {
		path := cue.ParsePath(loc)
		field := module.Root.LookupPath(path)
		if field.Exists() {
			if err := r.extractManifests(field, manifestSet, ""); err != nil {
				return nil, fmt.Errorf("extracting manifests from %s: %w", loc, err)
			}
			foundManifests = true
			break
		}
	}

	// If no common locations found, try to find any Kubernetes-like objects
	if !foundManifests {
		if err := r.findKubernetesObjects(module.Root, manifestSet); err != nil {
			return nil, fmt.Errorf("finding Kubernetes objects: %w", err)
		}
	}

	if manifestSet.Len() == 0 {
		return nil, ErrNoManifests
	}

	return manifestSet, nil
}

// RenderBundle generates manifests from a bundle (all modules combined).
func (r *Renderer) RenderBundle(ctx context.Context, bundle *Bundle) (*ManifestSet, error) {
	manifestSet := NewManifestSet(ModuleMetadata{
		Name:    bundle.Metadata.Name,
		Version: bundle.Metadata.Version,
	})

	// Render each module in the bundle
	for name, module := range bundle.Modules {
		moduleManifests, err := r.RenderModule(ctx, module)
		if err != nil {
			return nil, fmt.Errorf("rendering module %s: %w", name, err)
		}

		// Add module manifests with component name
		for _, m := range moduleManifests.Manifests {
			componentName := m.ComponentName
			if componentName == "" {
				componentName = name
			}
			manifestSet.AddWithWeight(m.Object, componentName, m.Weight)
		}
	}

	if manifestSet.Len() == 0 {
		return nil, ErrNoManifests
	}

	return manifestSet, nil
}

// extractManifests extracts manifests from a CUE value (list or struct).
func (r *Renderer) extractManifests(v cue.Value, ms *ManifestSet, componentName string) error {
	// Check if it's a list
	iter, err := v.List()
	if err == nil {
		// It's a list, iterate through elements
		for iter.Next() {
			elem := iter.Value()
			obj, err := r.valueToUnstructured(elem)
			if err != nil {
				return fmt.Errorf("converting list element: %w", err)
			}
			if obj != nil {
				ms.Add(obj, componentName)
			}
		}
		return nil
	}

	// Check if it's a struct with fields that are Kubernetes objects
	iter2, err := v.Fields()
	if err != nil {
		return fmt.Errorf("iterating fields: %w", err)
	}

	for iter2.Next() {
		fieldValue := iter2.Value()

		// Check if this field is a Kubernetes object
		if r.looksLikeKubernetesObject(fieldValue) {
			obj, err := r.valueToUnstructured(fieldValue)
			if err != nil {
				return fmt.Errorf("converting field %s: %w", iter2.Label(), err)
			}
			if obj != nil {
				compName := componentName
				if compName == "" {
					compName = iter2.Label()
				}
				ms.Add(obj, compName)
			}
		} else {
			// Recursively check nested structures
			if fieldValue.Kind() == cue.StructKind || fieldValue.Kind() == cue.ListKind {
				compName := componentName
				if compName == "" {
					compName = iter2.Label()
				}
				if err := r.extractManifests(fieldValue, ms, compName); err != nil {
					// Ignore errors in nested structures
					continue
				}
			}
		}
	}

	return nil
}

// findKubernetesObjects searches the entire CUE value for Kubernetes objects.
func (r *Renderer) findKubernetesObjects(v cue.Value, ms *ManifestSet) error {
	return r.walkValue(v, ms, "")
}

// walkValue recursively walks a CUE value looking for Kubernetes objects.
func (r *Renderer) walkValue(v cue.Value, ms *ManifestSet, componentName string) error {
	// Check if this value is a Kubernetes object
	if r.looksLikeKubernetesObject(v) {
		obj, err := r.valueToUnstructured(v)
		if err != nil {
			return err
		}
		if obj != nil {
			ms.Add(obj, componentName)
		}
		return nil
	}

	// Handle lists
	if v.Kind() == cue.ListKind {
		iter, _ := v.List()
		for iter.Next() {
			if err := r.walkValue(iter.Value(), ms, componentName); err != nil {
				continue // Ignore errors in list elements
			}
		}
		return nil
	}

	// Handle structs
	if v.Kind() == cue.StructKind {
		iter, err := v.Fields()
		if err != nil {
			return nil
		}

		for iter.Next() {
			label := iter.Label()
			// Skip hidden fields and definitions
			if len(label) > 0 && (label[0] == '_' || label[0] == '#') {
				continue
			}

			compName := componentName
			if compName == "" {
				compName = label
			}
			if err := r.walkValue(iter.Value(), ms, compName); err != nil {
				continue // Ignore errors
			}
		}
	}

	return nil
}

// looksLikeKubernetesObject checks if a CUE value looks like a Kubernetes object.
func (r *Renderer) looksLikeKubernetesObject(v cue.Value) bool {
	if v.Kind() != cue.StructKind {
		return false
	}

	// Check for required Kubernetes fields
	apiVersion := v.LookupPath(cue.ParsePath("apiVersion"))
	kind := v.LookupPath(cue.ParsePath("kind"))

	return apiVersion.Exists() && kind.Exists()
}

// valueToUnstructured converts a CUE value to an unstructured Kubernetes object.
func (r *Renderer) valueToUnstructured(v cue.Value) (*unstructured.Unstructured, error) {
	if !r.looksLikeKubernetesObject(v) {
		return nil, nil
	}

	// Validate the value is concrete (no non-concrete values)
	if err := v.Validate(cue.Concrete(true)); err != nil {
		return nil, fmt.Errorf("%w: value is not concrete: %v", ErrRenderFailed, err)
	}

	// Convert to JSON
	jsonBytes, err := v.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("%w: marshaling to JSON: %v", ErrRenderFailed, err)
	}

	// Parse into unstructured
	var obj map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &obj); err != nil {
		return nil, fmt.Errorf("%w: unmarshaling JSON: %v", ErrRenderFailed, err)
	}

	return &unstructured.Unstructured{Object: obj}, nil
}
