## MODIFIED Requirements

### Requirement: Public release processing orchestrates the full render pipeline
The `pkg/render` package SHALL export `ProcessModuleRelease` and `ProcessBundleRelease` functions (previously in `internal/releaseprocess`) that orchestrate config validation, CUE finalization, matching, and engine invocation. These are the top-level entry points for rendering.

#### Scenario: Process a module release end-to-end
- **WHEN** `render.ProcessModuleRelease(ctx, release, values, provider)` is called
- **THEN** it SHALL validate config, finalize CUE values, compute a match plan, invoke the renderer, and return a `*render.ModuleResult`

#### Scenario: Process a bundle release end-to-end
- **WHEN** `render.ProcessBundleRelease(ctx, release, values, provider)` is called
- **THEN** it SHALL process each child module release and return a `*render.BundleResult`

### Requirement: Public module release synthesis
The `pkg/render` package SHALL export `SynthesizeModule` (previously `SynthesizeModuleRelease`) that constructs a `*render.ModuleRelease` from a raw module CUE value plus values.

#### Scenario: Synthesize a module release from raw values
- **WHEN** `render.SynthesizeModule(cueCtx, modVal, values, name, namespace)` is called
- **THEN** it SHALL return a `*render.ModuleRelease` with metadata, config, and data components populated

### Requirement: Public config validation
The `pkg/render` package SHALL export `ValidateConfig` that validates user-supplied values against a module's `#config` schema.

#### Scenario: Valid config values
- **WHEN** `render.ValidateConfig(schema, values, context, name)` is called with valid values
- **THEN** it SHALL return the merged CUE value and nil error

#### Scenario: Invalid config values
- **WHEN** `render.ValidateConfig(schema, values, context, name)` is called with values that violate the schema
- **THEN** it SHALL return a `*errors.ConfigError` with grouped diagnostics
