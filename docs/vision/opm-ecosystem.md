# The Open Platform Model: Vision and Ecosystem

| Field       | Value                  |
|-------------|------------------------|
| **Status**  | Draft                  |
| **Created** | 2026-02-12             |
| **Authors** | OPM Contributors       |

## Table of Contents

1. [The Problem](#1-the-problem)
2. [The Vision: Three Layers](#2-the-vision-three-layers)
3. [Layer 1: Application Model](#3-layer-1-application-model)
4. [Layer 2: Platform Model](#4-layer-2-platform-model)
5. [Layer 3: The Ecosystem](#5-layer-3-the-ecosystem)
6. [Commodity vs Specialised Services](#6-commodity-vs-specialised-services)
7. [Multi-Provider Coexistence](#7-multi-provider-coexistence)
8. [Roadmap](#8-roadmap)
9. [Design Principles](#9-design-principles)

---

## 1. The Problem

### The Hyperscaler Lock-In

Three companies — Amazon, Google, and Microsoft — control roughly two thirds of the
global cloud market. Their dominance isn't just about scale. It's about the APIs.
Every service on AWS, Azure, and GCP exposes a proprietary interface. Once you build
against those interfaces, you're locked in. Moving means rewriting.

This isn't a theoretical problem. It shapes real decisions every day:

- A startup picks AWS because "everyone does," then discovers egress fees consuming
  20% of their infrastructure budget.
- A European company needs data sovereignty but can't extract their workloads from
  GCP because every service is wired to Google-specific APIs.
- A government agency wants to use a domestic provider but their entire stack is
  written against Azure Resource Manager.

### The Alternative Providers Exist — But Can't Compete

The cloud market is not actually a monopoly. There are dozens of capable providers:
Hetzner, Vultr, DigitalOcean, OVHcloud, Scaleway, Linode (Akamai), Civo, and many
more. The "neocloud" trend is accelerating — CoreWeave grew to over $3.5B in revenue
in 2025, Lambda and Crusoe raised hundreds of millions, and Oracle's cloud backlog
exceeded half a trillion dollars.

These providers offer real infrastructure: VMs, Kubernetes, managed databases, object
storage, networking. The hardware is there. The capacity is there. What's missing is
a **standard way for applications to consume services across providers**.

Today, if you're a smaller cloud provider, you have to convince every customer to
rewrite their application configuration to use your specific APIs. That's not a
technical problem — it's a market structure problem. There's no standard application
model that lets a workload move between providers without modification.

### The Tooling Gap

Existing cloud-native tools solve important problems, but none solve this one:

- **Helm** is the de facto Kubernetes package manager, but it's a Go template engine
  over YAML. No type safety, no composability, no portability beyond "it produces K8s
  manifests." The Bitnami deprecation in 2025 exposed how fragile the Helm ecosystem is
  when a single maintainer walks away.

- **KubeVela** is a powerful delivery platform with workflows, multi-cluster management,
  and a CUE-based definition system. But it's Kubernetes-only and requires an in-cluster
  controller. It solves application delivery, not application portability across
  providers.

- **Crossplane** extends Kubernetes with infrastructure-as-code CRDs. It provisions
  cloud resources, but through provider-specific APIs. You still write against AWS or GCP
  interfaces — just through a Kubernetes CRD instead of a CLI.

- **Terraform/OpenTofu** provisions infrastructure declaratively, but the module format
  is HCL-based, provider-specific, and has no concept of application composition
  (components, traits, policies).

None of these tools answer the fundamental question: *How do you define an application
once and deploy it across different service providers without rewriting?*

That's what OPM sets out to solve.

---

## 2. The Vision: Three Layers

OPM is designed as three complementary layers, each building on the one below:

```text
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Layer 3: ECOSYSTEM                                             │
│  Multi-provider marketplace. Customers assemble environments    │
│  from multiple service providers. Fair market with commodity    │
│  and specialised services.                                      │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 2: PLATFORM MODEL                                        │
│  Providers register capabilities. Standardised commodity        │
│  services (Compute, DNS, Database, Storage) alongside           │
│  specialised offerings. PlatformRegistry bridges models.        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Layer 1: APPLICATION MODEL                    ← Current work   │
│  Modules, Bundles, Resources, Traits, Policies.                 │
│  Type-safe CUE definitions. OCI distribution.                   │
│  Replaces Helm for defining and deploying applications.         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Each layer has a distinct purpose:

- **Layer 1** (Application Model) answers: *How do I define, distribute, and deploy
  applications with type safety and composability?*

- **Layer 2** (Platform Model) answers: *How do service providers expose their
  capabilities in a standard format, and how do platform operators assemble them?*

- **Layer 3** (Ecosystem) answers: *How do multiple providers coexist in the same
  environment, serving the same customer, competing on quality and price rather than
  lock-in?*

The work proceeds bottom-up. Layer 1 must be solid before Layer 2 can build on it.
Layer 2 must be solid before Layer 3 becomes practical. The current focus is Layer 1.

---

## 3. Layer 1: Application Model

The Application Model is where OPM lives today. It defines how applications are
written, composed, configured, and deployed.

### Replacing Helm

OPM's Application Model is designed to be a successor to Helm Charts for Kubernetes
application packaging. The core improvements:

| Problem with Helm                               | OPM's Answer                                                                                              |
|-------------------------------------------------|-----------------------------------------------------------------------------------------------------------|
| Go templates over YAML — no type safety         | CUE-native definitions with compile-time validation                                                       |
| Monolithic chart structure — all concerns mixed | Resource + Trait + Policy composition with clear ownership                                                |
| `values.yaml` is untyped — any key accepted     | `#config` schema constrains what's configurable; CUE rejects invalid values                               |
| No separation between author and consumer       | Module Author writes `#Module` with `#config` + `values`; End User writes `#ModuleRelease` with overrides |
| Subcharts are fragile and poorly composable     | Modules compose via CUE unification; Bundles group modules                                                |
| No built-in policy enforcement                  | `#Policy` / `#PolicyRule` with block/warn/audit semantics                                                 |
| OCI distribution bolted on after the fact       | OCI-native distribution from day one (CUE registry + ORAS)                                                |

### Planned Capabilities

The Application Model will grow to include capabilities inspired by KubeVela's delivery
platform, adapted to OPM's build-time-first architecture:

**Workflow orchestration** — Declarative multi-step deployment with support for staged
rollouts, approval gates, and conditional execution. Unlike KubeVela's controller-based
workflow engine, OPM's approach will define workflows in CUE and execute them from the
CLI or a lightweight controller, keeping the "render, then apply" philosophy intact.

**Multi-cluster topology** — First-class support for targeting multiple clusters and
namespaces from a single Module or Bundle definition. Topology and override policies
will allow per-cluster customisation without duplicating ModuleReleases.

**Continuous reconciliation** — An optional in-cluster controller that watches for
drift and re-applies the desired state. This complements the CLI's one-shot apply model
for environments that need continuous enforcement. The controller will use CUE evaluation
(the same pipeline as the CLI) so that behaviour is identical whether you apply from
your laptop or the controller reconciles in-cluster.

**Bundle implementation** — The `#Bundle` type will group multiple Modules into a
single distributable and deployable unit, with shared configuration and coordinated
lifecycle management.

These additions will make OPM a complete delivery system — not just a definition format.
The key differentiator from KubeVela is that all of these capabilities will be built on
top of the same CUE evaluation pipeline, ensuring that what you test locally is exactly
what runs in production.

---

## 4. Layer 2: Platform Model

The Platform Model bridges the Application Model to target platforms. Today this is
the Provider/Transformer system that converts abstract components into Kubernetes
manifests. Tomorrow it becomes a capability marketplace.

### Providers as Capability Registries

A Provider in OPM is not just "Kubernetes." It's any entity that offers platform
capabilities. A cloud provider, a managed service company, an internal infrastructure
team — all can be Providers.

Each Provider registers the capabilities it offers through Transformers. A Transformer
declares what it can handle (required labels, resources, traits) and how it converts
OPM components into platform-specific output.

The Platform Model's future role is to standardise this registration so that:

1. Providers can publish their capabilities in a discoverable format
2. Platform Operators can assemble capabilities from multiple providers
3. End Users deploy against abstract interfaces without knowing (or caring) which
   provider fulfills each capability

### The PlatformRegistry

The PlatformRegistry is the bridge between Application Model and Platform Model. It:

- Catalogues which Modules and Bundles are available for deployment
- Maps abstract capabilities to concrete Providers
- Enforces organisational policies across all deployments
- Provides discovery: "What can I deploy? Which providers support it?"

When a Platform Operator sets up an environment, they configure a PlatformRegistry
that declares which Providers handle which capabilities. An End User deploying a
Module doesn't need to know the details — the registry resolves abstract service
requests to concrete providers.

### Commodity Services

The Platform Model will define a set of **standardised service interfaces** — the
"commodities" of the ecosystem. These are services with well-known APIs that any
conforming Provider can implement:

- **StatelessWorkload** — containerised applications (Deployments)
- **StatefulWorkload** — applications with stable identity and storage (StatefulSets)
- **VirtualMachine** — compute instances
- **DatabaseAsAService** — managed relational databases (Postgres, MySQL)
- **ObjectStorage** — S3-compatible storage
- **DNS** — domain name resolution
- **LoadBalancer** — traffic distribution
- **MessageQueue** — asynchronous messaging
- **KeyValueStore** — managed Redis/Memcached-compatible stores
- **CertificateAuthority** — TLS certificate management

Each commodity defines an abstract interface — a CUE schema that specifies what the
service accepts and what it provides. Any Provider that implements that interface can
fulfill the capability. The customer doesn't need to know whether their
DatabaseAsAService is running on Hetzner, AWS, or a local Postgres operator — the
interface is the same.

### Specialised Services

Not everything is a commodity. Providers will also offer **specialised services** —
capabilities with unique APIs that don't conform to a standard interface.

Specialised services allow providers to differentiate. A GPU cloud might offer a
`#MLTrainingPipeline` that has no equivalent elsewhere. A security company might
offer a `#ZeroTrustMesh` with proprietary features. A data analytics firm might
offer a `#StreamProcessingEngine` with unique performance characteristics.

The Platform Model accommodates both:

```text
┌───────────────────────────────────────────────────┐
│                 PlatformRegistry                  │
│                                                   │
│  Commodities (standardised, interchangeable):     │
│  ┌──────────┐ ┌──────────┐ ┌───────────────────┐  │
│  │ Compute  │ │   DNS    │ │ DatabaseAsAService│  │
│  │ (any     │ │ (any     │ │ (any provider)    │  │
│  │ provider)│ │ provider)│ │                   │  │
│  └──────────┘ └──────────┘ └───────────────────┘  │
│                                                   │
│  Specialised (unique, provider-specific):         │
│  ┌───────────────────┐ ┌───────────────────────┐  │
│  │ MLTrainingPipeline│ │ ZeroTrustMesh         │  │
│  │ (Provider X only) │ │ (Provider Y only)     │  │
│  └───────────────────┘ └───────────────────────┘  │
└───────────────────────────────────────────────────┘
```

This dual model ensures a **fair market**: commodities create a level playing field
where providers compete on quality, price, and reliability. Specialised services
ensure that innovation is rewarded — a provider with a genuinely unique capability
can attract customers who need that specific feature.

Over time, successful specialised services may become commodities as the market
matures and multiple providers implement compatible interfaces. This is a natural
evolution — today's innovation becomes tomorrow's baseline.

---

## 5. Layer 3: The Ecosystem

The Ecosystem is the long-term vision: a world where customers assemble their
infrastructure from multiple service providers, each contributing different
capabilities to the same environment.

### Multi-Provider Environments

Today, most organisations pick one cloud provider and use it for everything. This
isn't because one provider is best at everything — it's because mixing providers is
operationally impossible. There's no standard way to wire services from Provider A
to services from Provider B.

The OPM Ecosystem changes this. A customer's environment might look like:

```text
┌────────────────────────────────────────────────────┐
│              Customer Environment                  │
│                                                    │
│  Compute:     Hetzner  (StatelessWorkload,         │
│                         StatefulWorkload)          │
│  DNS:         Cloudflare (DNS, CertificateAuth)    │
│  Database:    Neon     (DatabaseAsAService)        │
│  Storage:     Wasabi   (ObjectStorage)             │
│  Monitoring:  Grafana  (Observability)  [special.] │
│  ML:          Lambda   (MLPipeline)     [special.] │
│                                                    │
└────────────────────────────────────────────────────┘
```

The Platform Operator configures this environment by selecting Providers for each
capability in the PlatformRegistry. An End User deploying a Module that needs a
StatelessWorkload, a DatabaseAsAService, and ObjectStorage doesn't need to know
that three different companies are fulfilling those requests.

### Democratising the Platform

The hyperscaler model concentrates power: one company provides everything, sets all
the prices, controls all the APIs. The OPM Ecosystem distributes power:

- **Any provider** can offer any commodity service. A small hosting company in
  Germany can compete with AWS for Compute if their interface is conformant.
- **Customers choose** based on price, performance, locality, and values — not based
  on which APIs they've already committed to.
- **Providers coexist** in the same environment. Switching from Provider A to
  Provider B for a specific capability is a configuration change, not a rewrite.
- **Innovation is rewarded** through specialised services. A provider with a
  genuinely better approach to ML training can offer that as a specialised service
  that attracts customers — without having to replicate the entire hyperscaler stack.

This is not about replacing the hyperscalers. AWS, Azure, and GCP will remain
excellent providers of many services. The goal is to create a market structure where
they compete on equal terms with smaller providers — where the customer's application
definition is portable and the choice of provider is a deployment decision, not an
architectural one.

---

## 6. Commodity vs Specialised Services

### What Makes a Service a Commodity

A commodity service has three properties:

1. **Standardised interface** — There exists a well-understood, stable API contract
   that multiple implementations can satisfy. The interface describes inputs, outputs,
   and behaviour without prescribing implementation details.

2. **Interchangeable providers** — Switching from Provider A to Provider B for a
   commodity service requires changing provider configuration, not changing the
   application that consumes the service.

3. **Baseline expectations** — The commodity definition includes non-functional
   requirements (availability SLAs, latency bounds, security baseline) that all
   conforming providers must meet.

### Examples

| Commodity          | Interface Summary                                              | Providers (examples)                                    |
|--------------------|----------------------------------------------------------------|---------------------------------------------------------|
| StatelessWorkload  | Container image + ports + scaling + probes → running instances | Any K8s provider, any container platform                |
| DatabaseAsAService | Engine type + version + size → connection string               | Neon, PlanetScale, Aiven, AWS RDS, self-hosted operator |
| ObjectStorage      | Bucket name + region → S3-compatible endpoint                  | Wasabi, Cloudflare R2, MinIO, Backblaze B2, AWS S3      |
| DNS                | Domain + records → resolvable names                            | Cloudflare, Route53, Hetzner DNS, NS1                   |
| VirtualMachine     | Image + size + network → running VM                            | Hetzner, Vultr, DigitalOcean, AWS EC2, GCP Compute      |

### What Makes a Service Specialised

A specialised service is anything that **doesn't fit a commodity interface**. This
isn't a negative label — it means the service offers something genuinely unique:

- Proprietary algorithms or optimisations
- Novel hardware (custom ASICs, specialised GPUs)
- Unique operational models (serverless edge compute, etc.)
- Domain-specific functionality (genomics processing, financial tick data, etc.)

Specialised services use the same OPM primitives (Resources, Traits, Transformers)
but define their own interfaces. They're discoverable through the PlatformRegistry
but not interchangeable with other providers.

### The Commodity Lifecycle

Services evolve:

```text
Innovation          Adoption             Standardisation
     │                  │                      │
     ▼                  ▼                      ▼
  Specialised  →  De facto standard  →    Commodity
  (1 provider)    (few providers,        (many providers,
                   similar APIs)          standard interface)
```

Container orchestration followed this path: Docker Swarm, Mesos, Kubernetes were all
specialised. Kubernetes won adoption. Now "run a container with these properties" is
effectively a commodity — multiple providers offer Kubernetes-compatible compute.

The OPM ecosystem is designed to support this lifecycle. A provider can introduce a
specialised service today. If the market validates it and other providers implement
compatible interfaces, OPM can codify it as a commodity with a standard interface.
The original provider's early investment is rewarded by being first to market; the
ecosystem benefits from standardisation.

---

## 7. Multi-Provider Coexistence

### The Technical Challenge

Running multiple providers in the same environment isn't just a configuration
problem. It requires solving real infrastructure challenges:

**Networking** — When DNS comes from Cloudflare and Compute comes from Hetzner, those
systems need to communicate. Service discovery, internal DNS resolution, network
policies, and TLS termination must work across provider boundaries. Today this is
solved ad-hoc with external load balancers and manual DNS configuration. OPM's
Platform Model will need to define networking contracts that providers implement.

**Storage** — When ObjectStorage comes from Wasabi and the application runs on Vultr,
data paths cross provider boundaries. Latency, consistency, and access control become
cross-provider concerns. The Platform Model will need storage interface contracts that
handle these realities.

**Identity and access control** — A single customer identity must authorize operations
across multiple providers. The Platform Model will need a cross-provider authentication
and authorization model — likely based on existing standards like OIDC and SPIFFE, but
formalised as OPM interfaces.

**Observability** — Logs, metrics, and traces from multiple providers must be
aggregated into a coherent view. The Platform Model will define observability contracts
so that each provider emits data in a standard format.

### How OPM's Architecture Enables This

The Provider/Transformer model is designed from the start to support multiple providers
in the same environment:

- **Each Provider** registers the capabilities it offers
- **The PlatformRegistry** maps capabilities to providers, allowing different
  providers for different services
- **Transformers** are provider-specific, but the components they consume are
  provider-agnostic
- **A single ModuleRelease** can result in resources being created across multiple
  providers, each handling the capabilities it's registered for

The hard work — networking, storage, identity — is not in the application definition.
It's in the Platform Model and the provider implementations. This is deliberate:
application authors should not need to think about which provider runs their DNS.
That's a platform concern.

This is also where the most significant engineering challenges lie. The Application
Model (Layer 1) is tractable — it's fundamentally about CUE schemas and CLI tooling.
The Platform Model (Layer 2) requires solving cross-provider infrastructure problems
that the industry hasn't fully standardised yet. This is why the roadmap is
bottom-up: Layer 1 first, Layer 2 next, Layer 3 when the foundations are solid.

---

## 8. Roadmap

### Phase 1: Application Model and CLI — Current Focus

**What's built:**

- Core CUE definitions: Module, ModuleRelease, Component, Resource, Trait, Policy
- Catalog of definitions: 5 resources, 20+ traits, 5 blueprints
- Kubernetes Provider with 12 transformers (Deployment, StatefulSet, DaemonSet, Job,
  CronJob, Service, PVC, ConfigMap, Secret, ServiceAccount, HPA, Ingress)
- CLI: `opm mod init`, `opm mod build`, `opm mod apply`, `opm mod delete`,
  `opm mod status`
- OCI distribution design
- CUE registry integration (`opmodel.dev`)

**What's next:**

Full lifecycle management of applications using the CLI. The priority items are:

- Secrets injection — [RFC-0002](../rfc/0002-sensitive-data-model.md): `#Secret` type
  as a first-class concept, `@` tag injection from CLI, external reference support,
  and platform-appropriate output (K8s Secrets, ExternalSecrets, CSI volumes)
- Immutable config — [RFC-0003](../rfc/0003-immutable-config.md): content-hash suffixed
  ConfigMaps and Secrets with `spec.immutable: true`, automatic rolling updates on
  config change, garbage collection of old resources
- Garbage collection — [RFC-0001](../rfc/0001-release-inventory.md): release inventory
  Secret that tracks which resources belong to a ModuleRelease, enabling automatic
  pruning of stale resources during `opm mod apply`
- Policy definitions — first-class `#Policy` / `#PolicyRule` enforcement with
  block/warn/audit semantics across the rendering pipeline
- Stable CLI rendering pipeline — deterministic, composable render pipeline as the
  foundation for all downstream phases. This is the most critical deliverable: every
  future capability (controller, platform registry, multi-provider) depends on a
  reliable, well-tested rendering pipeline

### Phase 2: Kubernetes Controller — Next Phase

The same Module lifecycle as the CLI, but delivered through an in-cluster controller
and a custom resource. This phase marks the transition from developer tooling to
platform infrastructure.

- In-cluster controller that watches ModuleRelease custom resources and reconciles
  them using the same CUE rendering pipeline as the CLI
- Continuous reconciliation — drift detection and re-apply to maintain desired state
- Platform registry of providers and modules — a curated catalog, ready for
  consumption by end-users, managed by Platform Operators
- Beginning of the Platform Model — the ability to describe the whole platform,
  not just individual applications. PlatformRegistry as a CUE-defined catalog that
  maps capabilities to providers
- Multi-cluster topology and override policies — targeting multiple clusters and
  namespaces from a single definition

### Phase 3: Platform Model — Long-Term Vision

The full realisation of the multi-provider platform. Providers register standardised
capabilities, Platform Operators assemble environments from multiple providers, and
end-users deploy against abstract interfaces without knowing which provider fulfills
each capability.

- PlatformRegistry specification with commodity service interface definitions
  (StatelessWorkload, DatabaseAsAService, ObjectStorage, DNS, VirtualMachine,
  LoadBalancer, MessageQueue, KeyValueStore, CertificateAuthority)
- Provider certification model — verifying that a provider correctly implements a
  commodity interface
- Cross-provider contracts for networking, identity and access control, and
  observability
- Multi-provider rendering pipeline — a single ModuleRelease producing resources
  across multiple providers, each handling the capabilities it's registered for
- Multi-provider marketplace with discovery and selection
- Provider onboarding and certification pipeline
- Community-contributed commodity definitions and provider implementations

---

## 9. Design Principles

The following principles guide OPM's design. They're not aspirational — they're the
constraints that shape every technical decision.

### Type Safety First

All definitions are written in CUE. Validation happens at definition time, not at
deployment time. If a configuration is invalid, you find out on your laptop, not in
production at 3am.

This matters for the ecosystem because providers and consumers need a trustworthy
contract. A commodity interface defined in CUE is machine-verifiable: a provider can
prove their implementation satisfies the interface. A consumer can prove their
application conforms to the interface. There's no "it worked in dev but breaks in
production because Provider B interprets the YAML differently."

### Portability by Design

Definitions must be provider-agnostic. A Module describes WHAT an application needs
(containers, scaling, storage, DNS) not HOW a specific provider implements it.
Provider-specific details live exclusively in Transformers.

This is the architectural foundation of the entire ecosystem. Without it, you're just
building another tool that locks people into your specific implementation.

### Separation of Concerns

Four personas, four responsibilities, clear boundaries:

- **Infrastructure Operator** — runs the physical/virtual infrastructure
- **Module Author** — defines applications with sane defaults
- **Platform Operator** — curates the module catalog, selects providers, enforces policy
- **End User** — deploys modules with concrete values

In a multi-provider ecosystem, these boundaries become even more important. The Module
Author must not know or care which provider runs the infrastructure. The Platform
Operator must be able to swap providers without touching Module definitions. The End
User must be able to deploy without understanding provider internals.

### Simplicity and YAGNI

Justify complexity. Prefer explicit over implicit. Don't add abstractions until they
earn their place through real use cases.

The ecosystem vision is ambitious. But the path there is incremental. Each layer must
be useful on its own before the next one begins. Layer 1 must be a better Helm before
Layer 2 introduces provider marketplaces. Layer 2 must demonstrate multi-provider
value before Layer 3 builds a full ecosystem.

### Declarative Intent

Express WHAT, not HOW. The application model declares intent ("I need a database with
these properties"). The platform model resolves intent to implementation ("Neon
provides a Postgres database that satisfies those properties"). The ecosystem enables
choice ("you can also use PlanetScale, Aiven, or self-hosted — same interface").

This principle is what makes the commodity/specialised distinction possible. If your
definitions described HOW (specific API calls, provider-specific templates), portability
would be impossible. Because they describe WHAT, the resolution to HOW is a separate,
swappable concern.
