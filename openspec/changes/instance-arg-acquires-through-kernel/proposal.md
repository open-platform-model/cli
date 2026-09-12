## Why

`cmdutil.ResolveInstanceArg` resolves a path argument to `opm instance status|tree|events|delete` by calling `pkg/loader.LoadInstanceFile`, which re-implements the kernel's package load in a `cue.Context` of its own and passes the registry by setting `CUE_REGISTRY` on the process (`os.Setenv`, restored on return). That mutates global state the library forbids and the rest of the CLI no longer does, and it is the last CLI load path that does not go through a kernel acquire verb. `Kernel.AcquireInstanceFromDir` loads the same package with the registry supplied at construction and hands back decoded metadata. Slice 07 of the kernel diet.

## What Changes

- A path argument (an `instance.cue` file or a directory holding one) is acquired through the kernel: the CLI resolves the package directory (the path itself, or the file's parent), constructs the invocation's kernel with the resolved registry, calls `AcquireInstanceFromDir`, and reads the instance name and namespace from the acquired instance's metadata.
- `--namespace` keeps precedence over the namespace in the file. A path whose package is not a valid instance (wrong kind, no concrete `metadata.name` or `metadata.namespace`, build failure) is refused with exit 1, as a load failure is today. The best-effort namespace fallback goes: core requires a concrete namespace on every instance, so it only ever served packages that cannot render.
- The instance-directory resolution `internal/workflow/render` keeps privately moves to `cmdutil` so both callers share it.
- **BREAKING (Go importers of `pkg/`):** `pkg/loader/instance_file.go` is deleted (`LoadInstanceFile`, `LoadOptions`, the `instance.cue` directory lookup) with its tests. `pkg/loader/provenance.go` (`ModuleRootFrom`, `LocalReplacements`, `HasLocalModuleReplacement`) stays: it is CLI provenance, not a kernel copy.

**Not in this change:** filling the selector's UUID from the file (name and namespace stay the lookup key); the render-side instance path (already through the kernel); `pkg/loader.LoadValuesFile` (change `vet-validates-through-kernel`).

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `inst-commands`: "instance cluster-query commands accept an instance identifier as positional argument" gains the path form: acquired through the kernel, name and namespace from the acquired instance, `--namespace` precedence, invalid package refused.
- `instance-file-loading`: "Instance file loader lives in `pkg/loader/`" is removed.

## Impact

**SemVer:** PATCH for the command surface: no flag or syntax change. Behaviour change: a path whose instance carries no concrete namespace is refused instead of falling back to the configured default. For Go importers of `pkg/loader.LoadInstanceFile` this is a breaking removal.

**Packages:** `internal/cmdutil` (instance argument resolution, target resolution takes a context), `internal/cmd/instance` (four commands pass their context), `internal/workflow/render` (directory helper moves out), `pkg/loader` (one file deleted). Commands affected: `opm instance status`, `tree`, `events`, `delete` with a path argument.

**Depends on:** `one-kernel-constructor` (`config.NewKernel`).

**Complexity justification (Principle VII):** net negative, about 150 lines deleted; the kernel call replaces a loader.
