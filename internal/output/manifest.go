package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/pkg/weights"
)

// ManifestOptions controls manifest output formatting.
type ManifestOptions struct {
	// Format specifies output format: "yaml" or "json"
	Format Format
	// Writer is the output destination
	Writer io.Writer
}

// ResourceInfo provides information about a resource for output formatting.
// This interface allows the output package to work with resources without
// importing the build package.
type ResourceInfo interface {
	GetObject() *unstructured.Unstructured
	GetGVK() schema.GroupVersionKind
	GetKind() string
	GetName() string
	GetNamespace() string
	GetComponent() string
	GetTransformer() string
}

// WriteManifests writes resources to the writer in the specified format.
// Resources are sorted by weight for consistent output.
func WriteManifests(resources []ResourceInfo, opts ManifestOptions) error {
	if len(resources) == 0 {
		return nil
	}

	// Sort resources by weight then by name for deterministic output
	sortResourceInfos(resources)

	switch opts.Format {
	case FormatJSON:
		return writeJSON(resources, opts.Writer)
	case FormatYAML:
		return writeYAML(resources, opts.Writer)
	case FormatTable, FormatDir:
		return fmt.Errorf("format %s not supported for manifest output", opts.Format)
	}
	return writeYAML(resources, opts.Writer) // Default to YAML
}

// WriteUnstructuredManifests writes unstructured resources directly.
// This is a convenience function when you only have the raw objects.
func WriteUnstructuredManifests(objects []*unstructured.Unstructured, opts ManifestOptions) error {
	if len(objects) == 0 {
		return nil
	}

	// Sort by weight then by name
	sort.Slice(objects, func(i, j int) bool {
		wi := weights.GetWeight(objects[i].GroupVersionKind())
		wj := weights.GetWeight(objects[j].GroupVersionKind())
		if wi != wj {
			return wi < wj
		}
		nsi := objects[i].GetNamespace()
		nsj := objects[j].GetNamespace()
		if nsi != nsj {
			return nsi < nsj
		}
		return objects[i].GetName() < objects[j].GetName()
	})

	switch opts.Format {
	case FormatJSON:
		return writeJSONObjects(objects, opts.Writer)
	case FormatYAML:
		return writeYAMLObjects(objects, opts.Writer)
	case FormatTable, FormatDir:
		return fmt.Errorf("format %s not supported for manifest output", opts.Format)
	}
	return writeYAMLObjects(objects, opts.Writer) // Default to YAML
}

// sortResourceInfos sorts resources by weight, then by namespace, then by name.
func sortResourceInfos(resources []ResourceInfo) {
	sort.Slice(resources, func(i, j int) bool {
		// Primary: sort by weight
		wi := weights.GetWeight(resources[i].GetGVK())
		wj := weights.GetWeight(resources[j].GetGVK())
		if wi != wj {
			return wi < wj
		}

		// Secondary: sort by namespace
		nsi := resources[i].GetNamespace()
		nsj := resources[j].GetNamespace()
		if nsi != nsj {
			return nsi < nsj
		}

		// Tertiary: sort by name
		return resources[i].GetName() < resources[j].GetName()
	})
}

// writeYAML writes resources as YAML documents separated by ---.
// The yaml.v3 encoder automatically adds document separators between documents.
func writeYAML(resources []ResourceInfo, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)

	for _, res := range resources {
		if err := encoder.Encode(res.GetObject().Object); err != nil {
			return fmt.Errorf("encoding resource %s/%s: %w",
				res.GetKind(), res.GetName(), err)
		}
	}

	return encoder.Close()
}

// writeYAMLObjects writes unstructured objects as YAML documents.
// The yaml.v3 encoder automatically adds document separators between documents.
func writeYAMLObjects(objects []*unstructured.Unstructured, w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)

	for _, obj := range objects {
		if err := encoder.Encode(obj.Object); err != nil {
			return fmt.Errorf("encoding resource %s/%s: %w",
				obj.GetKind(), obj.GetName(), err)
		}
	}

	return encoder.Close()
}

// writeJSON writes resources as a JSON array.
func writeJSON(resources []ResourceInfo, w io.Writer) error {
	objects := make([]map[string]any, len(resources))
	for i, res := range resources {
		objects[i] = res.GetObject().Object
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(objects); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// writeJSONObjects writes unstructured objects as JSON array.
func writeJSONObjects(objects []*unstructured.Unstructured, w io.Writer) error {
	arr := make([]map[string]any, len(objects))
	for i, obj := range objects {
		arr[i] = obj.Object
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(arr); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// WriteResource writes a single resource to the writer.
func WriteResource(resource *unstructured.Unstructured, format Format, w io.Writer) error {
	switch format {
	case FormatJSON:
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(resource.Object)
	case FormatYAML:
		encoder := yaml.NewEncoder(w)
		encoder.SetIndent(2)
		err := encoder.Encode(resource.Object)
		if closeErr := encoder.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		return err
	case FormatTable, FormatDir:
		return fmt.Errorf("format %s not supported for single resource output", format)
	}
	// Default to YAML
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	err := encoder.Encode(resource.Object)
	if closeErr := encoder.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}
