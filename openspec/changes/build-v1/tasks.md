## 1. Infrastructure Setup

- [ ] 1.1 Create `cli/internal/render/` package directory
- [ ] 1.2 Create `cli/internal/render/types.go` with core data structures (Pipeline, Provider, Transformer, Component, Match types)
- [ ] 1.3 Create `cli/internal/render/errors.go` with error types (UnmatchedComponentError, MultipleMatchError, UnhandledTraitError, ValuesConflictError)
- [ ] 1.4 Create `cli/internal/render/context.go` with TransformerContext construction

## 2. Values Resolution

- [ ] 2.1 Create `cli/internal/render/values.go` with values loading logic
- [ ] 2.2 Implement `values.cue` auto-discovery at module root (FR-041)
- [ ] 2.3 Implement `--values` flag handling for additional values files
- [ ] 2.4 Implement CUE unification for multiple values files (FR-042)
- [ ] 2.5 Implement `--namespace` precedence over `#Module.metadata.defaultNamespace` (FR-043)
- [ ] 2.6 Implement `--name` precedence over `#Module.metadata.name` (FR-044)

## 3. Module Loading

- [ ] 3.1 Create `cli/internal/render/loader.go` with module loading logic
- [ ] 3.2 Implement `#Module` loading via cue/load
- [ ] 3.3 Implement `#ModuleRelease` construction from local module + values (FR-040)
- [ ] 3.4 Extract and validate release metadata
- [ ] 3.5 Build base TransformerContext with OPM tracking labels (FR-016)

## 4. Provider Loading

- [ ] 4.1 Create `cli/internal/render/provider.go` with provider loading logic
- [ ] 4.2 Implement provider determination from `--provider` flag or default
- [ ] 4.3 Access provider definition from loaded config
- [ ] 4.4 Index transformers for matching

## 5. Component Matching

- [ ] 5.1 Create `cli/internal/render/matcher.go` with matching logic
- [ ] 5.2 Implement `#Matches` predicate evaluation in CUE
- [ ] 5.3 Implement required labels matching
- [ ] 5.4 Implement required resources matching
- [ ] 5.5 Implement required traits matching
- [ ] 5.6 Compute `matchedTransformers` map grouping components by transformer
- [ ] 5.7 Handle multiple transformers matching single component

## 6. Parallel Transformer Execution

- [ ] 6.1 Create `cli/internal/render/worker.go` with worker pool
- [ ] 6.2 Implement Worker struct with isolated cue.Context per worker (FR-015)
- [ ] 6.3 Implement Job and Result types for worker communication
- [ ] 6.4 Implement TransformerContext injection into each execution
- [ ] 6.5 Execute `#transform` unification for each component
- [ ] 6.6 Decode transformer output to unstructured resources

## 7. Error Handling

- [ ] 7.1 Implement fail-on-end pattern - aggregate all errors (FR-024)
- [ ] 7.2 Implement unmatched component error with available transformers list (FR-019)
- [ ] 7.3 Implement unhandled trait error in `--strict` mode (FR-020)
- [ ] 7.4 Implement unhandled trait warning in normal mode (FR-021)
- [ ] 7.5 Implement multiple exact match error
- [ ] 7.6 Wrap CUE validation errors for values conflicts

## 8. Output Formatting

- [ ] 8.1 Create `cli/internal/render/output.go` with output formatting
- [ ] 8.2 Implement YAML output format (FR-017)
- [ ] 8.3 Implement JSON output format (FR-017)
- [ ] 8.4 Implement `--split` mode with separate files per resource (FR-018)
- [ ] 8.5 Implement file naming pattern `<lowercase-kind>-<resource-name>.yaml` (FR-026)
- [ ] 8.6 Implement deterministic output ordering (FR-023)
- [ ] 8.7 Implement sensitive data redaction in verbose logging (FR-027)

## 9. Pipeline Orchestration

- [ ] 9.1 Create `cli/internal/render/pipeline.go` with Pipeline struct
- [ ] 9.2 Implement Phase 1: Module Loading & Validation
- [ ] 9.3 Implement Phase 2: Provider Loading
- [ ] 9.4 Implement Phase 3: Component Matching
- [ ] 9.5 Implement Phase 4: Parallel Transformer Execution
- [ ] 9.6 Implement Phase 5: Aggregation & Output

## 10. CLI Integration

- [ ] 10.1 Create `cli/internal/cmd/mod/build.go` with build command
- [ ] 10.2 Add `--values` flag for additional values files
- [ ] 10.3 Add `--namespace` flag for target namespace
- [ ] 10.4 Add `--name` flag for release name
- [ ] 10.5 Add `--provider` flag for provider selection
- [ ] 10.6 Add `-o` / `--output` flag for format (yaml/json)
- [ ] 10.7 Add `--split` flag for separate files
- [ ] 10.8 Add `--out-dir` flag for split output directory
- [ ] 10.9 Add `--strict` flag for strict trait handling
- [ ] 10.10 Add `--verbose` flag with human/json modes (FR-025)
- [ ] 10.11 Register build command in mod command group

## 11. Verbose Output

- [ ] 11.1 Implement human-readable verbose output showing transformer matching decisions
- [ ] 11.2 Implement structured JSON verbose output (`--verbose=json`)
- [ ] 11.3 Show which transformers matched and why
- [ ] 11.4 Explain why transformers didn't match (missing labels/resources/traits)

## 12. Tests

- [ ] 12.1 Add unit tests for values resolution
- [ ] 12.2 Add unit tests for module loading
- [ ] 12.3 Add unit tests for component matching logic
- [ ] 12.4 Add unit tests for transformer execution
- [ ] 12.5 Add unit tests for output formatting
- [ ] 12.6 Add integration tests for full render pipeline
- [ ] 12.7 Add test fixtures with sample modules

## 13. Validation Gates

- [ ] 13.1 Run `cue fmt ./...` - verify all CUE files formatted
- [ ] 13.2 Run `cue vet ./...` - verify all CUE files validate
- [ ] 13.3 Run `task test` - verify all Go tests pass
