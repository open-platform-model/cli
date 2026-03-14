## REMOVED Requirements

### Requirement: SynthesizeModuleRelease builds a ModuleRelease without a release.cue file

**Reason**: The `SynthesizeModuleRelease` function (later renamed to `SynthesizeModule` and moved to `pkg/render`) is removed. Its only caller was the `FromModule` workflow function, which is also removed. The function duplicated validation and finalization steps that `ProcessModuleRelease` already performs in the release-file path.

**Migration**: Use a `release.cue` file that imports the module. The `opm release build -r <release-file>` path uses `LoadReleasePackage` + `LoadModuleReleaseFromValue` + `ProcessModuleRelease` to handle all rendering.
