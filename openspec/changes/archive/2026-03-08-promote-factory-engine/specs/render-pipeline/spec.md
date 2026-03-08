## REMOVED Requirements

### Requirement: Pipeline interface
**Reason**: Replaced by concrete `ModuleRenderer` and `BundleRenderer` structs in `pkg/engine/`. The interface provided no value with only one implementation.
**Migration**: Replace `pipeline.NewPipeline().Render()` calls with `engine.NewModuleRenderer().Render()` or `engine.NewBundleRenderer().Render()`.

### Requirement: RenderOptions struct
**Reason**: Loading is no longer driven by the pipeline — the loader handles release loading separately, and the engine receives a pre-loaded `*ModuleRelease`.
**Migration**: Callers load the release via `pkg/loader/` and pass it to the engine. CLI flags and values resolution stay in `internal/cmdutil/`.

### Requirement: Five-phase pipeline orchestration
**Reason**: The five phases (preparation, provider load, build, matching, generate) are replaced by two distinct steps: load (via `pkg/loader/`) and render (via `pkg/engine/`). Orchestration moves to `internal/cmdutil/render.go`.
**Migration**: `cmdutil.RenderRelease()` calls `loader.LoadReleasePackage()`, `loader.LoadModuleReleaseFromValue()`, then `engine.ModuleRenderer.Render()`.

### Requirement: RenderResult with Unstructured resources
**Reason**: `RenderResult.Resources` changes from `[]*core.Resource` wrapping `*unstructured.Unstructured` to `[]*core.Resource` wrapping `cue.Value`. The new `Resource` has conversion methods.
**Migration**: Call `resource.ToUnstructured()` at the cmdutil boundary before passing to K8s or inventory packages.
