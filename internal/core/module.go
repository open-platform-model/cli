package core

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
)

// Module represents the #Module type before it is built.
type Module struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	// Metadata is the module metadata extracted from the module definition.
	Metadata *ModuleMetadata `json:"metadata"`

	// Must preserve the original order of the #Module.#components map for deterministic output and to support index-based inventory tracking.
	Components map[string]*Component `json:"#components,omitempty"`

	// The schema (#Module.#config)
	Config cue.Value `json:"#config,omitempty"`

	// The default values (#Module.values)
	Values cue.Value `json:"values,omitempty"`

	ModulePath string `json:"modulePath,omitempty"`

	// pkgName is the CUE package name from the module, set by module.Load().
	// Accessed via PkgName().
	pkgName string

	// value is the fully evaluated CUE value for the module, set by module.Load()
	// via setCUEValue(). Access via CUEValue().
	value cue.Value
}

// CUEValue returns the fully evaluated CUE value for the module.
// Returns the zero value if the module has not been fully loaded via module.Load().
func (m *Module) CUEValue() cue.Value {
	return m.value
}

// SetCUEValue stores the fully evaluated CUE value. Called by module.Load().
// This is intentionally a package-accessible setter following the same pattern
// as SetPkgName — internal packages call this; external callers use module.Load().
func (m *Module) SetCUEValue(v cue.Value) {
	m.value = v
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
	if !m.CUEValue().Exists() {
		return fmt.Errorf("module CUE value is not set — ensure the module was fully loaded via module.Load()")
	}
	return nil
}

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the release it is deployed as.
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name.
	// Distinct from the release name when --name overrides the default.
	Name string `json:"name"`

	// DefaultNamespace is the default namespace from the module definition.
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

	// Components lists the component names rendered in this release.
	Components []string `json:"components,omitempty"`
}
