## Why

`opm module build` renders a module against `~/.opm/platform/` (or `--platform`), whose catalog pins are not the ones the module was written against. A first build can therefore print version skew for a module that is internally consistent, and it fails outright on a machine that never ran `opm config init`. The author's question is "does my module render with the catalogs it declares"; the deployer's question is "does it render on my platform". Today one command answers the second question for both (cli issue 229).

## What Changes

- The platform line is drawn between module and instance commands. An author's render uses the module's own deps; a render for a deployment uses a platform.
- `opm module build` and `opm module vet` render, by default, against a platform module the CLI generates from the module's committed `cue.mod/module.cue`: one registry entry per `opmodel.dev/catalogs/*` dependency, pinned at the version the module pins, with the dependency closure derived from the published module files. They read no cluster and no `~/.opm/platform/`.
- `--platform <dir>` stays on `opm module build` and `opm module vet` as the author's explicit switch to a platform (for example one pulled with `opm platform pull`). It wins over the deps platform.
- The module's own `cue.mod/local-module.cue` replacements of paths the generated platform pins are carried into that platform, so a module redirected to a local catalog or core checkout renders the checkout. Today the local default platform names those paths and the redirect is reported as ignored.
- `opm module vet` renders (synthesis, matching, transformers) after its identity and `#config` checks and reports the rendered objects without printing manifests, so `module vet` and `module build` reach the same verdict.
- The deps are read from the module's acquired source, so the same rule covers a module acquired from a directory and a published module acquired from the registry (`opm module build <published-path>`, added by `split-module-and-instance-inputs`).
- Unchanged: `opm instance build` and `opm instance vet` (`--platform`, else `~/.opm/platform/`), and `opm instance apply`, `opm instance diff` and `opm module apply` (`--platform`, else the cluster Platform, else the local default).

Release class: MINOR. No flag or command is removed. `opm module build` output changes only in its provenance line and the absence of skew warnings; `opm module vet` newly refuses a module whose components do not render against its own catalogs, which is a stricter check of the same command, not a removed interface. For the same reason it newly refuses what `opm module build` already refused at synthesis: a `debugValues` left open (`_`); the release note names the fix, `debugValues: {}`. The default synthetic instance name hyphenates the module name (`my_app` becomes `my-app-debug`): the old `my_app-debug` failed the instance-name pattern, so `module build` and `module apply` refused every multi-word module without `--name`, and `module vet` would have too.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `platform-resolution`: the precedence gains the module-deps source for `module build` and `module vet`; a new requirement defines how the deps platform is generated (catalog selection, closure, carried replacements, provenance, no skew).
- `mod-vet`: `module vet` renders against the deps platform (or `--platform`) instead of stopping at `#config` validation; `--platform`, `-n` and `--instance-name` now affect it.
- `kernel-render`: a module-side replacement of a path the deps platform pins is carried and honoured, no longer reported as ignored.
- `module-synthetic-instance`: the default synthetic instance name hyphenates the module name, so it is a valid instance name for every valid module name.

## Impact

- Commands: `internal/cmd/module/build.go`, `internal/cmd/module/vet.go` (long help, wiring). `module apply` and every `instance` command are untouched.
- Packages: `internal/platform` (new `SourceModuleDeps`, deps-platform generation beside `GenerateClusterModule`, `Resolve` option), `internal/workflow/render` (`ModuleOpts`, `FromModule`, skew-policy note, replacement warnings, refusal hints).
- Library: none. Uses the existing `platformmodule.Closure` and `platformmodule.Generate` and the module's `Source` overlay.
- Cache: generated modules land under `~/.opm/cache/platforms/<hash>/`, as the cluster CR module does.
- Related changes: `split-module-and-instance-inputs` edits the same `FromModule` and `module build` files; neither change gates the other, whichever lands second rebases. `add-dependency-tidy` (landed) provides `opm module tidy`, which the refusal for a missing catalog pin names.
- Docs in this repo: `QUICKSTART.md`, `AGENTS.md` (render path note). Pages in `opm`, `core`, `library` and `catalog_opm` listed in cli issue 229 follow in their own repos.
