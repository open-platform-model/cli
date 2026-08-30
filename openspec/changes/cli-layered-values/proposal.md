## Why

Every `-f` / `--values` file the CLI accepts is loaded on its own and folded into one value by a plain `Unify` loop (`internal/workflow/render/values.go`, and a sibling list in `internal/cmd/module/vet.go`). When two files disagree, or one file sets a field `#config` does not allow, the error the user sees names the merged value, not the files: there is no per-file position, because nothing set one at compile time. The kernel has shipped the surface built for exactly this since `redesign-config-validation` (`kernel.Source`, `LoadSourceFromFile`, `ValidateConfigDetailed`), and the CLI is its first real consumer. The sibling library change `library-phase-and-values-prune` removes every other unused validation entry and keeps this one; adopting it here is what justifies keeping it.

## What Changes

- `opm module build` / `opm module apply` / `opm instance apply` (`internal/workflow/render`): replace `unifyValuesFiles` with one `kernel.Source` per `-f` file built by `Kernel.LoadSourceFromFile` (which already auto-unwraps a top-level `values:`), merged and validated in declaration order by `k.ValidateConfigDetailed(mod.ConfigSchema(), sources)` before `ProcessModuleInstance` / `SynthesizeInstance`. The merged value flows on unchanged; the fallback to `debugValues` when no `-f` is given is unchanged.
- `opm module vet` (`internal/cmd/module/vet.go`): the same `Source` list; validation uses `ValidateConfigDetailed` with `kernel.Partial()` when vet's existing partial mode applies, so per-file positions reach the grouped validation output.
- A conflict between two values files, or a disallowed field in one of them, reports each contributing file and line in the grouped error output (`cmdutil.PrintValidationError` walks the CUE error tree; positions now carry the originating file).
- `pkg/loader.LoadValuesFile` loses its last callers in `internal/` and is removed if nothing outside the CLI depends on it (it is in `pkg/`, so check `opm-operator` first; if a consumer exists it stays, deprecated).
- No new flags, no flag semantics change. Declaration order of `-f` files keeps its later-wins meaning.

SemVer: MINOR for the CLI (a diagnostics improvement, no command or flag surface change). Kernel surface consumed is already released; this change depends on `library-phase-and-values-prune` only in that both were written together and that change keeps the surface this one names.

Complexity justification (Principle VII): this removes CLI code (a hand-rolled unify loop in two places) in favour of one kernel call, and the per-file attribution is behaviour the kernel already computes and the CLI currently discards.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `kernel-render`: the "CLI entry points map onto kernel entry points" requirement gains the values-file path: `-f` files enter as `kernel.Source` via `LoadSourceFromFile` and are validated by `ValidateConfigDetailed`; no CLI-side unification of values files remains.
- `module-synthetic-instance`: the "Values selection mirrors `opm module vet`" requirement's `-f` scenario loads through the kernel's source loader and layered validation instead of `loader.LoadValuesFile` plus unification.
- `mod-vet`: the "mod vet accepts values files for validation" requirement states that a conflict between values files, or a disallowed field, is reported with each file's position in the grouped output.

## Impact

- Packages: `internal/workflow/render` (`values.go`, `render.go`, `module.go`), `internal/cmd/module/vet.go`, `pkg/loader/instance_file.go` (`LoadValuesFile` removal or deprecation), their tests, and the e2e validation-output fixtures that assert grouped errors.
- Library surface consumed: `kernel.Source`, `Kernel.LoadSourceFromFile`, `Kernel.ValidateConfigDetailed`, `kernel.Partial`, `module.Module.ConfigSchema`. All present on `library` `main`; kept by `library-phase-and-values-prune`.
- Sibling: `library/openspec/changes/library-phase-and-values-prune` (removes the typed `ValidateModuleValues*` wrappers this change deliberately does not use; the spelling here is the primitive plus `ConfigSchema()`).
- Users: error output for conflicting or invalid `-f` files gains file and line attribution; successful renders are byte-identical.
