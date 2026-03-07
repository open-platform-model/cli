// Package module defines the Module and ModuleMetadata types, mirroring the
// #Module definition in the CUE catalog (v1alpha1). A Module represents the
// parsed module definition before it is built into a release.
//
// In v1alpha1, concrete author defaults live in values.cue alongside the
// module definition. The Module struct does not carry values — all values
// resolution happens in the builder phase.
package module

import (
	"cuelang.org/go/cue"
)

// Module represents the #Module type before it is built.
type Module struct {
	// Metadata is the module metadata extracted from the module definition.
	Metadata *ModuleMetadata `json:"metadata"`

	// Config is the #config schema from the module definition (#Module.#config).
	// It defines the constraints and defaults for module values.
	Config cue.Value `json:"#config,omitempty"`

	// Raw is the fully evaluated CUE value for the module, set by module.Load().
	Raw cue.Value

	// Path to the module
	ModulePath string

	// pkgName is the CUE package name from the module, set by module.Load().
	// Accessed via PkgName().
	pkgName string
}

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the release it is deployed as.
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name (kebab-case).
	// Distinct from the release name when --name overrides the default.
	Name string `json:"name"`

	// Description is a brief description of the module.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path from metadata.modulePath.
	// This is the registry path (e.g., "opmodel.dev/modules"), NOT a filesystem path.
	// Distinct from Module.ModulePath which is the local filesystem directory.
	ModulePath string `json:"modulePath"`

	// DefaultNamespace is the default namespace from the module definition.
	DefaultNamespace string `json:"defaultNamespace"`

	// FQN is the fully qualified module name (v1alpha1: modulePath/name:version).
	// Example: "opmodel.dev/modules/my-app:1.0.0"
	// Extracted directly from CUE evaluation of metadata.fqn.
	FQN string `json:"fqn"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition (pre-build, author-declared).
	// Populated at LoadModule time from metadata.labels in the CUE value.
	// Distinct from ReleaseMetadata.Labels which is the fully merged set computed by CUE at build time.
	// Not injected into TransformerContext — transformers receive the merged ReleaseMetadata.Labels.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	// TODO: not yet implemented. Populate from CUE metadata.annotations once a CUE path is defined.
	Annotations map[string]string `json:"annotations,omitempty"`
}
