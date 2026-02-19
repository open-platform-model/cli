// Package module provides module loading and AST inspection utilities.
package module

// Inspection contains metadata extracted from a module directory
// via AST inspection without CUE evaluation.
type Inspection struct {
	Name             string // metadata.name (empty if not a string literal)
	DefaultNamespace string // metadata.defaultNamespace (empty if not a string literal)
	PkgName          string // CUE package name from inst.PkgName
}

// MetadataPreview contains lightweight module metadata extracted
// before the full overlay build. Used for name/namespace resolution.
type MetadataPreview struct {
	Name             string
	DefaultNamespace string
}

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the release it is deployed as.
//
//nolint:revive // ModuleMetadata is intentional: the type is re-exported as build.ModuleMetadata without stutter.
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name.
	// Distinct from the release name when --name overrides the default.
	Name string `json:"name"`

	// DefaultNamespace is the default namespace from the module definition.
	// TODO: not yet consumed after extraction. Currently the pipeline reads namespace from
	// MetadataPreview.DefaultNamespace (internal/build/pipeline.go) instead of this field.
	// Either remove this duplicate or wire it up as the canonical source and drop MetadataPreview.DefaultNamespace.
	DefaultNamespace string `json:"defaultNamespace"`

	// FQN is the fully qualified module name.
	FQN string `json:"fqn"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition.
	// TODO: not yet consumed after extraction. TransformerContext.ToMap (internal/build/transform/context.go)
	// currently injects ReleaseMetadata.Labels into CUE instead of these module-level labels.
	// Decide whether module-level labels should be passed separately to transformers and implement accordingly.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	// TODO: not yet implemented. Neither set in extractModuleMetadata (internal/build/release/metadata.go)
	// nor consumed anywhere. Populate from CUE metadata.annotations, then wire into
	// TransformerContext.ToMap (internal/build/transform/context.go) alongside module Labels.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Components lists the component names in the module.
	// TODO: not yet consumed after extraction. Verbose output reads ReleaseMetadata.Components
	// (internal/cmdutil/output.go) instead of this field. Decide whether module-level components
	// should be surfaced separately and implement accordingly.
	Components []string `json:"components,omitempty"`
}
