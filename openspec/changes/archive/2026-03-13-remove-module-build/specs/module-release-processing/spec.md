## REMOVED Requirements

### Requirement: Public module release synthesis

**Reason**: The `SynthesizeModule` function is removed because its only caller (`FromModule` in the render workflow) is removed. The function mixed skeleton construction with processing work that `ProcessModuleRelease` repeated, creating redundant validation and component finalization. If module-source synthesis is needed in the future, it should be redesigned to produce output compatible with the `#ModuleRelease` CUE schema.

**Migration**: Use a `release.cue` file that imports the module. The `opm release build -r <release-file>` path handles all rendering via `ProcessModuleRelease`.
