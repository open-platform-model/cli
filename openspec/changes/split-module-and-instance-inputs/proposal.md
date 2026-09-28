## Why

`opm module build` and `opm module apply` only take a module directory on disk, so rendering or deploying a published module means cloning its source first. The operator already deploys straight from registry coordinates (`ModuleInstance.spec.module.{path,version}`); the CLI cannot. Separately, `opm instance build` picks its render path from `os.Stat`: any directory is treated as a module and synthesized. That collides with instance packages, which are directories too. An instance directory inside a module tree, or a standalone instance package, is refused or misread today ("is an instance package, not a module"). The planned `opm instance init` writes exactly such packages, so the collision has to go first.

## What Changes

- `opm module build` and `opm module apply` accept a published module path (major-free, for example `opmodel.dev/modules/web_app`) in place of a directory, with a new `--version` flag: `vN` takes the newest release within major N, an exact SemVer pins that tag, and no `--version` takes the newest release of the highest major whose core dependency major matches the CLI's core major. The resolved version, and every higher major skipped with its reason, is printed before the render.
- The module is acquired from the registry through the kernel and synthesized exactly as a local module is; values still come from `-f` or the module's `debugValues`. An apply from a published module carries no local-render provenance.
- A new shared resolver (`internal/modref`) owns the path grammar and the version selection, so `opm instance init` (planned change `add-instance-init`, enhancement 0016) reuses it instead of growing a second one.
- **BREAKING**: `opm instance build` no longer synthesizes from a module directory. Its argument is an instance file or an instance package directory, and the command decides by what the package is (`#ModuleInstance`), not by whether the path is a directory. A module package is refused with a hint to `opm module build`.
- **BREAKING**: `opm instance build` drops `--name`; it only named synthetic instances, which now come from `opm module build` alone.

Release class: MAJOR under Principle VI, because the last section removes behavior; its commit carries `!`. The CLI is on the 1.0.0 prerelease line (`1.0.0-alpha.21`).

This change does not depend on `add-dependency-tidy`: nothing here writes or tidies a `cue.mod`.

## Capabilities

### New Capabilities

- `published-module-resolution`: the published-module path grammar, the `--version` selector, the core-compatible major walk, and the resolution report shared by every command that takes a published module.

### Modified Capabilities

- `cmd-structure`: `opm module build` and `opm module apply` accept a published module path and `--version`; `opm instance build` accepts only instance packages and decides by package kind; `--name` leaves `opm instance build`.
- `module-synthetic-instance`: synthesis sources a module from a directory or from the registry; `opm instance build <dir>` no longer synthesizes.
- `instance-building`: the kernel entry points gain the published-module path and lose `opm instance build <dir>` as a synthesis caller.

## Impact

- Commands: `internal/cmd/module/build.go`, `internal/cmd/module/apply.go` (argument classification, `--version`), `internal/cmd/instance/build.go` (single instance path, `--name` removed).
- Packages: new `internal/modref` (path grammar, selector, version selection over the registry's published versions, core-major probe of each candidate's module file); `internal/workflow/render/module.go` (`FromModule` gains a registry-acquire branch); `internal/cmdutil/path_guard.go` (instance-package guard reworded for the new instance build).
- Library: none. `Kernel.AcquireModuleFromRegistry` and `Kernel.SynthesizeInstance` are used as they are; the operator already combines them this way.
- Users: scripts calling `opm instance build <module-dir>` switch to `opm module build <module-dir>` (same flags, same output). Scripts passing `--name` to `opm instance build` drop it or move to `opm module build`.
- Docs: `README.md` command examples; the docs site's CLI reference regenerates from the cobra tree.
