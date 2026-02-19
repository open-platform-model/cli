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

// extractReleaseMetadata extracts release-level metadata from the overlay-computed
// #opmReleaseMeta definition.
//
// Fields extracted: Name (from opts), Namespace (from opts), UUID (from
// #opmReleaseMeta.identity), Labels (from #opmReleaseMeta.labels with fallback
// to metadata.labels).
func extractReleaseMetadata(concreteRelease cue.Value, opts Options) ReleaseMetadata {
	relMeta := ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	// Extract from overlay-computed #opmReleaseMeta
	opmRelMeta := concreteRelease.LookupPath(cue.ParsePath("#opmReleaseMeta"))
	if opmRelMeta.Exists() && opmRelMeta.Err() == nil {
		// Release identity (computed by CUE uuid.SHA1)
		if v := opmRelMeta.LookupPath(cue.ParsePath("identity")); v.Exists() {
			if str, err := v.String(); err == nil {
				relMeta.UUID = str
			}
		}

		// Labels (includes module labels + standard release labels)
		if labelsVal := opmRelMeta.LookupPath(cue.ParsePath("labels")); labelsVal.Exists() {
			iter, err := labelsVal.Fields()
			if err == nil {
				for iter.Next() {
					if str, err := iter.Value().String(); err == nil {
						relMeta.Labels[iter.Selector().Unquoted()] = str
					}
				}
			}
		}
	} else {
		// Fallback: extract from module metadata directly (no overlay)
		extractReleaseMetadataFallback(concreteRelease, &relMeta)
	}

	return relMeta
}

// extractModuleMetadata extracts module-level metadata from the CUE value.
//
// Fields extracted:
//   - Name: from metadata.name
//   - DefaultNamespace: from metadata.defaultNamespace
//   - FQN: from #opmReleaseMeta.fqn, fallback metadata.fqn, fallback metadata.apiVersion
//   - Version: from #opmReleaseMeta.version, fallback metadata.version
//   - UUID: from metadata.identity
//   - Labels: same source as release labels (behavioral parity)
func extractModuleMetadata(concreteRelease cue.Value) module.ModuleMetadata {
	modMeta := module.ModuleMetadata{
		Labels: make(map[string]string),
	}

	extractModuleMetadataFromCUE(concreteRelease, &modMeta)

	opmRelMeta := concreteRelease.LookupPath(cue.ParsePath("#opmReleaseMeta"))
	if opmRelMeta.Exists() && opmRelMeta.Err() == nil {
		extractModuleMetadataFromOpmRelMeta(opmRelMeta, &modMeta)
	} else {
		// Fallback: extract from module metadata directly (no overlay)
		extractModuleMetadataFallback(concreteRelease, &modMeta)
	}

	return modMeta
}

// extractModuleMetadataFromCUE extracts module metadata fields directly from the CUE value.
// These fields are always read from module metadata, regardless of overlay presence.
func extractModuleMetadataFromCUE(v cue.Value, modMeta *module.ModuleMetadata) {
	if f := v.LookupPath(cue.ParsePath("metadata.name")); f.Exists() {
		if str, err := f.String(); err == nil {
			modMeta.Name = str
		}
	}

	if f := v.LookupPath(cue.ParsePath("metadata.defaultNamespace")); f.Exists() {
		if str, err := f.String(); err == nil {
			modMeta.DefaultNamespace = str
		}
	}

	if f := v.LookupPath(cue.ParsePath("metadata.identity")); f.Exists() {
		if str, err := f.String(); err == nil {
			modMeta.UUID = str
		}
	}
}

// extractModuleMetadataFromOpmRelMeta extracts FQN, Version, and Labels from the
// overlay-computed #opmReleaseMeta definition.
func extractModuleMetadataFromOpmRelMeta(opmRelMeta cue.Value, modMeta *module.ModuleMetadata) {
	if v := opmRelMeta.LookupPath(cue.ParsePath("fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			modMeta.FQN = str
		}
	}

	if v := opmRelMeta.LookupPath(cue.ParsePath("version")); v.Exists() {
		if str, err := v.String(); err == nil {
			modMeta.Version = str
		}
	}

	// Labels (same source as release for behavioral parity)
	if labelsVal := opmRelMeta.LookupPath(cue.ParsePath("labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					modMeta.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}
}

// extractReleaseMetadataFallback extracts release metadata from module fields
// when overlay is not available.
func extractReleaseMetadataFallback(concreteRelease cue.Value, relMeta *ReleaseMetadata) {
	if labelsVal := concreteRelease.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					relMeta.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}
}

// extractModuleMetadataFallback extracts module metadata from module fields
// when overlay is not available.
func extractModuleMetadataFallback(concreteRelease cue.Value, modMeta *module.ModuleMetadata) {
	if v := concreteRelease.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			modMeta.Version = str
		}
	}

	if v := concreteRelease.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			modMeta.FQN = str
		}
	}
	if modMeta.FQN == "" {
		if v := concreteRelease.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				modMeta.FQN = str
			}
		}
	}

	if labelsVal := concreteRelease.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					modMeta.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}
}
