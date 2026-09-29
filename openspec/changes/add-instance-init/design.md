## Context

See proposal.md for motivation and specs/instance-init/spec.md for the behavior contract. Enhancement 0016 (`enhancements/0016/03-decisions.md`, D1 to D9) is the design intent this change implements; its experiments 01 to 08 back the choices below.

Current state that shapes the approach:

- `opm module init` (`internal/cmd/module/init.go`, `internal/scaffold`) is the sibling this command mirrors: positional classification by shape, `--from`, prompting through `stdinReader`/`promptModulePath`, refusals through `publish.Refusal` (exit 2), `publish.ConnectivityError` (exit 3), and a report ending in a vet hint.
- `internal/modref` (from `split-module-and-instance-inputs`, section 1) parses a major-free path and a `--version` selector, and resolves one published version with the core-compatible walk. It returns the major-qualified import path, the tag, and every skipped major.
- `internal/cuemod.Tidy(ctx, dir, TidyOptions{Registry})` (from `add-dependency-tidy`, section 1) runs `cue mod tidy` in-process on an explicit module root. It honors the CLI's registry and restores the working directory and `CUE_REGISTRY` afterwards.
- The instance shape to emit is the one `library/opm/internal/synth/render.go` stages in memory (`package instance`, `core` and `opmModule` imports, `core.#ModuleInstance`, `metadata`, `#module: opmModule`). 0016 experiment 03 retyped it by hand into a standalone package, tidied it, and built it through `opm instance build`.
- `render.DebugValuesSource` already renders a module field back to CUE with `Syntax(cue.Final(), cue.Concrete(false))` plus `format.Node`. That is the serialization 0016 experiment 04 measured for non-concrete `initValues`: defaults resolve, disjunctions survive, optionals vanish.
- `pkg/loader.ModuleRootFrom(dir)` walks up to the nearest `cue.mod/module.cue`.

## Goals / Non-Goals

**Goals:**

- A pure, deterministic renderer (typed input in, three files out) that tests can pin byte for byte.
- One write path with all-or-nothing semantics that also covers the tidy step.
- Only the existing primitives decide which version to pin (`modref`) and which dependencies to list (`cuemod.Tidy`); init adds no resolution logic of its own.

**Non-Goals:**

- Validating the package (0016 D8), upgrading an existing package, and colocated instances inside a module tree.
- Registry search or discovery (0016 non-goal).
- Any library change (0016 D7).

## Decisions

### Command syntax and flags

```text
opm instance init [instance-name] [module-path] [flags]
```

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--from` | string | `""` | Module path to deploy; alternative to the positional, never both |
| `--version` | string | `""` | `vN` floats within major N; `X.Y.Z` pins; empty takes the highest core-compatible major |
| `--namespace`, `-n` | string | `""` | Instance namespace; required, prompted on a terminal |
| `--dir` | string | instance name | Target directory; must not exist or sit inside a CUE module |
| `--module-path` | string | `instance.local/<name>@v0` | The package's own module path, never published |

### Package layout and signatures

```go
// internal/instinit
package instinit

// ValuesSource names what populated values.cue.
type ValuesSource string

const (
	FromInitValues  ValuesSource = "initValues"
	FromDebugValues ValuesSource = "debugValues"
	FromEmpty       ValuesSource = "empty"
)

// Input is everything Render needs; every field is resolved before Render runs.
type Input struct {
	Name, Namespace   string
	PackageModulePath string         // instance.local/<name>@v0 or --module-path
	Module            module.Version // opmodel.dev/modules/web_app@v1 at v1.0.4
	Core              module.Version // opmodel.dev/core@v2 at the module's declared version
	Values            []byte         // CUE expression for `values:`
	Source            ValuesSource
}

// Files maps a package-relative path to its bytes.
type Files map[string][]byte

// Render is pure and deterministic: the same Input yields byte-identical files.
func Render(in Input) (Files, error)

// PickValues walks the ladder over an acquired module's package value
// (initValues, else debugValues when the whole value is concrete once
// defaults apply, else empty) and serializes the winner with
// Syntax(cue.Final(), cue.Concrete(false)).
func PickValues(pkg cue.Value) ([]byte, ValuesSource, error)

// Write stages files in a sibling temp directory, tidies the staged module
// through cuemod.Tidy, and renames the result onto dir. On any error
// nothing remains at dir or beside it.
func Write(ctx context.Context, dir string, files Files, registry string) error
```

`internal/cmd/instance/init.go` parses arguments, runs the refusals, and calls `modref`, the kernel acquire and `instinit`. The terminal and prompt helpers move from `internal/cmd/module/init.go` to `internal/cmdutil` so both init commands share one copy.

### Data flow

```text
opm instance init [name] [module-path] [flags]
  1. classify positionals; merge --from          refuse(2) on duplicate
  2. prompt for missing name / module path / namespace   refuse(2) without a terminal
  3. validate name, namespace (#NameType), --module-path, modref.ParsePath/ParseSelector   refuse(2), no I/O
  4. dir: refuse if it exists; refuse if ModuleRootFrom(parent(abs(dir))) != ""             refuse(2)
  5. modref.Resolve(src, path, sel, coreMajor)   ConnectivityError(3) | Refusal(2)
  6. k.AcquireModuleFromRegistry(res.Import(), res.Version)   identity-verified fetch
  7. src.ModFile(selected) -> core dependency version (module's own pin)
  8. instinit.PickValues(mod.Package) -> bytes, source
  9. instinit.Render(Input{...}) -> cue.mod/module.cue (module + core pins), instance.cue, values.cue
 10. instinit.Write(ctx, dir, files, cfg.Registry)
       mkdir <parent>/.opm-instance-init-<name>-<rand>
       write files; cuemod.Tidy(staging, {Registry}) adds the rest of the closure
       os.Rename(staging, dir); any error -> RemoveAll(staging)
 11. report (see Example output)
```

Step 9 writes the core pin explicitly at the module's own version, and tidy only completes the closure. That matches 0016 experiment 03, which wrote the module and core pins and tidy added one catalog pin. It also makes D9's "core at the module's major" hold regardless of which core release is newest.

### Exit codes and messages

| Condition | Exit | Message shape |
| --- | --- | --- |
| Package written | 0 | report ending `Validate it:  opm instance vet <dir>/instance.cue` |
| Duplicate module path, major suffix, bad name, namespace or selector | 2 | refusal naming the input and the accepted form |
| Name, module path or namespace missing without a terminal | 2 | `standard input is not a terminal`; action: pass `--namespace`, the module path (positional or `--from`), or the name |
| Target exists | 2 | `<dir> already exists`; action: choose another `--dir` |
| Target inside a CUE module | 2 | `<dir> is inside the CUE module at <root>`; action: initialize outside it |
| Pin not published, nothing compatible | 2 | refusal from `modref`, listing skipped majors |
| Registry unreachable (resolve, acquire) | 3 | `listing published versions of <path> (registry <registry>): <cause>` |
| Registry unreachable while tidy resolves the closure | 3 | `resolving the dependencies of <dir> (registry <registry>): <cause>`; nothing left behind |
| Any other tidy or write failure after staging | 1 | `initializing <dir>: <cause>`; nothing left behind |

### Example output

```text
$ opm instance init cert-manager opmodel.dev/modules/cert_manager -n cert-manager
Resolved opmodel.dev/modules/cert_manager -> v2 2.0.1 (highest major on core v2)
Values template: debugValues (module declares no initValues; review before deploying)
Initialized instance cert-manager/
  cue.mod/module.cue
  instance.cue
  values.cue

Validate it:  opm instance vet cert-manager/instance.cue

$ opm instance init web opmodel.dev/modules/web_app
Error: standard input is not a terminal, so the namespace cannot be asked
  Pass it:  opm instance init web opmodel.dev/modules/web_app --namespace <ns>
(exit 2)
```

## Research & Decisions

### Computing the dependency closure

**Context**: 0016 D9 requires a module file equivalent to a tidied one. The CUE SDK's tidy is internal.
**Explored**: `add-dependency-tidy` design (options: wrap `cmd/cue/cmd` in-process, fork `internal/mod`, extend `library/opm/helper/platformmodule.Closure`); the platform-module generator's `Closure`; the synth overlay.
**Options considered**:
1. `platformmodule.Closure` (MVS over published module files): no pruning, no default-major markers, and it drifts from `cue mod tidy --check`; `add-dependency-tidy` rejected it for the same reason.
2. `cuemod.Tidy` on the staged package: bit-for-bit `cue mod tidy`, already planned with `opm instance init` as a named consumer.
**Decision**: Option 2, gated on `add-dependency-tidy` section 1.
**Rationale**: One tidy implementation in the CLI; `opm module tidy <dir>` later reproduces init's module file exactly.

### Where the instance and values text comes from

**Context**: An earlier sketch read `instance.cue` and `values.cue` out of `Kernel.SynthesizeInstance`'s overlay.
**Options considered**:
1. Synth overlay bytes: no duplicated template, but synthesis fails on non-concrete values (0016 D4 allows them) and validates the package (0016 D8 says init does not).
2. CLI-side renderer (0016 D7): the instance-file shape exists in two repos; drift is caught by the e2e test that builds init's output through `opm instance build`.
**Decision**: Option 2.

### The package's own module path

**Context**: The package is a CUE main module that is never published.
**Options considered**:
1. A reserved-unpublished `opmodel.dev/instances/<name>@v0`, like `opmodel.dev/platforms/local@v0`: needs a namespace claim and a publish-gate change.
2. `instance.local/<name>@v0` with `--module-path` (0016 D9): nothing to reserve; experiment 03 showed no registry lookup of it.
**Decision**: Option 2 (user decision 2026-09-28: 0016 wins where it and the conversation differ).

### Refusing a target inside a CUE module

**Context**: 0016 D5 refuses only an existing directory. A standalone package nested in another CUE module's tree is legal CUE, but the enclosing module's tooling may walk into it.
**Decision**: Refuse (user decision 2026-09-28). The enhancement records it as 0016 D10, which amends D5.
**Rationale**: Relaxing a refusal later is non-breaking; tightening one later is breaking. Colocated instances, which need no `cue.mod` of their own, stay out of scope.

### What counts as a concrete `debugValues`

**Context**: 0016 D6 sends a `debugValues` that is "not concrete" to `values: {}` without defining the word for a partly concrete struct such as `{image: "nginx:1.27", replicas: int}`. Separately, `module-renders-against-own-deps` gives fixtures `debugValues: {}`, which is concrete but renders an empty file.
**Options considered**:
1. Render a partial `debugValues` the way `initValues` renders, sending only `_` or an absent field to empty: keeps the author's values, but narrows D6 and would need an amending decision in 0016.
2. The whole value must be concrete once defaults apply (chosen): D6 as written. `opm module build` already needs a concrete `debugValues`, so a published module rarely has a partial one.
**Decision**: Option 2 (user decision 2026-09-29): `debugValues.Validate(cue.Final(), cue.Concrete(true))` decides the rung. A source that renders as an empty struct keeps its name in the report (0016 D2:R2) and also gets the empty-file warning.

### A registry failure during tidy exits 3

**Context**: 0016 D5:R7 gives an unreachable registry exit 3, at any stage. `cuemod.Tidy` returns cmd/cue's error as it came, with no connectivity type, and `opm module tidy` maps every resolution or registry failure to exit 1.
**Options considered**:
1. Exit 1 for every tidy failure: matches `opm module tidy`, but breaks D5:R7 when the registry drops between acquire and tidy.
2. Classify a tidy error as a connectivity failure and return a `*publish.ConnectivityError` (exit 3): `errors.As` to `net.Error` first, cmd/cue's error text as the fallback.
**Decision**: Option 2 (user decision 2026-09-28). The classifier sits beside `cuemod.Tidy` so `opm module tidy` can adopt it later; this change leaves that command's exit codes as they are.
**Rationale**: A script that retries on exit 3 then treats a registry outage the same at every stage of init.

## Risks / Trade-offs

- [Assumption: `cuemod.Tidy` keeps the explicit module and core pins and only adds the rest of the closure] -> Proven in section 1 against the podinfo fixture (`TestE2E_InstanceInit_GeneratedPackageTidiesAndLoads`): both pins survive tidy unchanged and tidy adds the catalog pin; the tidied package loads through the kernel, passes a tidy check, and loads from a warm cache with the registry unreachable.
- [Assumption: tidy keeps the written language version (`v0.17.0`, the seeded platform's)] -> Proven in section 1: tidy leaves it as written.
- [Staging and rename across filesystems] -> Staging lives in the target's parent, so the rename never crosses a device.
- [`debugValues` may hold throwaway credentials or hostnames] -> The report warns to review it (0016 D2:R3). `initValues` is the author's fix, landing in section 4.
- [Assumption: a registry failure inside `cuemod.Tidy` is recognisable as one (`errors.As` to `net.Error`, else cmd/cue's error text)] -> Section 1 found (`TestE2E_InstanceInit_TidyRegistryFailureShape`, empty module cache, registry `127.0.0.1:1`): `errors.As` to `net.Error` does NOT match, because cmd/cue flattens the cause into a plain `*errors.errorString`: `failed to resolve "opmodel.dev/core@v2": cannot fetch opmodel.dev/core@v2.0.0-alpha.6: module opmodel.dev/core@v2.0.0-alpha.6: cannot do HTTP request: Get "http://127.0.0.1:1/...": dial tcp 127.0.0.1:1: connect: connection refused`. The classifier therefore keeps the `net.Error` check for a future unflattened error and matches the text `cannot do HTTP request`, the registry client's wording for a transport failure (refused, DNS, timeout), which an HTTP-status failure (404, 401) does not carry. The test pins the text against the embedded CUE version.
- [Section 4 waits on a core release] -> Sections 1 to 3 are releasable on their own. If core lags, section 4 splits into its own change rather than holding the PR; that change then claims 0016 D3 and D4 in its `enhancement.yaml`, and they are removed from this one.
- [Tidy mutates process state (working directory, `CUE_REGISTRY`)] -> Contained and restored inside `cuemod.Tidy`; init makes exactly one call.
- [0016 D1:R2 names `opm instance apply` too, and `apply --dry-run` is server-side, so it needs a cluster] -> `apply`, `vet` and `diff` load a package through the same `render.FromInstanceFile` as `build`, so the e2e `build` and `vet` of the generated package cover what `apply` loads. Only the cluster write goes untested here, and that step is the same for every instance package.

## Migration Plan

Additive command; nothing to migrate. Rollback is reverting the change. After release, the opm docs listed in proposal.md replace `cue mod init`/`cue mod tidy` with `opm instance init`.
