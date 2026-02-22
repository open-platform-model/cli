// Package modulerelease defines the ModuleRelease and ReleaseMetadata types,
// mirroring the module_release.cue definition in the CUE catalog. A
// ModuleRelease represents the built module release after the build phase,
// before any transformations are applied.
package modulerelease

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/core/module"
	opmerrors "github.com/opmodel/cli/internal/errors"
)

// ModuleRelease represents the built module release after the build phase, before any transformations are applied.
// Contains the fully concrete components with all metadata extracted and values merged, ready for matching and transformation.
type ModuleRelease struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// Metadata contains release-level identity information for a deployed module.
	// This metadata is used for labeling resources, inventory tracking, and verbose output.
	Metadata *ReleaseMetadata `json:"metadata"`

	// The original module definition, preserved for reference and verbose output.
	Module module.Module `json:"module"`

	// Concrete components with all metadata extracted and values merged, ready for matching and transformation.
	// Must preserve the original order of the #ModuleRelease.components map for deterministic output and to support index-based inventory tracking.
	Components map[string]*component.Component `json:"components,omitempty"`

	// The values from the Module Release. End-user values.
	Values cue.Value `json:"values,omitempty"`
}

// ValidateValues validates the user-supplied Values field against the Module.Config CUE schema.
// Uses recursive CUE field walking on the already-populated cue.Value fields.
// Returns nil if Module.Config or Values are not present (nothing to validate).
// This is a pure read operation — it does not mutate any field on ModuleRelease.
func (rel *ModuleRelease) ValidateValues() error {
	if !rel.Module.Config.Exists() || !rel.Values.Exists() {
		return nil
	}
	combined := validateFieldsRecursive(rel.Module.Config, rel.Values, []string{"values"}, nil)
	if combined == nil {
		return nil
	}
	return &opmerrors.ValidationError{
		Message: "values do not satisfy #config schema",
		Cause:   combined,
	}
}

// Validate checks that all components in Components are concrete CUE values,
// confirming the release is ready for transformer matching.
// This is a readiness gate, not a schema check; it does not re-run ValidateValues.
// Returns nil if Components is empty (a module with no components is valid).
// This is a pure read operation — it does not mutate any field on ModuleRelease.
func (rel *ModuleRelease) Validate() error {
	var concreteErrors []error
	for name, comp := range rel.Components {
		if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
			concreteErrors = append(concreteErrors, fmt.Errorf("component %q: %w", name, err))
		}
	}
	if len(concreteErrors) > 0 {
		return &opmerrors.ValidationError{
			Message: fmt.Sprintf("%d component(s) have non-concrete values - check that all required values are provided", len(concreteErrors)),
			Cause:   concreteErrors[0],
		}
	}
	return nil
}

// ReleaseMetadata contains release-level identity information for a deployed module.
// This metadata is used for labeling resources, inventory tracking, and verbose output.
type ReleaseMetadata struct {
	// Name is the release name (resolved from RenderOptions.Name or module.metadata.name).
	Name string `json:"name"`

	// Namespace is the target namespace.
	// From default config.cue or --namespace flag.
	Namespace string `json:"namespace"`

	// UUID is the release identity UUID.
	// Deterministic UUID5 computed from ModuleMetadata.FQN+ReleaseMetadata.Name+ReleaseMetadata.Namespace.
	UUID string `json:"uuid"`

	// Labels from the module release.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module release.
	// TODO: not yet implemented. Populate this in extractReleaseMetadata (internal/build/release/metadata.go)
	// once CUE annotation extraction is added, then wire it into TransformerContext.ToMap
	// (internal/build/transform/context.go) alongside Labels.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Components lists the component names rendered in this release.
	// This is a CLI only field used for verbose output and inventory tracking; not part of the core render contract.
	Components []string `json:"components,omitempty"`
}
