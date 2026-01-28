# 003-hybrid-render

**Hybrid Go+CUE render pipeline** that combines the strengths of experiments 001 and 002.

## Goal

Validate the optimal architecture for the CLI render system by combining:

- **Go orchestration** (Phases 1-2, 4-5) for parallel execution and performance
- **CUE matching logic** (Phase 3) for declarative, extensible transformer selection

This hybrid approach gets the best of both worlds:

- Go's performance for parallel transformer execution (experiment 001)
- CUE's expressiveness for complex matching logic (experiment 002)

## Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                       Hybrid Render Pipeline                    │
├─────────────────────────────────────────────────────────────────┤
│  Phase 1: Module Loading & Validation                     [Go]  │
│           ├─ Load CUE via cue/load                              │
│           ├─ Extract release metadata                           │
│           └─ Build base TransformerContext                      │
├─────────────────────────────────────────────────────────────────┤
│  Phase 2: Provider Loading                                [Go]  │
│           └─ Access provider.transformers from CUE              │
├─────────────────────────────────────────────────────────────────┤
│  Phase 3: Component Matching                             [CUE]  │
│           ├─ CUE evaluates #Matches predicate                   │
│           ├─ CUE computes #matchedTransformers map              │
│           └─ Go reads back the computed matching plan           │
├─────────────────────────────────────────────────────────────────┤
│  Phase 4: Parallel Transformer Execution                  [Go]  │
│           ├─ Iterate CUE-computed matches                       │
│           ├─ For each (transformer, component):                 │
│           │   ├─ Unify transformer.#transform + inputs          │
│           │   ├─ Export unified AST (thread-safe)               │
│           │   └─ Send Job to worker goroutine                   │
│           └─ Workers: isolated cue.Context → Decode output      │
├─────────────────────────────────────────────────────────────────┤
│  Phase 5: Aggregation & Output                            [Go]  │
│           ├─ Collect results from workers                       │
│           ├─ Aggregate errors (fail-on-end)                     │
│           └─ Output YAML manifests                              │
└─────────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. CUE Matching (Phase 3)

Instead of hardcoded Go logic (experiment 001's `switch` statement), we delegate to CUE:

```cue
// matching.cue
#Matches: {
    transformer: core.#Transformer
    component:   core.#Component
    
    // Check requiredLabels, requiredResources, requiredTraits
    result: len(_missingLabels) == 0 && len(_missingResources) == 0 && len(_missingTraits) == 0
}

// provider_extension.cue
#matchedTransformers: {
    for tID, t in transformers {
        let matches = [
            for cID, c in #module.components
            if (#Matches & {transformer: t, component: c}).result {
                c
            },
        ]
        if len(matches) > 0 {
            (tID): {
                transformer: t
                components:  matches
            }
        }
    }
}
```

**Benefits:**

- Declarative matching rules live alongside transformer definitions
- Providers can extend matching logic without Go code changes
- Full support for label/resource/trait-based matching (spec Section 4)

### 2. AST Transport Pattern (Phase 4)

From experiment 001 — the proven thread-safe parallel execution:

1. **Main thread:** Unify `transformer.#transform` with `#component` + `context`
2. **Export:** `unified.Syntax(cue.Final(), cue.Concrete(true))` → `ast.Expr`
3. **Worker thread:** `workerCtx.BuildExpr(ast)` in isolated `cue.Context`

**Benefits:**

- True parallelism (no shared `cue.Context`)
- All CUE evaluation happens in main thread (no race conditions)
- Workers just decode concrete outputs

### 3. Self-Contained CUE Package

Uses local `pkg/` from experiment 002 (not registry dependencies):

```
pkg/
├── core/              # Core type definitions
├── blueprints/        # All 6 blueprint types
├── resources/         # Resource definitions
├── schemas/           # Common schemas
└── traits/            # Trait definitions
```

**Benefits:**

- Simpler for experimentation (no registry setup)
- Complete type definitions for validation
- Easy to iterate on definitions

## Test Coverage

Inherits full test suite from experiment 002:

| Component | Blueprint | Workload Label | Transformer | K8s Resource |
|-----------|-----------|----------------|-------------|--------------|
| `web` | StatelessWorkload + Expose | `stateless` | deployment + service | Deployment + Service |
| `api` | StatelessWorkload | `stateless` | deployment | Deployment |
| `database` | SimpleDatabase | `stateful` | statefulset | StatefulSet |
| `cache` | StatefulWorkload | `stateful` | statefulset | StatefulSet |
| `log-agent` | DaemonWorkload | `daemon` | daemonset | DaemonSet |
| `migration` | TaskWorkload | `task` | job | Job |
| `backup` | ScheduledTaskWorkload | `scheduled-task` | cronjob | CronJob |

**Expected output:** 8 Kubernetes resources (7 workloads + 1 service for `web`)

## Running

```bash
cd cli/experiments/003-hybrid-render

# Clean YAML output to stdout
go run .

# Verbose mode (shows all 5 phases)
go run . -v

# Output to file
go run . -o manifests.yaml

# Combine: verbose + file output
go run . -v -o manifests.yaml
```

Verify output is valid Kubernetes YAML:

```bash
go run . | kubectl apply --dry-run=client -f -
```

## Performance Characteristics

### Matching (Phase 3)

- **CUE evaluation:** O(T × C) where T = transformers, C = components
- **One-time cost:** Happens once in main thread before parallel execution
- **Trade-off:** Slightly slower than hardcoded Go logic, but vastly more flexible

### Transformation (Phase 4)

- **Parallel execution:** O(M / N) where M = matches, N = CPU cores
- **AST transport overhead:** Minimal (one-time export per match)
- **Same performance as experiment 001**

## Spec Requirements Validated

| Requirement | Status | Implementation |
|-------------|--------|----------------|
| FR-001 Provider transformer registry | ✓ | CUE `transformers` map |
| FR-005 Transformer matching criteria | ✓ | `#Matches` predicate |
| FR-010 Unified effective labels | ✓ | CUE label inheritance |
| FR-011 Multiple transformers per component | ✓ | `web` matches both deployment + service |
| FR-015 Parallel execution | ✓ | AST transport + worker pool |
| FR-016 OPM tracking labels | ✓ | `#TransformerContext` derives labels |
| FR-017 YAML output | ✓ | go.yaml.in/yaml/v3 |
| FR-019 Unmatched component errors | ✓ | Aggregated and reported |
| FR-024 Fail-on-end | ✓ | All components processed before exit |

## Comparison to Previous Experiments

### vs. Experiment 001 (Go-only)

| Aspect | 001 | 003 |
|--------|-----|-----|
| Phase 3 Matching | Hardcoded `switch` | CUE `#Matches` predicate |
| Extensibility | Requires Go code changes | Add transformers in CUE |
| Matching Logic | 2 workload types | 6 workload types + traits |
| Phase 4 Execution | ✓ AST transport | ✓ AST transport (same) |

**Verdict:** 003 is strictly better — same performance, vastly more flexible.

### vs. Experiment 002 (CUE-only)

| Aspect | 002 | 003 |
|--------|-----|-----|
| Phase 3 Matching | ✓ CUE `#Matches` | ✓ CUE `#Matches` (same) |
| Phase 4 Execution | CUE `#rendered` (sequential) | Go AST transport (parallel) |
| Parallelism | None | Full parallelism |
| Performance | O(M) sequential | O(M / N) parallel |

**Verdict:** 003 adds parallel execution for large modules, keeping CUE's declarative matching.

## Files

| File | Purpose |
|------|---------|
| `main.go` | Go orchestrator (Phases 1-2, 4-5) |
| `matching.cue` | CUE matching predicate (Phase 3) |
| `provider_extension.cue` | `#matchedTransformers` map |
| `demo.cue` | Provider with 6 transformers |
| `components.cue` | 7 test components |
| `basic_module.cue` | Test module + release |
| `pkg/` | Self-contained CUE type definitions |

## Conclusion

This hybrid approach is the **recommended architecture** for the CLI render system:

1. **Go handles orchestration** → Fast, parallel, production-ready
2. **CUE handles matching** → Declarative, extensible, type-safe
3. **Best of both worlds** → Performance + flexibility

Next steps:

- Benchmark with 100+ components
- Add policy matching (Phase 2.5)
- Implement `--split` output mode
- Add structured JSON verbose output (`--verbose=json`)
