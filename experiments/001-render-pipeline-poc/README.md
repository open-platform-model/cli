# 001-render-pipeline-poc

Proof of concept for the render pipeline design specified in [013-cli-render-spec](../../../opm/specs/013-cli-render-spec/).

## Purpose

This experiment validates the key architectural decisions for the CLI render system:

1. **Parallel CUE execution** using isolated `cue.Context` per worker
2. **AST-based data transport** between goroutines (thread-safe)
3. **Transformer contract** matching the spec
4. **Full data preservation** through the transformation pipeline
5. **Fail-on-end error aggregation**

## Key Findings

### AST Transport Pattern

The CUE Go API (`cue.Context`) is **not thread-safe**. The solution is:

1. **Main thread**: Load module, match transformers, unify inputs
2. **Export**: `unified.Syntax(cue.Final(), cue.Concrete(true))` produces thread-safe `ast.Expr`
3. **Worker thread**: `workerCtx.BuildExpr(ast)` re-hydrates in isolated context

**Critical insight**: The unification must happen in the main context BEFORE exporting to AST. Exporting a transformer definition with unresolved `#component` references will fail in the worker.

### Transformer Contract

Transformers follow the spec contract:

```cue
#Transformer: {
    metadata: { name, description, version }
    requiredLabels: { ... }
    requiredResources: { ... }
    #transform: {
        #component: _
        #context: TransformerContext
        output: { apiVersion, kind, metadata, spec, ... }
    }
}
```

### TransformerContext

Injected into every transform with OPM tracking labels:

```go
type TransformerContext struct {
    Name      string            // Module release name
    Namespace string            // Target namespace
    Version   string            // Module version
    Provider  string            // Provider name
    Timestamp string            // RFC3339 timestamp
    Strict    bool              // Strict mode flag
    Labels    map[string]string // OPM tracking labels
}
```

## Running

```bash
cd cli/experiments/001-render-pipeline-poc

# Output clean YAML to stdout (default)
go run .

# Output to file
go run . -o manifests.yaml

# Verbose mode (shows pipeline phases on stderr)
go run . -v

# Combine: verbose + file output
go run . -v -o manifests.yaml
```

Output is valid Kubernetes YAML that can be applied:

```bash
go run . | kubectl apply --dry-run=client -f -
```

### Flags

| Flag | Description |
|------|-------------|
| `-o <file>` | Write YAML output to file (default: stdout) |
| `-v` | Verbose mode - show pipeline phases on stderr |

## Files

- `main.go` - Go implementation of the render pipeline
- `poc.cue` - CUE definitions (module release + transformers)
- `cue.mod/module.cue` - CUE module configuration

## Spec Requirements Validated

| Requirement | Status | Notes |
|-------------|--------|-------|
| FR-015 Parallel execution | Pass | Worker pool with isolated contexts |
| FR-016 OPM tracking labels | Pass | Injected via TransformerContext |
| FR-017 YAML output | Pass | Multi-doc YAML with separators |
| FR-019 Unmatched component errors | Pass | Aggregated at end |
| FR-024 Fail-on-end | Pass | All components processed before reporting errors |

## Architecture Decision Record

**Decision**: Unify transformer + component + context in main thread, export unified AST to workers.

**Rationale**:

- Exporting transformer definition alone leaves `#component` references unresolved
- Workers receive fully-concrete AST, avoiding CUE evaluation in parallel
- Minimizes work in workers (just decode output)

**Trade-off**: Main thread does more work, but this is necessary for correctness.
