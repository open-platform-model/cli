# Tasks: CLI Build Command

## Phase 1: Core Types & Interfaces

### 1.1 Shared Types (from render-pipeline-v1)

- [x] 1.1.1 Create `internal/build/types.go` with shared types from render-pipeline-v1
  - [x] RenderOptions struct
  - [x] RenderResult struct
  - [x] Resource struct
  - [x] ModuleMetadata struct
  - [x] MatchPlan and TransformerMatch structs

- [x] 1.1.2 Create `internal/build/errors.go` with error types
  - [x] RenderError interface
  - [x] UnmatchedComponentError
  - [x] UnhandledTraitError
  - [x] TransformError
  - [x] TransformerSummary
  - [x] ReleaseValidationError

### 1.2 Pipeline Interface

- [x] 1.2.1 Create `internal/build/pipeline.go` with Pipeline interface
- [x] 1.2.2 Implement `NewPipeline(cfg *config.OPMConfig) Pipeline`
- [x] 1.2.3 Implement `Render(ctx, opts) (*RenderResult, error)` method

## Phase 2: Module Loading & Release Building

### 2.1 ModuleLoader Implementation

- [x] 2.1.1 Create `internal/build/module.go`
- [x] 2.1.2 Implement `LoadedModule` and `LoadedComponent` types
- [x] 2.1.3 Implement module loading via cue/load
- [x] 2.1.4 Implement values.cue auto-discovery (required)
- [x] 2.1.5 Implement `--values` file unification
- [x] 2.1.6 Implement `--namespace` precedence over defaultNamespace
- [x] 2.1.7 Implement `--name` precedence over module name
- [x] 2.1.8 Extract metadata (name, namespace, version, labels)

### 2.2 ReleaseBuilder Implementation

- [x] 2.2.1 Create `internal/build/release_builder.go`
- [x] 2.2.2 Implement `ReleaseBuilder` struct with CUE context
- [x] 2.2.3 Implement `ReleaseOptions` struct (Name, Namespace)
- [x] 2.2.4 Implement `BuiltRelease` struct (Value, Components, Metadata)
- [x] 2.2.5 Implement `ReleaseMetadata` struct
- [x] 2.2.6 Implement `Build()` method:
  - [x] Extract values from module.values
  - [x] FillPath(#config, values) injection
  - [x] Extract components from #components
  - [x] Validate components are concrete
  - [x] Extract release metadata
- [x] 2.2.7 Implement `extractComponents()` helper
- [x] 2.2.8 Implement `extractComponent()` for single component
- [x] 2.2.9 Implement `extractMetadata()` helper
- [x] 2.2.10 Add `ReleaseValidationError` to errors.go
- [x] 2.2.11 Add unit tests for ReleaseBuilder

## Phase 3: Provider Loading

### 3.1 Provider Loader Implementation

- [x] 3.1.1 Create `internal/build/provider.go`
- [x] 3.1.2 Implement `LoadedProvider` and `LoadedTransformer` types
- [x] 3.1.3 Load provider from config.providers by name
- [x] 3.1.4 Index transformers by FQN for matching
- [x] 3.1.5 Extract transformer requirements (labels, resources, traits)

## Phase 4: Component Matching

### 4.1 Matcher Implementation

- [x] 4.1.1 Create `internal/build/matcher.go`
- [x] 4.1.2 Implement `MatchResult` and `MatchDetail` types
- [x] 4.1.3 Implement required labels matching
- [x] 4.1.4 Implement required resources matching
- [x] 4.1.5 Implement required traits matching
- [x] 4.1.6 Build MatchResult grouping components by transformer
- [x] 4.1.7 Identify unmatched components
- [x] 4.1.8 Track unhandled traits per match
- [x] 4.1.9 Implement `ToMatchPlan()` conversion

## Phase 5: Transformer Execution

### 5.1 Executor Implementation

- [x] 5.1.1 Create `internal/build/executor.go`
- [x] 5.1.2 Implement `Job` and `JobResult` types
- [x] 5.1.3 Implement worker pool with configurable size
- [x] 5.1.4 Implement `ExecuteWithTransformers` main entry point
- [x] 5.1.5 Implement `executeJob` for single transformation
- [x] 5.1.6 Use FillPath for #component injection
- [x] 5.1.7 Use FillPath for #context field injection

### 5.2 Context Construction

- [x] 5.2.1 Create `internal/build/context.go`
- [x] 5.2.2 Implement `NewTransformerContext(release *BuiltRelease, component *LoadedComponent)`
- [x] 5.2.3 Implement `TransformerContext` struct with proper JSON tags
- [x] 5.2.4 Implement `TransformerModuleMetadata` and `TransformerComponentMetadata`
- [x] 5.2.5 Implement `ToMap()` for CUE encoding

## Phase 6: Output Formatting

### 6.1 Manifest Output

- [x] 6.1.1 Create `internal/output/manifest.go`
- [x] 6.1.2 Implement YAML output format
- [x] 6.1.3 Implement JSON output format
- [x] 6.1.4 Implement deterministic ordering

### 6.2 Split Output

- [x] 6.2.1 Create `internal/output/split.go`
- [x] 6.2.2 Implement file naming pattern `<kind>-<name>.yaml`
- [x] 6.2.3 Implement directory creation
- [x] 6.2.4 Handle filename collisions

### 6.3 Verbose Output

- [x] 6.3.1 Create `internal/output/verbose.go`
- [x] 6.3.2 Implement human-readable matching decisions
- [x] 6.3.3 Implement JSON verbose output

## Phase 7: CLI Command

### 7.1 Command Implementation

- [x] 7.1.1 Create `internal/cmd/mod_build.go`
- [x] 7.1.2 Replace stub command with implementation
- [x] 7.1.3 Add `--values` / `-f` flag (repeatable)
- [x] 7.1.4 Add `--namespace` / `-n` flag
- [x] 7.1.5 Add `--name` flag
- [x] 7.1.6 Add `--provider` flag
- [x] 7.1.7 Add `--output` / `-o` flag (yaml, json)
- [x] 7.1.8 Add `--split` flag
- [x] 7.1.9 Add `--out-dir` flag
- [x] 7.1.10 Add `--strict` flag
- [x] 7.1.11 Add `--verbose` / `-v` flag
- [x] 7.1.12 Add `--verbose-json` flag

### 7.2 Error Handling

- [x] 7.2.1 Implement error aggregation display
- [x] 7.2.2 Implement exit codes (0, 1, 2)
- [x] 7.2.3 Implement actionable error messages

## Phase 8: Resource Ordering

- [x] 8.1 Ensure `pkg/weights/weights.go` exists with weight definitions
- [x] 8.2 Implement resource sorting by weight in pipeline
- [x] 8.3 Verify ordering in output

## Phase 9: Testing

### 9.1 Unit Tests

- [x] 9.1.1 Add unit tests for ModuleLoader
- [x] 9.1.2 Add unit tests for ReleaseBuilder
- [x] 9.1.3 Add unit tests for ProviderLoader
- [x] 9.1.4 Add unit tests for Matcher
- [x] 9.1.5 Add unit tests for Executor
- [x] 9.1.6 Add unit tests for TransformerContext
- [x] 9.1.7 Add unit tests for output formatting

### 9.2 Integration Tests

- [x] 9.2.1 Create test fixtures with sample modules
- [x] 9.2.2 Add integration tests for full render pipeline
- [x] 9.2.3 Add integration tests for CLI command
- [x] 9.2.4 Test deterministic output
- [x] 9.2.5 Test #config pattern with blog module

## Phase 10: Validation Gates

- [x] 10.1 Run `task fmt` - verify Go files formatted
- [x] 10.2 Run `task lint` - verify golangci-lint passes
- [x] 10.3 Run `task test` - verify all tests pass
- [x] 10.4 Manual testing with test module (testing/blog)
