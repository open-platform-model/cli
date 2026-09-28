## Why

An OPM module or catalog is a CUE module, and today the only way to resolve, pin and prune its `cue.mod/module.cue` dependencies is the separate `cue` binary (`cue mod tidy`). The CLI has refused shell-outs to `cue` since the start (`TODO.md`), and the tidy engine in the CUE SDK (`internal/mod/modload`) cannot be imported. Research on `cuelang.org/go` v0.17.1 found a third route: the public `cuelang.org/go/cmd/cue/cmd` package runs `cue mod tidy` in-process. A spike tidied a module against GHCR through it, and it adds 2.3 MB to the `opm` binary. `opm` can therefore own the whole authoring loop without a second binary.

## What Changes

- New command `opm module tidy [path] [--check]` (alias `opm mod tidy`): tidies the module's `cue.mod/module.cue` (and `cue.mod/local-module.cue`) the way `cue mod tidy` does. It adds missing dependencies at their latest version, removes unused ones and applies minimum version selection.
- New command `opm catalog tidy [path] [--check]`: the same operation against a catalog module.
- `--check` fails without writing when the module is not tidy (exit 2), as a CI gate matching `cue mod tidy --check`.
- Registry routing follows the CLI's existing precedence (`--registry` > `OPM_REGISTRY` > config `registry`), so tidy resolves from the same registry every other `opm` command reads.
- A reusable in-process tidy primitive in a new `internal/cuemod` package. The command layer is only glue. The planned `opm instance init` will call the same primitive to generate an instance's `cue.mod`.
- New direct dependency on `cuelang.org/go/cmd/cue/cmd` (same module and version as the existing `cuelang.org/go` requirement; transitive additions only).

This is a MINOR release (new commands, no changed behavior).

Delivered later, by other changes: instance tidy belongs to the planned `opm instance init`, which creates the enclosing `cue.mod` in the first place. A `--check` gate inside `opm module publish` is a separate decision, because it adds a registry round-trip to publish and would refuse artifacts that publish today.

## Capabilities

### New Capabilities

- `dependency-tidy`: `opm module tidy` and `opm catalog tidy`. What they write, `--check` semantics, registry routing, outcome reporting and exit codes.

### Modified Capabilities

- `platform-resolution`: the local default platform's maintenance loop names `opm module tidy` for pinning what a bumped catalog build needs; no behavior of platform resolution changes.

## Impact

- Commands: `internal/cmd/module` (new `tidy.go`, registered in `mod.go`), `internal/cmd/catalog` (new `tidy.go`, registered in `catalog.go`).
- Packages: new `internal/cuemod` (tidy primitive), new `internal/cmdutil/tidy.go` (shared command body, the `RunVersionSet` pattern).
- Dependencies: `go.mod` gains the `cmd/cue/cmd` import; indirect additions come from that package's own graph (for example `github.com/coder/websocket` via the LSP server). The binary grows by about 2.3 MB (79.1 MB to 81.4 MB, measured).
- Process state: the primitive changes the working directory and `CUE_REGISTRY` for the duration of one tidy call and restores both. This is the first `os.Setenv` in the CLI and is confined to that one function (see design.md).
- Docs: `README.md` command list and `AGENTS.md` package map; `TODO.md` line 6 item closed; `docs/roadmap.md` records tidy as delivered; `opm config init` help and the seeded platform `module.cue` comment name `opm module tidy` in the pin-bump loop. The docs site's CLI reference is generated from the cobra tree, so it picks up the commands automatically.
