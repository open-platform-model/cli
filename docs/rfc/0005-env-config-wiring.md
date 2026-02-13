# RFC-0005: Environment & Config Wiring

| Field        | Value                                                                                                                                              |
|--------------|----------------------------------------------------------------------------------------------------------------------------------------------------|
| **Status**   | Draft                                                                                                                                              |
| **Created**  | 2026-02-12                                                                                                                                         |
| **Authors**  | OPM Contributors                                                                                                                                   |
| **Related**  | RFC-0001 (Release Inventory), RFC-0002 (Sensitive Data), RFC-0003 (Immutable Config), RFC-0004 (Interface Architecture), K8s Coverage Gap Analysis |

## Summary

Define the transformation rules that connect `#config` fields to Kubernetes
container environment variables, bulk injection, downward API references, and
volume-mounted secrets. This RFC is the **output side** of the data flow whose
input side is defined by [RFC-0002 (Sensitive Data Model)](0002-sensitive-data-model.md).

Today, `#ContainerSchema.env` supports only literal `name`/`value` pairs. This
RFC introduces the full `#EnvVarSchema` with four source types (`value`, `from`,
`fieldRef`, `resourceFieldRef`), bulk injection via `envFrom`, volume-mounted
secrets, the `#Secret` contract type with `$secretName`/`$dataKey` routing
fields, auto-discovery of secrets from `#config` via CUE comprehensions, and
the auto-generated `spec.secrets` schema that eliminates the manual bridging
layer.

The design was validated in
[Experiment 002: Secret Discovery](../../../catalog/experiments/002-secret-discovery/).

## Motivation

### The Problem

`#ContainerSchema.env` only supports inline string values:

```cue
env?: [string]: {
    name:  string
    value: string
}
```

This is the single highest-impact gap identified in the
[K8s Coverage Gap Analysis](../../../catalog/docs/k8s-coverage-gap-analysis.md)
(item 1.1). Missing capabilities:

1. **`valueFrom.secretKeyRef`** — reference a key in a Secret.
2. **`valueFrom.configMapKeyRef`** — reference a key in a ConfigMap.
3. **`valueFrom.fieldRef`** — downward API (pod name, namespace, node name).
4. **`valueFrom.resourceFieldRef`** — container resource limits/requests.
5. **`envFrom`** — bulk inject all keys from a ConfigMap or Secret.
6. **Volume-mounted secrets** — TLS certs, credential files, service account
   keys mounted as files rather than env vars.

Without these, module authors must hardcode sensitive values, cannot reference
pre-existing cluster resources, and have no access to pod metadata.

### Why Now

RFC-0002 defines how secrets enter OPM (`#Secret` type, input paths, provider
handlers). RFC-0003 defines immutable config with content-hash naming. Both
depend on workload transformers knowing the Kubernetes resource name for
`secretKeyRef` and `configMapKeyRef`. This RFC closes the loop by defining:

- How `#Secret` identity (`$secretName`, `$dataKey`) flows through to
  transformer output.
- Auto-discovery: `spec.secrets` is generated from `#config`, not hand-written.
- The full `#EnvVarSchema` replacing the current `{ name, value }` struct.
- Bulk injection, downward API, and volume mount wiring.

## Design

### #Secret Type Definition

`#Secret` is a **contract type** — a disjunction of fulfillment variants.
Module authors annotate `#config` fields with `#Secret`; users provide values
that resolve to one of the variants.

```cue
#Secret: #SecretLiteral | #SecretRef

#SecretLiteral: {
    $opm:         "secret"
    $secretName!: #NameType    // K8s Secret resource name (grouping key)
    $dataKey!:    string       // data key within that K8s Secret
    description?: string
    value!:       string
}

#SecretRef: {
    $opm:         "secret"
    $secretName!: #NameType
    $dataKey!:    string
    description?: string
    source!:      *"k8s" | "esc"
    path!:        string       // K8s Secret name (k8s) or external path (esc)
    remoteKey!:   string       // key within the referenced secret
}
```

Design rationale:

- **`$opm: "secret"` discriminator.** A concrete value present on every
  `#Secret` variant. Enables CUE-native auto-discovery via the negation test
  (see [Auto-Discovery](#auto-discovery)). No external tooling or tags needed.

- **`$secretName` and `$dataKey` are routing info.** The module author sets
  these in the `#config` schema declaration. `$secretName` names the K8s Secret
  resource (and acts as the grouping key). `$dataKey` names the data key within
  that K8s Secret. Multiple `#config` fields sharing the same `$secretName` are
  grouped into one K8s Secret.

- **Users never set `$secretName`/`$dataKey`.** CUE unification propagates the
  author's values through. Users only provide `value` (for literals) or
  `source`/`path`/`remoteKey` (for refs).

- **`$`-prefixed fields.** These are regular CUE fields (visible in iteration),
  not hidden. The `$` prefix is a naming convention to visually distinguish
  author-set routing fields from user-set fulfillment fields.

- **No `#SecretDeferred`.** Unfulfilled `#Secret` fields are CUE incompleteness
  errors, caught at evaluation time. If a user omits a required secret value,
  CUE itself reports the error — no special deferred variant needed.

- **`#SecretRef.remoteKey` is separate from `$dataKey`.** The external secret's
  key (in Vault, ESO, or another K8s Secret) may differ from the logical data
  key that the module uses. `$dataKey` is the output key; `remoteKey` is the
  input key.

- **`source: *"k8s" | "esc"`.** Simplified from the previous design's
  `vault`/`aws-sm`/`gcp-sm`/`k8s` enumeration. ESO (External Secrets Operator)
  abstracts over external providers, so `"esc"` covers all external sources.

Example usage:

```cue
// Module author writes (once, in definition):
#config: {
    db: {
        host:     string
        password: #Secret & {
            $secretName: "db-credentials"
            $dataKey:    "password"
        }
        username: #Secret & {
            $secretName: "db-credentials"
            $dataKey:    "username"
        }
    }
    apiKey: #Secret & {
        $secretName: "api-credentials"
        $dataKey:    "api-key"
    }
}

// User writes (no $secretName or $dataKey needed):
values: {
    db: password: { value: "my-secret" }
    db: username: { value: "admin" }
    apiKey: {
        source: "esc", path: "prod/api", remoteKey: "token"
    }
}

// CUE unifies to:
// db: password: {
//     $opm: "secret", $secretName: "db-credentials",
//     $dataKey: "password", value: "my-secret"
// }
// db: username: {
//     $opm: "secret", $secretName: "db-credentials",
//     $dataKey: "username", value: "admin"
// }
// apiKey: {
//     $opm: "secret", $secretName: "api-credentials",
//     $dataKey: "api-key", source: "esc",
//     path: "prod/api", remoteKey: "token"
// }
```

### Auto-Discovery

The fundamental shift from the previous design: **`spec.secrets` is
auto-generated from `#config`**, not hand-written by the module author. The
old "Layer 2" bridging step is eliminated entirely.

#### The Negation Test

To discover `#Secret` fields in a resolved `#config`, we test each value:

```cue
(v & {$opm: !="secret", ...}) == _|_
```

This produces bottom (error) ONLY when `$opm` is already concretely set to
`"secret"` on the value. For any other value:

- **Scalars** (`string`, `int`, `bool`): fail the struct unification —
  correctly skipped.
- **Anonymous open structs** (e.g., `{host: "localhost", port: 5432}`): the
  constraint `$opm: !="secret"` is added without conflict (the struct has no
  `$opm` field, so the constraint is satisfied) — correctly skipped.
- **Closed definition structs**: `$opm` is rejected as a disallowed field —
  correctly skipped.

The `...` (open struct marker) in the test ensures it doesn't conflict with
closed structs during the initial unification step.

**Why other approaches fail.** A simpler test like `v.$opm == "secret"` gives
false positives on anonymous open structs, because an open struct accepts any
field — `v.$opm` would evaluate to an unconstrained string, not bottom. The
negation test avoids this entirely.

#### Three-Level Traversal

CUE has no recursion, so the discovery comprehension manually traverses up
to 3 levels deep. This covers practical nesting patterns:

- Level 1: `#config.dbUser`
- Level 2: `#config.cache.password`
- Level 3: `#config.integrations.payments.stripeKey`

The `_#discoverSecrets` definition (from the experiment):

```cue
_#discoverSecrets: {
    #in: {...}
    out: {
        // Level 1: direct fields
        for k1, v1 in #in
        if ((v1 & {$opm: !="secret", ...}) == _|_)
        if ((v1 & {...}) != _|_) {
            (k1): v1
        }

        // Level 2: one level of nesting
        for k1, v1 in #in
        if ((v1 & {$opm: !="secret", ...}) != _|_)
        if ((v1 & {...}) != _|_) {
            for k2, v2 in v1
            if ((v2 & {$opm: !="secret", ...}) == _|_)
            if ((v2 & {...}) != _|_) {
                ("\(k1)/\(k2)"): v2
            }
        }

        // Level 3: two levels of nesting
        for k1, v1 in #in
        if ((v1 & {$opm: !="secret", ...}) != _|_)
        if ((v1 & {...}) != _|_) {
            for k2, v2 in v1
            if ((v2 & {$opm: !="secret", ...}) != _|_)
            if ((v2 & {...}) != _|_) {
                for k3, v3 in v2
                if ((v3 & {$opm: !="secret", ...}) == _|_)
                if ((v3 & {...}) != _|_) {
                    ("\(k1)/\(k2)/\(k3)"): v3
                }
            }
        }
    }
}
```

#### Auto-Grouping

Discovered secrets are grouped by `$secretName`, keyed by `$dataKey`:

```cue
_#groupSecrets: {
    #in: {...}
    out: {
        for _k, v in #in {
            (v.$secretName): (v.$dataKey): v
        }
    }
}
```

The result is the K8s Secret resource layout — ready for the
SecretTransformer to consume:

```cue
_discovered: (_#discoverSecrets & {#in: values}).out
spec: secrets: (_#groupSecrets & {#in: _discovered}).out
```

#### Pipeline Diagram

```text
┌──────────────────────────────────────────────────────────────────────────┐
│  AUTO-DISCOVERY PIPELINE                                                 │
│                                                                          │
│  #config (resolved with user values)                                     │
│       │                                                                  │
│       ▼                                                                  │
│  _#discoverSecrets                                                       │
│    Traverses up to 3 levels                                              │
│    Tests: (v & {$opm: !="secret", ...}) == _|_                          │
│    Collects all #Secret fields into flat map                             │
│       │                                                                  │
│       ▼                                                                  │
│  _#groupSecrets                                                          │
│    Groups by $secretName, keyed by $dataKey                              │
│    Produces K8s Secret resource layout                                   │
│       │                                                                  │
│       ▼                                                                  │
│  spec.secrets (auto-generated)                                           │
│    "db-credentials":                                                     │
│      username: {$opm: "secret", value: "admin", ...}                    │
│      password: {$opm: "secret", source: "k8s", ...}                    │
│    "api-credentials":                                                    │
│      api-key: {$opm: "secret", value: "sk_live_...", ...}                │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### SecretTransformer Variant Dispatch

The SecretTransformer reads `spec.secrets` (auto-generated) and inspects the
`#Secret` variant of each data entry. **Each entry is dispatched
independently** — mixed variants within a group are supported.

```text
┌──────────────────────────────┬───────────────────────────────────────────┐
│ Entry variant                │ SecretTransformer action                  │
├──────────────────────────────┼───────────────────────────────────────────┤
│ #SecretLiteral               │ Include in K8s Secret                     │
│                              │   data[entry.$dataKey]: base64(value)     │
│                              │                                           │
│ #SecretRef (esc)             │ ExternalSecret CR                         │
│                              │   metadata.name: $secretName              │
│                              │   spec.data: [{secretKey: $dataKey,       │
│                              │     remoteRef: {key: path,                │
│                              │       property: remoteKey}}]              │
│                              │   spec.target.name: $secretName           │
│                              │                                           │
│ #SecretRef (k8s)             │ Skip. Resource already exists.            │
│                              │                                           │
│ Mixed variants in group      │ Supported. Each entry dispatched          │
│                              │ independently. Literal entries create     │
│                              │ a K8s Secret; ref entries handled         │
│                              │ per-variant.                              │
└──────────────────────────────┴───────────────────────────────────────────┘
```

**Mixed variants are supported.** Unlike the previous design which rejected
mixed variants, each entry within a `spec.secrets` group is dispatched on its
own variant. A group like `"db-credentials"` can contain a `#SecretLiteral`
for `username` and a `#SecretRef (k8s)` for `password`. The literal creates a
K8s Secret data entry; the k8s ref is skipped. This was validated in
experiment 002.

**Silent skip for `source: "k8s"`.** When an entry resolves to `#SecretRef`
with `source: "k8s"`, the SecretTransformer emits nothing for that entry.
The referenced Secret already exists in the cluster.

### #EnvVarSchema

The full environment variable schema replaces the current `{ name, value }`
struct:

```cue
#EnvVarSchema: {
    name!: string

    // Source — exactly one must be set
    value?:            string               // inline literal (non-sensitive)
    from?:             #Secret              // reference to a #Secret in #config
    fieldRef?:         #FieldRefSchema
    resourceFieldRef?: #ResourceFieldRefSchema
}

#FieldRefSchema: {
    fieldPath!: string   // e.g., "metadata.name", "metadata.namespace",
                         //       "metadata.labels['app']", "spec.nodeName",
                         //       "status.podIP"
    apiVersion?: string | *"v1"
}

#ResourceFieldRefSchema: {
    resource!:      string   // e.g., "limits.cpu", "requests.memory"
    containerName?: string   // defaults to current container
    divisor?:       string   // e.g., "1m" for millicores, "1Mi" for mebibytes
}
```

Mutual exclusivity: exactly one of `value`, `from`, `fieldRef`, or
`resourceFieldRef` should be set. The workload transformer validates this
constraint — setting two sources on a single env var is an error.

The `name` field is auto-set from the map key, following the existing OPM
pattern:

```cue
env?: [envName=string]: #EnvVarSchema & {name: envName}
```

The `from:` field carries the resolved `#Secret` value with all routing info
(`$opm`, `$secretName`, `$dataKey`, and the variant-specific fields). The
workload transformer reads the variant and dispatches accordingly.

### Env Var Wiring

The module author wires env vars by referencing `#config` values:

```cue
spec: container: env: {
    DB_USER:     { from: values.dbUser }
    DB_PASSWORD: { from: values.dbPassword }
    LOG_LEVEL:   { value: values.logLevel }
}
```

For `value:` fields, the transformer emits an inline `value`. For `from:`
fields, the transformer reads the `#Secret` variant and dispatches to the
appropriate `valueFrom` structure. No manual `spec.secrets` wiring needed —
the auto-discovery pipeline handles resource creation.

### Workload Transformer Behavior

The workload transformer (DeploymentTransformer, StatefulSetTransformer, etc.)
handles each env var source type:

```text
┌──────────────────────────┬──────────────────────────────────────────────┐
│ Source                   │ K8s Output                                   │
├──────────────────────────┼──────────────────────────────────────────────┤
│ value: "info"            │ env:                                         │
│                          │   - name: LOG_LEVEL                          │
│                          │     value: "info"                            │
│                          │                                              │
│ from: #SecretLiteral     │ env:                                         │
│   ($secretName: "db-c")  │   - name: DB_PASSWORD                       │
│   ($dataKey: "password") │     valueFrom:                              │
│                          │       secretKeyRef:                          │
│                          │         name: db-credentials                 │
│                          │         key: password                        │
│                          │                                              │
│ from: #SecretRef (esc)   │ env:                                         │
│   ($secretName: "cache") │   - name: CACHE_PW                          │
│   ($dataKey: "password") │     valueFrom:                              │
│                          │       secretKeyRef:                          │
│                          │         name: cache-credentials              │
│                          │         key: password                        │
│                          │ (ESC creates Secret named $secretName)       │
│                          │                                              │
│ from: #SecretRef (k8s)   │ env:                                         │
│   (path: "existing-sec") │   - name: DB_PASSWORD                       │
│   (remoteKey: "pw")      │     valueFrom:                              │
│                          │       secretKeyRef:                          │
│                          │         name: existing-sec                   │
│                          │         key: pw                              │
│                          │                                              │
│ fieldRef:                │ env:                                         │
│   fieldPath: "meta..."   │   - name: POD_NAME                          │
│                          │     valueFrom:                              │
│                          │       fieldRef:                              │
│                          │         fieldPath: metadata.name             │
│                          │                                              │
│ resourceFieldRef:        │ env:                                         │
│   resource: "limits.cpu" │   - name: CPU_LIMIT                         │
│                          │     valueFrom:                              │
│                          │       resourceFieldRef:                     │
│                          │         resource: limits.cpu                 │
└──────────────────────────┴──────────────────────────────────────────────┘
```

**Name resolution for `secretKeyRef`:**

```text
#SecretLiteral       -> secretKeyRef: { name: $secretName, key: $dataKey }
#SecretRef (esc)     -> secretKeyRef: { name: $secretName, key: $dataKey }
                        (ESC creates the target Secret named $secretName)
#SecretRef (k8s)     -> secretKeyRef: { name: path, key: remoteKey }
```

The transformer converts env from OPM's struct-keyed map to Kubernetes' list
format, applying the appropriate `valueFrom` structure based on which source
field is set. The `envFrom` list passes through with minimal transformation.

### envFrom — Bulk Injection

Inject all keys from a ConfigMap or Secret as environment variables:

```cue
#ContainerSchema: {
    // ... existing fields ...
    env?:     [envName=string]: #EnvVarSchema & {name: envName}
    envFrom?: [...#EnvFromSource]
}

#EnvFromSource: {
    secretRef?:    { name!: string }    // K8s Secret name (= $secretName)
    configMapRef?: { name!: string }    // full K8s ConfigMap name
    prefix?:       string               // optional prefix for injected keys
}
```

`secretRef.name` and `configMapRef.name` use the **full Kubernetes resource
name**. For secrets declared in `#config`, this is the `$secretName` value:

```cue
spec: container: {
    envFrom: [{
        secretRef: name: "db-credentials"   // matches $secretName
    }]
}
```

When referencing a pre-existing Kubernetes resource (not managed by OPM), the
raw name is used directly:

```cue
spec: container: {
    envFrom: [{
        configMapRef: name: "shared-feature-flags"
        prefix: "FF_"
    }]
}
```

### fieldRef — Downward API

Expose pod and container metadata as environment variables:

```cue
spec: container: env: {
    POD_NAME:      { fieldRef: fieldPath: "metadata.name" }
    POD_NAMESPACE: { fieldRef: fieldPath: "metadata.namespace" }
    NODE_NAME:     { fieldRef: fieldPath: "spec.nodeName" }
    POD_IP:        { fieldRef: fieldPath: "status.podIP" }
}
```

Supported `fieldPath` values (Kubernetes v1 API):

| fieldPath | Description |
|-----------|-------------|
| `metadata.name` | Pod name |
| `metadata.namespace` | Pod namespace |
| `metadata.uid` | Pod UID |
| `metadata.labels['<KEY>']` | Specific pod label |
| `metadata.annotations['<KEY>']` | Specific pod annotation |
| `spec.nodeName` | Node the pod is scheduled on |
| `spec.serviceAccountName` | Service account name |
| `status.podIP` | Pod IP address |
| `status.hostIP` | Node IP address |

### resourceFieldRef — Container Resources

Expose container resource limits and requests as environment variables:

```cue
spec: container: env: {
    CPU_LIMIT:    { resourceFieldRef: resource: "limits.cpu" }
    MEMORY_LIMIT: { resourceFieldRef: { resource: "limits.memory", divisor: "1Mi" } }
    CPU_REQUEST:  { resourceFieldRef: resource: "requests.cpu" }
}
```

Supported `resource` values:

| resource | Description |
|----------|-------------|
| `limits.cpu` | CPU limit |
| `limits.memory` | Memory limit |
| `limits.ephemeral-storage` | Ephemeral storage limit |
| `requests.cpu` | CPU request |
| `requests.memory` | Memory request |
| `requests.ephemeral-storage` | Ephemeral storage request |

The `divisor` field controls the unit of the output value. Without a divisor,
CPU is reported in cores and memory in bytes.

### Volume-Mounted Secrets

Secrets can be mounted as files via `from:` on volume mounts. The same
`#Secret` type used for env vars drives the volume wiring:

```cue
#config: tls: #Secret & {
    $secretName: "tls-cert"
    $dataKey:    "tls.crt"
}

spec: container: {
    volumeMounts: "tls": {
        mountPath: "/etc/tls"
        from:      #config.tls
    }
}
```

When the workload transformer encounters `from:` on a volume mount resolving
to `#Secret`, it emits:

1. A `volumes[]` entry on the pod spec referencing the Secret by its
   `$secretName` (or `path` for `source: "k8s"`).
2. A `volumeMount` on the container at the specified `mountPath`.

The K8s Secret resource itself is emitted by the SecretTransformer from the
auto-generated `spec.secrets`.

```text
from: #config.tls  ($secretName: "tls-cert")

Workload transformer emits on pod spec:
  volumes:
    - name: tls
      secret:
        secretName: tls-cert

  containers[0].volumeMounts:
    - name: tls
      mountPath: /etc/tls
```

For `#SecretRef` with `source: "k8s"`, the volume references the `path`
(pre-existing Secret name) and uses `remoteKey` for item selection:

```text
from: #config.tls  (source: "k8s", path: "wildcard-tls", remoteKey: "tls.crt")

Workload transformer emits on pod spec:
  volumes:
    - name: tls
      secret:
        secretName: wildcard-tls
        items:
          - key: tls.crt
            path: tls.crt

  containers[0].volumeMounts:
    - name: tls
      mountPath: /etc/tls
```

### #ConfigRef (Placeholder)

This RFC focuses on `#Secret` wiring. A parallel `#ConfigRef` type for
non-sensitive ConfigMap-backed values may be introduced in a future RFC. The
design would mirror `#Secret` — a struct type with `$configMapName` and
`$dataKey` fields that the ConfigMapTransformer and WorkloadTransformer both
read.

For now, non-sensitive env vars use `value:` (inline strings). If
ConfigMap-backed config is needed, use `spec.configMaps` directly combined
with `envFrom` or the existing ConfigMapTransformer.

## Interaction with Other RFCs

### RFC-0001: Release Inventory

The inventory tracks all resources emitted by transformers, including Secrets
generated from `spec.secrets` entries. When a secret value changes (and the
secret is immutable per RFC-0003), the old hash-suffixed resource appears in
the stale set and is pruned automatically.

### RFC-0002: Sensitive Data Model

This RFC is the output counterpart to RFC-0002's input model:

```text
┌─────────────────────────────────────────────────────────────────────────┐
│  RFC-0002 (Input)                    RFC-0005 (Output)                   │
│  ═══════════════                     ════════════════                    │
│  #Secret type definition             #Secret contract type               │
│  #SecretLiteral | #SecretRef         $opm discriminator                  │
│  $secretName + $dataKey routing      Auto-discovery pipeline             │
│  Provider handler interface          Auto-generated spec.secrets         │
│  Input paths (literal, ref)          SecretTransformer variant dispatch  │
│                                      #EnvVarSchema (value, from,         │
│                                        fieldRef, resourceFieldRef)       │
│                                      envFrom bulk injection              │
│                                      Volume mount transformer output     │
└─────────────────────────────────────────────────────────────────────────┘
```

RFC-0002 defines `#Secret` with `$secretName!` and `$dataKey!`, the `#Secret`
variants, and the `from?: #Secret` field on `#EnvVarSchema`. This RFC extends
the schema with `fieldRef` and `resourceFieldRef`, defines auto-discovery,
and specifies how all source types transform to Kubernetes output.

### RFC-0003: Immutable ConfigMaps and Secrets

The `#Secret.$secretName` field introduced here is the base name that
RFC-0003's `#ImmutableName` definition uses for hash-suffix computation. When
a `spec.secrets` entry is marked immutable, the SecretTransformer and
WorkloadTransformer both compute:

```text
#SecretImmutableName & {
    #baseName:  secret.$secretName
    #data:      secret.data
    #immutable: true
}
-> out: "{$secretName}-{content-hash}"
```

Both transformers arrive at the same hashed name because they read the same
`$secretName` and data from the same auto-generated `spec.secrets`.

### RFC-0004: Interface Architecture

Interface shapes include `#Secret` fields (e.g., `#Postgres.password`). The
wiring pattern is identical — `from:` references the interface field:

```cue
requires: "db": #Postgres

spec: container: env: {
    DB_HOST:     { value: requires.db.host }
    DB_PASSWORD: { from:  requires.db.password }
}
```

The platform fulfillment provides the `#Secret` value (with `$secretName`
and `$dataKey` set by the interface definition). The module author's wiring
is unchanged regardless of how the secret was fulfilled.

## Scenarios

### Scenario A: Literal Env Var [x]

```text
spec: container: env: {
    LOG_LEVEL: { value: "info" }
}

Transformer emits:
  env:
    - name: LOG_LEVEL
      value: "info"

No K8s resource generated. [x]
```

### Scenario B: Secret Env Var via #SecretLiteral [x]

```text
Schema:
  #config: db: password: #Secret & {
      $secretName: "db-credentials", $dataKey: "password"
  }
  values: db: password: { value: "my-secret" }

Env wiring:
  spec: container: env: { DB_PASSWORD: { from: values.db.password } }

Auto-discovery:
  _discovered: { "db/password": {$opm: "secret", $secretName: "db-credentials",
      $dataKey: "password", value: "my-secret"} }

Auto-grouped spec.secrets:
  "db-credentials": { password: { ... value: "my-secret" } }

SecretTransformer emits:
  Secret/db-credentials  { data: { password: base64("my-secret") } }

WorkloadTransformer emits:
  env:
    - name: DB_PASSWORD
      valueFrom:
        secretKeyRef:
          name: db-credentials
          key: password

Both transformers use "db-credentials" ($secretName) independently. [x]
```

### Scenario C: Multi-Key Secret [x]

```text
Schema:
  #config: db: {
      password: #Secret & { $secretName: "db-credentials", $dataKey: "password" }
      username: #Secret & { $secretName: "db-credentials", $dataKey: "username" }
  }
  values: db: { password: { value: "secret" }, username: { value: "admin" } }

Env wiring:
  spec: container: env: {
      DB_PASSWORD: { from: values.db.password }
      DB_USERNAME: { from: values.db.username }
  }

Auto-grouped spec.secrets:
  "db-credentials": {
      password: { ... value: "secret" }
      username: { ... value: "admin" }
  }

SecretTransformer emits:
  Secret/db-credentials  {
      data: { password: base64("secret"), username: base64("admin") }
  }

WorkloadTransformer emits:
  env:
    - name: DB_PASSWORD
      valueFrom: { secretKeyRef: { name: db-credentials, key: password } }
    - name: DB_USERNAME
      valueFrom: { secretKeyRef: { name: db-credentials, key: username } }

Two #config fields, one K8s Secret, two env vars. [x]
```

### Scenario D: Pre-existing K8s Secret via #SecretRef [x]

```text
Schema:
  #config: db: password: #Secret & {
      $secretName: "db-credentials", $dataKey: "password"
  }
  values: db: password: {
      source: "k8s", path: "existing-db-secret", remoteKey: "pw"
  }

Auto-grouped spec.secrets:
  "db-credentials": {
      password: { ... source: "k8s", path: "existing-db-secret", remoteKey: "pw" }
  }

SecretTransformer: source: "k8s" -> skip, emit nothing.

WorkloadTransformer emits:
  env:
    - name: DB_PASSWORD
      valueFrom:
        secretKeyRef:
          name: existing-db-secret    (uses path, not $secretName)
          key: pw                     (uses remoteKey, not $dataKey)

Module definition unchanged regardless of how users fulfill secrets. [x]
```

### Scenario E: ExternalSecret via #SecretRef (esc) [x]

```text
Schema:
  #config: cache: password: #Secret & {
      $secretName: "cache-credentials", $dataKey: "password"
  }
  values: cache: password: {
      source: "esc", path: "production/redis", remoteKey: "password"
  }

Auto-grouped spec.secrets:
  "cache-credentials": {
      password: { ... source: "esc", path: "production/redis",
          remoteKey: "password" }
  }

SecretTransformer emits:
  ExternalSecret/cache-credentials
    spec.data: [{ secretKey: "password",
        remoteRef: { key: "production/redis", property: "password" } }]
    spec.target.name: "cache-credentials"

WorkloadTransformer emits:
  env:
    - name: CACHE_PASSWORD
      valueFrom:
        secretKeyRef:
          name: cache-credentials     (ESC creates this Secret)
          key: password

[x]
```

### Scenario F: Mixed Variants in Same Group [x]

```text
Schema:
  #config: db: {
      username: #Secret & { $secretName: "db-credentials", $dataKey: "username" }
      password: #Secret & { $secretName: "db-credentials", $dataKey: "password" }
  }
  values: db: {
      username: { value: "admin" }                                    // literal
      password: { source: "k8s", path: "myapp-secrets", remoteKey: "pw" } // k8s ref
  }

Auto-grouped spec.secrets:
  "db-credentials": {
      username: { $opm: "secret", value: "admin", ... }              // literal
      password: { $opm: "secret", source: "k8s", path: "myapp-secrets", ... }
  }

SecretTransformer dispatches each entry independently:
  username (literal) -> K8s Secret data entry: username: base64("admin")
  password (k8s ref) -> skip (pre-existing)

  Emits: Secret/db-credentials { data: { username: base64("admin") } }
  (Only literal entries included in the K8s Secret)

WorkloadTransformer emits:
  env:
    - name: DB_USERNAME
      valueFrom: { secretKeyRef: { name: db-credentials, key: username } }
    - name: DB_PASSWORD
      valueFrom: { secretKeyRef: { name: myapp-secrets, key: pw } }

Mixed variants within a group: supported. [x]
```

### Scenario G: Nested Secrets (Level 2 and Level 3) [x]

```text
Schema:
  #config: {
      cache: {
          password: #Secret & {
              $secretName: "cache-credentials", $dataKey: "password"
          }
      }
      integrations: payments: {
          stripeKey: #Secret & {
              $secretName: "stripe-credentials", $dataKey: "secret-key"
          }
          webhookSecret: #Secret & {
              $secretName: "stripe-credentials", $dataKey: "webhook-secret"
          }
      }
  }
  values: {
      cache: password: { value: "redis-pw" }
      integrations: payments: {
          stripeKey:     { value: "sk_live_abc" }
          webhookSecret: { value: "whsec_xyz" }
      }
  }

Auto-discovery traverses:
  Level 2: finds cache/password
  Level 3: finds integrations/payments/stripeKey,
           integrations/payments/webhookSecret

Auto-grouped spec.secrets:
  "cache-credentials":  { password: { ... value: "redis-pw" } }
  "stripe-credentials": {
      secret-key:     { ... value: "sk_live_abc" }
      webhook-secret: { ... value: "whsec_xyz" }
  }

SecretTransformer emits:
  Secret/cache-credentials  { data: { password: base64("redis-pw") } }
  Secret/stripe-credentials {
      data: { secret-key: base64("sk_live_abc"),
              webhook-secret: base64("whsec_xyz") }
  }

[x]
```

### Scenario H: Downward API via fieldRef [x]

```text
spec: container: env: {
    POD_NAME:      { fieldRef: fieldPath: "metadata.name" }
    POD_NAMESPACE: { fieldRef: fieldPath: "metadata.namespace" }
}

Transformer emits:
  env:
    - name: POD_NAME
      valueFrom: { fieldRef: { fieldPath: metadata.name } }
    - name: POD_NAMESPACE
      valueFrom: { fieldRef: { fieldPath: metadata.namespace } }

No K8s resources generated. Pure pass-through. [x]
```

### Scenario I: Bulk Injection via envFrom [x]

```text
spec: container: {
    envFrom: [
        { secretRef: name: "db-credentials" },
        { configMapRef: name: "shared-feature-flags", prefix: "FF_" },
    ]
}

Transformer emits:
  envFrom:
    - secretRef: { name: db-credentials }
    - configMapRef: { name: shared-feature-flags }
      prefix: FF_

secretRef.name matches $secretName from #config declarations.
Resources must exist (managed by OPM via auto-discovery or pre-existing). [x]
```

### Scenario J: Volume-Mounted Secret [x]

```text
#config: tls: #Secret & { $secretName: "tls-cert", $dataKey: "tls.crt" }
values:  tls: { source: "k8s", path: "wildcard-tls", remoteKey: "tls.crt" }

spec: container: {
    volumeMounts: "tls": { mountPath: "/etc/tls", from: values.tls }
}

SecretTransformer: source: "k8s" -> nothing emitted.

WorkloadTransformer emits:
  volumes:
    - name: tls
      secret:
        secretName: wildcard-tls   (uses path, not $secretName)
  containers[0].volumeMounts:
    - name: tls
      mountPath: /etc/tls

[x]
```

### Scenario K: Mixed — All Wiring Types [x]

```text
#config: {
    logLevel: string
    db: {
        host:     string
        password: #Secret & {
            $secretName: "db-credentials", $dataKey: "password"
        }
    }
    tls: #Secret & { $secretName: "tls-cert", $dataKey: "tls.crt" }
}

values: {
    logLevel: "info"
    db: {
        host:     "db.prod.internal"
        password: { value: "my-secret" }
    }
    tls: { source: "k8s", path: "wildcard-tls", remoteKey: "tls.crt" }
}

spec: container: {
    env: {
        LOG_LEVEL:   { value: values.logLevel }
        DB_HOST:     { value: values.db.host }
        DB_PASSWORD: { from:  values.db.password }
        POD_NAME:    { fieldRef: fieldPath: "metadata.name" }
        CPU_LIMIT:   { resourceFieldRef: resource: "limits.cpu" }
    }
    envFrom: [{
        configMapRef: name: "shared-feature-flags"
    }]
    volumeMounts: "tls": { mountPath: "/etc/tls", from: values.tls }
}

Auto-generated spec.secrets:
  "db-credentials": { password: { ... value: "my-secret" } }
  "tls-cert":       { tls.crt: { ... source: "k8s", path: "wildcard-tls" } }

Result:
  5 env vars (2 literal, 1 secretKeyRef, 1 fieldRef, 1 resourceFieldRef)
  1 envFrom (configMapRef)
  1 volume mount (secret)
  1 K8s Secret emitted (db-credentials); tls-cert skipped (k8s ref)
  All coexist cleanly. [x]
```

## Open Questions

### Q1: envFrom Shorthand for #config-Declared Secrets

Should there be a shorthand for bulk-injecting all keys from a `#config`-
declared secret without repeating the name? For example:

```cue
// Hypothetical shorthand
envFrom: [{ fromSecret: #config.db.creds }]

// vs. current (explicit $secretName)
envFrom: [{ secretRef: name: "db-credentials" }]
```

### Q2: $opm Field Visibility

The `$opm: "secret"` discriminator field appears in the resolved output of
`spec.secrets` entries. Should it?

Options:
1. **Accept it.** It is the discriminator — transformers need it to dispatch.
   Strip it only in final K8s resource output (the transformer already
   controls what goes into `data:`).
2. **Strip in transformer output.** The transformer reads `$opm` for dispatch
   but excludes it from the emitted K8s Secret `data` map.
3. **Hidden field approach.** Use `_$opm` (CUE hidden field) instead, which
   would not appear in normal evaluation output. Downside: hidden fields
   are not visible during iteration in comprehensions.

Current recommendation: option 2. The transformer uses `$opm` for dispatch
but only emits `$dataKey: base64(value)` pairs into K8s Secret data.

### Q3: Deep Nesting Beyond 3 Levels

The CUE-based discovery traverses 3 levels, which covers practical nesting
patterns. If deeper nesting is needed, the Go SDK can handle arbitrary depth
via programmatic traversal.

Is 3 levels sufficient? Current evidence says yes — module configs rarely
nest secrets deeper than `integrations.payments.stripeKey` (level 3).

## Deferred Work

### #ConfigRef Full Design

A `#ConfigRef` type parallel to `#Secret` for ConfigMap-backed non-sensitive
values. Would enable `from:` syntax for ConfigMaps with the same
transformer-independent name resolution. Deferred until demand is clear.

### ConfigMap Aggregation

When multiple non-sensitive config values exist, should they be aggregated
into a single ConfigMap per component? This optimization is provider-specific
and deferred to the `#ConfigRef` design.

### @opm() Attribute

A future alternative to the `$opm` discriminator field. CUE attributes
(e.g., `@opm(secret)`) are metadata that don't affect the value graph.
This could provide a cleaner discovery mechanism via the Go SDK's attribute
API, avoiding the `$opm` field visibility question entirely. Deferred until
the Go-side processing pipeline is more mature.

### Deep Traversal via Go SDK

If the 3-level nesting limit in CUE comprehensions proves insufficient, the
Go SDK can walk the CUE value tree to arbitrary depth. This would replace
the `_#discoverSecrets` comprehension with programmatic traversal in the
build pipeline. Deferred until a concrete use case demands deeper nesting.

## References

- [RFC-0001: Release Inventory](0001-release-inventory.md) — Inventory-based pruning for stale resources
- [RFC-0002: Sensitive Data Model](0002-sensitive-data-model.md) — `#Secret` type, input paths, provider handlers
- [RFC-0003: Immutable ConfigMaps and Secrets](0003-immutable-config.md) — Content-hash naming and K8s `immutable` flag
- [RFC-0004: Interface Architecture](0004-interface-architecture.md) — `provides`/`requires` with typed shapes
- [Experiment 002: Secret Discovery](../../../catalog/experiments/002-secret-discovery/) — Auto-discovery validation
- [K8s Coverage Gap Analysis](../../../catalog/docs/k8s-coverage-gap-analysis.md) — Item 1.1: env valueFrom references
- [Kubernetes Downward API](https://kubernetes.io/docs/concepts/workloads/pods/downward-api/) — Pod and container metadata as env vars
- [Kubernetes env and envFrom](https://kubernetes.io/docs/tasks/inject-data-application/define-environment-variable-container/) — Environment variable configuration
