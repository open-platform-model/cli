## REMOVED Requirements

### Requirement: Module type location

**Reason**: The CLI no longer declares module or instance metadata types; the library's `opm/schema` types, carried by every kernel-acquired artifact, are the single declaration, and `pkg/module` is deleted.

**Migration**: Import `github.com/open-platform-model/library/opm/module` (or `opm/schema`) for `ModuleMetadata` and `InstanceMetadata`. The canonical `spec.module` reference derivation lives in the CLI's render workflow as `CanonicalModuleRef(m module.ModuleMetadata)`.
