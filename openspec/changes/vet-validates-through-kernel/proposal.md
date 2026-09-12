## Why

`opm module vet` validates values through `pkg/validate`, a byte-for-byte copy of the library kernel's validator (`opm/kernel/validate.go`), and loads `-f` files through `pkg/loader.LoadValuesFile`, a copy of `Kernel.LoadSourceFromFile`. The render workflow already does the same job through the kernel (`internal/workflow/render/values.go`: `-f` files and `debugValues` become kernel values sources, checked by `Kernel.ValidateConfigDetailed`), so `vet` and `build` can disagree on a verdict, and every validator fix in the library has to be ported by hand. This is slice 07 of the kernel diet: the CLI stops carrying copies of kernel code.

## What Changes

- `vet` resolves its values exactly as `opm module build` does: `-f` files as file-backed kernel sources, else the module's `debugValues` rendered to a source attributed to `<module>/debugValues`. The resolver that `FromModule` uses today becomes an exported, value-based function both callers share.
- `vet` validates through `Kernel.ValidateConfigDetailed` against the module's `#config`. The kernel asserts concreteness of the merged value, so vet's separate per-file "not fully concrete" pre-check and its message go; a non-concrete input is reported as a `#config` violation with positions, exit 2 as today.
- **BREAKING (Go importers of `pkg/`):** `pkg/validate` is deleted with its testdata; `pkg/loader.LoadValuesFile` is deleted; the `ConfigError` type in `pkg/errors` is deleted (its only producer was `pkg/validate`). The grouped-error helpers in `pkg/errors` stay. `cmdutil.PrintValidationError` keeps its grouped CUE-position output through the existing fallback branch, which already handles the kernel's error tree.
- Unchanged: the identity and coordinate checks, exit codes, the "Values satisfy #config" and "Module config valid" lines, `-f` precedence over `debugValues`, the error when neither exists.

**Not in this change:** `pkg/loader.LoadInstanceFile` (change `instance-arg-acquires-through-kernel`); the `pkg/module` types (change `library-metadata-types`); anything in the library.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mod-vet`: "mod vet does not use the render pipeline" - steps 2 to 4 become: resolve values as kernel sources, validate them through the kernel's layered validation, which includes the concreteness check.
- `cmdutil`: "PrintValidationError formats render validation errors consistently" - no `ConfigError` branch; grouped output is driven by CUE positions carried by any error.
- `errors-domain`: "ConfigError type for gate validation" is removed.
- `instance-building`: "`opm mod vet` uses `debugValues` by default" - the unconstrained-`debugValues` scenario reports through the standard validation output instead of a bespoke message.

## Impact

**SemVer:** PATCH for the command surface: no flag or syntax change. One failure's wording changes: a non-concrete values input prints the standard "values do not satisfy #config" block instead of "<source> not concrete". For Go importers of `pkg/validate`, `pkg/loader.LoadValuesFile` or `pkg/errors.ConfigError` this is a breaking removal.

**Packages:** `internal/cmd/module` (vet), `internal/workflow/render` (resolver exported and made value-based), `internal/cmdutil` (one branch removed), `pkg/validate` (deleted), `pkg/loader` (one function), `pkg/errors` (one type). Commands affected: `opm module vet`.

**Complexity justification (Principle VII):** net negative, about 300 lines deleted; one exported function added where a private one existed.
