## Context

OPM currently operates as a CLI tool: `opm mod apply` runs a render pipeline (CUE load → build → match → generate) and applies the resulting Kubernetes resources via Server-Side Apply. There is no in-cluster controller — the CLI must be re-run manually whenever the desired state changes.

Metacontroller is a Kubernetes add-on that lets you write custom controllers as simple webhook servers. You define a `CompositeController` (parent CRD → child resources) and provide a sync webhook. Metacontroller handles the watch/reconcile loop, child ownership via ControllerRef, garbage collection, and apply. Your webhook is a pure function: receive observed state, return desired state.

The `minecraft-java` OPM module produces: StatefulSet, Service, Secret, PVC — exactly the pattern CompositeController is designed for.

This experiment is **completely detached** from the CLI codebase. It lives under `experiments/metacontroller/` with its own `go.mod` and does not import any `internal/` packages. It reimplements the necessary CUE evaluation pipeline directly against `cuelang.org/go`.

## Goals / Non-Goals

**Goals:**
- Prove that Metacontroller can serve as a reconciliation engine for OPM module output
- Build a working `MinecraftServer` CRD → webhook → StatefulSet/Service/Secret/PVC flow
- Demonstrate the reconcile loop pattern: spec translation → CUE evaluation → child generation → status computation
- Keep the experiment self-contained (own go.mod, own binary, no CLI imports)
- Validate CUE's `load.Config.Overlay` for in-memory module loading

**Non-Goals:**
- General-purpose OPM operator (arbitrary modules, ModuleRelease CRD)
- Full minecraft-java config surface (all 11 server types, backup, all storage options)
- Production readiness (HA, metrics, RBAC hardening, upgrade testing)
- Modifying any existing CLI code or internal packages
- Supporting Metacontroller's DecoratorController pattern

## Decisions

### 1. Standalone Go project with own go.mod

**Decision:** The experiment gets its own `go.mod` under `experiments/metacontroller/`, depending only on `cuelang.org/go` and `k8s.io/apimachinery`.

**Why:** The CLI's `internal/` packages are tightly coupled to the full pipeline (config loading from `~/.opm/config.cue`, provider loading from evaluated CUE config, filesystem-based module loading). Importing them would require either: (a) modifying internal APIs to support the webhook use case, or (b) carrying the full CLI dependency tree into the webhook binary. Both violate the experiment's purpose of being lightweight and self-contained.

**Alternative considered:** Import `internal/loader`, `internal/builder`, `internal/pipeline` directly. Rejected because those packages assume filesystem paths for both modules and values, require a `GlobalConfig` struct populated from disk, and would make the experiment dependent on CLI release cycles.

### 2. CUE evaluation via load.Config.Overlay

**Decision:** Embed the `minecraft-java` module files via `//go:embed` and load them through CUE's `load.Config.Overlay` mechanism — a `map[string]load.Source` that provides virtual filesystem entries to the CUE loader.

**Why:** Avoids writing module files to a temp directory at startup. The Overlay is built once from the embedded FS and reused across all requests. Only the per-request values file needs to touch disk (written to a temp file because CUE's `load.Instances` API requires file paths for overlay entries or real files for non-overlay loading).

**Alternative considered:** Write embedded files to a real temp dir at startup. Simpler but operationally worse — relies on writable temp filesystem, creates cleanup concerns on crash. Overlay is the CUE-native solution.

### 3. Provider definition embedded as CUE string

**Decision:** The Kubernetes provider definition (transformer CUE code) is embedded as a Go string constant, compiled at startup via `ctx.CompileString()`, and passed as a `cue.Value` to the transformer matching/execution code.

**Why:** `LoadProvider` in the CLI operates on `map[string]cue.Value` — it never touches disk. We can construct this map directly without any config file. This means zero disk I/O for provider loading.

**Alternative considered:** ConfigMap-mounted provider config. More flexible but adds deployment complexity for an experiment.

### 4. Fresh cue.Context per request

**Decision:** Each sync webhook call creates a new `cue.Context`, loads the module (from overlay), compiles the provider, and runs the full evaluation pipeline.

**Why:** CUE contexts accumulate memory from evaluated values and are not documented as goroutine-safe. The OPM CLI itself uses "fresh CUE context per command" as an explicit design pattern. For a webhook handling concurrent requests, fresh-per-request is the safe choice.

**Alternative considered:** Cache the loaded module across requests. Risky — `cue.Value` is tied to its parent `cue.Context`, so a cached module value from context A can't be mixed with a fresh context B. Would require careful lifecycle management.

### 5. Preserve children on render failure

**Decision:** When the CUE render pipeline fails (invalid spec, evaluation error), the webhook returns HTTP 200 with the observed children echoed back as desired state and an error status on the CR.

**Why:** Returning `children: []` on error would cause Metacontroller to delete all running resources — a bad spec change would take down the server. Returning HTTP 500 would trigger Metacontroller retries but not surface the error in the CR status. Echoing observed children back preserves the running state while surfacing the error.

### 6. Immediate finalize (no ordered teardown)

**Decision:** The finalize hook returns `finalized: true, children: []` immediately.

**Why:** The StatefulSet already has `terminationGracePeriodSeconds: 60`, which handles graceful JVM shutdown. Metacontroller deletes children in its own order. Ordered teardown (e.g., wait for StatefulSet deletion before PVC deletion) adds complexity with no real benefit in phase 1 — PVCs with `Retain` reclaim policy survive pod deletion anyway.

### 7. Phase 1 scope: Paper server, minimal spec

**Decision:** The `MinecraftServer` CRD covers only: server version, JVM memory, basic server properties (motd, maxPlayers, difficulty, mode, pvp, onlineMode), service type, and storage type (emptyDir or PVC). No RCON, no backup, no alternate server types.

**Why:** The goal is proving the Metacontroller integration, not reimplementing the full minecraft-java config surface. The minimal spec exercises all the critical paths: CUE evaluation, StatefulSet generation, Service generation, conditional PVC generation.

## Risks / Trade-offs

- **[CUE evaluation latency]** → Each sync call runs full CUE load + evaluate. If this takes >1s, Metacontroller's 10s webhook timeout leaves little headroom for slow clusters. Mitigation: benchmark early; the module is small (~500 lines of CUE), so evaluation should be fast.

- **[CUE Overlay compatibility]** → `load.Config.Overlay` is used for virtual filesystem entries, but the minecraft-java module's `cue.mod/` tree includes registry dependencies. Mitigation: embed the full `cue.mod/` tree (including vendor cache) so all dependencies resolve locally.

- **[Temp file per request for values]** → CUE's `load.Instances` requires file paths; there's no in-memory values injection API. Mitigation: a single small temp file per request (~30 lines), cleaned up immediately via `defer`. Negligible I/O.

- **[Metacontroller SSA behavior]** → Metacontroller uses 3-way merge by default, not Server-Side Apply. Field ownership may differ from the CLI's SSA approach. Mitigation: Metacontroller has added SSA support; test both paths.

- **[No inventory tracking]** → The CLI uses an inventory Secret for state tracking. The webhook has no equivalent — Metacontroller's ControllerRef-based ownership replaces it. This means no change history, no digest-based idempotency. Acceptable for an experiment.
