# Design: Pluggable CUE-based Transformer Matching

## Context

The current matching system in `internal/build/matcher.go` uses a concrete `Matcher` struct with hardcoded Go logic. The pipeline instantiates it directly via `NewMatcher()`, making it impossible to:

- Inject custom matchers for testing
- Allow providers to define custom matching predicates
- Extend matching behavior without modifying CLI source

The matching algorithm currently checks three criteria in Go:

1. `requiredLabels` - exact key-value matches
2. `requiredResources` - FQN existence in component.#resources
3. `requiredTraits` - FQN existence in component.#traits

This change introduces a `ComponentMatcher` interface and a CUE-based implementation that evaluates `#Matches` predicates defined in transformer definitions.

## Goals / Non-Goals

**Goals:**

- Extract `ComponentMatcher` interface for pluggable matching
- Support CUE `#Matches` predicates for custom matching logic
- Maintain backwards compatibility (no `#Matches` → use Go fallback)
- Enable matcher injection via functional options for testing
- Preserve existing verbose output format

**Non-Goals:**

- Changing the Pipeline interface or RenderResult contract
- Supporting runtime-loaded Go plugins
- Adding new CLI flags for matcher selection
- Modifying how MatchResult is consumed by callers

## Decisions

### Decision 1: Interface in same package as implementations

**Choice**: Define `ComponentMatcher` interface in `internal/build/interface.go` alongside implementations.

**Alternatives considered**:

- Separate `pkg/matcher/` package → Rejected: adds complexity, matcher is tightly coupled to build types
- Interface in `internal/build/types.go` → Acceptable but clutters existing file

**Rationale**: Keeps related code together. The interface depends on `LoadedComponent`, `LoadedTransformer`, and `MatchResult` which are all in `internal/build/`.

### Decision 2: Functional options for Pipeline configuration

**Choice**: Use functional options pattern for matcher injection.

```go
type PipelineOption func(*pipeline)

func WithMatcher(m ComponentMatcher) PipelineOption {
    return func(p *pipeline) { p.matcher = m }
}

func NewPipeline(cfg *config.OPMConfig, opts ...PipelineOption) Pipeline
```

**Alternatives considered**:

- Constructor with matcher parameter → Rejected: breaks existing callers
- Setter method → Rejected: allows mutation after construction
- Builder pattern → Rejected: overkill for single option

**Rationale**: Functional options are idiomatic Go, backwards compatible, and extensible for future options.

### Decision 3: Single hybrid matcher (not separate CUE + Go matchers)

**Choice**: One `DefaultMatcher` that checks for `#Matches` and falls back to Go logic.

```go
type DefaultMatcher struct{}

func (m *DefaultMatcher) Match(...) *MatchResult {
    for _, tf := range transformers {
        if tf.HasMatchesPredicate() {
            // Evaluate CUE #Matches
        } else {
            // Use Go-based label/resource/trait matching
        }
    }
}
```

**Alternatives considered**:

- Separate `CUEMatcher` and `GoMatcher` with delegation → Rejected: over-engineered
- Always require `#Matches` → Rejected: breaks backwards compatibility

**Rationale**: Simplest approach that satisfies requirements. Transformer-level decision, not provider-level.

### Decision 4: Fresh CUE context per evaluation

**Choice**: Create new `cue.Context` for each `#Matches` evaluation.

**Alternatives considered**:

- Reuse single context → Rejected: memory accumulation, potential interference
- Context pool → Rejected: premature optimization

**Rationale**: CUE contexts accumulate values. Fresh context ensures isolation and predictable behavior. Performance is acceptable for typical workloads (< 100ms per evaluation per NFR-CM-001).

### Decision 5: Component context injection structure

**Choice**: Inject component as a struct in the CUE evaluation context.

```cue
// Available in #Matches evaluation:
component: {
    name: string
    labels: {[string]: string}
    "#resources": {[string]: _}
    "#traits": {[string]: _}
}
```

**Alternatives considered**:

- Flat namespace (labels at top level) → Rejected: collision risk, less clear
- Full component CUE value → Rejected: may expose internal structure

**Rationale**: Clear namespace, minimal exposure, matches how transformers think about components.

### Decision 6: Error handling for CUE evaluation failures

**Choice**: Treat CUE errors as non-match, record error in MatchDetail.

**Alternatives considered**:

- Fail entire matching → Rejected: one bad predicate shouldn't block all
- Silent ignore → Rejected: users need to know why matching failed

**Rationale**: Resilient behavior with visibility. Users see "Not matched: CUE error: <details>" in verbose output.

## File Changes

| File | Change |
|------|--------|
| `internal/build/interface.go` | NEW: `ComponentMatcher` interface definition |
| `internal/build/matcher.go` | MODIFY: Rename `Matcher` → `DefaultMatcher`, implement interface, add CUE evaluation |
| `internal/build/pipeline.go` | MODIFY: Use interface, add `WithMatcher` option, update `NewPipeline` |
| `internal/build/provider.go` | MODIFY: Extract `#Matches` field from transformer CUE value |

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| CUE evaluation performance | NFR-CM-001 sets 100ms limit; fresh context approach is O(n*m) but acceptable for typical sizes |
| Complex `#Matches` predicates hard to debug | Verbose output includes CUE errors; future: dedicated `opm mod match-test` command |
| Interface adds abstraction overhead | Minimal: single method interface, Go compiler inlines well |
| Breaking change if interface evolves | NFR-CM-002 commits to stability for one minor version |

## Open Questions

1. **Should `#Matches` have access to module metadata?** Current design only exposes component. May need `module.name`, `module.namespace` for some predicates.

2. **Should we support `#Matches` returning a struct with match + reason?** Would enable richer verbose output from CUE side.

3. **Caching evaluated predicates?** If same component matches multiple transformers, we re-inject context. Likely premature optimization.
