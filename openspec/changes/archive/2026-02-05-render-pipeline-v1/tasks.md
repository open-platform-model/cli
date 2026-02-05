## 1. Core Types Setup

- [x] 1.1 Create `internal/build/types.go` with Pipeline interface, RenderOptions, and RenderResult types
- [x] 1.2 Add Validate() method to RenderOptions with ModulePath required check
- [x] 1.3 Add helper methods to RenderResult (HasErrors, HasWarnings, ResourceCount)

## 2. Resource Type

- [x] 2.1 Define Resource struct with Object (*unstructured.Unstructured), Component, Transformer fields
- [x] 2.2 Add accessor methods: GVK(), Kind(), Name(), Namespace(), Labels()

## 3. Supporting Types

- [x] 3.1 Define ModuleMetadata struct with Name, Namespace, Version, Labels, Components
- [x] 3.2 Define MatchPlan struct with Matches map and Unmatched slice
- [x] 3.3 Define TransformerMatch struct with TransformerFQN and Reason

## 4. Error Types

- [x] 4.1 Create `internal/build/errors.go` with RenderError interface (error + Component() string)
- [x] 4.2 Implement UnmatchedComponentError with ComponentName and Available fields
- [x] 4.3 Implement UnhandledTraitError with ComponentName, TraitFQN, Strict fields
- [x] 4.4 Implement TransformError with ComponentName, TransformerFQN, Cause fields
- [x] 4.5 Define TransformerSummary struct for error guidance

## 5. Unit Tests

- [x] 5.1 Create `internal/build/types_test.go` with table-driven tests for RenderOptions.Validate()
- [x] 5.2 Add tests for RenderResult helper methods (HasErrors, HasWarnings, ResourceCount)
- [x] 5.3 Add tests for Resource accessor methods (GVK, Kind, Name, Namespace, Labels)
- [x] 5.4 Create `internal/build/errors_test.go` with tests for error message formatting
- [x] 5.5 Add tests verifying error types implement RenderError interface

## 6. Validation Gates

- [x] 6.1 Run `task fmt` and fix any formatting issues
- [x] 6.2 Run `task lint` and resolve any linter warnings
- [x] 6.3 Run `task test` and ensure all tests pass
