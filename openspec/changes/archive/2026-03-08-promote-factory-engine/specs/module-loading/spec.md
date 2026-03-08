## REMOVED Requirements

### Requirement: LoadModule function
**Reason**: `loader.LoadModule(cueCtx, modulePath, registry)` which loaded a module directory, filtered values files, and built a `*Module` is replaced by release-centric loading. The new loader operates on release packages (`release.cue + values.cue`) not module directories. Module metadata is extracted from the `#module` hidden field within the release.
**Migration**: Replace `loader.LoadModule()` with `loader.LoadReleasePackage()` + `loader.LoadModuleReleaseFromValue()`, which extracts module info from the release's `#module` field.
