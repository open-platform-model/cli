## Context

The library change `render-skips-unprovided-provider-demands` adds `kernel.RenderInput.SkipUnprovided`, `kernel.RenderDiagnostics.Skipped []kernel.SkippedDemand` and `errors.UnresolvedDemand.Unprovided` (the exact surface is in `orchestration.md`, "Interface B → D"; code against what B's worker reports under `surface`). The rule they implement is core's: a demand is skippable when its contract is provider-fulfilled and no enabled catalog provides it; a skipped trait leaves the component rendering its other pairs; a skipped resource drops the whole component; every skip is reported; everything else still refuses.

This change starts after `retire-local-default-platform` merged, so the resolver, the deps fallback and `refusalHint` are that change's versions. Today every render goes through `newRenderInput` (`internal/workflow/render/render.go`), refusals through `renderInstance` → `printValidationError` + `refusalHint` (`internal/workflow/render/validation.go`), and applies write the ModuleInstance spec through `inventory.ApplySpec` (`internal/inventory/store.go`), where the render-provenance annotation `module-instance.opmodel.dev/source` is stamped or omitted.

## Goals / Non-Goals

**Goals:**

- One flag, `--skip-unprovided`, on every render-bearing command, passed straight to the kernel.
- Skips are always visible: warnings on every run, an annotation on every applied instance.
- The refusal hint names the flag whenever it would help, on every platform source.

**Non-Goals:**

- Deciding what is skippable. That is the kernel's rule.
- Showing the annotation in `opm instance status` or `list` (possible follow-up).
- Any operator behaviour: the operator never sets the switch.

## Decisions

### Flag plumbing

```go
// internal/cmdutil/flags.go
type RenderFlags struct {       // module build, vet, apply
	// ...
	SkipUnprovided bool
}
type InstanceFileFlags struct { // instance build, vet, diff, apply
	// ...
	SkipUnprovided bool
}

const skipUnprovidedHelp = "Render what the platform can: skip provider-fulfilled contracts nothing on the platform provides, and report each one"
```

Both `AddTo` methods register `--skip-unprovided`. The render opts (`InstanceFileOpts`, the module render opts) carry it to `renderEnv`, and `newRenderInput` sets `kernel.RenderInput.SkipUnprovided`. `render.Result` gains `Skipped []kernel.SkippedDemand` so apply can record them.

### Wording the skipped rows

A new formatter beside `formatRenderDiagnostics` groups rows by component, in build order:

```go
// formatSkipped returns one warning line per skipped trait, and one line per
// omitted component naming all of its skipped resources.
func formatSkipped(rows []kernel.SkippedDemand) []string
```

```text
WARN component "db": skipped provider-fulfilled trait "opmodel.dev/catalogs/opm/traits/backup@v1alpha1" (no provider on this platform)
WARN component "archive" not rendered: provider-fulfilled resource "example.dev/catalogs/k8up/resources/backup-store@v1alpha1" has no provider on this platform
```

A row whose `Alternatives` is non-empty appends `; implemented at: <keys>`. The lines go through `output.Warn`, so `opm instance build -o yaml > out.yaml` keeps a clean manifest on stdout.

### The refusal hint

`refusalHint` gains a first branch, independent of the platform source:

```go
if errors.As(err, &unresolved) && anyUnprovided(unresolved.Demands) {
	return unprovidedHint // "a provider-fulfilled contract has no provider on this platform: install one, pass --platform <dir> with a platform that carries one, or pass --skip-unprovided to render the rest"
}
```

The existing deps-source hints stay for the cases the kernel does not mark unprovided (for example, a provider that exists but did not match).

### Recording skips on apply

```go
// internal/inventory/store.go
const AnnotationSkippedContracts = "module-instance.opmodel.dev/skipped-contracts"

type SpecInput struct {
	// ...
	// SkippedContracts are "<component>=<fqn>" pairs; empty omits the
	// annotation so server-side apply removes any prior value.
	SkippedContracts []string
}
```

`ApplySpec` builds one annotations map holding the source annotation (when `SourceLocal`) and the skipped annotation (when non-empty), sorted and deduplicated, and sets it once. The apply workflow fills `SkippedContracts` from `Result.Skipped`.

### Operator-managed instances

`executeThinEditor` writes the spec and lets the operator render; the operator never skips. With `--skip-unprovided`, apply refuses before `executeThinEditor` writes anything:

```text
Error: --skip-unprovided has no effect on instance "hello": the opm-operator renders it and does not skip provider-fulfilled contracts. Install a provider for the contract instead.
```

Exit code 2 (validation).

## Command Syntax

```text
opm module build|vet|apply [module] [flags]
opm instance build|vet|diff|apply <instance> [flags]
      --skip-unprovided   Render what the platform can: skip provider-fulfilled contracts nothing on the platform provides, and report each one (default false)
```

## Error Handling and Exit Codes

| Case | Result | Exit |
| --- | --- | --- |
| Flag off, unprovided demand | refusal, rows printed, unprovided hint | 2 |
| Flag on, only skippable gaps | success, one warning per skip | 0 |
| Flag on, a catalog-fulfilled or matched-but-refused gap remains | refusal as without the flag; the unprovided hint only if another row is unprovided | 2 |
| Flag on, operator-managed instance on apply | refusal before any write | 2 |

## Data Flow

```text
--skip-unprovided -> render opts -> kernel.RenderInput.SkipUnprovided
                                          |
                               Kernel.Render (library rule)
                                  |                  |
                     refusal: Unprovided rows    success: Diagnostics.Skipped
                     -> unprovided hint          -> WARN lines
                                                 -> apply: SpecInput.SkippedContracts
                                                    -> annotation on the ModuleInstance
```

## Research & Decisions

### Where skips are recorded on the cluster

**Context**: A skipped `backup` trait means an instance runs without backups; that must be visible after the terminal output is gone.
**Explored**: The ModuleInstance already carries the CLI's render-provenance annotation, written by the one spec writer `inventory.ApplySpec` and cleared by omission under server-side apply. The ModuleInstance CRD is opm-operator's.
**Options considered**:
1. A status field on the ModuleInstance: typed, but it changes the operator's CRD, which adds a repo to the change set.
2. An annotation beside the provenance one. Chosen.
**Decision**: The `module-instance.opmodel.dev/skipped-contracts` annotation.
**Rationale**: It needs no CRD change, follows an existing pattern, and clears itself on the next complete apply.

### One flag for every render-bearing command

**Context**: The flag could be limited to the author's commands, or to clusters without the operator.
**Explored**: The user decision (2026-09-29) asks for a flag on the commands to "render what can be rendered"; clusters without the operator are the main deploy target, and authors need it on `module build` to see the rest of their output.
**Decision**: All seven render-bearing commands; refuse only where it cannot take effect (an operator-managed instance).
**Rationale**: A flag that silently does nothing is worse than a refusal; everywhere else the kernel does exactly what the flag says.

### Developing before the library release

**Context**: This change needs library API that is not released until B merges.
**Decision**: Develop against B's pushed head as a Go pseudo-version (`go get github.com/open-platform-model/library@<sha>`); the last section pins the released version.
**Rationale**: Lets wave 2 start as soon as B's branch exists without shipping a pseudo-version on `main`.

## Risks / Trade-offs

- [A user deploys with skips and forgets] → the annotation stays on the instance until a complete apply; the warning prints on every run.
- [Handoff to the operator after a skipping apply] → the operator's render refuses the instance until a provider is installed; the annotation says which contracts. Not guarded here beyond the thin-editor refusal.
- [`instance diff --skip-unprovided` shows an omitted component's objects as removals] → that is what an apply with the flag would do; the warning names the component.
