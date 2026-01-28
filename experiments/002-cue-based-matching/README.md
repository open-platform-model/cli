# Experiment 002: CUE-Based Matching Logic

This experiment implements Phase 3 of the Render Pipeline (Component Analysis & Matching) entirely in CUE, with comprehensive coverage of all 6 blueprint types and their corresponding Kubernetes transformers.

## Goal

Validate if the complex matching logic required by the OPM Render Specification (013-cli-render-spec) can be expressed declaratively in CUE, allowing providers to be self-contained regarding their matching rules. This experiment tests the full render pipeline from blueprints → components → matching → transformation → Kubernetes resources.

## Architecture

### Core Infrastructure (unchanged)

1. **`matching.cue`**: Defines the `#Matches` predicate that checks if a component satisfies a transformer's:
   - `requiredLabels` (must match exactly)
   - `requiredResources` (must be present via FQN)
   - `requiredTraits` (must be present via FQN)

2. **`provider_extension.cue`**: Extends `core.#Provider` with:
   - `#matchedTransformers` - builds the execution plan via CUE comprehensions
   - `#rendered` - produces the final Kubernetes resources

### Test Implementation

3. **`components.cue`**: Defines 7 components using all 6 blueprint types:

   | Component | Blueprint | Workload Label | Description |
   |---|---|---|---|
   | `web` | `#StatelessWorkload` | `stateless` | Nginx web server with Expose trait |
   | `api` | `#StatelessWorkload` | `stateless` | API server without Expose (tests trait differentiation) |
   | `database` | `#SimpleDatabase` | `stateful` | Postgres database with persistence |
   | `cache` | `#StatefulWorkload` | `stateful` | Redis cache with volumes |
   | `log-agent` | `#DaemonWorkload` | `daemon` | Node exporter for logging |
   | `migration` | `#TaskWorkload` | `task` | One-time migration job |
   | `backup` | `#ScheduledTaskWorkload` | `scheduled-task` | Cron-based backup job |

4. **`basic_module.cue`**: Defines `AllBlueprintsModule` that:
   - Composes all 7 components
   - Provides value schemas for configuration
   - Includes concrete default values
   - Has a `ModuleRelease` targeting the `production` namespace

5. **`demo.cue`**: Defines a provider with 6 transformers:

   | Transformer | K8s Kind | Matching Criteria |
   |---|---|---|
   | `deployment` | `Deployment` | `requiredLabels: "core.opm.dev/workload-type": "stateless"` + Container resource |
   | `statefulset` | `StatefulSet` | `requiredLabels: "core.opm.dev/workload-type": "stateful"` + Container resource |
   | `daemonset` | `DaemonSet` | `requiredLabels: "core.opm.dev/workload-type": "daemon"` + Container resource |
   | `job` | `Job` | `requiredLabels: "core.opm.dev/workload-type": "task"` + Container resource |
   | `cronjob` | `CronJob` | `requiredLabels: "core.opm.dev/workload-type": "scheduled-task"` + Container resource |
   | `service` | `Service` | `requiredTraits: "opm.dev/traits/networking@v0#Expose"` |

## How to Run

Ensure you have the CUE CLI installed.

```bash
# Set cache directory
export CUE_CACHE_DIR=../../../.cue-cache/

# Validate all CUE definitions
cue vet .

# View the matching results (execution plan)
cue eval . -e 'provider.#matchedTransformers'

# View all rendered Kubernetes resources
cue eval . -e result

# View YAML output
cue eval . -e yaml
```

## Results

**✅ Experiment successful**. The system renders **8 Kubernetes resources** from the 7 components:

- `deployment/web` → Deployment (nginx, 2 replicas)
- `deployment/api` → Deployment (api-server, 3 replicas)
- `statefulset/database` → StatefulSet (postgres:14.5, PVC 20Gi)
- `statefulset/cache` → StatefulSet (redis:7.0, PVC 5Gi)
- `daemonset/log-agent` → DaemonSet (node-exporter)
- `job/migration` → Job (migrations:v2.0.0)
- `cronjob/backup` → CronJob (postgres backup at 2am daily)
- `service/web` → Service (ClusterIP, port 80)

### Key Validations

1. **Blueprint Composition**: All 6 blueprint types correctly compose resources and traits into components.

2. **Label-Based Matching**: The `#Matches` predicate correctly identifies which transformers apply to which components based on `core.opm.dev/workload-type` labels.

3. **Trait-Based Matching**: The Service transformer correctly matches only the `web` component (via the Expose trait), not the `api` component (which is also stateless but lacks Expose).

4. **Transformer Execution**: Each transformer produces valid Kubernetes resources with:
   - Correct metadata (name, namespace, labels)
   - Proper spec fields (replicas, containers, volumes, schedules)
   - Appropriate context propagation (module labels, component labels, controller labels)

5. **Scope Management**: CUE's scoping rules handle the complex cross-referencing between transformers, components, blueprints, resources, and traits without external Go code.

6. **Type Safety**: All definitions validate successfully via `cue vet`, ensuring type correctness at definition time.

## Conclusions

This experiment demonstrates that:

1. **CUE is sufficient** for implementing the complete OPM render pipeline declaratively.
2. **Blueprints work** as a composition mechanism for standardizing common workload patterns.
3. **Label-based matching** provides a clean, extensible way to connect components to transformers.
4. **Trait-based differentiation** allows fine-grained control over which components receive additional transformations (e.g., Service creation).
5. **The CLI can be simple** - just load the Provider + Module and invoke `cue eval` to get the execution plan and rendered resources.

### Next Steps

- Benchmark performance with larger module sizes (100+ components)
- Implement Policy matching (Phase 2.5 of the render pipeline)
- Add Scope evaluation for multi-component coordination
- Extend to support Bundle rendering (multiple modules)
