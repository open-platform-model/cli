package release

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/build/module"
)

// extractComponentsFromDefinition extracts components from #components (definition).
func extractComponentsFromDefinition(concreteRelease cue.Value) (map[string]*module.LoadedComponent, error) {
	componentsValue := concreteRelease.LookupPath(cue.ParsePath("#components"))
	if !componentsValue.Exists() {
		return nil, fmt.Errorf("module missing '#components' field")
	}

	components := make(map[string]*module.LoadedComponent)

	iter, err := componentsValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		compValue := iter.Value()

		comp := extractComponent(name, compValue)
		components[name] = comp
	}

	return components, nil
}

// extractComponent extracts a single component with its metadata.
func extractComponent(name string, value cue.Value) *module.LoadedComponent {
	comp := &module.LoadedComponent{
		Name:        name,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Resources:   make(map[string]cue.Value),
		Traits:      make(map[string]cue.Value),
		Value:       value,
	}

	// Extract metadata.name if present, otherwise use field name
	metaName := value.LookupPath(cue.ParsePath("metadata.name"))
	if metaName.Exists() {
		if str, err := metaName.String(); err == nil {
			comp.Name = str
		}
	}

	// Extract #resources
	resourcesValue := value.LookupPath(cue.ParsePath("#resources"))
	if resourcesValue.Exists() {
		iter, err := resourcesValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Resources[fqn] = iter.Value()
			}
		}
	}

	// Extract #traits
	traitsValue := value.LookupPath(cue.ParsePath("#traits"))
	if traitsValue.Exists() {
		iter, err := traitsValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Traits[fqn] = iter.Value()
			}
		}
	}

	// Extract annotations from metadata
	extractAnnotations(value, comp.Annotations)

	// Extract labels from metadata
	labelsValue := value.LookupPath(cue.ParsePath("metadata.labels"))
	if labelsValue.Exists() {
		iter, err := labelsValue.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					comp.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	return comp
}

// extractAnnotations extracts annotations from component metadata into the target map.
func extractAnnotations(value cue.Value, annotations map[string]string) {
	annotationsValue := value.LookupPath(cue.ParsePath("metadata.annotations"))
	if !annotationsValue.Exists() {
		return
	}
	iter, err := annotationsValue.Fields()
	if err != nil {
		return
	}
	for iter.Next() {
		if str, err := iter.Value().String(); err == nil {
			annotations[iter.Selector().Unquoted()] = str
		}
	}
}

// extractReleaseMetadata extracts release metadata from the overlay-computed
// #opmReleaseMeta definition and module metadata.
//
// The overlay computes identity, labels, fqn, and version via CUE.
// Module identity comes from metadata.identity (computed by #Module).
func extractReleaseMetadata(concreteRelease cue.Value, opts Options) Metadata {
	metadata := Metadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	// Extract from overlay-computed #opmReleaseMeta
	relMeta := concreteRelease.LookupPath(cue.ParsePath("#opmReleaseMeta"))
	if relMeta.Exists() && relMeta.Err() == nil {
		// Version
		if v := relMeta.LookupPath(cue.ParsePath("version")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.Version = str
			}
		}

		// FQN
		if v := relMeta.LookupPath(cue.ParsePath("fqn")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}

		// Release identity (computed by CUE uuid.SHA1)
		if v := relMeta.LookupPath(cue.ParsePath("identity")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.ReleaseIdentity = str
			}
		}

		// Labels (includes module labels + standard release labels)
		if labelsVal := relMeta.LookupPath(cue.ParsePath("labels")); labelsVal.Exists() {
			iter, err := labelsVal.Fields()
			if err == nil {
				for iter.Next() {
					if str, err := iter.Value().String(); err == nil {
						metadata.Labels[iter.Selector().Unquoted()] = str
					}
				}
			}
		}
	} else {
		// Fallback: extract from module metadata directly (no overlay)
		extractMetadataFallback(concreteRelease, &metadata)
	}

	// Extract module identity from metadata.identity (always from module, not release)
	if v := concreteRelease.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Identity = str
		}
	}

	return metadata
}

// extractMetadataFallback extracts metadata from module fields when overlay is not available.
func extractMetadataFallback(concreteRelease cue.Value, metadata *Metadata) {
	if v := concreteRelease.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Version = str
		}
	}

	if v := concreteRelease.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.FQN = str
		}
	}
	if metadata.FQN == "" {
		if v := concreteRelease.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}
	}

	if labelsVal := concreteRelease.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					metadata.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}
}
