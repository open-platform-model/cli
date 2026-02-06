# Tasks: Pluggable CUE-based Transformer Matching

## 1. Interface Definition

- [ ] 1.1 Create `internal/build/interface.go` with `ComponentMatcher` interface
- [ ] 1.2 Define interface method: `Match(components []*LoadedComponent, transformers []*LoadedTransformer) *MatchResult`
- [ ] 1.3 Add interface documentation explaining pluggability purpose

## 2. Pipeline Refactoring

- [ ] 2.1 Change `pipeline.matcher` field type from `*Matcher` to `ComponentMatcher`
- [ ] 2.2 Create `PipelineOption` type for functional options
- [ ] 2.3 Implement `WithMatcher(m ComponentMatcher) PipelineOption`
- [ ] 2.4 Update `NewPipeline()` signature to accept `...PipelineOption`
- [ ] 2.5 Apply options in `NewPipeline()`, defaulting to `NewDefaultMatcher()`
- [ ] 2.6 Update all existing `NewPipeline()` call sites (no changes needed if backwards compatible)

## 3. Provider #Matches Extraction

- [ ] 3.1 Add `MatchesPredicate cue.Value` field to `LoadedTransformer` struct
- [ ] 3.2 Update `extractTransformer()` to extract `#Matches` field from transformer CUE value
- [ ] 3.3 Add `HasMatchesPredicate() bool` method to `LoadedTransformer`

## 4. Default Matcher Implementation

- [ ] 4.1 Rename `Matcher` struct to `DefaultMatcher` in `matcher.go`
- [ ] 4.2 Rename `NewMatcher()` to `NewDefaultMatcher()`
- [ ] 4.3 Ensure `DefaultMatcher` implements `ComponentMatcher` interface
- [ ] 4.4 Refactor `evaluateMatch()` to check `tf.HasMatchesPredicate()` first
- [ ] 4.5 Add `evaluateCUEMatch()` method for CUE predicate evaluation
- [ ] 4.6 Add `evaluateGoMatch()` method for legacy label/resource/trait matching
- [ ] 4.7 Update `evaluateMatch()` to call appropriate method based on transformer

## 5. CUE Predicate Evaluation

- [ ] 5.1 Create `buildComponentContext()` to construct CUE evaluation context
- [ ] 5.2 Include `component.name` in context
- [ ] 5.3 Include `component.labels` map in context
- [ ] 5.4 Include `component.#resources` map in context
- [ ] 5.5 Include `component.#traits` map in context
- [ ] 5.6 Implement `evaluateCUEMatch()` to unify transformer `#Matches` with context
- [ ] 5.7 Handle CUE evaluation errors (return false, record in MatchDetail)
- [ ] 5.8 Extract boolean result from unified CUE value

## 6. Match Detail Updates

- [ ] 6.1 Add `EvaluationError string` field to `MatchDetail` for CUE errors
- [ ] 6.2 Update `buildReason()` to handle CUE match reasons
- [ ] 6.3 Add reason text: "Matched: #Matches predicate evaluated true"
- [ ] 6.4 Add reason text: "Not matched: #Matches predicate evaluated false"
- [ ] 6.5 Add reason text: "Not matched: CUE error: <error details>"

## 7. Unit Tests

- [ ] 7.1 Create `internal/build/interface_test.go` with interface compliance tests
- [ ] 7.2 Add test: `DefaultMatcher` implements `ComponentMatcher`
- [ ] 7.3 Add test: `WithMatcher` option injects custom matcher
- [ ] 7.4 Add test: Pipeline uses injected matcher
- [ ] 7.5 Add test: Transformer with `#Matches: true` matches all components
- [ ] 7.6 Add test: Transformer with `#Matches: false` matches no components
- [ ] 7.7 Add test: Transformer with label-based `#Matches` predicate
- [ ] 7.8 Add test: Transformer with resource-based `#Matches` predicate
- [ ] 7.9 Add test: Transformer without `#Matches` uses Go fallback
- [ ] 7.10 Add test: CUE evaluation error results in non-match
- [ ] 7.11 Add test: Mixed transformers (some CUE, some Go) work correctly

## 8. Integration Tests

- [ ] 8.1 Create test fixture: module with components having various labels
- [ ] 8.2 Create test fixture: provider with transformers using `#Matches`
- [ ] 8.3 Add integration test: End-to-end render with CUE matching
- [ ] 8.4 Add integration test: Backwards compatibility (no `#Matches` in provider)
- [ ] 8.5 Add integration test: Verbose output shows CUE match reasons

## 9. Documentation

- [ ] 9.1 Update inline code comments for new types and methods
- [ ] 9.2 Add example `#Matches` predicates in code comments

## 10. Validation Gates

- [ ] 10.1 Run `task fmt` - verify Go files formatted
- [ ] 10.2 Run `task lint` - verify golangci-lint passes
- [ ] 10.3 Run `task test` - verify all tests pass
- [ ] 10.4 Manual testing with sample module and provider
