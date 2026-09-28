## Context

See proposal.md for motivation; specs/dependency-tidy/spec.md for the behavior contract.

Current state that shapes the approach:

- The CLI already embeds `cuelang.org/go v0.17.1` for loading, publishing and registry I/O. Its registry handling carries the resolved mapping on config structs and never calls `os.Setenv` (`internal/publish/registry.go`, `registryEnv` in `internal/publish/load.go`).
- `cue mod tidy` is about 60 lines of glue in `cmd/cue/cmd/modtidy.go` over `modload.Tidy`. That function and its dependency graph (`modimports`, `modpkgload`, `modrequirements`, `mvs`, about 7.8k lines) live under `internal/` and cannot be imported.
- `cuelang.org/go/cmd/cue/cmd` is a public package: `New(args []string) (*Command, error)` builds the whole `cue` command tree and `(*Command).Run(ctx)` executes it. `Command` embeds `*cobra.Command`, sets `SilenceErrors`, and returns a subcommand's error as a Go error instead of printing it.
- Inside `cmd/cue`, the module root is found from the process working directory (`findModuleRoot` walks up from `os.Getwd`), and the registry comes from `modconfig.Config{Env: nil}`, meaning `os.Environ()` at the moment the registry is first used.
- Exit-code conventions: `internal/exit` (1 general, 2 validation/refusal, 3 connectivity, 5 not found). Shared command bodies live in `internal/cmdutil` (`RunVersionSet`, `RunPublish`).

## Goals / Non-Goals

**Goals:**

- One in-process tidy primitive, `internal/cuemod.Tidy`, with a typed result and typed errors. Commands and the future `opm instance init` call it; nothing else touches `cmd/cue`.
- Every piece of process state the primitive changes is restored before it returns, including on error and panic.
- Hermetic unit tests against an in-memory registry; no test depends on GHCR.

**Non-Goals:**

- Reimplementing any part of tidy. The CLI adds no dependency-resolution logic of its own.
- Concurrent tidies in one process. The primitive serializes; the CLI runs one command per process.
- Exposing other `cue` subcommands (`mod get`, `mod edit`) now. The same wrapper can carry them later.

## Decisions

### Command syntax and flags

```text
opm module tidy [path] [flags]     (alias: opm mod tidy)
opm catalog tidy [path] [flags]
```

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--check` | bool | `false` | Fail with exit 2 if tidying would change any file; write nothing. |
| `--registry` | string | resolved | Existing root persistent flag; routes resolution (spec: "Tidy resolves through the CLI's registry"). |

`path` defaults to `.` via `cmdutil.ResolveModulePath`. No `--local-only`: it only matters when a replaced module is unpublished, and nobody has asked for it (Principle VII). The error text for that case already names the situation; see Risks.

### Package layout and signatures

```go
// internal/cuemod/tidy.go
package cuemod

// TidyOptions configures one tidy.
type TidyOptions struct {
	// Registry is a CUE_REGISTRY-syntax mapping. Empty inherits the
	// process environment and CUE's default.
	Registry string
	// Check resolves without writing and fails with *NotTidyError when
	// tidying would change a file.
	Check bool
}

// TidyResult says which files a tidy changed. Both false means already tidy.
type TidyResult struct {
	ModuleUpdated bool // cue.mod/module.cue rewritten
	LocalUpdated  bool // cue.mod/local-module.cue rewritten, created or removed
}

var ErrNotModuleRoot = errors.New("not a CUE module root")

// NotTidyError reports a --check failure; Reason is CUE's explanation, may be empty.
type NotTidyError struct{ Reason string }

func Tidy(ctx context.Context, dir string, opts TidyOptions) (TidyResult, error)
```

```go
// internal/cmdutil/tidy.go: shared body, the RunVersionSet pattern
func RunTidy(ctx context.Context, cfg *config.GlobalConfig, kind publish.Kind, pathArgs []string, check bool) error
```

`internal/cmd/module/tidy.go` and `internal/cmd/catalog/tidy.go` only parse flags and call `RunTidy`.

### Data flow

```text
opm module tidy [path] [--check]
  -> cmdutil.RunTidy(cfg.Registry, kind, path, check)
       -> cuemod.Tidy(ctx, dir, {Registry, Check})
            1. abs(dir); require dir/cue.mod/module.cue  -> ErrNotModuleRoot (exit 2)
            2. snapshot bytes of module.cue, local-module.cue (or absent)
            3. lock processMu
            4. save cwd; chdir(dir)                         (defer restore)
            5. if Registry != "": save CUE_REGISTRY; setenv (defer restore/unset)
            6. cuecmd.New(["mod","tidy"(,"--check")]); SetOut/SetErr(buf); Run(ctx)
            7. classify error: "module is not tidy" prefix -> *NotTidyError; else wrap
            8. re-read both files; compare with snapshot -> TidyResult
       -> print outcome line / map error to exit code
```

Step 4 uses our own `os.Chdir`, not `cmd/cue`'s `-C` flag, because `-C` calls `os.Exit(1)` when the directory is bad (`handleChdirFlag` in `root.go`), which would bypass deferred restores and the CLI's error funnel. Step 1 already guarantees the directory exists, and our own `chdir` returns an error either way.

### Error handling and exit codes

| Condition | Exit | Message shape |
| --- | --- | --- |
| Tidied, or already tidy, or check passed | 0 | outcome line |
| Path missing, not a dir, or no `cue.mod/module.cue` | 2 | `<abs path> is not a CUE module root (no cue.mod/module.cue)`; hint `opm module init` when the dir exists and the kind is module (there is no `opm catalog init`) |
| `--check` and not tidy | 2 | `module is not tidy: <CUE reason>`; hint `run 'opm module tidy'` (or `opm catalog tidy`) |
| Resolution or registry failure | 1 | CUE's resolver text, prefixed `tidying <abs path>:` |

Registry failures are not split out as exit 3: `cmd/cue` returns them as plain wrapped strings, and classifying on message fragments beyond the one stable prefix would be guesswork. CUE's own `use 'cue mod tidy'` suffix is replaced by the `opm` command; its `run 'cue mod fix'` suggestion for a missing language version passes through verbatim (see Risks).

### Example output

```text
$ opm module tidy
[x] Tidied module: updated cue.mod/module.cue

$ opm module tidy
Module already tidy; nothing written

$ opm catalog tidy --check
Error: catalog is not tidy: missing dependency providing package opmodel.dev/core@v2
Hint: run 'opm catalog tidy'
(exit 2)

$ opm module tidy ./nope
Error: /home/me/src/app/nope is not a CUE module root (no cue.mod/module.cue)
Hint: run 'opm module init' to create one
(exit 2)
```

The exact success glyph comes from `output.FormatCheckmark`, matching `version set`.

## Research & Decisions

### Getting tidy without the cue binary

**Context**: The CLI does not shell out to `cue`, and `modload.Tidy` is internal.
**Explored**: `cuelang.org/go@v0.17.1` source (`cmd/cue/cmd/modtidy.go`, `internal/mod/modload/tidy.go`, `cmd/cue/cmd/root.go`), `research/cue/sdk/mod.md`, `research/cue/cli/mod.md`, and a spike: a Go program calling `cuecmd.New([]string{"-C", dir, "mod", "tidy"}).Run(ctx)` against GHCR. `--check` failed with `missing dependency providing package opmodel.dev/core@v2`, tidy wrote `v2.0.0-alpha.10`, and `--check` then passed. A `-overlay` build of `opm` with the package linked in measured 79.1 MB to 81.4 MB.
**Options considered**:
1. Wrap `cmd/cue/cmd` in-process: proven, exact parity with `cue mod tidy --check` in CI; costs process-global state and string-typed errors.
2. Ask CUE upstream to export `modload.Tidy`: the right long-term API, but not available in any release.
3. Fork `internal/mod` into the repo (Apache-2.0): typed API, but about 7.8k lines to re-sync on every CUE bump, and v0.17 alone reworked tidy for module replaces.
4. Extend the library's `platformmodule.Closure` into a tidy: it already does MVS over published module files, but pruning needs a package-level import graph (`@ignore`, default majors, replaces); a home-grown version would drift from `cue mod tidy --check`.
**Decision**: Option 1, behind `internal/cuemod` so a later swap to option 2 touches one file.
**Rationale**: Only option 1 is available today and gives bit-for-bit parity by construction.

### Where the primitive lives

**Context**: The workspace rule is that frontends rely on the library for structure. But the library constitution forbids process-model behavior, and this primitive mutates the process working directory and environment.
**Options considered**:
1. `library/opm/helper`: shareable with the operator, but brings process mutation into the library.
2. `cli/internal/cuemod`: CLI-owned authoring tooling.
**Decision**: Option 2.
**Rationale**: Tidy is authoring tooling, not kernel semantics (load, validate, match, execute), and the operator never edits source trees. If upstream exports tidy (option 2 above), a library helper becomes cheap and this decision can be revisited.

### Process state: working directory and CUE_REGISTRY

**Context**: `cmd/cue` reads both from the process. The CLI has so far kept the registry on config structs.
**Options considered**:
1. Require users to export `CUE_REGISTRY` themselves: no mutation, but `--registry` and `OPM_REGISTRY` would silently not apply to tidy.
2. Set and restore both inside `Tidy`, under a package mutex, with deferred restoration.
**Decision**: Option 2. When `opts.Registry` is empty, the environment is left untouched.
**Rationale**: The spec requires tidy to honor the CLI's registry precedence. Keeping the mutation inside one function, under one lock, with deferred restores makes it the single audited exception to the no-`os.Setenv` convention; a code comment at the call says so.

### Classifying "not tidy"

**Context**: `modload.ErrModuleNotTidy` is internal, so `errors.As` cannot reach it. `cmd/cue`'s `suggestModCommand` flattens it to `module is not tidy, use 'cue mod tidy'[: <reason>]`.
**Options considered**:
1. Match the stable `module is not tidy` prefix and extract the reason after `: `.
2. Compare file snapshots after a non-check run: cannot work for `--check`, which must not write.
**Decision**: Option 1, pinned by a unit test that fails if a CUE upgrade changes the wording.

### Explicit root, no walk-up

**Context**: `cmd/cue` walks up to the nearest `cue.mod`; `opm module publish` and `version set` take an explicit root.
**Decision**: Require `path` itself to be the root; `Tidy` changes into it, so `cmd/cue`'s walk-up finds it first.
**Rationale**: Explicit over inferred (Principle VII), consistent with the other authoring commands. Walking up from an instance file to its enclosing module is `opm instance init`'s job and can add a `FindModuleRoot` helper then.

### Publish gate

**Context**: `opm module publish` does not reproduce `cue mod publish`'s tidiness check (recorded in `cli-publish-pipeline`).
**Decision**: Not in this change. `cuemod.Tidy(..., Check: true)` makes it a small follow-up, but it adds a registry round-trip and refuses artifacts that publish today, so it deserves its own proposal.

## Risks / Trade-offs

- [`cmd/cue/cmd` is public but not a designed API; upstream has a TODO to reshape it] -> All calls go through `internal/cuemod`; its tests exercise add, prune, check pass/fail and error wording, so a CUE bump that breaks the contract fails CI in one place.
- [Process-global `chdir` and `setenv` break under concurrent callers] -> Package mutex; deferred restore; a test asserts cwd and `CUE_REGISTRY` match their pre-call values after success and after failure. Tests calling `Tidy` must not use `t.Parallel`.
- [Binary grows by 2.3 MB and pulls `cmd/cue`'s graph (LSP, websocket) into `go.mod`] -> Measured and accepted. `task lint` runs with readonly module mode, so `go.mod`/`go.sum` land in the same commit as the import.
- [A replaced module that is published nowhere makes tidy fail, with CUE's hint naming `cue mod tidy --local-only`] -> Accepted for now; add `--local-only` when someone hits it. The hint text is passed through so the user can still act.
- [Tidy reaches the network and the user's CUE module cache (`CUE_CACHE_DIR`)] -> The same cache every other `opm` read uses; tests set `CUE_CACHE_DIR` to a temp dir.
- [Unverified until section 1: `New` can be called repeatedly in one process; nothing is printed on success; the registry is read from the environment inside `Run` and not at `New`] -> Section 1 is a spike that proves each in a unit test before any command exists.

## Migration Plan

Additive commands; nothing to migrate. Rollback is reverting the change. `TODO.md` line 6 is closed when the commands land.
