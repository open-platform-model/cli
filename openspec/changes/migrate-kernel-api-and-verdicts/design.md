# Design: migrate-kernel-api-and-verdicts

## Context

See `proposal.md` § Why for the motivation. The design-relevant state:

- The CLI already constructs its kernel with the registry mapping once (`render.NewKernel`: `kernel.WithRegistry(cfg.Registry)` plus a matching `schema.OCILoader`). Every `loaderfile.LoadOptions{Registry: cfg.Registry}` at an acquire call site therefore repeats a mapping the kernel already holds; deleting the argument removes a duplicate, not a capability.
- Two acquire sites still go through the removed raw tier: `internal/config/platform.go` (`LoadPlatformPackage` + `NewPlatformFromValue`) and `internal/workflow/render/module.go` (`LoadModulePackage` + `NewModuleFromValue`). Both discard the intermediate value; neither reads it.
- `internal/publish/kernel_gate.go` is the one caller of a package-level `loaderfile.LoadModulePackage` free function rather than a kernel method. It has no kernel in scope today.
- `internal/scaffold/{scaffold,repair}.go` call `LoadModulePackage` purely as a shape gate — they discard the value and route on the sentinels.
- The CLI's `render.Result.Warnings []string` is read by `output_internal.go` and `cmd/instance/diff.go` and constructed in exactly one place (`render.go:201`). It is a CLI type, not a library type, so removing the library's field does not force the CLI's field to change shape.
- `internal/cmdutil/output.go` already owns the unresolved-demand wording; it just takes the aggregate as its parameter and reconstructs one in `validation.go` to call it.

Constraint: the target library alpha does not exist yet. Every task must be verifiable against a local `replace` directive and must leave `go.mod` pinned to a real published version until the alpha lands.

## Goals / Non-Goals

**Goals:**

- One green tree at the end, with the intermediate states green wherever the compiler allows it.
- The CLI's own public surface (commands, flags, output format, exit codes) is unchanged: this is a dependency migration, not a UX change.
- Every kernel-authored string the CLI used to pass through has a CLI-owned replacement that names the same facts.
- The candidate evidence the kernel now puts on the diagnostics is actually used, rather than decoded and dropped.

**Non-Goals:**

- Changing what the CLI renders, how it matches, or the render digest. The parity integration program is a guard on this, not a target.
- Reshaping `render.Result` or `pkg/errors`. `Result.Warnings []string` stays a `[]string`; only its producer moves.
- Adopting anything else new in the target library alpha beyond what these two changes force.
- Re-wording the CLI's existing skew sentence for its own sake. It keeps its current wording (the library's copy was the one that had to go, not the words).

## Decisions

### The two library migrations land as one change, not two

**Context**: `one-api-tier` and `cue-owned-verdicts` are separate library changes, and the repo's constitution prefers tiny batches.
**Explored**: (A) two CLI changes, acquire-surface first; (B) one change.
**Decision**: B.
**Rationale**: both library changes ship in the same alpha, so there is no `go.mod` pin at which only the first is present. Under (A) the first change would have to be verified against a library commit rather than a release, and the four files under `internal/workflow/render/` that both changes touch would be edited twice. The batch is still small — seventeen files, almost all one-line deletions — and the task groups below are individually reviewable, which is what the small-batch principle is actually protecting.

### The publish gate takes a kernel rather than growing one

**Context**: `kernel_gate.go` calls the free function `loaderfile.LoadModulePackage`, which no longer exists; the replacement is a method on `*kernel.Kernel`, and the gate has none.
**Explored**: (A) construct a throwaway kernel inside the gate; (B) thread the caller's kernel in through the existing options struct.
**Decision**: B — add the kernel to the gate's options struct, supplied by the publish command that already builds one.
**Rationale**: (A) would create a second kernel per invocation and a second schema cache, which the `kernel-render` spec forbids ("exactly one `kernel.Kernel` per invocation"). The gate's options struct already carries `Context` and `Registry`; the registry field becomes redundant once the kernel carries the mapping and is dropped with it.

### `Result.Warnings` keeps its type; only its producer moves

**Context**: the library removed `RenderResult.Warnings []string`; the CLI has its own field of the same name and type.
**Explored**: (A) reshape `render.Result` to carry typed advisory rows and format at the print site; (B) keep `[]string` and add one CLI formatter at the single construction site.
**Decision**: B, with the formatter a private function in `internal/workflow/render`.
**Rationale**: every reader of `Result.Warnings` wants a line to print. (A) would push formatting into two call sites (`output_internal.go` and `diff.go`) and change a struct that six files read, for no behavior difference. If a future change wants structured warnings in `--output json`, it can add a field beside this one; nothing here blocks that.

### The unmatched formatter uses the candidates it now receives

**Context**: `formatRenderDiagnostics` prints `component %q: no transformer matched` per unmatched component. The rows now carry each evaluated candidate with the labels it lacked.
**Decision**: keep the existing one-line-per-component output as the default, and print the candidate verdicts underneath it in verbose mode only.
**Rationale**: the default output must not grow for users who were not asking; but decoding evidence and discarding it is exactly the waste the library change was made to end. Verbose is where the CLI already prints match reasons, so the evidence lands beside its sibling.

### Sentinel comparisons move package, not shape

**Context**: `loaderfile.ErrWrongKind` and friends are now `liberrors.Err*`.
**Decision**: mechanical rename at the six comparison sites; the CLI keeps its own refusal wording and its own exit codes.
**Rationale**: the sentinels are the same values with the same meaning; the library only moved where they are declared (`opm/internal/loader` declares the gate, `opm/errors` declares its sentinels). Nothing about the CLI's classification changes.

## Risks / Trade-offs

- [The target library alpha does not exist, so the work cannot be finished in one sitting] → every task is staged against a local `replace github.com/open-platform-model/library => ../library`, with the pin bump as the last task; a reviewer can run the whole change before the release exists.
- [The render-parity digest could drift if an acquire-verb swap changes which bytes are staged] → the parity integration program is run explicitly as a task, not left to CI; `AcquireModuleFromDir` stages the same byte overlay `LoadModulePackage` built from, so a drift is a bug, not an expected difference.
- [The CLI's warning wording diverges from the operator's for the same fact] → accepted, and the point of the library change: the two frontends want different dedup keys and different phrasing. The facts named are pinned by the `kernel-render` spec so neither can silently drop one.
- [`kernel_gate.go` gaining a kernel field widens a struct the publish pipeline passes around] → it loses the `Registry` field in the same edit, so the struct does not grow.

## Migration Plan

1. Add `replace github.com/open-platform-model/library => ../library` locally. Land the acquire-surface group (tasks 1-2) — mechanical, compiler-guided, no behavior change.
2. Land the verdict group (tasks 3-4): the formatters, then the `Result.Warnings` producer.
3. Run the full gate plus the two integration programs (task 5).
4. When the library alpha is published, replace the `replace` directive with the real pin and re-run the gate (task 6). Rollback is a re-pin to `v1.0.0-alpha.26` together with a revert of this change; no persisted state, cluster resource or output format changes shape.

## Open Questions

- Which alpha number carries both library changes. It does not affect the specs, the approach or the task breakdown — only the literal in `go.mod` at the last task.
