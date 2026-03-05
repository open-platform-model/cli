## ADDED Requirements

### Requirement: MinecraftServer CRD definition
The experiment SHALL include a CustomResourceDefinition for `MinecraftServer` (group: `opm.example.com`, version: `v1alpha1`) with a namespaced scope. The CRD schema SHALL define the phase-1 subset of the minecraft-java module's config surface.

#### Scenario: CRD spec fields
- **WHEN** a user creates a MinecraftServer CR
- **THEN** the CRD SHALL accept the following spec fields: `version` (string), `jvm` (object with `memory` string and `useAikarFlags` boolean), `server` (object with `motd`, `maxPlayers`, `difficulty`, `mode`, `pvp`, `onlineMode`), `serviceType` (enum: ClusterIP/LoadBalancer/NodePort), and `storage` (object with `type` enum emptyDir/pvc, `size` string)

#### Scenario: CRD status subresource
- **WHEN** the CRD is defined
- **THEN** it SHALL enable the status subresource with fields: `phase` (string), `readyReplicas` (integer), `observedGeneration` (integer), `message` (string), and `conditions` (array of condition objects with type/status/reason/message/lastTransitionTime)

#### Scenario: CRD validation
- **WHEN** a user submits a MinecraftServer with `maxPlayers` outside 1-1000
- **THEN** the Kubernetes API server SHALL reject the CR via OpenAPI validation

### Requirement: CompositeController definition
The experiment SHALL include a Metacontroller `CompositeController` resource that watches `MinecraftServer` parents and manages StatefulSet, Service, Secret, and PersistentVolumeClaim children.

#### Scenario: CompositeController child resources
- **WHEN** the CompositeController is created
- **THEN** it SHALL declare child resources: `statefulsets` (apps/v1, InPlace update), `services` (v1, InPlace update), `secrets` (v1, InPlace update), and `persistentvolumeclaims` (v1, OnDelete update)

#### Scenario: Selector generation
- **WHEN** the CompositeController is created
- **THEN** it SHALL set `generateSelector: true` so MinecraftServer CRs do not require a `spec.selector` field

#### Scenario: Webhook target
- **WHEN** the CompositeController specifies its sync hook
- **THEN** it SHALL point to the webhook Service via `http://<webhook-service>.<namespace>/sync`

#### Scenario: Finalize hook
- **WHEN** the CompositeController specifies its finalize hook
- **THEN** it SHALL point to the same webhook endpoint as the sync hook

### Requirement: Webhook deployment manifests
The experiment SHALL include Kubernetes manifests for deploying the webhook server: a Deployment and a ClusterIP Service.

#### Scenario: Webhook Deployment
- **WHEN** the webhook manifests are applied
- **THEN** a Deployment SHALL be created with 1 replica running the webhook container on port 8080

#### Scenario: Webhook Service
- **WHEN** the webhook manifests are applied
- **THEN** a ClusterIP Service SHALL be created routing traffic to the webhook Deployment on port 80 → targetPort 8080

### Requirement: Sample MinecraftServer CR
The experiment SHALL include a sample MinecraftServer CR that demonstrates the CRD usage with sensible defaults.

#### Scenario: Sample CR creates a working server
- **WHEN** the sample MinecraftServer CR is applied (with Metacontroller and the webhook running)
- **THEN** it SHALL result in a Paper Minecraft server StatefulSet with 1 replica, a ClusterIP Service on port 25565, and emptyDir storage
