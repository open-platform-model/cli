## ADDED Requirements

### Requirement: Webhook serves sync and finalize hooks
The webhook server SHALL expose HTTP POST endpoints that Metacontroller calls for reconciliation. The server SHALL handle both sync (normal reconciliation) and finalize (deletion) requests on the same endpoint, distinguished by the `finalizing` field in the request body.

#### Scenario: Sync request for a new MinecraftServer
- **WHEN** Metacontroller sends a sync request with a MinecraftServer parent and no existing children
- **THEN** the webhook SHALL return a children list containing the rendered Kubernetes resources (StatefulSet, Service) and a status with `phase: "Pending"`

#### Scenario: Sync request for an existing MinecraftServer with running children
- **WHEN** Metacontroller sends a sync request with observed children (StatefulSet with readyReplicas >= 1)
- **THEN** the webhook SHALL return the desired children list and a status with `phase: "Running"` and `ready: true`

#### Scenario: Finalize request
- **WHEN** Metacontroller sends a request with `finalizing: true`
- **THEN** the webhook SHALL return `finalized: true` with an empty children list

### Requirement: Spec-to-CUE translation
The webhook SHALL translate the MinecraftServer CR `.spec` fields into a CUE values structure compatible with the minecraft-java module's `#config` schema. The translation SHALL map CRD spec fields to the module's expected value paths.

#### Scenario: Paper server with defaults
- **WHEN** a MinecraftServer spec contains `version: "LATEST"` and `jvm.memory: "2G"`
- **THEN** the webhook SHALL produce a CUE values structure with `paper: {version: "LATEST"}` and `jvm: {memory: "2G", useAikarFlags: true}`

#### Scenario: Custom server properties
- **WHEN** a MinecraftServer spec contains `server.maxPlayers: 50` and `server.difficulty: "hard"`
- **THEN** the CUE values SHALL include `server: {maxPlayers: 50, difficulty: "hard"}`

#### Scenario: PVC storage requested
- **WHEN** a MinecraftServer spec contains `storage.type: "pvc"` and `storage.size: "20Gi"`
- **THEN** the CUE values SHALL include `storage: {data: {type: "pvc", pvc: {size: "20Gi"}}}`

### Requirement: CUE evaluation pipeline
The webhook SHALL load the embedded minecraft-java CUE module via `load.Config.Overlay`, compile the embedded provider definition, evaluate the module with the translated values, match transformers to components, and execute them to produce Kubernetes resource manifests.

#### Scenario: Successful CUE evaluation
- **WHEN** a valid MinecraftServer spec is translated to CUE values
- **THEN** the pipeline SHALL produce a list of `*unstructured.Unstructured` Kubernetes objects including at minimum a StatefulSet and a Service

#### Scenario: Fresh CUE context per request
- **WHEN** multiple sync requests arrive concurrently
- **THEN** each request SHALL use its own `cue.Context` instance to avoid shared state corruption

### Requirement: Preserve children on render failure
The webhook SHALL NOT cause running resources to be deleted when the CUE evaluation fails. On any render pipeline error, the webhook SHALL return the observed children (from the request) as the desired children, preventing Metacontroller from deleting existing resources.

#### Scenario: Invalid spec field causes CUE error
- **WHEN** a MinecraftServer spec contains an invalid value that causes CUE evaluation to fail
- **THEN** the webhook SHALL return HTTP 200 with the observed children echoed back as desired state

#### Scenario: Error status is set on render failure
- **WHEN** CUE evaluation fails
- **THEN** the webhook SHALL return a status with `phase: "Error"`, a human-readable `message` containing the error, and a `Ready` condition with `status: "False"` and `reason: "RenderError"`

### Requirement: Status computation from observed state
The webhook SHALL compute the CR status exclusively from the observed children sent in the request, not from the desired state. Status SHALL report the current reality of the cluster.

#### Scenario: StatefulSet not yet ready
- **WHEN** the observed StatefulSet has `readyReplicas: 0`
- **THEN** status SHALL report `phase: "Pending"` and condition `Ready: False`

#### Scenario: StatefulSet fully ready
- **WHEN** the observed StatefulSet has `readyReplicas: 1`
- **THEN** status SHALL report `phase: "Running"` and condition `Ready: True`

#### Scenario: No observed children yet
- **WHEN** the children map contains no StatefulSet entry for this server
- **THEN** status SHALL report `phase: "Pending"` with `readyReplicas: 0`

### Requirement: Embedded module loading via Overlay
The webhook SHALL embed the minecraft-java CUE module files at compile time using `//go:embed` and load them via CUE's `load.Config.Overlay` mechanism. The module files SHALL NOT be written to disk at runtime.

#### Scenario: Module loads from embedded overlay
- **WHEN** the webhook starts up
- **THEN** it SHALL build a `load.Config.Overlay` map from the embedded filesystem and use a virtual base path for module loading

#### Scenario: Module dependencies resolve locally
- **WHEN** CUE evaluates the module
- **THEN** all `opmodel.dev` imports SHALL resolve from the embedded `cue.mod/` vendor cache without network access

### Requirement: Embedded provider definition
The webhook SHALL embed the Kubernetes provider definition as a CUE string constant and compile it at startup via `ctx.CompileString()`. No provider config file SHALL be read from disk.

#### Scenario: Provider compiles at startup
- **WHEN** the webhook process starts
- **THEN** it SHALL compile the embedded provider CUE string into a `cue.Value` and store it for use by sync handlers

### Requirement: Per-request temp file for values
The webhook SHALL write the translated CUE values to a temporary file for each sync request (required by CUE's `load.Instances` API) and SHALL clean up the file immediately after the request completes.

#### Scenario: Temp file lifecycle
- **WHEN** a sync request is processed
- **THEN** a temp directory SHALL be created, a `values.cue` file written into it, and the entire temp directory SHALL be removed via `defer` after the pipeline completes
