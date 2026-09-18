## Why

The cluster arm of platform resolution reads the Platform CR's `spec` and generates a platform module from its subscriptions alone. Since opm-operator `registration-driven-regeneration` (0015 D13, D17) the package the operator renders against is a function of the spec **plus** the active `TransformerRegistration` claims, and the operator records that resolved union on `status.registry` beside `status.packageIdentity`. So `opm module apply`, `opm instance apply` and `opm instance diff` render today against a platform missing every registered provider catalog: a module demanding a provider-fulfilled contract renders in-cluster and fails on the laptop, and `diff` compares against the wrong package. Enhancement 0015 D6's answer, reproducing a cluster's render by pulling the operator-generated package, was marked blocked because the package "never leaves the operator's disk". It does not need to: the CLI already runs the same library generator on the same closure derivation under the same module path, so a module regenerated from the CR's effective inputs is the operator's package, not a reconstruction of it. What was missing is reading the right inputs and a command that hands the result to the user.

## What Changes

- **The cluster arm reads the effective registry.** The cluster getter returns the Platform document (spec and status), and resolution generates from `status.registry` (catalog, version, enabled, as the operator resolved them from subscriptions and active claims) whenever the operator has recorded one, falling back to `spec.registry` only when it has not (a solo cluster, or an operator predating the field). The provenance line names which, and the recorded `status.packageIdentity`. Two warnings, never silent swaps: when `status.observedGeneration` lags `metadata.generation` the effective package is older than the spec being edited; when the Platform's `Ready` is `False` the effective package is the last good one and the message names the operator's reason.
- **`opm platform pull <dir>` (new).** Reads the cluster Platform through the same resolution, generates the module and writes it to `<dir>`: a build-local platform module a local `opm module build --platform <dir>` consumes, so the laptop reproduces the cluster's render (0015 D6). Prints the CR generation, the package identity, the operator version and every registry entry with its source. Refuses a non-empty target directory unless `--force`. It writes nothing to the cluster and publishes nothing (0019 D6's no-publish rule is about registries; a directory the user asked for is neither).
- **The `platform` group's help stops claiming it never contacts a cluster.** `check` stays offline; `pull` is cluster-facing and says so.
- **Not in this change.** A CLI-side gate on `routable` or `discriminated` for the cluster arm: `status.registry` describes only a package the operator already gated, so regenerating from it cannot produce an ungated platform. A gate for a local `--platform` directory: `opm platform check` is that pre-flight. Byte-level verification against the operator's package: the identity on status covers generation and claims, not the core pin the library carries, so exactness holds when CLI and operator embed the same library release; a content digest on Platform status is the operator follow-up that would make it checkable, recorded in design.md. `platform check --cluster`: pull the module, then check the directory.

## Impact

- **Affected commands and packages:** `opm module apply`, `opm instance apply`, `opm instance diff` (cluster resolution semantics; no flags change); new `opm platform pull`; `internal/platform` (`cluster.go`, `spec.go`, `resolve.go`), `internal/workflow/render` (getter type), `internal/cmd/platform/` (new `pull.go`, group help), `internal/cmd/{module,instance}` (getter constructor call sites).
- **Behaviour change:** on a cluster with active claims, the three cluster-facing commands now render against the provider catalogs too, which is what the operator renders against; on a cluster without claims the generated module is byte-identical to today's. A stale or refused Platform now warns where it was silent.
- **SemVer:** MINOR. One new command, one new flag with a safe default (`--force=false`), no removed surface. The `ClusterSpecGetter` type changes shape inside `internal/`.
- **Complexity (Principle VII):** one decoder over a status list the operator already writes, one provenance field, one thin command over the existing generator. Justified by the correctness gap: the cluster arm renders against the wrong package today.
- **Enhancement record:** 0015 D6 needs a Revised line saying "retrieve" means regenerate from the CR's effective inputs; that edit lives in `enhancements/` and is flagged, not made here.

## Capabilities

### New Capabilities

- `platform-pull`: what `opm platform pull` writes, prints and refuses, and the reproduction property it delivers.

### Modified Capabilities

- `platform-resolution`: the cluster arm generates from the effective registry the operator recorded, with spec as the fallback; provenance names the source and the package identity; a stale or not-Ready Platform warns.
