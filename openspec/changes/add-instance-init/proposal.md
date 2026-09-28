## Why

Deploying a published module needs a standalone instance package: `cue.mod/module.cue` pinning the module, core and the catalogs, an `instance.cue` wiring `#ModuleInstance`, and a `values.cue`. Nothing in `opm` writes one, so every user reaches for the `cue` binary (`cue mod init`, `cue mod tidy`, plus a hand-exported `CUE_REGISTRY`), or hand-writes pins that go stale with the next release (cli issue 230). Enhancement 0016 fixes the design: a new `opm instance init` generates the package from the published module, correct by construction.

## What Changes

- New command `opm instance init [instance-name] [module-path] [--from <module-path>] [--version <vN | X.Y.Z>] --namespace <ns> [--dir <dir>] [--module-path <path>]`, mirroring `opm module init` (0016 D5). The module path is major-free; with no `--version` the newest release of the highest major on the CLI's core major is chosen.
- It writes a standalone three-file package, `cue.mod/module.cue`, `instance.cue` and `values.cue`, all or nothing (0016 D1, D5). The module file pins the module exactly and core at the module's own core version, and carries the tidied dependency closure, so the first `opm instance build` needs no other step (0016 D9). The package's own module path defaults to `instance.local/<instance-name>@v0`.
- `values.cue` starts from the module's `initValues`, else its `debugValues`, else `values: {}` with a warning. The report names the source (0016 D2, D3, D6). Init does not validate the package; its report names `opm instance vet <dir>/instance.cue` (0016 D8).
- Init refuses a target directory that already exists (0016 D5), and also one that sits inside an existing CUE module, so a standalone package is never nested in another module's tree.
- The renderer is CLI-side, beside `opm module init` (0016 D7).

Release class: MINOR (a new command, no changed behavior).

**Gates** (details in tasks.md):

- `add-dependency-tidy`, section 1 (`internal/cuemod.Tidy`) is on main before section 1 of this change. The closure is computed by the in-process `cue mod tidy`, never by a home-grown resolver.
- `split-module-and-instance-inputs`, section 1 (`internal/modref`, the shared path grammar and version resolver) is on main before section 1.
- The core change that adds `initValues` to `#Module` (0016 D3, D4; planned as its own core OpenSpec change) is released before section 4. Sections 1 to 3 ship the `debugValues` and empty rungs without it.

Out of scope: moving an existing instance to a newer module version (an `upgrade` or instance-level tidy command), and colocated instances inside a module tree.

## Capabilities

### New Capabilities

- `instance-init`: `opm instance init`. Its arguments and prompting, version selection, the generated package and its module file, the values-source ladder, refusals, all-or-nothing writing, the report and exit codes.

### Modified Capabilities

(none)

## Impact

- Commands: new `internal/cmd/instance/init.go`, registered in the instance group.
- Packages: new `internal/instinit`, which renders the three files from typed input and picks the values source. It calls `internal/cuemod.Tidy` (from `add-dependency-tidy`) and `internal/modref` (from `split-module-and-instance-inputs`). The prompt and terminal helpers of `internal/cmd/module/init.go` move to `internal/cmdutil` so both init commands share them.
- Library: none (0016 D7). `AcquireModuleFromRegistry` is used as it is.
- Docs: `README.md`, and the `AGENTS.md` package map. Outside this repo, after release: the opm quickstart, `deploy-with-the-cli` and the glossary's "CUE module" entry drop `cue mod init`/`cue mod tidy` for `opm instance init`, as issue 230 lists. The docs site's CLI reference regenerates from the cobra tree.
