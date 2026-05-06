## Context

`opm release build <release.cue>` renders a `#ModuleRelease` to manifests. `opm module vet` validates a module's `#config` against `debugValues` or `-f` files but never renders. Module authors who want to see what their module produces have to author a release.cue first — even if their module is unpublished and only exists locally.

The render pipeline already supports two entry points (`internal/releasefile/GetReleaseFile` for standalone files; `pkg/loader/LoadReleasePackage` for release-package directories), both feeding into `pkg/loader.LoadModuleReleaseFromValue`. The missing entry point is "module package directory, no release.cue" — which has to synthesize a `#ModuleRelease` wrapper before reusing the existing pipeline.

The user's module typically does not declare `opmodel.dev/core/v1alpha1/modulerelease@v1` as a dependency — modules don't import their deployer schema. So we cannot just unify the loaded module with `#ModuleRelease` from the user's CUE module context. We bring `#ModuleRelease` into scope ourselves through a small synthetic CUE module whose only dep is the catalog at the same version the user's module already pins.

The catalog (`opmodel.dev/core/v1alpha1@v1`) is published to the OCI registry the project already uses, so the synth wrapper can import it like any other registry dep. CUE's loader resolves it via `CUE_REGISTRY` and caches in `~/.cache/cuelang/mod` — same machinery that already resolves the user module's own deps. No embed, no vendor tree, no offline-only path.

Two `load.Instances` calls — one for the user's module on disk, one for the synthetic wrapper in a temp anchor — share a single `*cue.Context`, and `Value.Unify`/`FillPath` compose them in Go. The synth wrapper's `cue.mod/module.cue` pins the catalog at the **same** version the user's module pins (parsed from the user's `cue.mod/module.cue` via `mod/modfile`), guaranteeing both loads see the same `#Module` definition.

## Goals / Non-Goals

**Goals:**

- Authors can run `opm release build ./my-module` (directory) or `opm module build [./my-module]` and get rendered manifests using `debugValues` or `-f` overrides.
- Module mode loads the **whole CUE package** (not a single file), matching `cue eval` / `cue vet` semantics.
- No filesystem writes inside the user's module dir; no synthetic-release files left behind.
- No registry round-trip for the synthesis wrapper itself (the user's module's own deps still resolve normally).
- Reuse `pkg/loader.LoadModuleReleaseFromValue` and the existing render pipeline unchanged.
- Use the catalog version the user's module already pins, so synth wrapper and user module load the same `#Module` definition by construction (no drift detection needed).

**Non-Goals:**

- Bundle releases (`#BundleRelease`) — `opm release build` does not support these today and this change does not change that.
- Watch/auto-rerun mode.
- Diffing synthetic output against a deployed release.
- Single-file `module.cue` arg for module mode — explicitly rejected; CUE packages span directories.
- Generating a real `release.cue` on disk as a side-effect.

## Decisions

### D1. Synthesize `#ModuleRelease` via a registry-resolved synthetic CUE module

The catalog is already published. The synth wrapper is a tiny CUE module declaring one dep on `opmodel.dev/core/v1alpha1@v1` and importing `modulerelease` from it. CUE's loader fetches it through the same `CUE_REGISTRY`/`~/.cache/cuelang/mod` machinery that already serves the user's module's own deps.

The wrapper lives in a temp anchor dir. Both files (`cue.mod/module.cue` and `wrapper.cue`) are served via `load.Config.Overlay` with `load.FromString` — no real file writes required. CUE's loader requires `Dir` to exist on disk (`research/cue/sdk/load.md` L143), but the contents can be entirely virtual.

```
<anchor>/                                          (real, os.MkdirTemp; defer RemoveAll)
<anchor>/cue.mod/module.cue                        (overlay)
    module: "opm.local/synth"
    language: { version: "v0.16.0" }
    source: { kind: "self" }
    deps: {
        "opmodel.dev/core/v1alpha1@v1": { v: "<copied from user module>" }
    }
<anchor>/wrapper.cue                               (overlay)
    package synth
    import mr "opmodel.dev/core/v1alpha1/modulerelease@v1"
    mr.#ModuleRelease
```

The wrapper applies `#ModuleRelease` at the top level — the same shape every real `release.cue` in `releases/<env>/<module>/` uses (verified against `releases/kind_opm_dev/jellyfin/release.cue`, `releases/kind_opm_dev/garage/release.cue`, `releases/mr_spel/wolf/release.cue`). The only structural difference is that real release files also import the module via CUE (e.g. `m "opmodel.dev/modules/jellyfin@v1"`) and write `#module: m`. The synth wrapper omits that import — composition is done in Go via `FillPath` against the loaded user-module value. This avoids requiring the user's module to be published to the registry.

The pinned catalog version is **copied** from the user's `cue.mod/module.cue` via `mod/modfile.Parse`. `mf.Deps["opmodel.dev/core/v1alpha1@v1"]` returns the user's pin (e.g. `v1.3.9`); the synth modfile reuses the same string. By construction, both `load.Instances` calls in D2 resolve the catalog to the same version, so the `#Module` definition the user satisfies is the same `#Module` definition the synth wrapper expects.

If the user's module pins no catalog dep at all, fail with: *"module must declare `opmodel.dev/core/v1alpha1@v1` as a dependency to be buildable"*. Realistic OPM modules always do — `#Module` lives there.

**Alternatives considered:**

- *Embed catalog schemas in the CLI binary* — works without network, but then the CLI carries a frozen schema snapshot that drifts from what users pin. Drift detection becomes mandatory and version-mismatch errors become a real category. Skipped: the catalog is a hard published dep already; we're not gaining offline-mode that doesn't exist for the user's own module.
- *Synth file inside user's module dir* — pollutes the user's working tree and requires adding `modulerelease` to the user's `cue.mod` deps.
- *`replace` directive pointing at user's module by path* — CUE's `mod/modfile` does not support `replace` (verified). Composition stays in Go via `FillPath`, so we don't need this.
- *Reimplement `#ModuleRelease` composition in Go* — duplicates schema logic (`_autoSecrets`, `unifiedModule`, label propagation, UUID derivation) and rots whenever the catalog evolves.

### D2. Two `load.Instances` calls share one `*cue.Context`

```
ctx := cuecontext.New()

// User's module — real on-disk; whole-package unification.
// Imports (incl. opmodel.dev/core/v1alpha1@v1) resolve via CUE_REGISTRY.
userInsts := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
userVal   := ctx.BuildInstance(userInsts[0])

// Synthetic wrapper — overlay-served files in a temp anchor.
// Pins the SAME catalog version; resolves via CUE_REGISTRY (cache hit on second fetch).
synthInsts := load.Instances([]string{"."}, &load.Config{Dir: anchor, Overlay: overlayMap})
synthVal   := ctx.BuildInstance(synthInsts[0])

// Compose in Go (FillPath on the package-level value, since `mr.#ModuleRelease`
// is applied at the top of wrapper.cue — same shape as a real release.cue).
spec := synthVal.
    FillPath(cue.MakePath(cue.Def("module")), userVal).
    FillPath(cue.ParsePath("metadata.name"),      synthName).
    FillPath(cue.ParsePath("metadata.namespace"), synthNamespace)
```

`Value` is a handle into its parent context; values from different contexts can't unify (`research/cue/sdk/value.md` L9-11). One `cuecontext.New()` for the whole synthesis is mandatory.

Both loads use the loader's default registry resolver (`CUE_REGISTRY` env). No special `Config.Registry` override needed — the user's environment already drives this for `module vet` and `release build` today.

### D3. Module-mode arg must be a directory; loaded as a CUE package

`opm release build <path>` branches on `os.Stat(path).IsDir()`:

- File → existing `FromReleaseFile` path, unchanged.
- Directory → new `FromModule` workflow.

`opm module build [path]` (`opm mod build` via existing cobra alias) accepts only directories. A file argument errors with: *"module build expects a directory; CUE packages span all files in a dir. Use 'opm release build <file>' for a release file."*

Whole-package loading uses `load.Instances([]string{"."}, &load.Config{Dir: modulePath})` — exactly what `cue eval` / `cue vet` do. No `module.cue`-specific shortcut.

### D4. Values selection mirrors `opm module vet`

- `-f`/`--values` files merged in declaration order (same as `module vet`).
- Otherwise the module's `debugValues` field.
- Neither present → error with hint: *"module does not define debugValues — add debugValues or provide values with -f"* (same string `module vet` uses today).

`pkg/loader.LoadModuleReleaseFromValue` already runs the Module Gate (validate values vs `#config`), concreteness check, metadata extraction, and finalisation. The synthesis path hands off the composed `cue.Value` and the caller-supplied values; downstream behaviour is identical to a real release file.

### D5. Synthetic metadata defaults

```
metadata.name      ← --name flag, else "<module.metadata.name>-debug"
metadata.namespace ← --namespace flag (already exists), else "default"
```

`--name` is a new flag specific to module-mode build. When module mode triggers (`opm release build <dir>` or `opm module build`), the flag fills `metadata.name` on the synthetic wrapper.

A banner prints before render output to make synthetic-build runs visually distinct: `Building synthetic release "<name>" for module "<module.metadata.name>"`.

### D6. Catalog version is copied from the user's modfile

Parse the user's `cue.mod/module.cue` via `mod/modfile.Parse` (`research/cue/sdk/mod.md` L45-64). Look up `mf.Deps["opmodel.dev/core/v1alpha1@v1"]`. Use the returned version string verbatim in the synth modfile. Effects:

- Both loads resolve the catalog to the same registry artifact → same `#Module` definition → no subsumption mismatches.
- "CLI ↔ catalog drift" disappears as a category. Users upgrading their module's catalog pin automatically get the matching `#ModuleRelease` semantics.
- No CLI release is needed when the catalog adds non-breaking modulerelease features.

Failure modes:

- *Module declares no catalog dep* → `DetailError` with hint to add `opmodel.dev/core/v1alpha1@v1` as a dep.
- *Pinned catalog version unreachable* (registry down, version yanked) → CUE's loader surfaces the registry error; we let it bubble up unwrapped beyond a context note (`fmt.Errorf("resolving catalog dep for synth wrapper: %w", err)`).

### D8. Surface change for `opm release build`

Today's command takes `Args: cobra.ExactArgs(1)` with help text *"Path to the release .cue file (required)"*. New shape: `Args: cobra.ExactArgs(1)` still, but the arg can be a release file or a module directory. Help text and examples updated accordingly. No flag removed; `--name` added.

### D9. Re-use `pkg/loader.LoadModuleReleaseFromValue`

The synthesis function returns a concrete `cue.Value` shaped as a `#ModuleRelease` (with `#module` filled, `metadata` filled, `values` left to be filled by the loader). `LoadModuleReleaseFromValue` then runs Module Gate → values fill → concreteness → metadata extraction. This keeps the `release-building` capability's Module Gate authoritative — the synthesis path doesn't bypass any validation.

## Risks / Trade-offs

- **[Synth build needs registry/cache reachable]** → identical constraint to `module vet` and `release build` today (the user's module already imports `core/v1alpha1@v1`). No new failure mode. First fetch lands in `~/.cache/cuelang/mod`; subsequent runs are cache hits.
- **[Anchor temp dir leaks if process is killed]** → `defer os.RemoveAll(anchor)`. Acceptable; same as any temp-dir use.
- **[User's module pins no catalog dep]** → defensive `DetailError` from D6 with a hint. Edge case in practice: every OPM module imports something from `core/v1alpha1`.
- **[Catalog version yanked or registry down at synth-fetch time]** → CUE loader's normal error path; we wrap with context. Not a new failure mode.
- **[Module's own deps fail to resolve]** → unchanged from `module vet` today. Surfaced via the existing CUE error path.
- **[Confusion between synthetic and real release output]** → banner (D5) plus a synthetic name suffix (`-debug`) make logs distinguishable. Document in `--help` text.
- **[`--name` flag collision]** → `opm release build` doesn't currently use `--name`; `opm module build` is new. No conflict in either group.
- **[Bundle authors might expect `opm release build <bundle-dir>` to synthesize a `BundleRelease`]** → out of scope; clear error: *"bundle synthesis is not supported; pass a single-module directory or a release file."*

## Migration Plan

No data migration. Behaviour change is additive:

1. Land `pkg/loader` synthesis function + unit tests against fixtures in `examples/modules/`.
2. Land `internal/workflow/render/FromModule` + unit tests.
3. Wire `opm release build` dir-branching.
4. Wire `opm module build` (and the `mod` alias).
5. Update CLI docs / examples / QUICKSTART note.

Rollback: revert the cobra wiring; the `pkg/loader` and `internal/workflow/render` additions are inert without a command surface.

## Open Questions

- Should the `--name` flag also be wired into `opm release build` for the file path (overriding `metadata.name` from a real release file), or strictly module-mode only? Lean: module-mode only — overriding a real release's name is a separate use case.
- Default synthetic namespace: `default` vs `opm-debug`? Lean: `default` — matches what most module debug runs target locally; users override via `--namespace`.
