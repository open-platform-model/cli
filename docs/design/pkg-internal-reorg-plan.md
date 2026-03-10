# pkg vs internal reorganization plan

## Goal

Make `pkg/` the intentional public API surface and move unstable implementation details, pipeline state, and CLI/runtime mechanics into `internal/`.

## Principles

- Public packages expose stable domain concepts, not phase-by-phase execution state.
- Public functions return `error`, with typed errors discoverable through `errors.As`.
- Public packages avoid process-global side effects.
- One concept has one owner; avoid alias re-exports across packages.
- Types that mainly shuttle data between internal phases belong in `internal/`.

## Target package map

### Keep public

- `pkg/module`
- `pkg/bundle`
- `pkg/provider`
- `pkg/loader`
- `pkg/errors` (reduced to library-facing errors)
- `pkg/core` for rendered resource primitives only

### Move internal

- `pkg/releaseprocess` -> `internal/releaseprocess`
- `pkg/match` -> `internal/match`
- `pkg/engine` -> `internal/engine`
- `pkg/modulerelease` -> `internal/runtime/modulerelease`
- `pkg/bundlerelease` -> `internal/runtime/bundlerelease`
- `pkg/core/weights.go` -> `internal/resourceorder/weights.go`
- CLI exit handling from `pkg/errors` -> `internal/exit`

## Package decisions

### `pkg/module`

- Keep public.
- Keep `ModuleMetadata` public.
- Revisit `Module.ModulePath` later; it is loader context rather than core domain state.
- Keep `Config` and `Raw` public for now because the current embedding surface is CUE-native.

### `pkg/bundle`

- Keep public.
- Keep `BundleMetadata` public.
- Keep `Data cue.Value` public for now, documented as low-level access.

### `pkg/provider`

- Keep public.
- Keep `ProviderMetadata` public.
- Keep `Data cue.Value` public for now, documented as low-level access.

### `pkg/core`

- Keep `Resource` and conversion helpers public.
- Move apply-order weights to `internal/resourceorder` because they are operational policy, not domain primitives.
- Keep label constants public for now because inventory and render output both rely on them and the labels can be useful to embedders.

### `pkg/loader`

- Keep public.
- Keep `LoadProvider`, `LoadReleaseFile`, `LoadModulePackage`, `LoadValuesFile`, and `DetectReleaseKind`.
- Introduce `LoadOptions` for loader configuration growth.
- Future cleanup: replace `LoadReleaseFile` registry environment mutation with an options-based, side-effect-free implementation.

### `pkg/errors`

- Keep library-facing validation and domain errors public.
- Keep `ConfigError`, `GroupedError`, `ErrorLocation`, `FieldError`, `TransformError`, `ValidationError`, `DetailError`, sentinel errors, and grouping helpers public.
- Move CLI exit codes and `ExitError` to `internal/exit`.

### `internal/runtime/modulerelease`

- Own mutable module release processing state.
- Keep `ModuleRelease`, `ReleaseMetadata`, and helper methods internal.

### `internal/runtime/bundlerelease`

- Own mutable bundle release processing state.
- Keep `BundleRelease` and `BundleReleaseMetadata` internal.

### `internal/match`

- Own the component-transformer matching algorithm and match-plan details.
- Do not expose match-plan internals as part of the supported public surface.

### `internal/engine`

- Own renderer execution and bundle/module rendering internals.
- Remove public alias re-exports of match-plan types.
- Keep display-oriented helpers such as `ComponentSummary` internal.

### `internal/releaseprocess`

- Own synthesis, validation, finalization, and orchestration helpers.
- Keep release processing internal.
- Future cleanup: return `error` instead of concrete `*errors.ConfigError` from `ValidateConfig`.

### `internal/exit`

- Own exit codes and `ExitError` used by the CLI command path.
- Keep CLI-specific exit behavior out of `pkg/errors`.

### `internal/resourceorder`

- Own resource ordering weights used for apply, delete, digest, and manifest sorting.

## Execution sequence

1. Write the plan and checklist documents.
2. Move clearly internal packages into `internal/` and update imports.
3. Split CLI exit handling out of `pkg/errors` into `internal/exit`.
4. Move Kubernetes apply-order weights out of `pkg/core`.
5. Run formatting and targeted tests.
6. Update the checklist with execution status.

## Success criteria

- `pkg/` contains only packages the project is comfortable versioning as public APIs.
- No exported public function is knowingly unimplemented.
- No public package exists solely to pass data between internal phases.
- No public package owns CLI-only exit semantics.
- Ownership is clear and there are no alias re-exports of another package's main types.
