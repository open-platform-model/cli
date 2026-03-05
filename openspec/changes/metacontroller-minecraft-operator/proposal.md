## Why

Validate whether Metacontroller can serve as a lightweight, in-cluster reconciliation engine for OPM modules — turning CLI-driven `opm mod apply` into a proper Kubernetes operator without writing a full controller in Go. The Minecraft module family provides a concrete, non-trivial test case with StatefulSets, Services, Secrets, and PVCs.

This is an **experiment** (`experiments/metacontroller/`), completely detached from the CLI codebase. It does not import or modify any `internal/` packages. The goal is to prove the architecture, not ship production code.

## What Changes

- New standalone Go project under `experiments/metacontroller/` with its own `go.mod`
- Implements a Metacontroller `CompositeController` sync webhook for the `minecraft-java` module
- Defines a `MinecraftServer` CRD (v1alpha1) whose spec mirrors a subset of the minecraft-java `#config` schema
- Embeds the `minecraft-java` CUE module files and a provider definition via `//go:embed`
- Uses `cuelang.org/go` directly (same version as CLI) to evaluate CUE and produce Kubernetes resources
- Includes Kubernetes manifests for deploying: CRD, CompositeController, webhook Deployment+Service
- Includes a sample `MinecraftServer` CR for testing

## Capabilities

### New Capabilities

- `metacontroller-webhook`: The sync/finalize webhook server — translates a MinecraftServer CR spec into rendered Kubernetes children via CUE evaluation. Covers request parsing, spec-to-CUE translation, CUE pipeline execution, error handling (preserve children on failure), status computation from observed children, and finalize (immediate teardown).
- `metacontroller-manifests`: Kubernetes manifests for the experiment — CRD definition, CompositeController resource, webhook Deployment+Service, and sample MinecraftServer CR.

### Modified Capabilities
<!-- None — this is a standalone experiment that doesn't touch existing CLI code or specs -->

## Impact

- **Code**: New standalone Go project under `experiments/metacontroller/`. Zero imports from `internal/`. Own `go.mod`, own binary.
- **Dependencies**: `cuelang.org/go` (CUE evaluation), `k8s.io/apimachinery` (unstructured types for JSON marshalling). No cobra, no charmbracelet, no OPM CLI dependencies.
- **Cluster requirements**: Metacontroller must be installed in the target cluster. The experiment includes install instructions but not the Metacontroller deployment itself.
- **Existing code**: No changes. No modifications to `internal/`, `cmd/`, or any existing packages.
- **SemVer**: N/A — this is an experiment, not a release artifact.
