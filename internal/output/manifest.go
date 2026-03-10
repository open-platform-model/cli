package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	resourceorder "github.com/opmodel/cli/internal/resourceorder"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WriteManifests writes resources to the writer in the specified format.
// Resources are sorted by weight for consistent output.
func WriteManifests(resources []*unstructured.Unstructured, opts ManifestOptions) error {
	if len(resources) == 0 {
		return nil
	}

	// Sort resources by weight then by name for deterministic output.
	sortResources(resources)

	switch opts.Format {
	case FormatJSON:
		return writeJSON(resources, opts.Writer)
	case FormatYAML:
		return writeYAML(resources, opts.Writer)
	case FormatTable, FormatDir, FormatWide:
		return fmt.Errorf("format %s not supported for manifest output", opts.Format)
	}
	return writeYAML(resources, opts.Writer) // Default to YAML
}

// ManifestOptions controls manifest output formatting.
type ManifestOptions struct {
	// Format specifies output format: "yaml" or "json"
	Format Format
	// Writer is the output destination
	Writer io.Writer
}

// sortResources sorts resources by weight, then by namespace, then by name.
// Intentional 3-key sort for display purposes (weight → namespace → name).
// Does not need to match the 5-key apply order (weight → group → kind → namespace → name)
// since this function only controls output file and table ordering.
func sortResources(resources []*unstructured.Unstructured) {
	sort.Slice(resources, func(i, j int) bool {
		// Primary: sort by weight
		wi := resourceorder.GetWeight(resources[i].GroupVersionKind())
		wj := resourceorder.GetWeight(resources[j].GroupVersionKind())
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
func writeYAML(resources []*unstructured.Unstructured, w io.Writer) error {
	for i, res := range resources {
		if i > 0 {
			if _, err := fmt.Fprint(w, "---\n"); err != nil {
				return fmt.Errorf("writing separator: %w", err)
			}
		}
		yamlBytes, err := marshalYAML(res)
		if err != nil {
			return fmt.Errorf("encoding resource %s/%s: %w", res.GetKind(), res.GetName(), err)
		}
		if _, err := w.Write(yamlBytes); err != nil {
			return fmt.Errorf("writing resource %s/%s: %w", res.GetKind(), res.GetName(), err)
		}
	}
	return nil
}

// writeJSON writes resources as a JSON array.
func writeJSON(resources []*unstructured.Unstructured, w io.Writer) error {
	objects := make([]json.RawMessage, len(resources))
	for i, res := range resources {
		j, err := json.Marshal(res.Object)
		if err != nil {
			return fmt.Errorf("encoding resource %s/%s: %w", res.GetKind(), res.GetName(), err)
		}
		objects[i] = j
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(objects); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	return nil
}

// marshalYAML serializes an Unstructured resource to YAML bytes.
func marshalYAML(res *unstructured.Unstructured) ([]byte, error) {
	return yaml.Marshal(res.Object)
}

// writeResource writes a single resource to the writer.
func writeResource(res *unstructured.Unstructured, format Format, w io.Writer) error {
	switch format {
	case FormatJSON:
		j, err := json.Marshal(res.Object)
		if err != nil {
			return err
		}
		var obj json.RawMessage = j
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(obj)
	case FormatYAML:
		// handled below
	case FormatTable, FormatDir, FormatWide:
		return fmt.Errorf("format %s not supported for single resource output", format)
	}
	// Default/YAML path
	y, err := marshalYAML(res)
	if err != nil {
		return err
	}
	_, err = w.Write(y)
	return err
}
