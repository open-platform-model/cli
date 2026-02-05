# Proposal: Pluggable CUE-based Transformer Matching

## Why

The current transformer-component matching system is implemented entirely in Go (`internal/build/matcher.go`) with hardcoded logic. Users cannot customize matching behavior without modifying CLI source code and recompiling. A CUE-based matching system would allow providers to define custom matching predicates declaratively, enabling extensibility without Go development.

## What Changes

- **New `ComponentMatcher` interface** in `internal/build/` for pluggable matching implementations
- **CUE-based `#Matches` predicate** evaluated at match time, allowing transformers to define arbitrary matching logic
- **Pipeline refactored** to accept matcher via interface, enabling injection and testing
- **Default CUE matcher** that evaluates transformer-defined `#Matches` predicates
- **Fallback Go matcher** preserved for backwards compatibility when no `#Matches` is defined

**SemVer**: MINOR - Adds new capability with backwards-compatible defaults.

## Capabilities

### New Capabilities

- `cue-matcher`: CUE-based component-transformer matching system with pluggable architecture. Covers the `ComponentMatcher` interface, CUE `#Matches` predicate evaluation, and the matching algorithm.

### Modified Capabilities

_None_ - The `render-pipeline` spec defines the Pipeline interface but not matching internals. Matching is an implementation detail that doesn't change the Pipeline contract.

## Impact

**Code Changes**:

- `internal/build/matcher.go` → Extract interface, refactor to `ComponentMatcher` implementations
- `internal/build/pipeline.go` → Use `ComponentMatcher` interface, add injection mechanism
- `internal/build/cue_matcher.go` → New CUE-based matcher implementation

**CUE Schema Changes**:

- Transformer definition schema needs `#Matches` field (optional, for custom predicates)
- `#Matches` receives component context and returns boolean

**Dependencies**:

- No new external dependencies
- Relies on existing CUE evaluation infrastructure

**Testing**:

- Unit tests for interface contract
- Integration tests for CUE predicate evaluation
- Backwards compatibility tests (no `#Matches` → use Go defaults)

**Justification** (Principle VII - Simplicity):
This complexity is justified because:

1. Users have requested custom matching logic (provider-specific requirements)
2. CUE predicates align with the project's CUE-first philosophy
3. Interface extraction improves testability regardless of CUE feature
4. Backwards compatible - no breaking changes to existing behavior
