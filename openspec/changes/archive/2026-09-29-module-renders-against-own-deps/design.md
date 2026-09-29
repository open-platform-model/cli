## Context

See proposal.md for motivation and specs/ for the behavior contract. Current state that shapes the approach:

- `internal/platform.Resolve` is the one precedence function: argument > `--platform` > cluster CR (only when a `ClusterPlatformGetter` is passed) > `~/.opm/platform/`. The cluster CR arm already turns catalog pins into a platform module: `platformmodule.Closure` derives the dependency list from published module files, `platformmodule.Generate` writes `cue.mod/module.cue` and `platform.cue`, and `writeCached` places them under `cache/platforms/<sha256>/` (`internal/platform/generate.go`).
- `render.FromModule` acquires the module (`Kernel.AcquireModuleFromDir`), resolves values, synthesizes the instance, then calls `resolvePlatformEnv` and renders. `module build`, `module apply` and `instance build <module-dir>` all enter here; only the first must change.
- The acquired module carries its tree as a `module.Source` in overlay mode, `cue.mod/module.cue` included, whether it was acquired from a directory or from the registry (`opm/module/source.go`). The committed dependency list is readable from `Source.Overlay[Root/cue.mod/module.cue]` with `cuelang.org/go/mod/modfile`, with no second load.
- The render build promotes the platform's `cue.mod` whole and the instance's only for paths the platform does not name; the same precedence governs `local-module.cue` replacements (`renderstage/promote.go`). Skew rows compare the instance's committed pins with the platform's (`renderstage/skew.go`).
- A module's tidied `cue.mod` is not a closure. `modules/apprise` pins `opmodel.dev/catalogs/opm@v4` and core, but not `cue.dev/x/k8s.io@v0`, which `catalog_opm/opm/cue.mod/module.cue` requires for its transformers. `cue mod tidy` prunes by imported package, and the platform imports the catalog's root package, not the module's subpackages.
- `opm module vet` builds in the identity schema's CUE context (`identitySchemaForVet`), validates `#config`, and stops. It already registers every `RenderFlags` flag, `--platform` included, and ignores all but `-f`.

## Goals / Non-Goals

**Goals:**

- One generator for the module-deps platform, beside the cluster CR generator, sharing its cache writer.
- The deps source is resolved inside `platform.Resolve`, so the "flag wins" rule and the provenance line stay in one place.
- The same code path for a directory module and a registry-acquired module, so `split-module-and-instance-inputs` needs no follow-up.
- Cheap failures first: identity, values and synthesis fail before any registry I/O for the closure.

**Non-Goals:**

- Catalogs outside `opmodel.dev/catalogs/`. No third-party catalog exists; `--platform` covers one.
- A cache-only or `--offline` mode. The closure reads the same CUE module cache every render uses.
- Changing `opm instance vet` or `opm instance build` inputs. `split-module-and-instance-inputs` owns instance inputs.
- Pruning the generated platform's closure the way `cue mod tidy` would. An unimported pinned path is harmless (`platformmodule.Closure` doc).
- The colocated-instance provenance gap (`SourceLocal` false for an instance inside its module's tree). Separate issue.

## Decisions

### Command syntax and flags

Syntax is unchanged:

```text
opm module build [path] [flags]
opm module vet [path] [flags]
```

| Flag | Type | Default | Description (new text) |
| --- | --- | --- | --- |
| `--platform` | string | `""` | `Render against this platform module directory instead of the module's own deps` |
| `-n`, `--namespace` | string | resolved | Namespace of the synthesized instance (now affects `vet` too) |
| `--instance-name` | string | module name | Name of the synthesized instance (now affects `vet` too) |
| `-f`, `--values` | []string | `nil` | Unchanged |

`RenderFlags.AddTo` keeps the shared help text for every other command; `module build` and `module vet` overwrite the `platform` flag's `Usage` after registering it. No flag is added or removed.

### Data flow

```text
opm module build [path] [--platform dir]
  -> render.FromModule(ModuleOpts{..., PlatformFromDeps: true})
       1. Kernel.AcquireModuleFromDir(path)          (or FromRegistry, split change)
       2. ResolveModuleValues, validate -f files, SynthesizeInstance
       3. resolvePlatformEnv(..., moduleDeps(mod, moduleRoot))
            platform.Resolve(ResolveOptions{PlatformFlag, ModuleDeps, Cluster: nil})
              flag set?        -> SourceFlagDir (unchanged)
              else             -> GenerateModuleDepsModule
                                    a. parse committed cue.mod/module.cue
                                    b. select opmodel.dev/catalogs/* deps  (none -> empty registry)
                                    c. roots = Roots(entries) + the module's core pin
                                    d. Closure(roots) over replacedModFiles{registry, module replacements}
                                    e. Generate + carried local-module.cue
                                    f. writeCached(cache/platforms)
                                 -> SourceModuleDeps
            Kernel.AcquirePlatformFromDir(dir)
       4. Kernel.Render -> renderInstance
            refusal + SourceModuleDeps + UnresolvedDemandsError -> provider hint
            refusal + SourceModuleDeps + no catalogs + UnmatchedComponentsError -> tidy hint
            replacementWarnings(rows, moduleRoot, carried)
```

`opm module vet` runs its identity and `#config` checks first, unchanged, then calls the same `render.FromModule` with `PlatformFromDeps: true` and prints the per-object lines and summary the way `instance vet` does.

### Package layout and signatures

```go
// internal/platform/resolve.go
const SourceModuleDeps Source = "module-deps"

type ResolveOptions struct {
	// ... existing fields ...

	// ModuleDeps, when non-nil, replaces the cluster and local-default
	// steps: a command rendering for its author (module build, module vet)
	// resolves --platform, else a platform generated from these deps.
	// Cluster MUST be nil when it is set.
	ModuleDeps *ModuleDeps
}

type Resolution struct {
	// ... existing fields ...

	// Catalogs names each registry entry of a module-deps platform as
	// "<path> <version>", in path order. Set only for SourceModuleDeps.
	Catalogs []string
}
```

```go
// internal/platform/moduledeps.go

// ModuleDepsPlatformModulePath is the generated module's identity, in the
// reserved, never-published platforms namespace beside
// ClusterPlatformModulePath.
const ModuleDepsPlatformModulePath = "opmodel.dev/platforms/module-deps@v0"

// CatalogPathPrefix selects the deps that become registry entries.
const CatalogPathPrefix = "opmodel.dev/catalogs/"

// ModuleDeps is what a module-deps platform is generated from.
type ModuleDeps struct {
	// ModFile is the module's committed cue.mod/module.cue, and
	// ModFileName names it in errors.
	ModFile     []byte
	ModFileName string
	// Replacements are the replaceWith entries of the module's
	// cue.mod/local-module.cue, with ModuleRoot resolving relative
	// directories. Both empty for a module with no local context
	// (a published module).
	Replacements []loader.LocalReplacement
	ModuleRoot   string
}

// GenerateModuleDepsModule writes the platform module for deps under
// opts.CacheDir and returns its directory, the registry entries it carries
// and the replacements it carried (path -> absolute target).
func GenerateModuleDepsModule(ctx context.Context, deps ModuleDeps, opts GenerateOptions) (
	dir string, entries []platformmodule.Entry, carried map[string]string, err error)

// replacedModFiles serves a replaced module's module file from its
// replacement (a directory's cue.mod/module.cue, or the target module's
// published file) and delegates every other lookup to base.
type replacedModFiles struct {
	base    platformmodule.ModFileSource
	targets map[string]string // major-qualified path -> absolute dir or module@version
}
```

```go
// internal/workflow/render/types.go
type ModuleOpts struct {
	// ... existing fields ...

	// PlatformFromDeps selects the author's platform: --platform, else a
	// platform generated from the module's own deps. Set by module build
	// and module vet; never by module apply.
	PlatformFromDeps bool
}
```

The generated platform's name is `module-deps` and its type `kubernetes`, matching the local default's informational type. The platform is never applied, so the name reaches no cluster; the write-if-absent seed runs only for `SourceLocalDefault` in apply.

### Error handling and exit codes

| Condition | Exit | Message shape |
| --- | --- | --- |
| No `opmodel.dev/catalogs/*` dep, module has components | 2 | kernel unmatched-components refusal + hint `the module pins no catalog under opmodel.dev/catalogs/: pin the catalogs it imports with 'opm module tidy', or pass --platform <dir>` |
| Committed `cue.mod/module.cue` unreadable or unparseable | 2 | the modfile error naming the file |
| Unpublished pin, unreachable registry during the closure | 1 | `module deps platform: resolving dependency <path>@<version>: <cause>` (the `Closure` wording, as the cluster arm) |
| Generated platform fails acquisition | 1 | existing `building platform module <dir> (source module-deps): ...` |
| Render refused, unresolved demands, source module-deps | 2 | kernel refusal + hint `the module's own catalogs do not implement this contract; a provider-fulfilled contract comes from a platform: pass --platform <dir>` |
| Any other render refusal | 2 | unchanged |

### Example output

```text
$ opm module build ./apprise
INFO Building synthetic instance "apprise" for module "apprise"
INFO platform: module deps (opmodel.dev/catalogs/opm@v4 v4.4.0; generated module /home/me/.opm/cache/platforms/3f2a...)
INFO m:apprise: > apprise <- opmodel.dev/catalogs/opm/transformers/deployment-transformer@4.4.0
...
kind: Deployment
...

$ opm module vet ./apprise
INFO m:apprise: [x] Identity conforms to #IdentityPackage  identity/identity.cue
INFO m:apprise: [x] Values satisfy #config  debugValues
INFO m:apprise: [x] Module config valid
INFO platform: module deps (opmodel.dev/catalogs/opm@v4 v4.4.0; generated module ...)
INFO m:apprise: Deployment default/apprise  valid
INFO m:apprise: [x] Module valid (4 resources)

$ opm module build ./no-catalog
INFO platform: module deps (no catalogs; generated module ...)
ERRO render failed: 1 component(s) matched no transformer: web
Hint: the module pins no catalog under opmodel.dev/catalogs/: pin the catalogs it imports with 'opm module tidy', or pass --platform <dir>
(exit 2)
```

The glyphs are whatever `output.FormatVetCheck`, `FormatCheckmark` and `FormatResourceLine` print today; the example shows ASCII stand-ins.

## Research & Decisions

### Where the generator lives

**Context**: The workspace rule is that frontends rely on the library for structure.
**Explored**: `library/opm/helper/platformmodule` (`Roots`, `Closure`, `Generate`), `cli/internal/platform/generate.go`.
**Options considered**:
1. A library helper (`platformmodule.FromModFile`): shareable, but the operator never renders from a module's own deps, and choosing which deps an author render subscribes to is frontend policy, not a kernel verdict.
2. `cli/internal/platform`, composing the library's public `Closure` and `Generate`: `Roots`'s doc already invites a caller needing a different core build to assemble roots itself.
**Decision**: Option 2.
**Rationale**: No new structure; the kernel still loads, matches and renders. The CLI only chooses inputs.

### Closure: `platformmodule.Closure`, not `opm module tidy`

**Context**: `add-instance-init` computes an instance package's closure with the in-process `cue mod tidy` (`internal/cuemod.Tidy`, landed with `add-dependency-tidy`).
**Options considered**:
1. `cuemod.Tidy` over a written platform directory: bit-for-bit `cue mod tidy` parity, but it changes the process working directory and `CUE_REGISTRY`, and it can add a missing dependency at its latest version.
2. `platformmodule.Closure`: pure, deterministic, already produces the cluster CR module byte-identical to the operator's.
**Decision**: Option 2.
**Rationale**: The generated platform is the same kind of artifact as the cluster CR module: derived, cached, never committed and never checked with `--check`. Tidy's parity argument is about authored files. Option 1's "latest" fallback would make the render depend on registry state instead of the module's pins.

### Which pins become roots

**Options considered**:
1. The module's whole dependency list: the platform would name every path the module names, so the module's own replacements of any of them would be overridden by the platform (inert), and a colocated or instance module's self entry would be pinned by the platform.
2. The catalog pins, plus core at the higher of the module's pin and the kernel's verified core (`platformmodule.Roots(entries)` plus the module's core pin as an extra root; `Closure` keeps the maximum per path), closed over.
**Decision**: Option 2.
**Rationale**: The platform names exactly what it imports and what that needs. Every other path stays the instance's, under the kernel's existing precedence. Core is the one path the kernel itself constrains: a module pinning an older core (the `tests/fixtures/valid/simple-module` fixture pins `v2.0.0-alpha.6`) must not pull the render below the release the kernel was verified against, and the module generated from the cluster CR carries the same floor through `Roots`. Every pinned path is at least the module's version, so skew cannot arise and the policy is not applied.

### No catalog dependency

**Decision**: Generate a platform with an empty `#registry` and render; do not refuse up front.
**Rationale**: A module without components (the `simple-module` fixture `module vet` is unit-tested on) renders zero objects and is valid. A module with components gets the kernel's own unmatched verdict, and the CLI adds the hint. `platformmodule.Generate` already emits an empty registry.

### Catalog selection by path prefix

**Options considered**:
1. Prefix `opmodel.dev/catalogs/`: the owned-prefix layout (`opmodel.dev/site/content/docs/reference/registry-namespaces.md`); every catalog the fleet and the fixtures use today.
2. Acquire every dependency as a catalog (`Kernel.AcquireCatalogFromRegistry`) and keep those that pass: finds third-party catalogs, at one catalog build per dependency.
3. `IsOPMPath` (any `*.opmodel.dev` host): would subscribe core and the testing domain's modules as catalogs.
**Decision**: Option 1.
**Rationale**: YAGNI. Option 2 is the upgrade path when a third-party catalog exists; `--platform` covers one until then.

### Carrying the module's replacements

**Context**: Today a module's `local-module.cue` redirect of a catalog is inert under the local default platform and the CLI says so. The deps platform is derived from the module, so the author's redirect is the author's intent.
**Options considered**:
1. Carry nothing: the author still has to edit a platform module to test a catalog checkout.
2. Carry every module replacement whose path the generated platform pins, with directory targets made absolute (the generated module lives in the cache, so a relative target would resolve against the wrong root), and serve those paths' module files to `Closure` from the replacement.
**Decision**: Option 2.
**Rationale**: The kernel promotes a platform's local view whole, and `modfile.FormatLocal` writes it. Serving the replaced module file from the directory lets an unpublished pin in a catalog under development render. The cache hash covers `local-module.cue`, and the closure output covers the checkout's own `cue.mod`, so a changed target or a changed checkout requirement lands in a new directory. The checkout's other files are read live at render time, as with any directory replacement.

### Where the deps come from

**Decision**: Read `cue.mod/module.cue` from the acquired module's `Source` overlay; read replacements with `loader.LocalReplacements(moduleContextRoot(path))` for a directory module only.
**Rationale**: One path for directory and registry acquisition (`split-module-and-instance-inputs`). A published module's committed file is exactly what its author tidied and published. Replacements need a local module context, which a published module does not have, matching how `split` already sets `moduleRoot` empty for it.

### `module vet` renders through `FromModule`

**Options considered**:
1. Call `render.FromModule` after the existing checks: a second CUE load of the module (the identity checks load it in the identity schema's context).
2. Share one acquisition: the identity checks deliberately build in the schema's context (`identitySchemaForVet`), while the render path builds in the kernel's; merging them is the behavior change the existing `nolint` comment warns about.
**Decision**: Option 1.
**Rationale**: One extra module load, measured in hundreds of milliseconds, against a verdict that is by construction the same as `module build`'s.

### Replacement warnings for carried paths

**Decision**: `replacementWarnings` gains the carried set. A carried row is worded with the module side's label (`instance`, the label module renders already use for their own rows), and a local-file entry is inert only when no row carries its path with the same target.
**Rationale**: Without it the author sees both "in effect" and "ignored" for one redirect.

### Relation to 0006:D21

0006:D21 says offline `build`/`render` use `--platform` or the local default only. This change keeps its load-bearing half (build never reads the cluster) and replaces the local default with the module's deps for `module build` and `module vet`. No enhancement decision is implemented here, so the change carries no `enhancement.yaml`.

### Spike findings

Run 2026-09-28 against library `v1.0.0-alpha.33`, `modules/apprise` (pins `opmodel.dev/catalogs/opm@v4` `v4.4.0`, `opmodel.dev/core@v2` `v2.0.0-alpha.10`) and GHCR.

- **Rendered objects are identical.** `Closure(Roots(entries) + the module's core pin)` then `Generate` produced a platform whose render of the synthesized `apprise` instance gave the same three objects (Deployment, Service, HTTPRoute), byte for byte after JSON normalisation, as `hack/platform`. Neither render reported a skew row or a replacement row.
- **The catalog's own dependency is closed over.** The generated `cue.mod/module.cue` pins `cue.dev/x/k8s.io@v0` at `v0.11.0`, which `apprise` does not list and `opmodel.dev/catalogs/opm@v4` requires.
- **The closure works offline with a warm cache.** With the cache warm, `Closure` returned the same three pins with `CUE_REGISTRY` pointing at an unresolvable host and again at a refused port. The module cache is keyed by module path and version, not by registry host, so the offline risk is closed.
- **A platform `local-module.cue` with an absolute target is honoured.** Both shapes work: a minimal file listing only the replaced path, and one listing every pinned path with only the catalog replaced. The edited transformer's label appears on the Deployment, and the kernel returns one row, `{Path: opmodel.dev/catalogs/opm@v4, Target: <absolute checkout dir>, By: "platform"}`. The generator writes the minimal shape.
- **The checkout must declare the pinned version.** `platformmodule.Generate` stamps each entry's pinned version as the entry's expected `version`, and the catalog reports its own `identity.Version`. A checkout at `4.4.1` behind a module pin of `v4.4.0` fails platform acquisition with `#registry."opmodel.dev/catalogs/opm@v4".version: conflicting values "4.4.1" and "4.4.0"`. That is the generator's existing tripwire, and a `--platform` directory replacing its catalog hits it the same way. So an author who redirects a catalog also pins the version the checkout declares. Today the same redirect is ignored with a warning; after this change it is honoured, so a mismatched pin now fails where it used to pass. The error names both versions.
- **A directory-acquired module keys its overlay under the real directory.** `Kernel.AcquireModuleFromDir` returns `Source.Root` as the absolute module directory, not a synthetic root, and `Source.Overlay` carries `cue.mod/module.cue` under it. `TestModuleSource_CarriesCommittedModFile` pins this.

### Coordination with sibling changes

- `split-module-and-instance-inputs` edits `render.FromModule`, `ModuleOpts` and `module build`. Neither change depends on the other: `PlatformFromDeps` reads the acquired `Source`, which the registry branch also produces. Whichever lands second rebases; the conflict is textual.
- `add-instance-init` is unaffected: instance commands keep the platform.

## Risks / Trade-offs

- [Verified in the spike: `Closure` through `modconfig.NewRegistry` serves module files from the CUE module cache with the registry unreachable, once a first build has run] -> See Spike findings.
- [Verified in the spike: a platform `local-module.cue` with an absolute directory target is honoured by the render build, and its row comes back `By: "platform"` with that absolute target] -> See Spike findings.
- [Verified in the spike: the dir-acquired module's `Source.Overlay` carries `cue.mod/module.cue` at `Root`] -> Pinned by a unit test.
- [A carried catalog checkout whose `identity.Version` differs from the module's pin fails platform acquisition on the entry's version tripwire, where the same redirect was ignored with a warning before this change] -> The error names both versions. The author pins the version the checkout declares, the same rule a `--platform` directory's own replacement already follows.
- [`module build` can now pass where `instance build` of the same module fails, for example a platform missing a catalog] -> Intended: the two answer different questions. The provenance line names the source every time.
- [`module vet` newly refuses modules whose components do not render against their own catalogs] -> `module build` already refused them. Release note states it.
- [`module vet` newly refuses what synthesis already refused for `module build`: an open `debugValues: _` (the kernel does not fill `#config` defaults into an incomplete `values`, even with no values sources)] -> Found in section 4; decided to keep one verdict rather than special-case vet. The `simple-module` fixture declares `debugValues: {}`. Release note names the fix. Filling defaults at synthesis is a library question, not this change.
- [The default synthetic instance name `<module name>-debug` failed the instance-name pattern for every module name with `_`, the only separator a CUE package name allows. `mod init example.com/modules/my_app@v0` followed by `module vet` exited 2 once vet rendered] -> Found in section 4 (`TestE2E_ModInit_ThenVet`). The default now hyphenates the module name (`my-app-debug`), which also fixes `module build` and `module apply` for such modules (`module-synthetic-instance` delta). An explicit `--name` is taken verbatim.
- [Provider-fulfilled contracts: a module attaching one (today only `#Backup`, attached by no fleet module) is refused by `module build` without `--platform`] -> Hint names `--platform`; designed behaviour of the fail-closed gate.
- [Cache growth: one directory per distinct pin and replacement set] -> Derived state, safe to delete, same policy as the cluster arm.

## Migration Plan

No migration. A user who relied on `module build` rendering against `~/.opm/platform/` passes `--platform ~/.opm/platform`. Rollback is reverting the change.

## Open Questions

- Whether to record the 0006:D21 amendment in `enhancements/` (a governance record; it changes nothing here).
