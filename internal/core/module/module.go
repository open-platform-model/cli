// Package module defines the Module and ModuleMetadata types, mirroring the
// #Module definition in the CUE catalog (v1alpha1). A Module represents the
// parsed module definition before it is built into a release.
//
// In v1alpha1, concrete author defaults live in values.cue alongside the
// module definition. The Module struct does not carry values — all values
// resolution happens in the builder phase.
package module

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core/component"
)

// Module represents the #Module type before it is built.
type Module struct {
	// Metadata is the module metadata extracted from the module definition.
	Metadata *ModuleMetadata `json:"metadata"`

	// Must preserve the original order of the #Module.#components map for deterministic output and to support index-based inventory tracking.
	Components map[string]*component.Component `json:"#components,omitempty"`

	// Config is the #config schema from the module definition (#Module.#config).
	// It defines the constraints and defaults for module values.
	Config cue.Value `json:"#config,omitempty"`

	ModulePath string `json:"modulePath,omitempty"`

	// pkgName is the CUE package name from the module, set by module.Load().
	// Accessed via PkgName().
	pkgName string

	// Raw is the fully evaluated CUE value for the module, set by module.Load().
	Raw cue.Value
}

// PkgName returns the CUE package name of the module.
// Set by module.Load(); empty if the module was not constructed via Load().
func (m *Module) PkgName() string {
	return m.pkgName
}

// SetPkgName sets the CUE package name. Called by module.Load().
// This is intentionally package-scoped via a friend-access pattern:
// internal packages may call this; external packages should use module.Load().
func (m *Module) SetPkgName(name string) {
	m.pkgName = name
}

// ResolvePath resolves ModulePath to an absolute path, verifies the directory
// exists, and verifies a cue.mod/ subdirectory is present.
// On success, ModulePath is updated in-place to the resolved absolute path.
func (m *Module) ResolvePath() error {
	absPath, err := filepath.Abs(m.ModulePath)
	if err != nil {
		return fmt.Errorf("resolving module path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("module directory not found: %s", absPath)
	}

	cueModPath := filepath.Join(absPath, "cue.mod")
	if _, err := os.Stat(cueModPath); os.IsNotExist(err) {
		return fmt.Errorf("not a CUE module: missing cue.mod/ directory in %s", absPath)
	}

	m.ModulePath = absPath
	return nil
}

// Validate checks that the Module has the required fields populated after
// module.Load(). This is a structural check only — it does not perform CUE
// concreteness validation.
func (m *Module) Validate() error {
	if m.ModulePath == "" {
		return fmt.Errorf("module path is empty")
	}
	if m.Metadata == nil {
		return fmt.Errorf("module metadata is nil")
	}
	if m.Metadata.Name == "" {
		return fmt.Errorf("module metadata.name is empty or not concrete — ensure metadata.name is defined in the module definition")
	}
	if m.Metadata.FQN == "" {
		return fmt.Errorf("module metadata.fqn is empty — ensure the module was fully loaded via module.Load()")
	}
	if !m.Raw.Exists() {
		return fmt.Errorf("module CUE value is not set — ensure the module was fully loaded via module.Load()")
	}
	return nil
}

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the release it is deployed as.
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name (kebab-case).
	// Distinct from the release name when --name overrides the default.
	Name string `json:"name"`

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

	// Components lists the component names rendered in this release.
	Components []string `json:"components,omitempty"`
}
