# OPM Mod Pipeline — Architecture Reference

> **Scope**: `opm mod apply`, `build`, `diff`, `delete`, `status`
> **Last updated**: 2026-02-23

---

## Overview

The five `opm mod` commands split into two fundamentally different shapes:

```mermaid
graph LR
    CMD([opm mod])

    subgraph RF["Render-first — need module source"]
        B[build]
        A[apply]
        D[diff]
    end

    subgraph IF["Inventory-first — cluster state only"]
        DEL[delete]
        ST[status]
    end

    CMD --> B
    CMD --> A
    CMD --> D
    CMD --> DEL
    CMD --> ST
```

Render-first commands read local `.cue` files and produce Kubernetes resources.
Inventory-first commands only consult the inventory Secret already on the cluster.

---

## The Render Pipeline (shared core)

Every render-first command runs through the same 5-phase pipeline implemented in
`internal/pipeline/pipeline.go`. The pipeline is invoked via the thin shared
wrapper `cmdutil.RenderRelease()` (`internal/cmdutil/render.go`).

```mermaid
flowchart TD
    START(["pipeline.Render()"])
    RESULT(["*RenderResult\n{ Resources, Release, Module, MatchPlan, Errors, Warnings }"])

    subgraph PH1["Phase 1 · PREPARATION — internal/loader/module.go"]
        P1A["enumerate .cue files, separate values*.cue"]
        P1B["load.Instances → BuildInstance → cue.Value"]
        P1C["extract metadata, #config, values, #components"]
        P1A --> P1B --> P1C
    end

    subgraph PH2["Phase 2 · PROVIDER LOAD — internal/loader/provider.go"]
        P2A["parse transformers from config.Providers"]
        P2B["extract requiredLabels, requiredResources, requiredTraits, #transform"]
        P2A --> P2B
    end

    subgraph PH3["Phase 3 · BUILD — internal/builder/builder.go"]
        P3A["load opmodel.dev/core@v0, extract #ModuleRelease schema"]
        P3B["select + validate values against #config"]
        P3C["FillPath: #module → name → namespace → values"]
        P3D["validate concreteness, read back metadata + components"]
        P3A --> P3B --> P3C --> P3D
    end

    subgraph PH4["Phase 4 · MATCHING — internal/core/provider/provider.go"]
        P4A["for each component × transformer pair (both sorted)"]
        P4B["check requiredLabels, requiredResources, requiredTraits"]
        P4C["build TransformerMatchPlan with matches and unmatched"]
        P4A --> P4B --> P4C
    end

    subgraph PH5["Phase 5 · GENERATE — internal/core/transformer/execute.go"]
        P5A["for each matched pair: FillPath #component + #context into #transform"]
        P5B["extract output field, decode to *unstructured.Unstructured"]
        P5C["sort resources by weight"]
        P5A --> P5B --> P5C
    end

    START --> PH1
    PH1 -->|"*module.Module"| PH2
    PH2 -->|"*provider.Provider"| PH3
    PH3 -->|"*modulerelease.ModuleRelease"| PH4
    PH4 -->|"*TransformerMatchPlan"| PH5
    PH5 --> RESULT
```

---

## Key Data Flow: CUE to Kubernetes

```mermaid
flowchart TD
    MC[/"module.cue"/]
    VC[/"values.cue / --values flags"/]

    MC --> LM["loader.LoadModule()"]
    VC --> SV["builder.selectValues()"]

    LM & SV --> BB["builder.Build() — FillPath chain into #ModuleRelease schema"]

    BB --> MR["*modulerelease.ModuleRelease\n.Components: map[name]*Component"]

    MR --> PM["provider.Match() — check labels, resources, traits\nper component × transformer pair"]

    PM --> EX["matchPlan.Execute() — FillPath #component + #context\nextract output, decode to Unstructured"]

    EX --> RES["[]*core.Resource — weight-sorted: CRDs first, webhooks last"]

    RES --> AP["apply\nSSA Patch"]
    RES --> DI["diff\nGET + compare"]
    RES --> BU["build\nstdout YAML"]
```

---

## Per-Command Pipelines

### `opm mod build`

The simplest path — render only, no cluster contact.

```mermaid
flowchart TD
    CMD(["opm mod build"])

    CMD --> RK["config.ResolveKubernetes()\nnamespace + provider only — no kubeconfig resolution"]
    RK  --> RR["cmdutil.RenderRelease()\nall 5 render phases"]
    RR  --> SR["cmdutil.ShowRenderOutput()\ntransformer matches + warnings"]
    SR  --> OUT{split?}

    OUT -->|"--split"| WS["output.WriteSplitManifests()\none file per resource to --out-dir"]
    OUT -->|"default"| WM["output.WriteManifests()\nstdout YAML / JSON"]
```

| | |
|---|---|
| **Packages** | `config`, `cmdutil`, `pipeline`, `loader`, `builder`, `core/*`, `output` |
| **Cluster** | not contacted |

---

### `opm mod apply`

The most complex path — render + 8-step inventory-aware cluster apply.

```mermaid
flowchart TD
    CMD(["opm mod apply"])

    CMD --> RK["config.ResolveKubernetes()\nkubeconfig + context + namespace + provider"]
    RK  --> RR["cmdutil.RenderRelease() — all 5 render phases"]
    RR  --> SR["cmdutil.ShowRenderOutput() — match output + warnings"]
    SR  --> K8S["cmdutil.NewK8sClient()"]
    K8S --> ENS["EnsureNamespace()\nonly if --create-namespace"]

    ENS --> S2["Step 2 · inventory.ComputeManifestDigest()\nSHA256 of rendered manifests"]
    S2  --> S3["Step 3 · inventory.ComputeChangeID()\nSHA1(path + version + values + digest)"]
    S3  --> S4["Step 4 · inventory.GetInventory()\nread previous inventory Secret from cluster"]
    S4  --> S5["Step 5 · Stale detection\nComputeStaleSet + ComponentRenameSafetyCheck + PreApplyExistenceCheck"]
    S5  --> S6["Step 6 · kubernetes.Apply()\nSSA Patch for each resource in weight order"]
    S6  --> S7["Step 7 · inventory.PruneStaleResources()\ndelete resources absent from current render"]
    S7  --> S8["Step 8 · inventory.WriteInventory()\nwrite new change entry to inventory Secret"]
```

| | |
|---|---|
| **Packages** | `config`, `cmdutil`, `pipeline`, `loader`, `builder`, `core/*`, `inventory`, `kubernetes`, `output` |
| **Cluster** | connected — read + write |

---

### `opm mod diff`

Render + semantic cluster comparison — no writes.

```mermaid
flowchart TD
    CMD(["opm mod diff"])

    CMD --> RK["config.ResolveKubernetes()\nkubeconfig + context + namespace + provider"]
    RK  --> RR["cmdutil.RenderRelease() — all 5 render phases\nNOTE: ShowRenderOutput is NOT called here"]
    RR  --> K8S["cmdutil.NewK8sClient()"]
    K8S --> GI["inventory.GetInventory()\nread inventory for orphan detection"]
    GI  --> DR["inventory.DiscoverResourcesFromInventory()\nfetch live resources from inventory"]
    DR  --> DEC{render errors?}

    DEC -->|"yes"| DPP["kubernetes.DiffPartial()\nbest-effort diff on partially rendered resources"]
    DEC -->|"no"| DP["kubernetes.Diff()\nfull semantic diff via dyff"]

    DPP & DP --> OUT["print diff output\n--- modified   +++ added   ~~~ orphaned"]
```

| | |
|---|---|
| **Packages** | `config`, `cmdutil`, `pipeline`, `loader`, `builder`, `core/*`, `inventory`, `kubernetes`, `output` |
| **Cluster** | connected — read only |

---

### `opm mod delete`

Inventory-first — no render pipeline involved.

```mermaid
flowchart TD
    CMD(["opm mod delete"])

    CMD    --> VAL["validate: --release-name or --release-id required"]
    VAL    --> RK["config.ResolveKubernetes()\nkubeconfig + context + namespace — no provider needed"]
    RK     --> K8S["cmdutil.NewK8sClient()"]
    K8S    --> CONF{--force?}

    CONF -->|"no"| PROMPT["confirmDelete() — interactive prompt"]
    CONF -->|"yes"| RI

    PROMPT --> RI["cmdutil.ResolveInventory()\nGetInventory OR FindInventoryByReleaseName\n+ DiscoverResourcesFromInventory"]
    RI     --> DEL["kubernetes.Delete()\ndelete all resources in reverse weight order\ninventory Secret deleted automatically"]
```

| | |
|---|---|
| **Packages** | `config`, `cmdutil`, `inventory`, `kubernetes`, `output` |
| **Cluster** | connected — read + write |
| **Pipeline** | not used — no module source needed |

---

### `opm mod status`

Inventory-first — no render pipeline involved.

```mermaid
flowchart TD
    CMD(["opm mod status"])

    CMD --> VAL["validate: --release-name or --release-id required"]
    VAL --> RK["config.ResolveKubernetes()"]
    RK  --> K8S["cmdutil.NewK8sClient()"]
    K8S --> RI["cmdutil.ResolveInventory()\nGetInventory OR FindInventoryByReleaseName\n+ DiscoverResourcesFromInventory"]
    RI  --> GRS["kubernetes.GetReleaseStatus()\nhealth check per resource type"]

    GRS --> WATCH{--watch?}

    WATCH -->|"no"| FS["kubernetes.FormatStatus()\ntable / yaml / json"]
    WATCH -->|"yes"| WM["runStatusWatch()\npoll every 2s + clear screen"]
```

| | |
|---|---|
| **Packages** | `config`, `cmdutil`, `inventory`, `kubernetes`, `output` |
| **Cluster** | connected — read only |
| **Pipeline** | not used — no module source needed |

---

## Shared Infrastructure

The following utilities are reused across commands without duplication.

| Utility | Used by | What it does |
|---|---|---|
| `config.ResolveKubernetes()`<br>`internal/config/resolver.go` | all five commands | Resolve kubeconfig / context / namespace / provider via precedence: flag > env > config > default |
| `cmdutil.RenderRelease()`<br>`internal/cmdutil/render.go` | build, apply, diff | Thin wrapper: resolve module path, build RenderOptions, call pipeline.Render, handle fatal errors |
| `cmdutil.ShowRenderOutput()`<br>`internal/cmdutil/render.go` | build, apply (not diff) | Check render errors, print transformer match log, emit warnings |
| `cmdutil.NewK8sClient()`<br>`internal/cmdutil/k8s.go` | apply, diff, delete, status | Create Kubernetes dynamic client from pre-resolved kubeconfig + context |
| `cmdutil.ResolveInventory()`<br>`internal/cmdutil/inventory.go` | delete, status | Lookup inventory by name or id, discover live resources from it |
| `cmdutil.RenderFlags`<br>`internal/cmdutil/flags.go` | build, apply, diff | `-f/--values`, `-n/--namespace`, `--provider`, `--release-name` |
| `cmdutil.K8sFlags`<br>`internal/cmdutil/flags.go` | apply, diff, delete, status | `--kubeconfig`, `--context` |
| `cmdutil.ReleaseSelectorFlags`<br>`internal/cmdutil/flags.go` | delete, status | `--release-name`, `--release-id`, `--namespace` |

---

## Package Dependency Map

```mermaid
graph TD
    ENTRY["cmd/opm/\nmain.go · root.go"]
    MOD["internal/cmd/mod/\napply · build · diff · delete · status"]
    CMDUTIL["internal/cmdutil/\nrender · inventory · k8s · flags · output"]
    PIPELINE["internal/pipeline/\npipeline.go"]
    LOADER["internal/loader/\nmodule.go · provider.go"]
    BUILDER["internal/builder/\nbuilder.go"]
    CORE["internal/core/\nmodule · modulerelease · component\nprovider · transformer · resource"]
    INVENTORY["internal/inventory/\ncrud · discover · stale · digest · changeid"]
    K8S["internal/kubernetes/\napply · delete · diff · status · client"]
    CONFIG["internal/config/\nconfig.go · resolver.go"]
    OUTPUT["internal/output/\nmanifest · table · log · styles"]

    ENTRY    --> MOD
    MOD      --> CMDUTIL
    MOD      --> CONFIG
    CMDUTIL  --> PIPELINE
    CMDUTIL  --> INVENTORY
    CMDUTIL  --> K8S
    CMDUTIL  --> OUTPUT
    PIPELINE --> LOADER
    PIPELINE --> BUILDER
    PIPELINE --> CORE
    PIPELINE --> INVENTORY
    LOADER   --> CORE
    BUILDER  --> CORE
    INVENTORY --> K8S
    INVENTORY --> CORE
    K8S      --> OUTPUT
```

---

## Notable Design Decisions

**`diff` skips `ShowRenderOutput`**

`apply` and `build` call `cmdutil.ShowRenderOutput()` which returns an error
(and stops) if the render has errors. `diff` intentionally does not — it calls
`kubernetes.DiffPartial()` instead to show a best-effort diff for the resources
that did render successfully, even when some components failed. The comment in
`diff.go:77-78` documents this explicitly.

**Single CUE context per command**

The same `*cue.Context` flows from `config.GlobalConfig.CueContext` into
`pipeline.NewPipeline()`, through `loader.LoadModule()`, `builder.Build()`, and
into `matchPlan.Execute()`. All CUE values (`mod.Raw`, `releaseSchema`,
`transformValue`, etc.) are bound to this context — mixing contexts causes a
runtime panic. This is why the config creates one fresh context per command
invocation (not a global singleton).

**Execute is sequential, not concurrent**

`matchPlan.Execute()` runs transformer matches one at a time. `*cue.Context` is
not safe for concurrent use, so parallelism is not possible here without
multiple contexts. The match plan iteration order is deterministic (components
and transformers are sorted alphabetically before matching).

**Inventory is the source of truth for cluster-side commands**

`delete` and `status` never read the module source. They recover all needed
information from the inventory Secret stored on the cluster. This means a
release can be deleted or inspected even after the module source is gone or
unavailable.

**Stale detection is manifest-only, not cluster-based**

The stale set is computed by diffing the previous inventory entries against the
current render output — no cluster GET is required for this step. This makes
stale detection fast and offline, but it requires the inventory to stay
consistent with what's actually on the cluster.

**Values isolation**

`values.cue` is loaded separately via `ctx.CompileBytes` and never unified into
`mod.Raw`. This keeps the default module definition clean. Extra `values_*.cue`
files in the module directory are silently excluded from the package load and
reported via DEBUG — they must be passed explicitly via `--values` to take
effect.
