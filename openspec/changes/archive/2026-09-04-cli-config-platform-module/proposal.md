## Why

Enhancement 0019 D5 makes a `#Platform` carry its catalogs by import: a registry entry becomes `{enable, #catalog}` with `version` derived from the imported bytes, and the platform's `cue.mod` is the resolution. The CLI's local default platform is the opposite shape today: `~/.opm/platform.cue` is a data-only file (no module, no imports, hand-pinned `version!` scalars) that the render path feeds through `SynthesizePlatform`, which the library wave deletes. `opm config init` must scaffold a real platform CUE module, and platform maintenance must move from "edit a version scalar in platform.cue" to "bump a dependency in the platform module's `cue.mod`".

This change is `cli-config-platform-module`, the CLI's config-surface slice of the 0019 Phase B wave. It covers `opm config init`, `opm config vet`, the local platform module contract, and the maintenance story. It does not rewire the render path.

## What Changes

- **BREAKING (user-facing config format)** `opm config init` stops writing the data-only `~/.opm/platform.cue` and writes `~/.opm/platform/` instead: a real CUE module (`cue.mod/module.cue` pinning core and both first-party catalogs; `platform.cue` embedding `core.#Platform`, importing both catalogs, and declaring one `#registry` entry per catalog with `#catalog: <import>`). The module path is `opmodel.dev/platforms/local@v0` in the reserved-unpublished namespace (0019 D6). `config.cue` is unchanged.
- Init stays normatively offline: it writes pins without resolving anything, exactly as today. A stale legacy `~/.opm/platform.cue` is removed when the module is written, with a printed note.
- `opm config vet`'s platform check stops validating the data-only projection schema and instead builds the platform module through the kernel's shape-gated loader: imports resolve (CUE module cache; network on a cold cache), the D5 key-binding and derived-version tripwires evaluate, and a bad pin or key/import drift fails vet naming the entry. A missing `~/.opm/platform/` stays non-fatal (noted). A leftover legacy `platform.cue` fails vet with a re-init hint (same pattern as the stale-`providers:` check).
- Maintenance contract: catalog pins live in `~/.opm/platform/cue.mod/module.cue` and are bumped by editing that file (or `cue mod get`), then verified with `opm config vet`. The template's comments document exactly this loop. No new bump command (Principle VII; revisit if the loop proves painful).
- The dev-cluster mirror `hack/platform.cue` becomes the module-form `hack/platform/`; the four-file pin-mirror rule in `CLAUDE.md` is updated (pins now live in the two platform modules' `cue.mod` files plus `hack/kind-platform.yaml`; the operator's sample Platform stays a CR and is out of this repo).

Out of scope, in the same wave as siblings: rewiring resolution and render to the new library surface (`cli-render-switch`: `--platform` becomes a module directory, `Resolve` returns a source-carrying platform, `Kernel.Render` + the skew flag, retirement of the `SynthesizePlatform`/`Materialize` mirror requirement) and generating a platform module from the cluster Platform CR (`cli-platform-cr-generation`, the CLI-side twin of the operator's 0019 D6 work).

## Sequencing (release train, not independent landings)

The seeded module only evaluates against a published core prerelease carrying D5 (`core-registry-import`) and the catalogs resolvable beside it. Between this change and `cli-render-switch` the CLI's render path cannot consume what init writes (the old pipeline reads `version!` subscriptions that no longer exist in the new shape), so the two changes merge in one release train with the library cutover re-pin; this change lands first only in branch order, never in a release alone.

## SemVer classification

MAJOR by behavior (the local platform config format changes shape and location; old files fail vet with a migration hint). Released under the repo's conventional-commit rules as part of the 0019 wave; the template pins remain shipped content (`fix(deps)` on later bumps).

## Affected commands and packages

- Commands: `opm config init`, `opm config vet`.
- Packages: `internal/config` (templates, paths: `PlatformFile` → platform module dir, platform loader/validator), `internal/cmd/config` (init, vet), `hack/` (mirror), docs (`README`/command help text).
- `internal/platform` (resolution) and `internal/workflow` are NOT touched here; they move in `cli-render-switch`.

## Complexity justification

No new abstractions: the module template is embedded strings exactly like today's templates; vet reuses the kernel loader the CLI already embeds. The one removed concept (the data-only projection schema and its import ban for the platform file) outweighs the added module scaffolding.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `config-commands`: init writes the platform module instead of the data file (and removes the legacy file); vet builds the platform module instead of checking the projection schema; migration scenarios for the legacy file.
- `platform-resolution`: the "local platform file is a data-only CR-spec projection" requirement is replaced by the local-default-platform-module contract (shape, module path, pins-in-cue.mod). The precedence, materialization-mirror and CR-related requirements are untouched here (they move in `cli-render-switch` / `cli-platform-cr-generation`).

## Impact

- `internal/config/templates.go` (platform module templates: `module.cue` + `platform.cue`), `internal/config/paths.go`, `internal/config/platform.go` (module build via kernel loader replaces projection-schema validation), `internal/cmd/config/init.go`, `internal/cmd/config/vet.go`, their tests, `internal/config/schema/platform.cue` (projection schema retired or reduced), `hack/platform.cue` → `hack/platform/`, `CLAUDE.md` mirror note.
- `enhancement.yaml` declares 0019 D5 (consumer side).
