# OPM Mod Pipeline — Architecture Reference

> **Scope**: `opm mod apply`, `build`, `diff`, `delete`, `status`
> **Last updated**: 2026-02-23

---

## Overview

The five `opm mod` commands split into two fundamentally different shapes:

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                         opm mod <command>                                   │
├────────────────────────┬────────────────────────────────────────────────────┤
│  RENDER-FIRST commands │  INVENTORY-FIRST commands                          │
│  (need module source)  │  (cluster state only, no source needed)            │
│                        │                                                    │
│  build                 │  delete                                            │
│  apply                 │  status                                            │
│  diff                  │                                                    │
└────────────────────────┴────────────────────────────────────────────────────┘
```

Render-first commands read local `.cue` files and produce Kubernetes resources.
Inventory-first commands only consult the inventory Secret already on the cluster.

---

## The Render Pipeline (shared core)

Every render-first command runs through the same 5-phase pipeline implemented in
`internal/pipeline/pipeline.go`. The pipeline is invoked via the thin shared
wrapper `cmdutil.RenderRelease()` (`internal/cmdutil/render.go`).

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                     pipeline.Pipeline.Render()                              │
│                     internal/pipeline/pipeline.go                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  Phase 1: PREPARATION                                                       │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ loader.LoadModule(cueCtx, modulePath, registry)                       │  │
│  │ internal/loader/module.go                                             │  │
│  │                                                                       │  │
│  │  1. Enumerate all top-level .cue files in the module directory        │  │
│  │  2. Separate values*.cue from module files                            │  │
│  │  3. load.Instances(moduleFiles) → CUE instance                        │  │
│  │  4. cueCtx.BuildInstance() → fully evaluated cue.Value                │  │
│  │  5. Extract metadata (name, fqn, version, uuid, defaultNamespace)     │  │
│  │  6. Extract #config schema                                            │  │
│  │  7. Extract values (Pattern A: values.cue / Pattern B: inline)        │  │
│  │  8. Extract #components                                               │  │
│  │  → *module.Module                                                     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│               │                                                             │
│               ▼                                                             │
│  Phase 2: PROVIDER LOAD                                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ loader.LoadProvider(cueCtx, providerName, providers)                  │  │
│  │ internal/loader/provider.go                                           │  │
│  │                                                                       │  │
│  │  Parse transformers from config.Providers CUE values                  │  │
│  │  Extract per transformer:                                             │  │
│  │    requiredLabels, requiredResources, requiredTraits                  │  │
│  │    optionalTraits                                                     │  │
│  │    #transform function (the actual generation logic)                  │  │
│  │  → *provider.Provider (with Transformers map)                         │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│               │                                                             │
│               ▼                                                             │
│  Phase 3: BUILD                                                             │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ builder.Build(cueCtx, mod, opts, valuesFiles)                         │  │
│  │ internal/builder/builder.go                                           │  │
│  │                                                                       │  │
│  │  1. Load opmodel.dev/core@v0 from the module's pinned dep cache       │  │
│  │  2. Extract #ModuleRelease schema                                     │  │
│  │  3. Select values (external files or mod.Values)                      │  │
│  │  4. Validate values against #config schema                            │  │
│  │  5. FillPath chain (order matters):                                   │  │
│  │       #module → metadata.name → metadata.namespace → values           │  │
│  │  6. Validate full concreteness of the result                          │  │
│  │  7. Read back: ReleaseMetadata (uuid, labels, components)             │  │
│  │  8. Read back: components map                                         │  │
│  │  → *modulerelease.ModuleRelease                                       │  │
│  │                                                                       │  │
│  │  + rel.ValidateValues()                                               │  │
│  │  + rel.Validate()                                                     │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│               │                                                             │
│               ▼                                                             │
│  Phase 4: MATCHING                                                          │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ provider.Match(rel.Components)                                        │  │
│  │ internal/core/provider/provider.go                                    │  │
│  │                                                                       │  │
│  │  For each component × transformer pair (both sorted for determinism): │  │
│  │    check requiredLabels match component labels                        │  │
│  │    check requiredResources exist in component.Resources               │  │
│  │    check requiredTraits exist in component.Traits                     │  │
│  │    collect unhandledTraits (warnings in non-strict mode)              │  │
│  │                                                                       │  │
│  │  → *TransformerMatchPlan                                              │  │
│  │      .Matches   = [(transformer, component, matched bool)]            │  │
│  │      .Unmatched = [component names with no matches]                   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│               │                                                             │
│               ▼                                                             │
│  Phase 5: GENERATE                                                          │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ matchPlan.Execute(ctx, rel)                                           │  │
│  │ internal/core/transformer/execute.go                                  │  │
│  │                                                                       │  │
│  │  For each matched (transformer, component) pair (sequential):         │  │
│  │    1. Resolve #transform CUE value from transformer                   │  │
│  │    2. FillPath: #component = component.Value                          │  │
│  │    3. Build #context (name, namespace, releaseMetadata,               │  │
│  │                        componentMetadata)                             │  │
│  │    4. FillPath: #context fields into transform value                  │  │
│  │    5. Extract output field (list / single resource / map)             │  │
│  │    6. Decode to []*unstructured.Unstructured                          │  │
│  │                                                                       │  │
│  │  → []*core.Resource (weight-sorted: CRDs first, webhooks last)        │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  → *RenderResult { Resources, Release, Module, MatchPlan, Errors, Warnings }│
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Key Data Flow: CUE to Kubernetes

```text
 module.cue                values.cue / --values flags
      │                            │
      ▼                            ▼
 loader.LoadModule()        builder.selectValues()
      │                            │
      └──────────────┬─────────────┘
                     ▼
            builder.Build()
            FillPath chain into #ModuleRelease schema
                     │
                     ▼
           *modulerelease.ModuleRelease
             .Components: map[name]*Component
                          (each has Resources, Traits, Labels, Value)
                     │
                     ▼
            provider.Match()
            check labels / resources / traits
            per (component × transformer) pair
                     │
                     ▼
           *TransformerMatchPlan
                     │
                     ▼
            matchPlan.Execute()
            FillPath: #component + #context → into #transform
            extract output → decode to Unstructured
                     │
                     ▼
           []*core.Resource
            (weight-sorted: CRDs first, webhooks last)
                     │
           ┌─────────┼─────────────┐
           ▼         ▼             ▼
        apply       diff         build
     SSA Patch   GET+compare  stdout YAML
```

---

## Per-Command Pipelines

### `opm mod build`

The simplest path — render only, no cluster contact.

```text
  cobra.Command
       │
       ├── config.ResolveKubernetes()      namespace + provider only
       │   internal/config/resolver.go     (no kubeconfig resolution)
       │
       ├── cmdutil.RenderRelease()         shared render wrapper
       │   internal/cmdutil/render.go
       │       └── pipeline.Render()       all 5 phases
       │
       ├── cmdutil.ShowRenderOutput()      transformer matches + warnings
       │   internal/cmdutil/render.go
       │
       └── output.WriteManifests()         stdout YAML / JSON
           output.WriteSplitManifests()    one file per resource (--split)
           internal/output/manifest.go

  Packages: config, cmdutil, pipeline, loader, builder, core/*, output
  Cluster:  [x] not contacted
```

---

### `opm mod apply`

The most complex path — render + 8-step inventory-aware cluster apply.

```text
  cobra.Command
       │
       ├── config.ResolveKubernetes()      kubeconfig + context + ns + provider
       │
       ├── cmdutil.RenderRelease()         [shared] all 5 render phases
       │
       ├── cmdutil.ShowRenderOutput()      [shared] match output + warnings
       │
       ├── cmdutil.NewK8sClient()          connect to cluster
       │   internal/cmdutil/k8s.go
       │
       ├── k8sClient.EnsureNamespace()     create namespace if --create-namespace
       │   internal/kubernetes/client.go
       │
       │── Step 2: inventory.ComputeManifestDigest()
       │           SHA256 of rendered resource manifests
       │           internal/inventory/digest.go
       │
       │── Step 3: inventory.ComputeChangeID()
       │           SHA1(path + version + values + manifestDigest)
       │           internal/inventory/changeid.go
       │
       │── Step 4: inventory.GetInventory()
       │           read previous inventory Secret from cluster
       │           internal/inventory/crud.go
       │
       │── Step 5a: inventory.ComputeStaleSet()
       │            resources in prev inventory but not current render
       │            internal/inventory/stale.go
       │
       │── Step 5b: inventory.ApplyComponentRenameSafetyCheck()
       │            guard against accidental rename-then-delete
       │            internal/inventory/stale.go
       │
       │── Step 5c: inventory.PreApplyExistenceCheck()
       │            first-time apply: check for pre-existing resources
       │            internal/inventory/stale.go
       │
       │── Step 6: kubernetes.Apply()
       │           server-side apply for all resources in weight order
       │           internal/kubernetes/apply.go
       │               └── Patch(ApplyPatchType) per resource (SSA)
       │
       │── Step 7a: inventory.PruneStaleResources()
       │            delete resources removed from current render
       │            internal/inventory/stale.go
       │            internal/kubernetes/delete.go
       │
       └── Step 8: inventory.WriteInventory()
                   write/update inventory Secret with new change entry
                   internal/inventory/crud.go

  Packages: config, cmdutil, pipeline, loader, builder, core/*,
            inventory, kubernetes, output
  Cluster:  [x] connected (read + write)
```

---

### `opm mod diff`

Render + semantic cluster comparison — no writes.

```text
  cobra.Command
       │
       ├── config.ResolveKubernetes()      kubeconfig + context + ns + provider
       │
       ├── cmdutil.RenderRelease()         [shared] all 5 render phases
       │   NOTE: does NOT call ShowRenderOutput — diff handles errors itself
       │         via DiffPartial (best-effort diff even on partial render)
       │
       ├── cmdutil.NewK8sClient()          connect to cluster
       │
       ├── inventory.GetInventory()        read inventory for orphan detection
       │   internal/inventory/crud.go
       │
       ├── inventory.DiscoverResourcesFromInventory()
       │   fetch live resources referenced by the inventory
       │   internal/inventory/discover.go
       │
       ├── kubernetes.Diff()               semantic diff via dyff
       │   kubernetes.DiffPartial()        (if render had errors, partial)
       │   internal/kubernetes/diff.go
       │       └── for each resource:
       │             GET live state from cluster
       │             compare with local render via dyff
       │             categorize: modified / added / orphaned / unchanged
       │
       └── print diff output
               --- kind/name [modified]
               +++ kind/name [new resource]
               ~~~ kind/name [orphaned]

  Packages: config, cmdutil, pipeline, loader, builder, core/*,
            inventory, kubernetes, output
  Cluster:  [x] connected (read only)
```

---

### `opm mod delete`

Inventory-first — no render pipeline involved.

```text
  cobra.Command
       │
       ├── rsf.Validate()                  --release-name or --release-id required
       │
       ├── config.ResolveKubernetes()      kubeconfig + context + namespace
       │                                   (no provider needed)
       │
       ├── cmdutil.NewK8sClient()          connect to cluster
       │
       ├── confirmDelete()                 interactive prompt (unless --force)
       │
       ├── cmdutil.ResolveInventory()      [shared with status]
       │   internal/cmdutil/inventory.go
       │       ├── inventory.GetInventory()                  by release-id
       │       │   OR inventory.FindInventoryByReleaseName() by name (label scan)
       │       └── inventory.DiscoverResourcesFromInventory()
       │
       └── kubernetes.Delete()             delete all resources in reverse weight order
           internal/kubernetes/delete.go
               └── inventory Secret is tracked in inventory, deleted automatically

  Packages: config, cmdutil, inventory, kubernetes, output
  Cluster:  [x] connected (read + write)
  Pipeline: [ ] not used — no module source needed
```

---

### `opm mod status`

Inventory-first — no render pipeline involved.

```text
  cobra.Command
       │
       ├── rsf.Validate()                  --release-name or --release-id required
       │
       ├── config.ResolveKubernetes()      kubeconfig + context + namespace
       │
       ├── cmdutil.NewK8sClient()          connect to cluster
       │
       ├── cmdutil.ResolveInventory()      [shared with delete]
       │   internal/cmdutil/inventory.go
       │       ├── inventory.GetInventory() OR FindInventoryByReleaseName()
       │       └── inventory.DiscoverResourcesFromInventory()
       │
       ├── kubernetes.GetReleaseStatus()   health check per resource type
       │   internal/kubernetes/status.go
       │       ├── Workloads (Deployment, StatefulSet, DaemonSet): Ready condition
       │       ├── Jobs: Complete condition
       │       ├── CronJobs: always healthy (scheduled)
       │       └── Passive / Custom: healthy on creation
       │
       └── kubernetes.FormatStatus()       table / yaml / json
           kubernetes.FormatStatusTable()  (watch mode: always table)
           + runStatusWatch()              poll every 2s with screen clear

  Packages: config, cmdutil, inventory, kubernetes, output
  Cluster:  [x] connected (read only)
  Pipeline: [ ] not used — no module source needed
```

---

## Shared Infrastructure

The following utilities are reused across commands without duplication.

```text
┌──────────────────────────────┬────────────────────────────────────────────────────────────┐
│ Utility                      │ Used by           │ What it does                           │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ config.ResolveKubernetes()   │ all five commands │ Resolve kubeconfig / context /         │
│ internal/config/resolver.go  │                   │ namespace / provider via precedence:   │
│                              │                   │ flag > env > config > default          │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.RenderRelease()      │ build, apply, diff│ Thin wrapper: resolve module path,     │
│ internal/cmdutil/render.go   │                   │ build RenderOptions, call pipeline.    │
│                              │                   │ Render, handle fatal errors            │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.ShowRenderOutput()   │ build, apply      │ Check render errors, print transformer │
│ internal/cmdutil/render.go   │ (not diff)        │ match log, emit warnings               │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.NewK8sClient()       │ apply, diff,      │ Create Kubernetes dynamic client from  │
│ internal/cmdutil/k8s.go      │ delete, status    │ pre-resolved kubeconfig + context      │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.ResolveInventory()   │ delete, status    │ Lookup inventory by name or id,        │
│ internal/cmdutil/inventory.go│                   │ discover live resources from it        │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.RenderFlags          │ build, apply, diff│ -f/--values, -n/--namespace,           │
│ internal/cmdutil/flags.go    │                   │ --provider, --release-name             │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.K8sFlags             │ apply, diff,      │ --kubeconfig, --context                │
│ internal/cmdutil/flags.go    │ delete, status    │                                        │
├──────────────────────────────┼───────────────────┼────────────────────────────────────────┤
│ cmdutil.ReleaseSelectorFlags │ delete, status    │ --release-name, --release-id,          │
│ internal/cmdutil/flags.go    │                   │ --namespace                            │
└──────────────────────────────┴───────────────────┴────────────────────────────────────────┘
```

---

## Package Dependency Map

```text
  cmd/opm/                      (entry: main.go, root.go)
      │
      └── internal/cmd/mod/     (command implementations)
              │
              ├── internal/cmdutil/         (shared command utilities)
              │       ├── render.go         → internal/pipeline
              │       ├── inventory.go      → internal/inventory
              │       ├── k8s.go            → internal/kubernetes
              │       ├── output.go         → internal/output
              │       └── flags.go
              │
              ├── internal/pipeline/        (render orchestration)
              │       └── pipeline.go       → loader, builder, core/*, inventory
              │
              ├── internal/loader/          (CUE loading)
              │       ├── module.go         → core/module, core/component
              │       └── provider.go       → core/provider, core/transformer
              │
              ├── internal/builder/         (CUE build / FillPath)
              │       └── builder.go        → core/modulerelease, core/component
              │
              ├── internal/core/            (domain types)
              │       ├── module/           Module, ModuleMetadata
              │       ├── modulerelease/    ModuleRelease, ReleaseMetadata
              │       ├── component/        Component, ComponentMetadata
              │       ├── provider/         Provider, Match()
              │       ├── transformer/      Transformer, Execute(), MatchPlan
              │       └── resource.go       Resource (wraps Unstructured)
              │
              ├── internal/inventory/       (inventory Secret CRUD + stale logic)
              │       ├── crud.go           GetInventory, WriteInventory
              │       ├── discover.go       DiscoverResourcesFromInventory
              │       ├── stale.go          ComputeStaleSet, PruneStaleResources
              │       ├── digest.go         ComputeManifestDigest
              │       ├── changeid.go       ComputeChangeID
              │       └── types.go          InventorySecret, ChangeEntry, ...
              │
              ├── internal/kubernetes/      (cluster operations)
              │       ├── apply.go          SSA Patch
              │       ├── delete.go         Delete in reverse weight order
              │       ├── diff.go           GET + dyff comparison
              │       ├── status.go         Health checks per resource type
              │       └── client.go         Dynamic client, EnsureNamespace
              │
              ├── internal/config/          (config loading + resolution)
              │       ├── config.go         GlobalConfig, CueContext, Providers
              │       └── resolver.go       ResolveKubernetes()
              │
              └── internal/output/          (terminal output)
                      ├── manifest.go       YAML/JSON serialization
                      ├── table.go          Status tables
                      ├── log.go            Structured logging
                      └── styles.go         lipgloss styles
```

---

## Notable Design Decisions

**`diff` skips `ShowRenderOutput`**

`apply` and `build` call `cmdutil.ShowRenderOutput()` which returns an error
(and stops) if the render has errors. `diff` intentionally does not — it calls
`kubernetes.DiffPartial()` instead to show a best-effort diff for the resources
that did render successfully, even when some components failed. The comment in
`diff.go:77-78` documents this explicitly.

**Single CUE context per command**:

The same `*cue.Context` flows from `config.GlobalConfig.CueContext` into
`pipeline.NewPipeline()`, through `loader.LoadModule()`, `builder.Build()`, and
into `matchPlan.Execute()`. All CUE values (`mod.Raw`, `releaseSchema`,
`transformValue`, etc.) are bound to this context — mixing contexts causes a
runtime panic. This is why the config creates one fresh context per command
invocation (not a global singleton).

**Execute is sequential, not concurrent**:

`matchPlan.Execute()` runs transformer matches one at a time. `*cue.Context` is
not safe for concurrent use, so parallelism is not possible here without
multiple contexts. The match plan iteration order is deterministic (components
and transformers are sorted alphabetically before matching).

**Inventory is the source of truth for cluster-side commands**:

`delete` and `status` never read the module source. They recover all needed
information from the inventory Secret stored on the cluster. This means a
release can be deleted or inspected even after the module source is gone or
unavailable.

**Stale detection is manifest-only, not cluster-based**:

The stale set is computed by diffing the previous inventory entries against the
current render output — no cluster GET is required for this step. This makes
stale detection fast and offline, but it requires the inventory to stay
consistent with what's actually on the cluster.

**Values isolation**:

`values.cue` is loaded separately via `ctx.CompileBytes` and never unified into
`mod.Raw`. This keeps the default module definition clean. Extra `values_*.cue`
files in the module directory are silently excluded from the package load and
reported via DEBUG — they must be passed explicitly via `--values` to take
effect.
