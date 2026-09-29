## Context

`internal/platform.Resolve` (`resolve.go:202-262`) resolves every render's platform by precedence: `[dir]` argument > `--platform` > module deps (module build/vet only) > cluster Platform CR (when a getter is passed) > `~/.opm/platform/`. Today:

| Command | Sources |
| --- | --- |
| `module build`, `module vet` | `--platform` > module deps |
| `instance build`, `instance vet` | `--platform` > local default; never the cluster |
| `instance apply`, `instance diff`, `module apply` | `--platform` > cluster CR > local default (warns) |
| `platform check` | `[dir]` > `--platform` > local default |
| `platform pull` | cluster CR only (`NoLocalFallback`) |

`opm config init` writes `~/.opm/platform/` from `internal/config/templates.go` (`DefaultCorePin`, `DefaultCatalogPins`), `opm config vet` builds it, and an apply that fell back to it seeds the cluster Platform (`internal/workflow/apply/apply.go:188-190` → `EnsureClusterPlatform`, fed by `SpecFromPlatform` through `Result.PlatformSpec`). `opm operator install` seeds its own Platform through `EnsureClusterPlatformForCatalog` and is unaffected.

The module-deps generator (`GenerateModuleDepsModule`, `internal/platform/moduledeps.go`) already turns a committed `cue.mod/module.cue` plus its `local-module.cue` replacements into a cached platform module. This change reuses it as the fallback for every render that is not the module author's.

## Goals / Non-Goals

**Goals:**

- No command writes or reads `~/.opm/platform/`.
- Instance commands and `module apply`: `--platform` > cluster Platform > the render's own deps.
- `instance build` and `instance vet` look at the cluster but never fail because of it; `--offline` skips the lookup.
- No render-bearing command creates a Platform.
- `platform check` resolves `[dir]` > `--platform` > cluster, and refuses otherwise.

**Non-Goals:**

- Skipping provider-fulfilled demands (`add-skip-unprovided-flag`, the next change in the set).
- Changing `platform pull`, `operator install`, the module-deps generator itself, or the cluster arm's effective-registry logic.
- Taking `--platform` as a pure-data `#Platform` file. That waits on enhancement 0026 landing in core.
- Deleting a user's existing `~/.opm/platform/`. It is left on disk; `config vet` says it is unused.

## Decisions

### Resolver shape

`ResolveOptions` drops the exclusivity between the module deps and the cluster, and the local-default step. The deps become the last step for every render that sets them.

```go
type ResolveOptions struct {
	Argument     string                // [dir] (platform check)
	PlatformFlag string                // --platform <dir>
	ConfigPath   string                // locates cache/platforms only
	Cluster      ClusterPlatformGetter // nil: no cluster step
	// ClusterOptional: any cluster failure (not only NotFound/Forbidden)
	// warns and falls through to Deps. Set by instance build and vet.
	ClusterOptional bool
	// Deps is the last step: a platform generated from committed pins.
	// Kind names whose pins they are, for the provenance line.
	Deps     *ModuleDeps
	DepsKind DepsKind // DepsModule | DepsInstance
	// NoFallback: a cluster that yields no Platform is an error
	// (ErrNoClusterPlatform). Set by platform pull and platform check.
	NoFallback bool
	Registry   string
	ModFiles   platformmodule.ModFileSource
}

func Resolve(ctx context.Context, opts ResolveOptions) (string, Resolution, error)
// 0. Argument  1. PlatformFlag  2. Cluster (if non-nil)  3. Deps (if non-nil)
// else: error naming the sources that were looked for.
```

`SourceLocalDefault` is removed. `SourceModuleDeps` stays, and `Resolution` gains the deps kind so `Describe()` prints `module deps` or `instance deps`:

```text
platform: instance deps (opmodel.dev/catalogs/opm@v4 v4.4.0; generated module /home/u/.opm/cache/platforms/3f2a...)
```

Cluster step outcomes:

| Getter result | Default | `ClusterOptional` | `NoFallback` |
| --- | --- | --- | --- |
| Platform read | cluster source | cluster source | cluster source |
| NotFound / Forbidden | warn, deps | warn, deps | `ErrNoClusterPlatform` |
| other error (unreachable, timeout) | fatal `ErrClusterRead` | warn, deps | fatal `ErrClusterRead` |

Fallback warning, replacing "falling back to the local default platform":

```text
WARN cluster Platform not used (no Platform CR in the cluster) — rendering against the instance's own deps
```

**Options considered:**

1. Keep `ModuleDeps` exclusive and add a second deps field for the fallback. Two fields meaning "generate from pins" invite drift.
2. One `Deps` field, used last by every render that sets it. Chosen: module build/vet set `Deps` with no `Cluster`; everything else sets both.

### Who passes what

| Command | `Cluster` | `ClusterOptional` | `Deps` |
| --- | --- | --- | --- |
| `module build`, `module vet` | nil | - | module (unchanged) |
| `instance build`, `instance vet` | getter, unless `--offline` or no kube context | true | instance package |
| `instance diff`, `instance apply` | getter | false | instance package |
| `module apply` | getter | false | module (same reader as `module build`) |
| `platform check` | getter | false | nil, `NoFallback` |
| `platform pull` | getter | false | nil, `NoFallback` (unchanged) |

The instance package's deps are read exactly as the module's are (`moduleDepsOf` in `internal/workflow/render/module.go`): its `cue.mod/module.cue` and `cue.mod/local-module.cue` under the module context root `render.go` already computes (`moduleContextRoot(instanceDir)`). A package under no module root cannot import its module, so that case is a validation error naming the directory.

### Optional cluster lookup for build and vet

`instance build` and `instance vet` gain `cmdutil.K8sFlags` (`--kubeconfig`, `--context`) and `--offline`. Building the getter never fails the command:

```go
// optionalClusterGetter returns nil when --offline is set or no kubeconfig
// context resolves (clientcmd's empty-config case: no warning). A client that
// cannot be built warns and returns nil. The getter it returns bounds the
// whole lookup with context.WithTimeout(ctx, clusterLookupTimeout).
func optionalClusterGetter(ctx context.Context, cfg *config.GlobalConfig, kf cmdutil.K8sFlags, offline bool) platform.ClusterPlatformGetter

const clusterLookupTimeout = 10 * time.Second
```

`instance diff` and `instance apply` build their client as today and fail on an unreachable cluster, because they need it for their own work.

**Options considered:**

1. Build/vet fail on an unreachable cluster, as apply does. A stopped VPN would break every offline render: rejected.
2. Warn and fall back on any cluster failure, bounded by a timeout. Chosen.
3. No cluster lookup unless a flag asks for it. Rejected by the user decision: the cluster comes first when it exists.

### Removing the local default and the seed

- `internal/config`: remove `PlatformDir`, `WritePlatformModule`, `Paths.PlatformDir`, the platform-module templates, `DefaultCorePin` and `DefaultCatalogPins` (their only reader outside tests is the template). Keep what `operator install` uses (`platform.DefaultCatalogPath` and its registry resolution).
- `opm config init`: writes `config.cue` only; keeps removing a legacy data-only `platform.cue` under `--force`; never touches `~/.opm/platform/`.
- `opm config vet`: drops the platform check; if a `platform/` directory sits beside the config file, prints `WARN ~/.opm/platform/ is no longer read by any command; pass it with --platform <dir> or delete it`.
- Seed removal: delete the seeding block in `internal/workflow/apply/apply.go`, `EnsureClusterPlatform`, `SpecFromPlatform` (`seed.go`), `renderEnv.spec` and `Result.PlatformSpec`. `createClusterPlatform` stays for `EnsureClusterPlatformForCatalog`.

### Refusal hints for the deps source

`refusalHint` (`internal/workflow/render/validation.go:75`) keys on the deps source for both kinds. The no-catalog hint names the tidy command for the kind: `opm module tidy` for a module, `cue mod tidy` in the package directory for an instance. The unresolved-demand hint keeps its wording (provider-fulfilled contracts come from a platform; `--platform <dir>`). `add-skip-unprovided-flag` rewrites that hint later.

### Tests

Tests that need a platform directory use the repo's maintained `hack/platform/` module instead of `config.WritePlatformModule`: `tests/e2e/instance_build_test.go` (including the skew test, which copies `hack/platform/` and lowers its catalog pin), `tests/e2e/mod_build_test.go`, `tests/integration/render-parity`, `tests/integration/module-apply`. `tests/integration/platform-build` checks the seeded default builds offline; retarget it to `hack/platform/` so the kind dev flow's platform keeps a build check.

## Command Syntax

```text
opm instance build <instance> [flags]
opm instance vet <instance> [flags]
  -f, --values strings      Additional values files
  -n, --namespace string    Target namespace
      --platform string     Platform module directory (overrides the cluster Platform and the instance's own deps)
      --kubeconfig string   Path to kubeconfig (for the cluster Platform lookup)
      --context string      Kubernetes context (for the cluster Platform lookup)
      --offline             Never contact a cluster; render against --platform or the instance's own deps (default false)

opm platform check [dir] [flags]
      --platform string     Platform module directory
      --kubeconfig string   Path to kubeconfig
      --context string      Kubernetes context
```

`instance diff`, `instance apply` and `module apply` keep their flags; only the `--platform` help text changes.

## Error Handling and Exit Codes

| Case | Command | Behaviour | Exit |
| --- | --- | --- | --- |
| No kube context | build, vet | deps, no warning | render's |
| Platform absent or forbidden | build, vet, diff, apply | warn, deps | render's |
| Cluster unreachable | build, vet | warn naming the error, deps | render's |
| Cluster unreachable | diff, apply, module apply | unchanged: today's error | unchanged |
| Instance package under no module root | instance commands | validation error naming the directory | 2 |
| Deps pin unpublished | any deps render | generation error naming path and version | 1 |
| No source for `platform check` | platform check | `no platform to check: pass [dir] or --platform <dir>, or point --context at a cluster with a Platform` | 5 (not found) |
| Leftover `~/.opm/platform/` | config vet | warning only | 0 |

Example, `opm instance build ./hello` on a laptop whose context points at a kind cluster without a Platform:

```text
WARN cluster Platform not used (no Platform CR in the cluster) — rendering against the instance's own deps
INFO platform: instance deps (opmodel.dev/catalogs/opm@v4 v4.4.0; generated module /home/u/.opm/cache/platforms/3f2a...)
```

## Data Flow

```text
instance build/vet
  --offline? ----------------------------------------------+
  kube context? --no------------------------------------+   |
     | yes                                               v   v
  GET Platform "cluster" (10s) --read--> cluster module   deps module
     | NotFound/Forbidden/error: warn ----------------->  (instance cue.mod
                                                           + local-module.cue)
                      --platform <dir> overrides all of the above
                                     |
                         kernel.AcquirePlatformFromDir -> Kernel.Render
```

## Research & Decisions

### Which deps an instance render uses

**Context**: The deps fallback needs a dependency list for an instance package.
**Explored**: `opm instance init` (`internal/instinit`) writes `cue.mod/module.cue` pinning the module and core, then completes the closure with `cuemod.Tidy`, so a tidied package lists the module's catalogs transitively. `render.go` already locates the package's module context root.
**Options considered**:
1. The instance package's own `cue.mod/module.cue`: the list the render build itself resolves through; carries the package's replacements. Needs a tidied package.
2. The acquired module's `cue.mod/module.cue`: what the module author tested, but ignores the instance package's own resolution and its replacements.
**Decision**: Option 1, the instance package's own list.
**Rationale**: It is what the build resolves, and the generator already handles a list with no catalogs (empty registry, hint to tidy). Section 1 verifies an `instance init` package lists its catalogs and renders against it.

### Seeding a Platform on apply

**Context**: 0006:D12/D22 seeded the cluster Platform from the local default so a later operator install adopted it.
**Explored**: `opm operator install` already seeds its own Platform (`EnsureClusterPlatformForCatalog`, `internal/cmd/operator/install.go:196`) from the registry-resolved catalog.
**Options considered**:
1. Seed from the deps platform. The deps are one module's pins, not a cluster's; it would make the first module applied decide the cluster's catalogs.
2. Stop seeding on apply. Chosen.
**Decision**: Remove the apply seed; keep the install seed.
**Rationale**: A Platform should be created by whoever installs the operator, from a registry-resolved catalog, not as a side effect of the first apply.

### Keeping `hack/platform/`

**Context**: Tests and the kind dev flow need a platform module directory.
**Explored**: `hack/platform/` is a real CUE module that the workspace root `task deps:update` already bumps; `hack/kind-platform.yaml` mirrors its pins for the cluster Platform.
**Decision**: Keep both, and make `hack/platform/` the test fixture that replaces `WritePlatformModule`.
**Rationale**: It stays maintained without the Go pins, and the tests keep exercising a real `--platform` directory.

## Risks / Trade-offs

- [`instance build` output now depends on the kube context] → the provenance line always names the source, and `--offline` gives a cluster-free render.
- [An unreachable cluster slows build/vet by up to 10 s] → only when a context exists and the server does not answer; `--offline` avoids it.
- [Users with a hand-edited `~/.opm/platform/` silently stop using it] → `config vet` warns, and the release note (the `feat!` commit body) names `--platform ~/.opm/platform`.
- [A CLI-only cluster previously got a Platform from its first apply] → it now has none until `opm operator install`; renders there use the deps, which is the supported mode for clusters without the operator.
