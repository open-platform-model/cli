# v1alpha1 — Definition Index

CUE module: `opmodel.dev/core@v1alpha1`

---

## Project Structure

```text
v1alpha1/
├── core/                        # Core OPM definition types
│   ├── types/                   # Shared primitive types and regex constraints
│   ├── primitives/              # Resource, Trait, Blueprint, PolicyRule base types
│   ├── component/               # #Component — deployable unit
│   ├── transformer/             # #Transformer, #TransformerContext
│   ├── policy/                  # #Policy
│   ├── module/                  # #Module
│   ├── modulerelease/           # #ModuleRelease
│   ├── bundle/                  # #Bundle
│   ├── provider/                # #Provider
│   └── helpers/                 # Internal helpers (e.g. auto-secrets wiring)
├── schemas/                     # Shared field schemas (reused across definitions)
│   └── kubernetes/              # Mirrored Kubernetes API types (transformer targets)
├── resources/                   # Resource implementations
│   ├── config/                  # ConfigMap, Secret
│   ├── extension/               # CRD
│   ├── security/                # ServiceAccount, Role
│   ├── storage/                 # Volume
│   └── workload/                # Container
├── traits/                      # Trait implementations
│   ├── network/                 # Expose, HttpRoute, GrpcRoute, TcpRoute
│   ├── security/                # SecurityContext, WorkloadIdentity, Encryption
│   └── workload/                # Scaling, Sizing, UpdateStrategy, Placement, ...
├── blueprints/                  # Blueprint implementations
│   ├── data/                    # SimpleDatabase
│   └── workload/                # Stateless, Stateful, Daemon, Task, ScheduledTask
├── providers/                   # Provider implementations
│   └── kubernetes/              # Kubernetes provider + transformers
└── examples/                    # Concrete usage examples (no exported definitions)
```

---

## Core

Base definition types that form the OPM type system. Each construct lives in its own subpackage under `core/`.

### `core/types/`

| Definition | Description |
|---|---|
| `#LabelsAnnotationsType`, `#NameType`, `#FQNType`, `#VersionType`, ... | Shared primitive types and regex constraints used across all definitions |

### `core/primitives/`

| Definition | Description |
|---|---|
| `#Resource` | Deployable resource definition with FQN, metadata, and OpenAPIv3-compatible spec |
| `#Trait` | Additional behavior attachable to components, with `appliesTo` constraints |
| `#Blueprint` | Reusable composition of resources and traits into a higher-level abstraction |
| `#PolicyRule` | Governance rule encoding security, compliance, or operational guardrails |

### `core/component/`

| Definition | Description |
|---|---|
| `#Component` | Deployable unit composing resources, traits, and blueprints into a closed spec |

### `core/transformer/`

| Definition | Description |
|---|---|
| `#Transformer` | Converts OPM components to platform-specific resources via label/resource/trait matching |
| `#TransformerContext` | Provider context injected into each transformer at render time |

### `core/policy/`

| Definition | Description |
|---|---|
| `#Policy` | Groups policy rules and targets them to components via label matching or explicit refs |

### `core/module/`

| Definition | Description |
|---|---|
| `#Module` | Portable application blueprint containing components, policies, and a config schema |

### `core/modulerelease/`

| Definition | Description |
|---|---|
| `#ModuleRelease` | Concrete deployment instance binding a module to values and a target namespace |

### `core/bundle/`

| Definition | Description |
|---|---|
| `#Bundle` | Collection of modules grouped for distribution |

### `core/provider/`

| Definition | Description |
|---|---|
| `#Provider` | Provider definition with a transformer registry for converting OPM components to platform resources |

### `core/helpers/`

| Definition | Description |
|---|---|
| `#OpmSecretsComponent` | Builds the auto-generated `opm-secrets` component from discovered `#Secret` fields |
| `#SecretsResourceFQN` | Canonical FQN for the secrets resource (must stay in sync with `resources/config/secret.cue`) |

---

## Schemas

Reusable field schemas shared across resource and trait definitions.

### `schemas/common.cue`

| Definition | Description |
|---|---|
| `#NameType`, `#LabelsAnnotationsSchema`, `#VersionSchema` | Primitive name, label/annotation, and version field schemas |

### `schemas/config.cue`

| Definition | Description |
|---|---|
| `#Secret` / `#SecretLiteral` / `#SecretK8sRef` / `#SecretEsoRef` | Discriminated union for secret sources (literal, K8s ref, ESO ref) |
| `#SecretSchema` / `#ConfigMapSchema` | Field schemas for Secret and ConfigMap resources |
| `#ContentHash` / `#SecretContentHash` | Content-hash based immutable naming helpers |
| `#DiscoverSecrets` / `#GroupSecrets` / `#AutoSecrets` | Auto-discovery pipeline for extracting secrets from component specs |

### `schemas/data.cue`

| Definition | Description |
|---|---|
| `#SimpleDatabaseSchema` | Schema for a simple database (postgres / mysql / mongodb / redis) with optional persistence |

### `schemas/extension.cue`

| Definition | Description |
|---|---|
| `#CRDSchema` / `#CRDVersionSchema` | Kubernetes CRD definition schemas for vendoring operator CRDs |

### `schemas/network.cue`

| Definition | Description |
|---|---|
| `#PortSchema` / `#IANA_SVC_NAME` | Port definition with name, number, and protocol |
| `#ExposeSchema` | Service exposure spec with typed port mappings |
| `#NetworkRuleSchema` / `#SharedNetworkSchema` | Network policy and shared-network schemas |
| `#HttpRouteSchema` / `#HttpRouteRuleSchema` / `#HttpRouteMatchSchema` | HTTP routing: matches, rules, and full route spec |
| `#GrpcRouteSchema` / `#GrpcRouteRuleSchema` / `#GrpcRouteMatchSchema` | gRPC routing: matches, rules, and full route spec |
| `#TcpRouteSchema` / `#TcpRouteRuleSchema` | TCP port-forwarding route spec |
| `#RouteHeaderMatch` / `#RouteRuleBase` / `#RouteAttachmentSchema` | Shared route primitives (header matching, gateway attachment) |

### `schemas/quantity.cue`

| Definition | Description |
|---|---|
| `#NormalizeCPU` / `#NormalizeMemory` | Normalize CPU and memory values to Kubernetes canonical formats |

### `schemas/security.cue`

| Definition | Description |
|---|---|
| `#WorkloadIdentitySchema` | Service account / workload identity for pod authentication |
| `#ServiceAccountSchema` | Standalone service account identity (name, automountToken) |
| `#PolicyRuleSchema` | Single RBAC permission rule (apiGroups, resources, verbs) |
| `#RoleSubjectSchema` | Role subject — embeds a WorkloadIdentity or ServiceAccount via CUE reference |
| `#RoleSchema` | RBAC role with scope (namespace/cluster), rules, and CUE-referenced subjects |
| `#SecurityContextSchema` | Pod and container security constraints (runAsNonRoot, privilege escalation, capabilities) |
| `#EncryptionConfigSchema` | At-rest and in-transit encryption requirements |

### `schemas/storage.cue`

| Definition | Description |
|---|---|
| `#VolumeSchema` | Volume definition supporting multiple source types |
| `#VolumeMountSchema` | Mount path and options for attaching a volume to a container |
| `#EmptyDirSchema` / `#HostPathSchema` / `#PersistentClaimSchema` | Concrete volume source schemas |

### `schemas/workload.cue`

| Definition | Description |
|---|---|
| `#ContainerSchema` / `#Image` | Container definition with image, command, args, ports, env, probes, and mounts |
| `#EnvVarSchema` / `#EnvFromSource` / `#FieldRefSchema` / `#ResourceFieldRefSchema` | Environment variable sources (literal, configMap, secret, field ref) |
| `#ResourceRequirementsSchema` | CPU and memory requests/limits |
| `#ProbeSchema` | Liveness, readiness, and startup probe spec |
| `#ScalingSchema` / `#AutoscalingSpec` / `#MetricSpec` / `#MetricTargetSpec` | Horizontal scaling: replica count and HPA autoscaling metrics |
| `#SizingSchema` / `#VerticalScalingSchema` | Vertical resource sizing (CPU/memory) |
| `#RestartPolicySchema` | Container restart policy (Always / OnFailure / Never) |
| `#UpdateStrategySchema` | Rollout update strategy (RollingUpdate / Recreate / OnDelete) |
| `#InitContainersSchema` / `#SidecarContainersSchema` | Init and sidecar container list schemas |
| `#JobConfigSchema` / `#CronJobConfigSchema` | Job completions/parallelism and CronJob schedule/concurrency |
| `#StatelessWorkloadSchema` | Full schema for a stateless (Deployment) workload |
| `#StatefulWorkloadSchema` | Full schema for a stateful (StatefulSet) workload |
| `#DaemonWorkloadSchema` | Full schema for a daemon (DaemonSet) workload |
| `#TaskWorkloadSchema` | Full schema for a one-time task (Job) workload |
| `#ScheduledTaskWorkloadSchema` | Full schema for a cron-scheduled (CronJob) workload |
| `#DisruptionBudgetSchema` | Availability constraints during voluntary disruptions |
| `#GracefulShutdownSchema` | Termination grace period and pre-stop hook |
| `#PlacementSchema` | Zone/region/host spreading and node selector requirements |

---

## Resources

Concrete resource definitions that can be attached to components.
Each follows the triple pattern: `#XxxResource` (definition) · `#Xxx` (mixin) · `#XxxDefaults` (defaults).

| Definition | File | Description |
|---|---|---|
| `#ContainerResource` | `resources/workload/container.cue` | Core workload resource: a container image definition requiring a workload-type label |
| `#ConfigMapsResource` | `resources/config/configmap.cue` | External key/value configuration via ConfigMaps |
| `#SecretsResource` | `resources/config/secret.cue` | Sensitive configuration via Secrets (literal, K8s ref, or ESO) |
| `#VolumesResource` | `resources/storage/volume.cue` | Persistent and ephemeral volume storage |
| `#CRDsResource` | `resources/extension/crd.cue` | Kubernetes CustomResourceDefinitions for vendoring operator CRDs |
| `#ServiceAccountResource` | `resources/security/service_account.cue` | Standalone service account identity (independent of WorkloadIdentity trait) |
| `#RoleResource` | `resources/security/role.cue` | RBAC Role with rules and CUE-referenced subjects; collapses k8s Role/ClusterRole + RoleBinding/ClusterRoleBinding |

---

## Traits

Behavioral extensions attachable to components.
Each follows the triple pattern: `#XxxTrait` (definition) · `#Xxx` (mixin) · `#XxxDefaults` (defaults).

### Network

| Definition | File | Description |
|---|---|---|
| `#ExposeTrait` | `traits/network/expose.cue` | Expose a workload via a Kubernetes Service with typed port mappings |
| `#HttpRouteTrait` | `traits/network/http_route.cue` | HTTP routing rules (Gateway API / Ingress) |
| `#GrpcRouteTrait` | `traits/network/grpc_route.cue` | gRPC routing rules (Gateway API / Ingress) |
| `#TcpRouteTrait` | `traits/network/tcp_route.cue` | TCP port-forwarding rules |

### Security

| Definition | File | Description |
|---|---|---|
| `#SecurityContextTrait` | `traits/security/security_context.cue` | Container and pod-level security constraints |
| `#WorkloadIdentityTrait` | `traits/security/workload_identity.cue` | Service account / workload identity for pod authentication |
| `#EncryptionConfigTrait` | `traits/security/encryption.cue` | At-rest and in-transit encryption requirements |

### Workload

| Definition | File | Description |
|---|---|---|
| `#ScalingTrait` | `traits/workload/scaling.cue` | Horizontal scaling: replica count and optional HPA autoscaling |
| `#SizingTrait` | `traits/workload/sizing.cue` | Vertical sizing: CPU and memory requests/limits |
| `#UpdateStrategyTrait` | `traits/workload/update_strategy.cue` | Rollout update strategy (RollingUpdate / Recreate / OnDelete) |
| `#PlacementTrait` | `traits/workload/placement.cue` | Zone/region/host spreading and node selector requirements |
| `#RestartPolicyTrait` | `traits/workload/restart_policy.cue` | Container restart policy (Always / OnFailure / Never) |
| `#InitContainersTrait` | `traits/workload/init_containers.cue` | Init containers to run before the main container starts |
| `#SidecarContainersTrait` | `traits/workload/sidecar_containers.cue` | Sidecar containers injected alongside the main workload |
| `#DisruptionBudgetTrait` | `traits/workload/disruption_budget.cue` | Availability constraints during voluntary disruptions |
| `#GracefulShutdownTrait` | `traits/workload/graceful_shutdown.cue` | Termination grace period and pre-stop lifecycle hooks |
| `#JobConfigTrait` | `traits/workload/job_config.cue` | Job settings: completions, parallelism, backoff, deadlines, TTL |
| `#CronJobConfigTrait` | `traits/workload/cron_job_config.cue` | CronJob settings: schedule, concurrency policy, history limits |

---

## Blueprints

Higher-level abstractions composing resources and traits into opinionated workload patterns.
Each follows the pair pattern: `#XxxBlueprint` (definition) · `#Xxx` (mixin).

### Data

| Definition | File | Description |
|---|---|---|
| `#SimpleDatabaseBlueprint` | `blueprints/data/simple_database.cue` | Opinionated stateful database (postgres / mysql / mongodb / redis) with auto-wired persistence and readiness probes |

### Workload

| Definition | File | Description |
|---|---|---|
| `#StatelessWorkloadBlueprint` | `blueprints/workload/stateless_workload.cue` | Stateless workload with no stable identity or persistent storage (Deployment) |
| `#StatefulWorkloadBlueprint` | `blueprints/workload/stateful_workload.cue` | Stateful workload with stable identity and persistent storage (StatefulSet) |
| `#DaemonWorkloadBlueprint` | `blueprints/workload/daemon_workload.cue` | Daemon workload running on all (or selected) nodes (DaemonSet) |
| `#TaskWorkloadBlueprint` | `blueprints/workload/task_workload.cue` | One-time task workload that runs to completion (Job) |
| `#ScheduledTaskWorkloadBlueprint` | `blueprints/workload/scheduled_task_workload.cue` | Cron-scheduled task workload (CronJob) |

---

## Providers

Provider and transformer definitions for converting OPM components to platform resources.

### Registry

| Definition | File | Description |
|---|---|---|
| `#Registry` | `providers/registry.cue` | Top-level provider registry mapping provider names to provider definitions |

### Kubernetes Provider

| Definition | File | Description |
|---|---|---|
| `#Provider` | `providers/kubernetes/provider.cue` | Kubernetes provider registering all K8s transformers |

### Kubernetes Transformers

| Definition | File | Description |
|---|---|---|
| `#DeploymentTransformer` | `transformers/deployment_transformer.cue` | Converts stateless workload components to Kubernetes Deployments |
| `#StatefulsetTransformer` | `transformers/statefulset_transformer.cue` | Converts stateful workload components to Kubernetes StatefulSets |
| `#DaemonSetTransformer` | `transformers/daemonset_transformer.cue` | Converts daemon workload components to Kubernetes DaemonSets |
| `#JobTransformer` | `transformers/job_transformer.cue` | Converts task workload components to Kubernetes Jobs |
| `#CronJobTransformer` | `transformers/cronjob_transformer.cue` | Converts scheduled task components to Kubernetes CronJobs |
| `#ServiceTransformer` | `transformers/service_transformer.cue` | Creates Kubernetes Services from components with the Expose trait |
| `#IngressTransformer` | `transformers/ingress_transformer.cue` | Converts HttpRoute trait to Kubernetes Ingress |
| `#HPATransformer` | `transformers/hpa_transformer.cue` | Converts Scaling autoscaling config to Kubernetes HorizontalPodAutoscalers |
| `#ConfigMapTransformer` | `transformers/configmap_transformer.cue` | Converts ConfigMaps resources to Kubernetes ConfigMaps (with content-hash naming) |
| `#SecretTransformer` | `transformers/secret_transformer.cue` | Converts Secrets resources to Kubernetes Secrets and ExternalSecrets (ESO) |
| `#PVCTransformer` | `transformers/pvc_transformer.cue` | Creates PersistentVolumeClaims from Volume resources |
| `#CRDTransformer` | `transformers/crd_transformer.cue` | Converts CRDs resources to Kubernetes CustomResourceDefinitions |
| `#ServiceAccountTransformer` | `transformers/serviceaccount_transformer.cue` | Converts WorkloadIdentity traits to Kubernetes ServiceAccounts |
| `#ServiceAccountResourceTransformer` | `transformers/sa_resource_transformer.cue` | Converts standalone ServiceAccount resources to Kubernetes ServiceAccounts |
| `#RoleTransformer` | `transformers/role_transformer.cue` | Converts Role resources to k8s Role+RoleBinding or ClusterRole+ClusterRoleBinding |
| `#ToK8sContainer` / `#ToK8sContainers` / `#ToK8sVolumes` | `transformers/container_helpers.cue` | Shared helpers converting OPM container/volume schemas to Kubernetes list format |
| `#ToK8sServiceAccount` | `transformers/sa_helpers.cue` | Shared helper converting an OPM identity spec (WorkloadIdentity or ServiceAccount) to a Kubernetes ServiceAccount |
