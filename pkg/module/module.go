// Package module defines the decoded metadata types for modules and module
// instances (ModuleMetadata, InstanceMetadata) plus the canonical module
// reference derivation. Loading, validation, and rendering live in the
// library kernel (enhancement 0006 C2); this package carries only the
// CLI-side identity shapes.
package module

import (
	"fmt"
	"strings"
)

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the instance it is deployed as.
//
//nolint:revive // stutter intentional: module.ModuleMetadata reads clearly at call sites
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name (kebab-case).
	Name string `json:"name"`

	// Description is a brief description of the module.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path from metadata.modulePath.
	// This is the registry path (e.g., "opmodel.dev/modules"), NOT a filesystem path.
	ModulePath string `json:"modulePath"`

	// DefaultNamespace is the default namespace from the module definition.
	DefaultNamespace string `json:"defaultNamespace"`

	// FQN is the fully qualified module name (modulePath/name:version).
	// Example: "opmodel.dev/modules/my-app:1.0.0"
	FQN string `json:"fqn"`

	// Version is the module version (semver).
	Version string `json:"version"`

	// NameSnakeCase is the snake_case projection of Name (core
	// #Module.metadata.nameSnakeCase), used as the module's registry-path leaf.
	// Absent on modules built against a pre-nameSnakeCase core; derived from
	// Name as a fallback (see CanonicalModuleRef).
	NameSnakeCase string `json:"nameSnakeCase,omitempty"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition (pre-build, author-declared).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CanonicalModuleRef returns the module's canonical registry import path and
// declared version — the reference a consumer would import or a
// ModuleInstance.spec.module would pin. The path follows the convention
// `modulePath + "/" + nameSnakeCase + "@v" + MAJOR(version)` (e.g.
// "opmodel.dev/modules/cert_manager@v0"); the version is the declared semver
// verbatim (e.g. "0.1.0"). It is never a filesystem path, so it is correct for
// local-directory and locally-replaced module resolution as well.
func (m ModuleMetadata) CanonicalModuleRef() (path, version string) {
	leaf := m.NameSnakeCase
	if leaf == "" {
		leaf = strings.ReplaceAll(m.Name, "-", "_")
	}
	path = fmt.Sprintf("%s/%s@%s", m.ModulePath, leaf, majorVersionTag(m.Version))
	return path, m.Version
}

// majorVersionTag returns the "vN" major-version tag for a semver string
// ("0.1.0" -> "v0"). An unparseable version yields "v0".
func majorVersionTag(version string) string {
	major := "0"
	if idx := strings.IndexByte(version, '.'); idx > 0 {
		major = version[:idx]
	}
	return "v" + major
}
