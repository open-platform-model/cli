## Why

Six sites construct the library kernel by hand (`internal/cmd/module/vet.go`, `internal/cmd/module/init.go` twice, `internal/config/platform.go`, `internal/cmdutil/publish.go`, `internal/workflow/render/kernel.go`), and two of them still pass `kernel.WithSchemaLoader(schema.OCILoader{Registry: ...})`, an option the library has seeded from `WithRegistry` itself since `v1.0.0-alpha.28`. The `kernel-render` spec requires the registry mapping to be supplied once per invocation; one constructor makes that a fact instead of a convention. The next change (`instance-arg-acquires-through-kernel`) needs a constructor `cmdutil` can call, and `render.NewKernel` is not one: `internal/workflow/render` imports `cmdutil`.

## What Changes

- `internal/config` gains `NewKernel(registry string) *kernel.Kernel`, the single construction site: `kernel.New(kernel.WithRegistry(registry))`, documented as the one place the resolved registry mapping enters the kernel (module acquisition and the schema loader both read it from there).
- The six construction sites call it. `render.NewKernel` is deleted, the two explicit `WithSchemaLoader` options go with it, and so do the `opm/schema` imports they alone justified.
- No behaviour changes: the same mapping reaches the same two places. After the change `grep -rn 'kernel.New(' internal/` finds exactly one site.

**Not in this change:** any new use of the kernel; the instance-argument path (next change); `tests/` programs that construct their own kernel.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. Pure refactor with no spec-level behaviour change; `.openspec.yaml` sets `skip_specs: true`.

## Impact

**SemVer:** PATCH. No command syntax or output changes.

**Packages:** `internal/config` (new function), `internal/cmd/module`, `internal/cmdutil`, `internal/workflow/render` (call sites). Commands affected: none visibly.

**Complexity justification (Principle VII):** one three-line function replaces six call sites and two redundant options; net negative.
