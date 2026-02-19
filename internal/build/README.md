# internal/build

The `build` package implements the OPM render pipeline. It exposes a single public entry point — `NewPipeline` — and organizes implementation into focused subpackages.

## Package Structure

```text
internal/build/
├── pipeline.go          # Pipeline interface + orchestration (NewPipeline, Render)
├── types.go             # Public API types (Pipeline, RenderOptions, RenderResult, ModuleReleaseMetadata)
├── errors.go            # Public error types (UnmatchedComponentError, NamespaceRequiredError, ModuleValidationError)
├── component.go         # Shared type: LoadedComponent (re-exported from module package)
├── transformer_adapter.go  # Shared types re-exported from transform package
├── release_adapter.go   # Shared types re-exported from release package
├── module/              # Module path resolution and AST metadata extraction
├── release/             # Release building: CUE overlay, values, metadata, validation
└── transform/           # Transformer logic: provider loading, matching, execution
```

## Subpackages

### `module/`

Responsible for locating a module on disk and extracting metadata from its CUE AST without full CUE evaluation.

- `loader.go` — `ResolvePath`, `ExtractMetadata`
- `inspector.go` — `ExtractMetadataFromAST`, `ExtractFieldsFromMetadataStruct`
- `types.go` — `LoadedComponent`, `ModuleInspection`

### `release/`

Responsible for building a `#ModuleRelease` CUE value from a module directory and user-provided values.

- `builder.go` — `Builder`, `NewBuilder`, `Build`, `InspectModule`
- `overlay.go` — AST overlay generation for values injection
- `metadata.go` — Release metadata extraction from CUE values
- `validation.go` — Values validation against `#config` schema
- `validation_format.go` — CUE error formatting helpers
- `types.go` — `ReleaseOptions`, `BuiltRelease`, `ReleaseMetadata`

### `transform/`

Responsible for loading provider transformers, matching components to transformers, and executing transformations.

- `provider.go` — `ProviderLoader`, `NewProviderLoader`, `Load`
- `matcher.go` — `Matcher`, `NewMatcher`, `Match`
- `executor.go` — `Executor`, `NewExecutor`, `ExecuteWithTransformers`
- `context.go` — `TransformerContext`, `NewTransformerContext`
- `types.go` — Internal job/result types

## Public API

Commands interact with this package exclusively through:

```go
pipeline := build.NewPipeline(cfg)
result, err := pipeline.Render(ctx, build.RenderOptions{...})
```

All subpackage types used by commands are re-exported at the root level via adapter files, so commands import only `"github.com/opmodel/cli/internal/build"`.

## Dependency Flow

```text
pipeline.go
  → build/module    (path resolution, AST inspection)
  → build/release   (CUE overlay, release building)
  → build/transform (provider loading, matching, execution)
```

Subpackages do not import each other. Shared types (`LoadedComponent`, `LoadedTransformer`) are defined in the subpackage that owns them and re-exported at root level to avoid circular imports.

## Testing

Each subpackage has its own tests alongside the implementation:

```text
go test ./internal/build/...          # all build tests
go test ./internal/build/module       # module tests only
go test ./internal/build/release      # release tests only
go test ./internal/build/transform    # transform tests only
```

Shared test fixtures are in `testdata/` at the root build level and accessed via relative paths from test files.
