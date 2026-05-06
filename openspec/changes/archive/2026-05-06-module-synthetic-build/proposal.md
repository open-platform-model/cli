## Why

Module authors currently have no way to render their module to manifests without first authoring a `release.cue`. `opm release build` requires a release file; `opm module vet` only validates `#config`. Authors fall back to hand-written test releases just to see what their module produces. This change lets them point at a module directory and get rendered output, using the module's own `debugValues` (or `-f` overrides), so the inner-loop matches `cue eval` ergonomics and works for unpublished local modules.

## What Changes

- Add module-mode to `opm release build`: when the positional arg is a directory, synthesize a `#ModuleRelease` around the loaded module package and render it through the existing pipeline. File arg behaviour (today's `release.cue` path) is unchanged.
- Add `opm module build [path]` (alias `opm mod build`) accepting a module directory only. File paths produce a clear error directing users to `opm release build <file>`.
- Module mode loads the module as a whole CUE package via `load.Instances(["."], Dir: path)` — no single-file `module.cue` shortcut. Mirrors `cue eval` / `cue vet` semantics.
- Synthesize `#ModuleRelease` via a small synthetic CUE module that imports `opmodel.dev/core/v1alpha1@v1` from the registry, pinned at the same catalog version the user's module already declares (parsed from the user's `cue.mod/module.cue` via `mod/modfile`). The synth files are served via `load.Config.Overlay` against a temp anchor; no filesystem writes inside the user's module dir.
- Values selection mirrors `opm module vet`: `-f`/`--values` files merged in order, otherwise the module's `debugValues`. Fail with a hint if neither is present.
- Synthetic metadata defaults: `metadata.name` = `<module.metadata.name>-debug`, `metadata.namespace` = `default`. Both overridable via existing `--namespace` flag and a new `--name` flag.
- Require the user's module to declare `opmodel.dev/core/v1alpha1@v1` as a dep (every realistic OPM module does); fail with an actionable hint otherwise.
- Bundle releases remain out of scope (consistent with `opm release build` today).

## Capabilities

### New Capabilities

- `module-synthetic-release`: synthesizing a concrete `*ModuleRelease` from a bare module package (no `release.cue`), using a registry-resolved synthetic CUE module pinned at the user-module's catalog version and values from `-f` or `debugValues`.

### Modified Capabilities

- `cmd-structure`: `internal/cmd/module/` (the `mod` group) gains a `build` subcommand; the existing scenario forbidding `build` here is replaced.
- `release-building`: `pkg/loader` gains a third entry path (synthesize from module-package directory) feeding the same `LoadModuleReleaseFromValue` pipeline used by release files and release packages.
- `build`: align the existing `opm mod build` spec language with the synthetic-release semantics (the command renders a synthesized release, not an ad-hoc Pipeline call).
- `loader-api`: add a new public function for synthesising a release `cue.Value` from a module-package directory + values, parallel to `LoadReleasePackage`.

## Impact

- **Code**:
  - `internal/cmd/release/build.go` — branch on dir vs file; forward dir path to the new workflow function.
  - `internal/cmd/module/` — add `build.go`; register it in `mod.go`.
  - `internal/workflow/render/` — add `FromModule` (sibling of `FromReleaseFile`).
  - `pkg/loader/` — new `SynthesizeModuleReleaseFromPackage` producing a `cue.Value` shaped like a `#ModuleRelease` for downstream `LoadModuleReleaseFromValue`. Reads the user's modfile via `mod/modfile.Parse` to copy the catalog version pin into the synth wrapper's modfile.
- **Flags**: new `--name` flag on `opm release build` and `opm module build`. Existing `--namespace`, `-f/--values`, `-o`, `--split`, `--out-dir`, `--provider` plumbing reused unchanged.
- **Docs**: CLI reference (auto-generated in `opmodel.dev/`) regenerates; QUICKSTART.md gets a note on the module inner-loop.
- **SemVer**: MINOR — adds a subcommand and broadens `opm release build` arg semantics without breaking existing release-file invocations.
- **Out of scope**: bundle synthesis, watch mode, diffing synthetic output against a deployed release.
