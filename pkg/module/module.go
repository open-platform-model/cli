// Package module defines the decoded metadata types for modules and module
// instances (ModuleMetadata, InstanceMetadata) plus the canonical module
// reference derivation. Loading, validation, and rendering live in the
// library kernel (enhancement 0006 C2); this package carries only the
// CLI-side identity shapes.
package module

// ModuleMetadata contains module-level identity and version information.
// This is the module's canonical metadata, distinct from the instance it is deployed as.
//
//nolint:revive // stutter intentional: module.ModuleMetadata reads clearly at call sites
type ModuleMetadata struct {
	// Name is the canonical module name from module.metadata.name
	// (snake_case, equal to the modulePath leaf under core v2).
	Name string `json:"name"`

	// Description is a brief description of the module.
	Description string `json:"description,omitempty"`

	// ModulePath is the CUE registry module path from metadata.modulePath.
	// Under core v2 this is the complete major-suffixed registry address
	// (e.g., "opmodel.dev/modules/podinfo@v0"), NOT a filesystem path.
	ModulePath string `json:"modulePath"`

	// FQN is the fully qualified module name; under core v2 it equals
	// ModulePath.
	FQN string `json:"fqn"`

	// Version is the module version (bare semver, per core #VersionType).
	Version string `json:"version"`

	// UUID is the module identity UUID (from #Module.metadata.identity).
	UUID string `json:"uuid"`

	// Labels from the module definition (pre-build, author-declared).
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations from the module definition.
	Annotations map[string]string `json:"annotations,omitempty"`
}

// CanonicalModuleRef returns the module's canonical registry import path and
// pinned version — the reference a consumer would import or a
// ModuleInstance.spec.module would pin. Under core v2 metadata.modulePath is
// already the complete major-suffixed registry address, so the path is read
// verbatim — no leaf or major tag is composed. The version is the declared
// semver with a "v" prefix (e.g. "v0.1.4"). It is never a filesystem path, so
// it is correct for local-directory and locally-replaced module resolution as
// well.
//
// The "v" prefix is load-bearing, not cosmetic. This pair is written verbatim
// to ModuleInstance.spec.module, which the operator reads and passes straight
// to the registry loader with no normalization of its own — and CUE rejects a
// bare "0.1.0" as a malformed module version. A bare version therefore produces
// a CR that the operator cannot resolve, and that `opm instance handoff` cannot
// verify, which defeats the point of both actors sharing one record. The
// operator's own CRD documents the v-prefixed form (`Example: "v0.2.1"`).
func (m ModuleMetadata) CanonicalModuleRef() (path, version string) {
	return m.ModulePath, ensureVPrefix(m.Version)
}

// ensureVPrefix normalizes a declared semver to the "v"-prefixed form the
// registry expects. Idempotent (either prefix case is recognized), and leaves
// an empty version empty so callers can still detect "no version declared".
func ensureVPrefix(version string) string {
	if version == "" || version[0] == 'v' || version[0] == 'V' {
		return version
	}
	return "v" + version
}
