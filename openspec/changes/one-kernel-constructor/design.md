## Context

See proposal.md for motivation. Constraints that shape the placement:

- `internal/cmdutil` imports `internal/config`; `internal/workflow/render` imports `cmdutil`; `internal/cmd/*` import all three. A constructor every site can reach without a cycle must sit at or below `config`.
- `internal/config/platform.go` already imports the library kernel (`BuildPlatformModule` constructs one), so `config` taking the dependency is not new.
- `BuildPlatformModule(ctx, dir, registry string)` receives a registry string, not a `*GlobalConfig`.
- Since library `v1.0.0-alpha.28`, `kernel.New` seeds its schema loader as `schema.OCILoader{Registry: k.registry}` when no loader option is given (`opm/kernel/kernel.go`, `New`), so `WithSchemaLoader` with the same registry is a no-op duplicate.

## Goals / Non-Goals

**Goals:**
- One function constructs every kernel the CLI's `internal/` tree uses.
- The redundant `WithSchemaLoader` option disappears.

**Non-Goals:**
- Changing which commands construct a kernel or when.
- Touching `tests/` programs.

## Research & Decisions

### Home of the constructor

**Context**: `cmdutil` will need a kernel in the next change; `render.NewKernel` is unreachable from `cmdutil`.
**Explored**: the import graph (`grep -ln 'cli/internal/cmdutil"' internal/workflow/render/*.go` lists four files; `cmdutil` imports `config`).
**Options considered**:
1. `cmdutil.NewKernel` - reachable from `render` and `cmd`, not from `config` (`config` cannot import `cmdutil`), so `BuildPlatformModule` would keep its own `kernel.New`.
2. `config.NewKernel` - reachable from every site; `config` already imports the kernel.
3. A new leaf package (`internal/kernelenv`) - reachable everywhere, one more package for a three-line function.
**Decision**: `config.NewKernel(registry string) *kernel.Kernel`.
**Rationale**: it is the lowest existing package on the graph, it already depends on the kernel, and the registry string is exactly the input the kernel takes.

### Signature: registry string, not `*GlobalConfig`

**Context**: five of six sites hold a `*GlobalConfig`; `BuildPlatformModule` holds only the string.
**Decision**: take the string; callers pass `cfg.Registry`.
**Rationale**: the kernel consumes one string. A `*GlobalConfig` parameter would force `BuildPlatformModule` to keep a second constructor.

### Dropping `WithSchemaLoader`

**Context**: `vet.go` and `cmdutil/publish.go` pass it explicitly; `render/kernel.go` passes it with a comment written before alpha.28.
**Decision**: rely on the library's seeding.
**Rationale**: the library documents the seeding as what makes `WithRegistry` reach the schema loader (`kernel.go` doc on `New`); passing the same value twice adds nothing.

```go
// internal/config/kernel.go
func NewKernel(registry string) *kernel.Kernel {
	return kernel.New(kernel.WithRegistry(registry))
}
```

## Risks / Trade-offs

- [A future library release stops seeding the schema loader from the registry] → the flow tests that resolve the core schema through a non-default mapping fail at the library bump; the fix is one line in `config.NewKernel`.
- [A new construction site appears elsewhere] → task 2.1's grep is the check; the `kernel-render` spec's "Single kernel per invocation" scenario is the rule reviewers apply.
