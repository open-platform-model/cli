// Package release provides release building functionality for OPM modules.
package release

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/build/module"
)

// Options configures release building.
type Options struct {
	Name      string // Required: release name
	Namespace string // Required: target namespace
	PkgName   string // Internal: CUE package name (set by InspectModule, skip detectPackageName)
}

// BuiltRelease is the result of building a release.
type BuiltRelease struct {
	Value      cue.Value                          // The concrete module value (with #config injected)
	Components map[string]*module.LoadedComponent // Concrete components by name
	Metadata   Metadata
}

// Metadata contains release-level metadata.
type Metadata struct {
	Name      string
	Namespace string
	Version   string
	FQN       string
	Labels    map[string]string
	// Identity is the module identity UUID (from #Module.metadata.identity).
	Identity string
	// ReleaseIdentity is the release identity UUID.
	// Computed by the CUE overlay via uuid.SHA1(OPMNamespace, "fqn:name:namespace").
	ReleaseIdentity string
}

// TransformerMetadata is the release metadata projected into transformer context.
// This is the single place where the ReleaseIdentity → Identity field rename occurs.
type TransformerMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	FQN       string            `json:"fqn"`
	Version   string            `json:"version"`
	Identity  string            `json:"identity"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// ReleaseMetadataForTransformer projects release metadata into transformer context form.
// This is the single authoritative location for the ReleaseIdentity → Identity rename.
func (m Metadata) ReleaseMetadataForTransformer() TransformerMetadata {
	return TransformerMetadata{
		Name:      m.Name,
		Namespace: m.Namespace,
		FQN:       m.FQN,
		Version:   m.Version,
		Identity:  m.ReleaseIdentity, // rename: ReleaseIdentity → Identity
		Labels:    m.Labels,
	}
}

// ModuleReleaseMetadata contains information about the source module and the release being deployed.
// This metadata is used for labeling resources and verbose output.
type ModuleReleaseMetadata struct {
	// Name is the release name (resolved from RenderOptions.Name or module.metadata.name).
	Name string

	// ModuleName is the canonical module name from module.metadata.name.
	// Distinct from Name when --release-name overrides the default.
	ModuleName string

	// Namespace is the target namespace.
	Namespace string

	// Version is the module version (semver).
	Version string

	// Labels from the module definition.
	Labels map[string]string

	// Components lists the component names in the module.
	Components []string

	// Identity is the module identity UUID (from #Module.metadata.identity).
	Identity string

	// ReleaseIdentity is the release identity UUID.
	ReleaseIdentity string
}

// ToModuleReleaseMetadata projects a BuiltRelease into a ModuleReleaseMetadata value.
// moduleName is the canonical module name from module.metadata.name, which may
// differ from r.Metadata.Name when --release-name overrides the default.
func (r *BuiltRelease) ToModuleReleaseMetadata(moduleName string) ModuleReleaseMetadata {
	names := make([]string, 0, len(r.Components))
	for name := range r.Components {
		names = append(names, name)
	}
	return ModuleReleaseMetadata{
		Name:            r.Metadata.Name,
		ModuleName:      moduleName,
		Namespace:       r.Metadata.Namespace,
		Version:         r.Metadata.Version,
		Labels:          r.Metadata.Labels,
		Components:      names,
		Identity:        r.Metadata.Identity,
		ReleaseIdentity: r.Metadata.ReleaseIdentity,
	}
}

// ValidationError indicates the release failed validation.
// This typically happens when values are incomplete or non-concrete.
type ValidationError struct {
	// Message describes what validation failed.
	Message string

	// Cause is the underlying error.
	Cause error

	// Details contains the formatted CUE error output with all individual
	// errors, their CUE paths, and source positions.
	Details string
}

func (e *ValidationError) Error() string {
	if e.Cause != nil {
		return "release validation failed: " + e.Message + ": " + e.Cause.Error()
	}
	return "release validation failed: " + e.Message
}

func (e *ValidationError) Unwrap() error {
	return e.Cause
}
