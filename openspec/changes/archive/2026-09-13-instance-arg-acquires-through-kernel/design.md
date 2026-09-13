## Context

See proposal.md for motivation. Relevant facts:

- `cmdutil.ResolveInstanceArg(arg, cfg)` detects a path (`isInstancePath`), then `resolveInstanceArgFromFile` calls `ValidateInstanceInputPath`, builds a bare `cuecontext.New()`, calls `loader.LoadInstanceFile(ctx, arg, LoadOptions{Registry: cfg.Registry})` and extracts `metadata.name` (required concrete) and `metadata.namespace` (best effort) by `LookupPath`.
- `ResolveInstanceTarget(identifier, cfg, kf, namespaceFlag)` is the only caller, itself called by `internal/cmd/instance/{status,tree,events,delete}.go`; none of them constructs a kernel today.
- `Kernel.AcquireInstanceFromDir(ctx, dir, values ...Source)` takes a directory, runs the shape gate (kind `ModuleInstance`, concrete `metadata.name` and `metadata.namespace`), builds in its own context with the kernel's registry, and returns `*module.Instance` with decoded `Metadata`.
- Core requires `metadata.namespace!` on every `#ModuleInstance`, so the best-effort fallback never applied to a valid instance.
- `internal/workflow/render/values.go` keeps a private `resolveInstanceDir(path)` (path if directory, else parent) that the render path uses for the same job; `render` imports `cmdutil`, not the reverse.
- `config.NewKernel(registry)` exists after `one-kernel-constructor` and is reachable from `cmdutil`.

## Goals / Non-Goals

**Goals:**
- The path form of the cluster-query commands resolves identity through the same acquire verb the render path uses.
- No `os.Setenv` anywhere in the CLI's loading code.

**Non-Goals:**
- Selecting the deployed instance by the file's UUID (name and namespace stay the key; a module UUID change would otherwise miss a deployed instance the name still finds).
- Hermetic CLI tests for local-module replacements on this path (covered by the library since alpha.29 and by the render path's tests).

## Research & Decisions

### Acquire through the kernel, not parse the file

**Context**: the path form only needs two strings.
**Explored**: the shape gate's required fields; core's `namespace!`; what today's loader does (a full `load.Instances` + `BuildInstance`, so imports and the registry are already required).
**Options considered**:
1. `AcquireInstanceFromDir` - same verb as render, same registry, decoded metadata, refuses what render would refuse; needs registry access, as today.
2. `parser.ParseFile` and walk `metadata.{name,namespace}` string literals - no registry, no build, but computed names and anything behind an import stop resolving.
**Decision**: option 1.
**Rationale**: identity for `status`/`delete` equals identity at `apply`; the cost is unchanged.

### Context threading

**Context**: the acquire verb takes a `context.Context`; `ResolveInstanceTarget` has none.
**Decision**: `ResolveInstanceArg(ctx, arg, cfg)` and `ResolveInstanceTarget(ctx, identifier, cfg, kf, namespaceFlag)`; the four commands pass `c.Context()`.
**Rationale**: the kernel ignores the context today but the signature is the library's; a `context.Background()` at the call site would hide a future cancellation.

### One kernel per cluster-query invocation

**Context**: the `kernel-render` spec requires a single kernel per render-bearing invocation; cluster-query commands built none.
**Decision**: `resolveInstanceArgFromFile` constructs `config.NewKernel(cfg.Registry)` for the acquire; nothing else in these commands needs one.
**Rationale**: the constructor is the single site the previous change introduced; the commands stay at one kernel.

### Shared directory resolution

**Context**: `render.resolveInstanceDir` and the deleted `resolveInstanceFile` both map a path to a package directory.
**Decision**: `cmdutil.InstanceDir(path string) (string, error)`; `render` calls it.
**Rationale**: one definition, in the package both can import.

Sketch:

```go
func resolveInstanceArgFromFile(ctx context.Context, arg string, cfg *config.GlobalConfig) (InstanceArg, error) {
	if err := ValidateInstanceInputPath(arg); err != nil { return InstanceArg{}, err }
	dir, err := InstanceDir(arg)
	if err != nil { return InstanceArg{}, err }
	inst, err := config.NewKernel(cfg.Registry).AcquireInstanceFromDir(ctx, dir)
	if err != nil { return InstanceArg{}, fmt.Errorf("loading instance %q: %w", arg, err) }
	return InstanceArg{Name: inst.Metadata.Name, Namespace: inst.Metadata.Namespace}, nil
}
```

## Risks / Trade-offs

- [A package with an open namespace that used to resolve with `--namespace` is now refused] → it cannot be applied or rendered either; the error names the path and the missing field.
- [A directory holding more than the instance file loads the whole package] → that is the package the render path loads too, so `status` and `apply` agree.
- [The deleted `pkg/loader` tests covered `replaceWith` resolution and a minimal instance file] → the library's acquire and replacement tests cover the mechanism; the CLI keeps a unit test for path detection and directory resolution and a registry-backed test that resolves a real instance package.
