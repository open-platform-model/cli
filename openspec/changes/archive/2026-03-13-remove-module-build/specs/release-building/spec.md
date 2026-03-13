## REMOVED Requirements

### Requirement: SynthesizeModuleRelease builds a ModuleRelease without a release.cue file

**Reason**: The `SynthesizeModule` function is removed because its only caller (`FromModule`) is removed. The function mixed skeleton construction with processing work that `ProcessModuleRelease` repeated, creating redundant validation and component finalization. If module-source synthesis is needed in the future, it should be redesigned to produce output compatible with the `#ModuleRelease` CUE schema.

**Migration**: Use a `release.cue` file that imports the module. The `opm release build -r <release-file>` path handles all rendering.

### Requirement: SynthesizeModuleRelease builds a ModuleRelease without a release.cue file

**Reason**: The `SynthesizeModuleRelease` function (now `SynthesizeModule` in `pkg/render`) is removed. Its only caller was the `FromModule` workflow function, which is also removed. The function's 8-step pipeline (Module Gate, FillPath, extract components, wrap, finalize, decode metadata, construct release metadata, return ModuleRelease) duplicated validation and finalization work already performed by `ProcessModuleRelease` in the release-file path.

**Migration**: Use a `release.cue` file that imports the module. The `opm release build -r <release-file>` path handles all rendering.

### Requirement: RenderRelease supports synthesis mode when release.cue is absent

**Reason**: The synthesis branch in `cmdutil.RenderRelease` that detected the absence of `release.cue` and called `SynthesizeModuleRelease` is removed along with the `FromModule` workflow entrypoint. The `hasReleaseFile` detection helper, `DebugValues` option, and the synthesis-specific release name resolution chain (`opts.ReleaseName` -> `module.metadata.name` -> `filepath.Base(modulePath)`) are all removed.

**Migration**: Always provide a `release.cue` file. Use `opm release build -r <release-file>` for rendering.
