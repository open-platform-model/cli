## Why

Deploying a published module needs a standalone instance package: `cue.mod/module.cue` pinning the module, core and the catalogs, an `instance.cue` wiring `#ModuleInstance`, and a `values.cue`. Nothing in `opm` writes one, so every user reaches for the `cue` binary (`cue mod init`, `cue mod tidy`, plus a hand-exported `CUE_REGISTRY`), or hand-writes pins that go stale with the next release (cli issue 230). Enhancement 0016 fixes the design: a new `opm instance init` generates the package from the published module, correct by construction.

## What Changes

- New command `opm instance init [instance-name] [module-path] [--from <module-path>] [--version <vN | X.Y.Z>] --namespace <ns> [--dir <dir>] [--module-path <path>]`, mirroring `opm module init` (enhancement 0016, "`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given"). The module path is major-free; with no `--version` the newest release of the highest major on the CLI's core major is chosen.
- It writes a standalone three-file package, `cue.mod/module.cue`, `instance.cue` and `values.cue`, all or nothing (enhancement 0016, "A new CLI command scaffolds an on-disk instance package from a published module reference" and "`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given"). The module file pins the module exactly and core at the module's own core version, and carries the tidied dependency closure, so the first `opm instance build` needs no other step (enhancement 0016, "The generated module file pins the module exactly, carries core at the module's major, and names a local placeholder module path"). The package's own module path defaults to `instance.local/<instance-name>@v0`.
- `values.cue` starts from the module's `initValues`, else its `debugValues`, else `values: {}` with a warning. The report names the source (enhancement 0016, "`debugValues` is the default template source for the generated `values.cue`", "`#Module` gains an optional field carrying the author-intended init template, taking precedence over `debugValues`" and "With no usable template source, init writes `values: {}` and warns"). Init does not validate the package; its report names `opm instance vet <dir>/instance.cue` (enhancement 0016, "Init does not validate the generated package; it tells the user how to").
- Init refuses a target directory that already exists (enhancement 0016, "`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given"), and also one that sits inside an existing CUE module (enhancement 0016, "Init refuses a target directory inside an existing CUE module"), so a standalone package is never nested in another module's tree.
- A registry that cannot be reached exits 3 at every stage, including while tidy resolves the dependency closure (enhancement 0016, "`opm instance init` mirrors `opm module init`, takes a major-free module path, and floats to the highest core-compatible major when no version is given").
- The renderer is CLI-side, beside `opm module init` (enhancement 0016, "The package renderer lives in the CLI; library ships nothing for this entry").

Release class: MINOR (a new command, no changed behavior).

**Gates** (details in tasks.md):

- `add-dependency-tidy`, section 1 (`internal/cuemod.Tidy`) is on main before section 1 of this change. The closure is computed by the in-process `cue mod tidy`, never by a home-grown resolver.
- `split-module-and-instance-inputs`, section 1 (`internal/modref`, the shared path grammar and version resolver) is on main before section 1.
- The core change that adds `initValues` to `#Module` (enhancement 0016 ("`#Module` gains an optional field carrying the author-intended init template, taking precedence over `debugValues`" and "The field is `initValues?: _`, open and optional, with no schema-side `#config` assertion"); planned as its own core OpenSpec change) is released before section 4. Sections 1 to 3 ship the `debugValues` and empty rungs without it.

Out of scope: moving an existing instance to a newer module version (an `upgrade` or instance-level tidy command), and colocated instances inside a module tree.

## Capabilities

### New Capabilities

- `instance-init`: `opm instance init`. Its arguments and prompting, version selection, the generated package and its module file, the values-source ladder, refusals, all-or-nothing writing, the report and exit codes.

### Modified Capabilities

(none)

## Impact

- Commands: new `internal/cmd/instance/init.go`, registered in the instance group.
- Packages: new `internal/instinit`, which renders the three files from typed input and picks the values source. It calls `internal/cuemod.Tidy` (from `add-dependency-tidy`) and `internal/modref` (from `split-module-and-instance-inputs`). `internal/cuemod` gains a classifier that recognises a registry failure in tidy's error; `opm module tidy` does not use it yet. The prompt and terminal helpers of `internal/cmd/module/init.go` move to `internal/cmdutil` so both init commands share them.
- Library: none (enhancement 0016, "The package renderer lives in the CLI; library ships nothing for this entry"). `AcquireModuleFromRegistry` is used as it is.
- Docs: `README.md`, and the `AGENTS.md` package map. Outside this repo, after release: the opm quickstart, `deploy-with-the-cli` and the glossary's "CUE module" entry drop `cue mod init`/`cue mod tidy` for `opm instance init`, as issue 230 lists. The docs site's CLI reference regenerates from the cobra tree.
