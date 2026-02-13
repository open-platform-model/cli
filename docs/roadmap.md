# OPM Roadmap

| Field       | Value            |
|-------------|------------------|
| **Status**  | Draft            |
| **Created** | 2026-02-13       |
| **Authors** | OPM Contributors |

## Table of Contents

1. [Current Status](#1-current-status)
2. [Phase 1: Application Model and CLI](#2-phase-1-application-model-and-cli)
3. [Phase 2: Kubernetes Controller](#3-phase-2-kubernetes-controller)
4. [Phase 3: Platform Model](#4-phase-3-platform-model)

---

## 1. Current Status

OPM's CLI foundation has been built over the first iteration of development. The
following capabilities are working today:

**Core definitions and rendering:**

- CUE-native definitions: Module, ModuleRelease, Component, Resource, Trait, Policy
- Catalog of definitions: 5 resources, 20+ traits, 5 blueprints
- 6-phase render pipeline with AST-based overlay generation
- Kubernetes Provider with 12 transformers (Deployment, StatefulSet, DaemonSet, Job,
  CronJob, Service, PVC, ConfigMap, Secret, ServiceAccount, HPA, Ingress)

**CLI commands:**

- `opm mod init` — scaffold a new module (simple, standard, advanced templates)
- `opm mod build` — render Kubernetes manifests from CUE definitions
- `opm mod apply` — server-side apply to a cluster with weighted ordering
- `opm mod delete` — label-based discovery with `--name` or `--release-id`
- `opm mod diff` — semantic diff of local vs live state (via dyff)
- `opm mod status` — health and readiness status (table/yaml/json, watch mode)
- `opm config init` — initialize `~/.opm/config.cue`
- `opm config vet` / `opm mod vet` — validation stubs (native rewrite planned)

**Infrastructure:**

- Deterministic release identity via UUID v5 labels
- CUE-native configuration with full precedence chain (Flag > Env > Config > Default)
- Provider auto-resolution when a single provider is configured
- Styled terminal output with charmbracelet/log and lipgloss
- CUE registry integration (`opmodel.dev`)
- OCI distribution design
- Kind cluster lifecycle tasks for local development

**RFCs authored (all Draft):**

- [RFC-0001: Release Inventory](rfc/0001-release-inventory.md) — Secret-based resource tracking
- [RFC-0002: Sensitive Data Model](rfc/0002-sensitive-data-model.md) — first-class `#Secret` type
- [RFC-0003: Immutable Config](rfc/0003-immutable-config.md) — content-hash suffixed ConfigMaps/Secrets
- [RFC-0004: Interface Architecture](rfc/0004-interface-architecture.md) — `provides`/`requires` contracts
- [RFC-0005: Env & Config Wiring](rfc/0005-env-config-wiring.md) — `#EnvVarSchema` and secret output paths

---

## 2. Phase 1: Application Model and CLI

Full lifecycle management of applications using the CLI. This is the current focus.
Everything in this phase must be solid before the controller (Phase 2) can build on it.

### Dependency Graph

The milestones within Phase 1 have the following dependency relationships. M1 is the
foundation — M2, M3, and M4 can proceed in parallel once M1 is complete, though each
has internal dependencies shown below.

```text
M1: CLI Stability & Validation
 │
 ├──► M2: Secrets & Config Lifecycle
 │     │
 │     ├── RFC-0001 (Release Inventory)
 │     ├── RFC-0002 (Sensitive Data)
 │     │     └──► RFC-0005 (Env & Config Wiring)
 │     └── RFC-0003 (Immutable Config)
 │           └──► RFC-0001 (for GC of old immutables)
 │
 ├──► M3: Distribution
 │     │
 │     ├── config-registries-v1 (MAJOR)
 │     ├── distribution-v1
 │     │     └──► config-registries-v1
 │     └── templates-v2
 │           └──► distribution-v1
 │
 └──► M4: Rendering Pipeline Maturity
       │
       ├── transformer-matching-v2
       ├── RFC-0004 (Interface Architecture)
       │     └──► transformer-matching-v2
       └── Policy definitions
```

### M1: CLI Stability and Validation

The foundation must be solid before adding new capabilities. This milestone focuses
on replacing stubs with real implementations, fixing correctness bugs, and filling
gaps in the existing command surface.

**Major deliverables:**

- **Native validation** — Replace the stub `opm mod vet` and `opm config vet`
  commands (which currently shell out to `cue vet`) with native Go CUE SDK
  implementations. 4-phase validation pipeline with custom error formatting,
  entity summaries, `--debug`/`--values`/`--package`/`--concrete` flags.
  *(openspec change: validation-v1)*

- **`#ModuleRelease.values` validation against `#Module.#config`** — During
  processing, validate that the values provided in a ModuleRelease satisfy the
  schema defined by the Module's `#config`. Leverage CUE's evaluator to support
  mandatory (`!`), optional (`?`), and default (`*`) fields. Error output must
  reference the correct file paths and line numbers relative to the execution
  directory.

- **Atomic apply** — An incorrectly configured module must not partially apply.
  Today, `opm mod apply` can create some resources before failing on others,
  leaving the cluster in an inconsistent state. All resources must be validated
  before any are applied. *(bug fix)*

- **Unchanged apply detection** — When `opm mod apply` is run against an already-
  applied module with no changes, the output should indicate that no changes were
  made instead of displaying "Module applied." *(bug fix)*

**Additional deliverables:**

- `opm mod eval` — print the raw CUE evaluation of a module
- `opm mod list` — list deployed modules in a namespace (`-A` for all namespaces),
  leveraging `release-id` labels for discovery
- `--ignore-not-found` flag for `opm mod delete` and `opm mod status` for
  idempotent operations (exit 0 when no resources match)
- `--create-namespace` flag for `opm mod apply`
- Remove `injectLabels()` — redundant with transformer-based label injection
- Archive completed openspec changes: diff-status-v1, improve-k8s-api-warnings

### M2: Secrets and Config Lifecycle

Full lifecycle management of configuration and sensitive data. This is the critical
path for making OPM usable for real applications — without secrets support, every
non-trivial application requires manual workarounds.

**Major deliverables:**

- **Release Inventory** — A lightweight Kubernetes Secret that tracks which resources
  belong to a ModuleRelease. Enables automatic pruning of stale resources during
  `opm mod apply` and provides a precise source of truth for `diff`, `delete`, and
  `status`. Maintains change history for future rollback.
  *([RFC-0001](rfc/0001-release-inventory.md))*

- **Sensitive Data Model** — Introduce `#Secret` as a first-class type that tags
  fields as sensitive at the schema level. This single annotation propagates through
  every layer — module definition, release fulfillment, transformer output — enabling
  the toolchain to redact, encrypt, and dispatch secrets to platform-appropriate
  resources (K8s Secrets, ExternalSecrets, CSI volumes). Supports literal values,
  external references, and CLI `@` tag injection.
  *([RFC-0002](rfc/0002-sensitive-data-model.md))*

- **Environment and Config Wiring** — Full `#EnvVarSchema` with four source types
  (literal, configMapKeyRef, secretKeyRef, fieldRef), bulk injection via `envFrom`,
  volume-mounted secrets, and auto-discovery of secrets from `#config`. This is the
  output side of RFC-0002's data flow.
  *([RFC-0005](rfc/0005-env-config-wiring.md))*

- **Immutable ConfigMaps and Secrets** — When `immutable: true` is set, the
  transformer appends a content-hash suffix to the resource name and sets
  `spec.immutable: true`. Content changes produce a new name, triggering workload
  rolling updates. Old resources are garbage collected via the Release Inventory
  (RFC-0001). The OPM equivalent of Kustomize's `configMapGenerator`.
  *([RFC-0003](rfc/0003-immutable-config.md))*

### M3: Distribution

OCI-native module distribution — the ability to publish, discover, and consume
modules from registries. This makes OPM usable as a package manager, not just a
local build tool.

**Major deliverables:**

- **Module distribution** — `opm mod publish` (push module to OCI registry),
  `opm mod get` (pull module), `opm mod update` (update dependencies),
  `opm mod tidy` (tidy CUE module dependencies without the external `cue` binary).
  Uses oras-go for OCI, Docker `config.json` for auth. Strict SemVer only — no
  `@latest`.
  *(openspec change: distribution-v1)*

- **Template distribution** — `opm template list`, `get`, `show`, `validate`,
  `publish`. Replaces the V1 embedded templates with registry-distributed templates.
  V1 templates remain during transition.
  *(openspec change: templates-v2)*

**Additional deliverables:**

- `opm config update` — extract current values, initialize latest config schema,
  reapply values. Helps users upgrade configuration across breaking changes.
- CUE-native CRD vendor — import Kubernetes CRDs into CUE using `cue import openapi`,
  similar to `timoni vendor crd`
- Rework tests to use CUE AST and pure CUE files for test data and comparison,
  eliminating string-based test fixtures

### M4: Rendering Pipeline Maturity

The rendering pipeline is the heart of OPM. Every future capability — the controller,
the platform registry, multi-provider rendering — depends on a deterministic,
composable, well-tested pipeline. This milestone hardens the pipeline and adds the
extensibility required for Phase 2.

**Major deliverables:**

- **Pluggable transformer matching** — Replace the hardcoded Go matcher with a
  `ComponentMatcher` interface and CUE `#Matches` predicates. Providers define
  matching logic declaratively in CUE. Fallback to Go matcher when no `#Matches`
  is defined. This enables provider extensibility without Go code changes.
  *(openspec change: transformer-matching-v2)*

- **Interface Architecture** — `provides`/`requires` model for typed contracts
  between components. Transforms OPM from a deployment tool into an application
  description language. Service communication, data dependencies, and infrastructure
  requirements expressed as typed contracts.
  *([RFC-0004](rfc/0004-interface-architecture.md))*

- **Policy definitions** — First-class `#Policy` / `#PolicyRule` enforcement with
  block/warn/audit semantics across the rendering pipeline. Policies are evaluated
  at render time, ensuring violations are caught before resources reach the cluster.

- **Staged apply and delete** — Wait for resource readiness before proceeding to
  dependent resources. Investigate whether staging should be configurable in the
  model (as a Policy or per-component).

**Milestone exit criteria:** The rendering pipeline must be deterministic (same input
always produces same output), composable (modules compose without implicit coupling),
and thoroughly tested. This is the stable foundation that Phase 2's controller will
embed.

---

## 3. Phase 2: Kubernetes Controller

The same Module lifecycle as the CLI, delivered through an in-cluster controller and
a custom resource. This phase marks the transition from developer tooling to platform
infrastructure. The controller uses the same CUE rendering pipeline as the CLI, so
behaviour is identical whether you apply from your laptop or the controller reconciles
in-cluster.

### M5: In-Cluster Controller

The core controller that watches ModuleRelease custom resources and reconciles them.

**Preliminary deliverables:**

- ModuleRelease CRD definition and controller scaffolding
- Embed the Phase 1 rendering pipeline — same CUE evaluation, same transformer
  matching, same output
- Continuous reconciliation — drift detection and re-apply to maintain desired state
- Controller-based status reporting via CRD status subresource and Kubernetes events
- Multi-cluster topology — target multiple clusters and namespaces from a single
  ModuleRelease definition, with override policies for per-cluster customisation

### M6: Platform Registry

A curated catalog of providers and modules, managed by Platform Operators, ready for
consumption by end-users.

**Preliminary deliverables:**

- PlatformRegistry as a CUE-defined catalog — declares which providers handle which
  capabilities
- Provider and module curation — Platform Operators select and approve modules for
  their environment
- Platform-level policy enforcement — organisational policies applied across all
  deployments
- Discovery — "What can I deploy? Which providers support it?"
- Beginning of the Platform Model — the ability to describe the whole platform, not
  just individual applications

---

## 4. Phase 3: Platform Model

The full realisation of the multi-provider platform. Providers register standardised
capabilities, Platform Operators assemble environments from multiple providers, and
end-users deploy against abstract interfaces without knowing which provider fulfills
each capability. For the complete vision, see
[The Open Platform Model: Vision and Ecosystem](vision/opm-ecosystem.md).

### M7: Commodity Interfaces

Standard service interfaces that any conforming provider can implement, creating a
level playing field where providers compete on quality, price, and reliability rather
than API lock-in.

**Preliminary deliverables:**

- Commodity service interface definitions — CUE schemas for StatelessWorkload,
  StatefulWorkload, VirtualMachine, DatabaseAsAService, ObjectStorage, DNS,
  LoadBalancer, MessageQueue, KeyValueStore, CertificateAuthority
- Provider certification model — machine-verifiable proof that a provider's
  implementation satisfies a commodity interface
- Cross-provider contracts for networking, identity and access control, and
  observability
- Multi-provider rendering pipeline — a single ModuleRelease producing resources
  across multiple providers, each handling the capabilities it's registered for

### M8: Multi-Provider Ecosystem

The ecosystem where customers assemble their infrastructure from multiple service
providers, each contributing different capabilities to the same environment.

**Preliminary deliverables:**

- Multi-provider marketplace with discovery and selection
- Provider onboarding and certification pipeline
- Cross-provider observability and monitoring aggregation
- Governance and compliance framework across providers
- Community-contributed commodity definitions and provider implementations

---

*This roadmap is a living document. Milestones are ordered by priority and dependency,
not by calendar. Phase 2 and Phase 3 milestones are preliminary and will be refined as
Phase 1 matures.*
