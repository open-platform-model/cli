package render

import "github.com/open-platform-model/library/opm/module"

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
// a CR that the operator cannot resolve, which defeats the point of both actors
// sharing one record. The operator's own CRD documents the v-prefixed form
// (`Example: "v0.2.1"`).
func CanonicalModuleRef(m module.ModuleMetadata) (path, version string) {
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
