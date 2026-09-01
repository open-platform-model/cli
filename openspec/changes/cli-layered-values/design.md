## Context

See `proposal.md` (Why). This section records only what the code actually does
today, verified against the tree and the pinned library, because three of the
decisions below turn on details the proposal states loosely.

Two independent values pipelines exist:

| | render path | vet path |
| --- | --- | --- |
| entry | `internal/workflow/render/render.go:76`, `module.go:187` | `internal/cmd/module/vet.go:144` |
| load | `pkg/loader.LoadValuesFile` per file | `pkg/loader.LoadValuesFile` per file |
| merge | `unifyValuesFiles`: `Unify` loop, then one `unified.Err()` check wrapped as `fmt.Errorf("unifying values files: %w", err)` | `validate.Config` merges after a per-file pass |
| validate | none of its own: the merged value goes to `ProcessModuleInstance` / `SynthesizeInstance` | `pkg/validate.Config` |
| print | `printValidationError` -> `cmdutil.PrintValidationError("render failed", err)` | `cmdutil.PrintValidationError("values do not satisfy #config", err)` |

Verified facts that shape the design:

1. `pkg/loader.LoadValuesFile` (`pkg/loader/instance_file.go`) and
   `Kernel.LoadSourceFromFile` (library `opm/kernel/source_loader.go:47`) are
   logically identical: `filepath.Abs`, `os.Stat`, `load.Instances` with
   `Dir` set to the parent, `BuildInstance`, then auto-unwrap of a top-level
   `values:` field. `load.Instances` already populates `token.Pos.Filename`
   with the absolute path. Positions therefore already carry a filename on
   both paths today; what the CLI loses is downstream, not at load time.
2. `pkg/validate.Config` (`pkg/validate/config.go:15`) is a fork of the
   kernel's `runValidate` / `appendSchemaErrors` / `walkDisallowed` /
   `fieldNotAllowedError` / `normalizeFieldPath`, with one addition: a
   staged shape. It runs a non-concrete pass per file, then unifies while
   collecting `merged.Err()`, and runs the concrete pass on the merged value
   only when both earlier stages were clean.
3. `Kernel.ValidateConfigDetailed` (library `opm/kernel/validate.go:174`) is
   not staged: it unifies every `Source` and runs one `runValidate` over the
   result, concrete unless `Partial()` is passed.
4. `validate.Config` has exactly one non-test caller, `vet.go:144`.
   `LoadValuesFile` has no consumer in `opm-operator`.
5. The pinned `github.com/open-platform-model/library v1.0.0-alpha.22`
   already exports `Source`, `LoadSourceFromFile`, `ValidateConfigDetailed`,
   `Partial` and `Module.ConfigSchema`. No dependency bump is part of this change.
6. `vet.go:127` runs `valuesVal.Validate(cue.Concrete(true))` per file,
   before `#config` is consulted.

## Governing constraint: full library reliance

**The CLI MUST NOT carry values-loading or values-validation logic of its own.**
Loading a values file, merging values, and validating them against `#config`
are kernel responsibilities in their entirety. The CLI's remaining job at
these call sites is to collect `-f` paths, hand them to the kernel, and render
whatever error tree comes back.

This is a hard constraint on every decision below, not a preference to be
traded against output stability. Where the kernel cannot currently produce
what the CLI needs, **the fix is a library change, not a CLI workaround**:
open or extend the sibling change in `library/openspec/changes/`, land the
kernel surface, re-pin, and come back. A CLI-side reimplementation of
anything the kernel does "because the kernel does it slightly differently" is
the failure mode this change exists to remove, and it is not an acceptable
outcome of this change.

The line between the two repos is the one the kernel already draws in its own
doc comment on `ValidateConfig`: *"Presentation is outside the kernel's
contract, frontends own their own formatting."* Error **structure** is the
library's. Error **rendering** is the CLI's. Nothing in the CLI may inspect,
re-derive, or supplement the structure.

## Goals / Non-Goals

**Goals:**

- Every `-f` values file at all four call sites is loaded by
  `Kernel.LoadSourceFromFile` and validated by `Kernel.ValidateConfigDetailed`.
  After this change, `grep -rn "walkDisallowed\|Unify()" internal/ pkg/`
  returns nothing on the values path.
- A conflict or a disallowed field across two `-f` files names the
  contributing file in the grouped output.
- The rendered error block is byte-identical wherever the diagnostic itself
  did not change. The section 1 and 2 baselines in `tasks.md` are the
  acceptance evidence, not a reviewer's reading.

**Non-Goals:**

- Component-level (`#components`) diagnostics. They come from the kernel
  loader and matcher, not from this path. The baselines capture them purely
  as an invariance check: any diff there is a regression, not an improvement.
- Changing what `-f` means. See D3.
- A new formatter. `cmdutil.PrintValidationError` and its grouping are
  untouched (D4).
- `opm module apply` / `opm instance apply` behavior beyond what they inherit
  from the shared render call sites.

## Research & Decisions

### D1: The CLI performs no values validation of its own

**Context**: `validate.Config` is staged (fact 2); `ValidateConfigDetailed` is
not (fact 3). Adopting the kernel primitive changes which errors are produced,
not only how they are positioned. Two concrete consequences: a values set that
is both schema-invalid and incomplete reports only the schema errors today but
reports both under the kernel's single pass; and a disallowed field present in
two files is reported once per file today and once against the merged value
after.

**Explored**: `pkg/validate/config.go:15-51` against library
`opm/kernel/validate.go:174-199` and `54-97`; the existing expectations in
`tests/e2e/vet_output_test.go` and `pkg/validate/config_diag_test.go`.

**Options considered**:

1. Adopt `ValidateConfigDetailed` as-is and accept whatever the error set
   becomes. Satisfies the constraint. Risks a worse diagnostic than today with
   no recourse inside this change.
2. Reproduce the staging in the CLI by calling `ValidateConfigDetailed` twice,
   once per source with `Partial()` and once over all sources. Preserves
   today's error sets exactly. **Rejected: it violates the governing
   constraint.** Merge-and-report ordering is validation policy, and policy
   that decides which errors a user sees does not belong in a frontend.
3. Adopt `ValidateConfigDetailed` as the only validation call, and treat any
   diagnostic the baselines show degrading as a **library defect**: land the
   missing behavior in the kernel (a `Staged()` `ValidateOption`, or per-source
   attribution inside the existing pass), re-pin, and re-run the baselines.

**Decision**: Option 3.

**Rationale**: Option 1 and option 3 produce the same CLI code; they differ
only in what happens when the baseline diff is unacceptable, and option 3
names that path in advance so it is not improvised into option 2 under
schedule pressure. The staging in `validate.Config` is not CLI knowledge that
the kernel lacks: it is the same walk, forked. If it is worth keeping, it is
worth keeping in the kernel where the operator and any future frontend get it
too.

**Escalation trigger (task 5.5)**: the baseline diff shows, for any case, a
lost diagnostic, a lost position, or a message a user cannot act on. That is a
library bug. Do not patch it in the CLI.

### D2: `pkg/validate` is deleted

**Context**: With D1, `pkg/validate.Config`'s entire body is either forked
kernel logic or the staging D1 rejects. Its only remaining contribution is the
`*oerrors.ConfigError` wrapper. It has exactly one non-test caller,
`vet.go:144` (fact 4).

**Decision**: The `pkg/validate` package MUST be deleted in full:
`config.go`, `config_test.go`, `config_diag_test.go`, `testdata/`. `vet.go`
MUST call the kernel directly and wrap the returned tree for display:

```go
// internal/cmd/module/vet.go
sources, err := loadValueSources(k, rf.Values)   // thin: LoadSourceFromFile per -f
if err != nil { /* exit 1 */ }

if _, vErr := k.ValidateConfigDetailed(mod.ConfigSchema(), sources); vErr != nil {
    cmdutil.PrintValidationError("values do not satisfy #config",
        &oerrors.ConfigError{Context: "module", Name: modName, RawError: vErr})
    return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: vErr, Printed: true}
}
```

`ConfigError.RawError` is a plain `error` field (`pkg/errors/config_error.go:24`),
so the kernel's raw tree drops in with no adaptation.

**Rationale**: The wrapper is presentation: it exists so
`PrintValidationError` takes its `*ConfigError` branch and prints the header
and grouped block it prints today (D4). Constructing it at the call site is
formatting, which the constraint assigns to the CLI. Keeping a whole package
to hold one struct literal is not.

`pkg/errors.ConfigError` itself stays: it is the printer's contract, not
validation logic.

### D3: `-f` is unification, not later-wins

**Context**: `proposal.md` states "Declaration order of `-f` files keeps its
later-wins meaning." That is not the current behavior. `unifyValuesFiles`
(`values.go:17`) calls `cue.Value.Unify`, and `ValidateConfigDetailed`
(`validate.go:190-193`) does the same. Under unification two conflicting
concrete values are an error, not an override. The order matters only for
which position CUE reports first.

**Decision**: The design MUST preserve unification semantics. Neither path
gains override behavior. The proposal sentence MUST be corrected to say that
declaration order is preserved and that conflicting concrete values across
files remain an error.

**Rationale**: Override semantics would be a breaking behavior change
disguised as a diagnostics change, and it is exactly the case the new
attribution is meant to explain rather than silently resolve. If later-wins is
wanted it is a separate change with its own proposal, and by D1 it would be a
library change first.

### D4: Error presentation is unchanged

**Context**: `cmdutil.PrintValidationError` (`internal/cmdutil/output.go:25`)
branches on `*pkgerrors.ConfigError` first, then `*ValidationError`, then
`*liberrors.UnresolvedDemandsError`, then falls back to
`GroupedErrorsFromError` gated on `hasPositions`. Both the first and the last
branch end in `groupCUEErrors`.

**Decision**: No change to `internal/cmdutil` or `pkg/errors`. The vet path
MUST keep producing a `*ConfigError` (D2); the render path MUST keep handing
the raw wrapped error to `printValidationError`. Attribution reaches the
output solely because `token.Pos.Filename` on the leaves now names the
contributing file, which `groupCUEErrors` already prints.

**Rationale**: The grouping already renders "one message, all positions that
report it", which is precisely the shape a cross-file conflict wants. Nothing
in the formatter needs to know about `Source`. This is the CLI's half of the
constraint's dividing line, and it is where CLI code is legitimate.

### D5: The render path validates where it did not before

**Context**: Today `unifyValuesFiles` merges and checks only `unified.Err()`;
`#config` validation on the render path happens later inside
`ProcessModuleInstance` / `SynthesizeInstance`. The proposal asks for
`ValidateConfigDetailed(mod.ConfigSchema(), sources)` to run before those.

**Decision**: The render path MUST call `ValidateConfigDetailed` with no
options (concrete required) before `ProcessModuleInstance`, and MUST pass the
returned merged value on unchanged. `unifyValuesFiles` is replaced by a
`[]kernel.Source` builder with no CUE operations in it.

**Rationale**: The kernel enforces concreteness in the same place today, so a
values set that passes now still passes; what moves is where the failure is
reported from and, with it, the message framing. This is the single riskiest
part of the change for output stability, and it is why the section 1 matrix
includes `build-two-file-conflict` and `instance-vet-conflict` and not only
the vet cases. If the captures show framing move, the fix is the wrapping at
the call site (presentation, CLI-side) or the kernel's message (structure,
library-side), never a re-added CLI merge step.

### D6: `LoadValuesFile` is deleted

**Context**: Fact 1 (identical logic) and fact 4 (no `opm-operator` consumer).

**Decision**: `pkg/loader.LoadValuesFile` MUST be deleted along with its
tests, and every caller MUST use `Kernel.LoadSourceFromFile`. The proposal's
"deprecate if a consumer exists" branch does not apply.

**Rationale**: Verified, not assumed: `grep -rn "LoadValuesFile\|cli/pkg/loader"`
over `opm-operator` returns nothing. A duplicate loader in the CLI is the same
category of defect as a duplicate validator.

### D7: Vet's per-file concreteness pre-check is removed

**Context**: `vet.go:127` runs `valuesVal.Validate(cue.Concrete(true))` on
each `-f` file individually, before `#config` is consulted (fact 6). This is
CLI-side validation logic, and it is also wrong: a field left abstract in
`base.cue` and filled by `prod.cue` is rejected before the merge is
considered.

**Decision**: The loop MUST be deleted. Concreteness is checked once, by
`ValidateConfigDetailed` on the merged value.

**Rationale**: Required by the governing constraint. The behavioral effect is
that a previously rejected invocation now succeeds, which is a fix, not a
regression; it is the only case in this change where the CLI accepts input it
used to reject, and the `vet-non-concrete` baseline case is what makes it
visible rather than silent. SemVer stays MINOR (no command or flag surface
change); the commit carrying it is a `fix`.

## Data flow

Before, per call site:

```
-f a.cue ,┐
-f b.cue ,┴─> LoadValuesFile x2 ─> Unify loop ─> cue.Value ─> ProcessModuleInstance
                                    (render: only unified.Err())
                                └─> validate.Config (staged) ─> ConfigError  (vet)
```

After, one shape, with no CUE operation left on the CLI side of the arrow:

```
-f a.cue ─> k.LoadSourceFromFile ─> Source{Value, Name: "a.cue", Origin: "/abs/a.cue"} ─┐
-f b.cue ─> k.LoadSourceFromFile ─> Source{...b.cue}                                    ├─> []Source
                                                                                        │
   render: k.ValidateConfigDetailed(mod.ConfigSchema(), sources) ─> merged ─────────────┴─> ProcessModuleInstance
   vet:    k.ValidateConfigDetailed(mod.ConfigSchema(), sources) ─> err
                                                    └─> &ConfigError{RawError: err} ─> PrintValidationError
```

`Source.Origin` is the absolute path, set by `LoadSourceFromFile` and matching
the `cue.Filename` baked in at compile time; that identity is the whole
mechanism by which positions survive the merge.

## Error handling

Command syntax, flags and exit codes are unchanged: `opm module vet [path]
[-f FILE]...`, `opm module build [path] [-f FILE]...`, `opm instance apply
FILE [-f FILE]...`. Exit codes stay 0 success, 1 usage or general error, 2
validation error (`internal/exit`: `ExitValidationError = 2`).

| condition | today | after |
| --- | --- | --- |
| `-f` file missing | `values file %q not found`, exit 1 | same text from `LoadSourceFromFile`, exit 1 |
| `-f` file does not compile | `loading values file: %w` / `building values file: %w`, exit 1 | same text, exit 1 |
| field not in `#config` | `field not allowed` + position, exit 2 | unchanged text; position names the file that set it |
| two files set the same field to different concrete values | one `conflicting values` group, exit 2 | same group, now listing both files' positions |
| one `-f` file non-concrete on its own, completed by a later file | rejected, exit 2 | accepted (D7) |
| merged values not concrete | `values are not fully concrete` (vet) / kernel message (render), exit 2 | kernel message on both paths; framing proven identical or corrected by the baseline diff |

Expected output for `vet-two-file-conflict`, the case that justifies the
change (`<testdata>` is the capture script's path normalization):

```
ERRO values do not satisfy #config: 1 issue
  conflicting values "pvc" and "emptyDir"
    <testdata>/layered/base.cue:5:11
    <testdata>/layered/conflict.cue:4:11
```

The header, the two-space message indent and the four-space position indent
all come from the existing `printGrouped`; only the second position line is new.

## Risks / Trade-offs

- **The kernel's single pass produces a worse diagnostic than today's staged
  one** -> D1 option 3. The baseline diff detects it; the response is a
  library change, and task 5.5 blocks the CLI change until it lands. This is
  the risk the governing constraint deliberately accepts in exchange for one
  implementation.
- **This change now depends on the library being right, and a library fix
  costs a release and a re-pin** -> accepted, and cheaper than the alternative
  the repo already lived with: two validators drifting apart, with the
  operator inheriting whichever one it happened to import.
- **The render path's new validation call changes the message framing** ->
  D5. Caught by `build-two-file-conflict` and `instance-vet-conflict`.
- **D7 makes a previously rejected invocation succeed** -> intended, recorded,
  and visible in the `vet-non-concrete` baseline. It is the only acceptance
  change in the change.
- **`pkg/validate` is a `pkg/` export being deleted** -> no consumer outside
  this repo (verified for `opm-operator`, the only sibling importing
  `cli/pkg`). SemVer stays MINOR per the proposal.

## Migration Plan

No user migration: no flag, command or file-format change. Rollback is a
revert of the implementation commits; the baseline captures and fixtures are
inert on their own.

Implementation order, each step green before the next, each its own commit
(Principle VIII):

1. `LoadSourceFromFile` replaces `LoadValuesFile` at all four call sites;
   `unifyValuesFiles` becomes a `[]kernel.Source` builder. Delete
   `LoadValuesFile` and its tests. (D6)
2. `vet.go` calls `ValidateConfigDetailed` directly and wraps for display;
   delete `pkg/validate` and the per-file concrete loop. (D1, D2, D7)
3. Render path calls `ValidateConfigDetailed` before `ProcessModuleInstance`.
   (D5)
4. Only if task 5.5 fires: pause, land the kernel fix in `library`, re-pin,
   re-run both baselines, resume.

## Open Questions

- Whether the two always-pass baseline tests (`values_baseline_test.go`,
  `print_validation_baseline_test.go`) stay after archive or are deleted with
  the change directory. Either is fine; it does not affect this design.
