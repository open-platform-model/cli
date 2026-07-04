// Package operator implements the opm-operator lifecycle surface: embedding
// the pinned operator manifest, deriving install/uninstall plans from it,
// applying and waiting for readiness, and the uninstall safety checks.
package operator

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// PinnedOperatorVersion is the opm-operator release tag whose dist/install.yaml
// is embedded below. Refresh both together with `task operator:sync VERSION=<tag>`.
const PinnedOperatorVersion = "v1.0.0-alpha.2"

//go:embed dist/install.yaml
var embeddedManifest []byte

// ParseManifest decodes a multi-document YAML manifest into unstructured objects.
func ParseManifest(data []byte) ([]*unstructured.Unstructured, error) {
	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)

	var objs []*unstructured.Unstructured
	for {
		var raw map[string]any
		if err := decoder.Decode(&raw); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decoding manifest document: %w", err)
		}
		if len(raw) == 0 {
			continue
		}
		objs = append(objs, &unstructured.Unstructured{Object: raw})
	}

	return objs, nil
}

// EmbeddedManifest parses and returns the objects from the embedded, pinned
// operator manifest.
func EmbeddedManifest() ([]*unstructured.Unstructured, error) {
	return ParseManifest(embeddedManifest)
}
