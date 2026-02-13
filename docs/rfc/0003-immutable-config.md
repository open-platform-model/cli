# RFC-0003: Immutable ConfigMaps and Secrets

| Field        | Value                                                                                    |
|--------------|------------------------------------------------------------------------------------------|
| **Status**   | Draft                                                                                    |
| **Created**  | 2026-02-11                                                                               |
| **Authors**  | OPM Contributors                                                                         |
| **Related**  | RFC-0001 (Release Inventory), RFC-0002 (Sensitive Data Model), RFC-0005 (Env & Config Wiring), K8s Coverage Gap Analysis |

## Summary

Add an `immutable` field to `#SecretSchema` and `#ConfigMapSchema`. When set to
`true`, the transformer appends a content-hash suffix to the Kubernetes resource
name and sets the native `spec.immutable: true` field on the emitted resource.
Content changes produce a new name, which triggers workload rolling updates (once
env wiring is implemented) and causes the old resource to be garbage collected
via the Release Inventory (RFC-0001).

This is the OPM equivalent of Kustomize's `configMapGenerator`/`secretGenerator`
and Timoni's `#ImmutableConfig`.

## Motivation

### The Problem

OPM currently treats ConfigMaps and Secrets as mutable resources updated in-place
via Server-Side Apply. This has three consequences:

1. **No rolling updates on config change.** When a ConfigMap or Secret's data
   changes, the resource is updated but workloads consuming it are not restarted.
   Running pods continue using stale configuration until they are manually cycled
   or happen to restart.

2. **No protection against accidental edits.** A `kubectl edit` on a Secret can
   silently break a running application. There is no guard against drift between
   the OPM-defined state and the live state.

3. **Performance cost.** The kubelet watches every Secret and ConfigMap that a
   pod references, polling for changes. For clusters with many Secrets, this is a
   measurable overhead. Kubernetes' native `spec.immutable: true` disables this
   watch, but OPM never sets it.

### The Industry Pattern: Rename-as-Update

The established solution across the Kubernetes ecosystem is to treat config
resources as immutable, append a content hash to the name, and let the name
change propagate through the system:

```text
┌──────────────────────────────────────────────────────────────────────────┐
│  THE RENAME-AS-UPDATE PATTERN                                            │
│                                                                          │
│  v1: Secret/db-creds-a3f8b2c1e9                                         │
│      Deployment envFrom → db-creds-a3f8b2c1e9                           │
│                                                                          │
│  Data changes (password rotated):                                        │
│                                                                          │
│  v2: Secret/db-creds-7c2d91f0b4   ← new hash, new resource              │
│      Deployment envFrom → db-creds-7c2d91f0b4  ← pod template changed   │
│                                                                          │
│  Result:                                                                 │
│  • Deployment sees pod template change → rolling update                  │
│  • Old pods finish with old config (no mid-request disruption)           │
│  • New pods start with new config                                        │
│  • Old Secret pruned after rollout completes                             │
│  • kubelet skips watching (immutable: true)                              │
│                                                                          │
│  This pattern is used by:                                                │
│  • Kustomize (configMapGenerator / secretGenerator)                      │
│  • Timoni (#ImmutableConfig)                                             │
│  • Various Helm community patterns                                       │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Why Now

Two concurrent developments make this the right time:

1. **RFC-0001 (Release Inventory)** provides the garbage collection mechanism.
   Without inventory-based pruning, old hash-suffixed resources would accumulate
   indefinitely. With it, they are automatically cleaned up as stale resources.

2. **The env valueFrom gap** (K8s Coverage Gap Analysis, item 1.1) is the next
   major schema change. Designing immutable now ensures that when env wiring is
   implemented, the hash-suffixed names flow naturally into workload references,
   completing the rolling update story.

## Prior Art

### Timoni `#ImmutableConfig`

Timoni is the closest architectural analog to OPM — both are CUE-based,
CLI-driven, and module-oriented. Timoni's approach:

- Module author wraps config in `#ImmutableConfig`, specifying `#Kind`
  (Secret or ConfigMap), `#Meta`, and `#Data`.
- Timoni computes a hash of the `#Data` content and appends it to the resource
  name: `<instance-name>-<data-hash>`.
- Sets `spec.immutable: true` on the Kubernetes resource.
- The module author explicitly passes the computed name to workload templates
  (e.g., `#secretName` parameter on Deployment).
- Old resources are garbage collected via Timoni's inventory system.
- A `#Suffix` field is available to disambiguate multiple configs per component.

**Key takeaway**: The module author is responsible for wiring the computed name
into workload references. This maps directly to OPM's duplicate hash computation
pattern.

### Kustomize `configMapGenerator` / `secretGenerator`

- Generates ConfigMaps and Secrets with a content-hash suffix (10 characters).
- Automatically rewrites all references across the kustomization: `configMapRef`,
  `secretRef`, `configMapKeyRef`, `secretKeyRef`, and volume references.
- Hash suffix can be disabled globally or per-resource via `generatorOptions`.
- No built-in garbage collection — relies on `kubectl apply --prune` or external
  tooling.

**Key takeaway**: Kustomize's automatic reference rewriting is powerful but
requires a global post-processing pass. OPM's transformer architecture does not
support cross-resource rewriting, making duplicate hash computation the better
fit.

### Helm Checksum Annotations

Helm uses a different approach — no name change, no immutability:

```yaml
spec:
  template:
    metadata:
      annotations:
        checksum/config: {{ include "mychart/configmap.yaml" . | sha256sum }}
```

The ConfigMap is updated in-place. The annotation on the pod template changes,
triggering a rolling update. This is simpler but has downsides:

- Running pods can see partial config updates (if mounted as volumes).
- No kubelet watch optimization (resource is mutable).
- No protection against manual edits.

**Key takeaway**: OPM should prefer true immutability over annotation-based
tricks. The rename-as-update pattern is strictly better when inventory-based
GC is available.

### Kubernetes Native `spec.immutable`

Kubernetes ConfigMaps and Secrets have an `immutable` field (stable since v1.21):

- Once set to `true`, the API server rejects any updates to `data`,
  `binaryData`, or the `immutable` field itself.
- The only way to change the data is to delete and recreate the resource.
- The kubelet stops watching the resource, reducing API server load.
- The field does not trigger rolling updates or manage resource naming.

**Key takeaway**: The native field is a performance and safety optimization.
OPM should set it on immutable resources for defense in depth, but the
rename-as-update pattern is what actually drives the lifecycle.

## Design

### Schema Changes

Add `immutable?: bool | *false` to both `#SecretSchema` and `#ConfigMapSchema`
in `schemas/config.cue`:

```cue
#SecretSchema: {
    type?:      string | *"Opaque"
    immutable?: bool | *false
    data:       [dataKey=string]: #Secret
}

#ConfigMapSchema: {
    immutable?: bool | *false
    data:       [string]: string
}
```

The field is **per-entry** — each secret or configmap in the map can
independently be immutable or mutable:

```cue
spec: secrets: {
    "db-creds": {
        immutable: true              // hash-suffixed, K8s immutable: true
        data: {
            password: #config.db.password   // #Secret reference
            username: #config.db.username   // #Secret reference
        }
    }
    "feature-flags": {
        // immutable defaults to false — mutable, updated in-place
        data: { enable_beta: #config.feature.beta }
    }
}

spec: configMaps: {
    "app-settings": {
        immutable: true
        data: {
            log_level:   "info"
            max_retries: "3"
        }
    }
}
```

### Content Hash Computation

A shared CUE definition computes the content hash. It lives in
`schemas/config.cue` alongside the schema definitions, since both the config
schemas and the transformers need access to it.

```cue
import (
    "crypto/sha256"
    "encoding/hex"
    "list"
    "strings"
)

// #ContentHash computes a deterministic 10-character hex hash of a string data map.
// Used by ConfigMapTransformer and as a building block for #SecretContentHash.
#ContentHash: {
    #data: [string]: string

    // Step 1: Extract and sort keys for deterministic ordering
    let _keys = [ for k, _ in #data { k } ]
    let _sorted = list.SortStrings(_keys)

    // Step 2: Concatenate sorted key=value pairs
    let _pairs = [ for _, k in _sorted { "\(k)=\(#data[k])" } ]
    let _concat = strings.Join(_pairs, "\n")

    // Step 3: SHA256 and take first 5 bytes (10 hex characters)
    out: hex.Encode(sha256.Sum256(_concat)[:5])
}

// #SecretContentHash normalizes #Secret entries to strings, then delegates
// to #ContentHash. The normalization is variant-aware:
//   #SecretLiteral  -> key=<value>
//   #SecretRef      -> key=ref:<source>:<path>:<remoteKey>
#SecretContentHash: {
    #data: [string]: #Secret

    // Normalize each #Secret entry to a string for hashing
    let _normalized = {
        for k, v in #data {
            if (v & #SecretLiteral) != _|_ {
                "\(k)": v.value
            }
            if (v & #SecretRef) != _|_ {
                "\(k)": "ref:\(v.source):\(v.path):\(v.remoteKey)"
            }
        }
    }

    out: (#ContentHash & { #data: _normalized }).out
}

// #ImmutableName computes the resource name, with or without hash suffix.
// Used by ConfigMapTransformer with string data.
#ImmutableName: {
    #baseName:  string
    #data:      [string]: string
    #immutable: bool | *false

    _hash: (#ContentHash & { #data: #data }).out

    if #immutable {
        out: "\(#baseName)-\(_hash)"
    }
    if !#immutable {
        out: #baseName
    }
}

// #SecretImmutableName computes the resource name for secrets with #Secret data.
// Used by SecretTransformer.
#SecretImmutableName: {
    #baseName:  string
    #data:      [string]: #Secret
    #immutable: bool | *false

    _hash: (#SecretContentHash & { #data: #data }).out

    if #immutable {
        out: "\(#baseName)-\(_hash)"
    }
    if !#immutable {
        out: #baseName
    }
}
```

**Algorithm details:**

- **Input**: For ConfigMaps, only the `data` field string values. For Secrets,
  the `#Secret` entries are normalized: `#SecretLiteral` hashes the `value`
  field, `#SecretRef` hashes `source:path:remoteKey` as a composite string.
  The `type` field (for Secrets) and `immutable` flag itself are excluded.
  Changing `type` from `Opaque` to `kubernetes.io/tls` without changing data
  does not create a new resource. For `#SecretRef`, the hash captures the
  reference metadata — changes to the external path trigger recreation, while
  runtime secret rotation by the external store does not.
- **Determinism**: Keys are explicitly sorted with `list.SortStrings` before
  concatenation. CUE struct field ordering is deterministic, but explicit
  sorting removes any dependency on declaration order.
- **Length**: 10 hex characters (5 bytes of SHA256). This matches Kustomize's
  suffix length and provides 2^40 (~1 trillion) possible values — collision
  probability is negligible for any realistic number of config resources per
  module.
- **Separator**: Newline between key=value pairs, preventing ambiguity when
  values contain `=` characters.

### Transformer Changes

Both `SecretTransformer` and `ConfigMapTransformer` are updated to use
`#SecretImmutableName` / `#ImmutableName` for resource naming and to set
`spec.immutable` when applicable.

#### SecretTransformer (updated)

```cue
#transform: {
    #component: _
    #context:   core.#TransformerContext

    _secrets: #component.spec.secrets

    output: {
        for _entryName, _secret in _secrets {
            // Skip entirely if all entries are source: "k8s" (pre-existing)
            // Each entry dispatched independently by variant

            let _outputName = (schemas.#SecretImmutableName & {
                #baseName:  _entryName
                #data:      _secret.data
                #immutable: *_secret.immutable | false
            }).out

            // #SecretLiteral -> K8s Secret
            // #SecretRef (esc) -> ExternalSecret CR
            // #SecretRef (k8s) -> skip
            // See RFC-0005 for full variant dispatch table.

            "\(_outputName)": k8scorev1.#Secret & {
                apiVersion: "v1"
                kind:       "Secret"
                metadata: {
                    name:      _outputName
                    namespace: #context.namespace
                    labels:    #context.labels
                }
                if _secret.immutable {
                    immutable: true
                }
                type: _secret.type
                // Extract values from #SecretLiteral entries for K8s Secret data
                data: {
                    for _dataKey, _entry in _secret.data {
                        "\(_dataKey)": _entry.value
                    }
                }
            }
        }
    }
}
```

> **Note**: The transformer above shows the `#SecretLiteral` path for clarity.
> The full variant dispatch (ExternalSecret for esc, skip for k8s)
> is specified in
> [RFC-0005](0005-env-config-wiring.md). The `#SecretImmutableName` hash
> computation applies to all variants — for `#SecretRef`, it hashes the
> reference metadata rather than the secret value.

#### ConfigMapTransformer (updated)

```cue
#transform: {
    #component: _
    #context:   core.#TransformerContext

    _configMaps: #component.spec.configMaps

    output: {
        for _entryName, _cm in _configMaps {
            let _outputName = (schemas.#ImmutableName & {
                #baseName:  _entryName
                #data:      _cm.data
                #immutable: *_cm.immutable | false
            }).out

            "\(_outputName)": k8scorev1.#ConfigMap & {
                apiVersion: "v1"
                kind:       "ConfigMap"
                metadata: {
                    name:      _outputName
                    namespace: #context.namespace
                    labels:    #context.labels
                }
                if _cm.immutable {
                    immutable: true
                }
                data: _cm.data
            }
        }
    }
}
```

When `immutable` is `false` (the default), `#ImmutableName` /
`#SecretImmutableName` returns the base name unchanged and no `immutable`
field is set on the K8s resource. The output is identical to today's
behavior — fully backward compatible.

### Cross-Transformer Reference Pattern

The critical design question: how does a workload transformer (e.g.,
DeploymentTransformer) know the hashed name of an immutable Secret when
generating `envFrom` or `volumeMount` references?

**Answer: Duplicate hash computation.** Both the config transformer and the
workload transformer independently compute the same hash from the same
`#component.spec` data. Since CUE evaluation is deterministic and both
transformers receive the same `#component` value, they produce identical
output.

```text
┌──────────────────────────────────────────────────────────────────────────┐
│  DUPLICATE HASH COMPUTATION                                              │
│                                                                          │
│  #component.spec.secrets: {                                              │
│      "db-creds": {                                                       │
│          immutable: true                                                 │
│          data: { password: { $opm: "secret",                            │
│            $secretName: "db-creds", $dataKey: "password",               │
│            value: "abc" } }                                             │
│      }                                                                   │
│  }                                                                       │
│       │                                              │                   │
│       │  SecretTransformer reads                     │  WorkloadTransf.  │
│       │  #component.spec.secrets                     │  reads same data  │
│       ▼                                              ▼                   │
│  ┌──────────────────────┐              ┌──────────────────────────────┐  │
│  │ #SecretImmutableName │              │ #SecretImmutableName         │  │
│  │   #baseName: "db-c…" │              │   #baseName: "db-creds"     │  │
│  │   #data: {password:  │              │   #data: {password:          │  │
│  │     {value: "abc"}}  │              │     {value: "abc"}}          │  │
│  │   #immutable: true   │              │   #immutable: true           │  │
│  │                       │              │                             │  │
│  │   out: "db-creds-…"  │              │   out: "db-creds-…"         │  │
│  └───────────┬──────────┘              └──────────────┬───────────────┘  │
│              │                                        │                  │
│              ▼                                        ▼                  │
│  Secret/db-creds-a3f8b2c1e9             envFrom:                        │
│    immutable: true                        secretRef:                     │
│    data: {password: "abc"}                  name: db-creds-a3f8b2c1e9   │
│                                                                          │
│  SAME HASH — guaranteed by CUE determinism.                              │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

This pattern requires no pipeline architecture changes, no multi-pass
rendering, and no shared mutable state between transformers. It relies on a
property CUE already guarantees: pure functions over the same input produce the
same output.

**Note**: The workload transformer side of this pattern is deferred to the env
valueFrom implementation (K8s Coverage Gap Analysis, item 1.1). This RFC only
defines the schema, hash computation, and config transformer behavior. The
workload transformer integration is designed here but implemented separately.

### Garbage Collection

Immutable resources rely on the Release Inventory (RFC-0001) for lifecycle
management. When data changes, the content hash changes, which changes the
resource name. From the inventory's perspective, this is identical to a resource
rename — the core scenario that motivated RFC-0001.

```text
┌──────────────────────────────────────────────────────────────────────────┐
│  GARBAGE COLLECTION VIA INVENTORY                                        │
│                                                                          │
│  Apply v1:                                                               │
│    Rendered: Secret/db-creds-a3f8b2c1e9                                  │
│    Inventory: [Secret/db-creds-a3f8b2c1e9, Deployment/app, ...]         │
│                                                                          │
│  Apply v2 (password changed):                                            │
│    Rendered: Secret/db-creds-7c2d91f0b4                                  │
│    Inventory: [Secret/db-creds-7c2d91f0b4, Deployment/app, ...]         │
│                                                                          │
│    stale = v1_inventory - v2_inventory                                   │
│          = {Secret/db-creds-a3f8b2c1e9}                                  │
│                                                                          │
│    Apply new resources → Prune stale → Write inventory                   │
│                                                                          │
│  Result: Old Secret cleaned up automatically.                            │
│  No special immutable-specific GC logic required.                        │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

**Dependency**: This RFC ships alongside RFC-0001. Without inventory-based
pruning, old immutable resources would accumulate as orphans. While label-based
discovery can detect them, only inventory pruning can automatically clean them
up.

## Interaction with Other RFCs

### RFC-0002: Sensitive Data Model

[RFC-0002](0002-sensitive-data-model.md) introduces `#Secret` as a type in
`#config` that tags fields as sensitive. This RFC adds `immutable` as a field
on `#SecretSchema` that controls K8s resource lifecycle. The `#SecretSchema.data`
field now holds `#Secret` entries (per RFC-0005), and the hash computation is
variant-aware:

```text
┌──────────────────────────────────────────────────────────────────┐
│  RFC-0002 (Sensitive Data)         Immutable Config (this RFC)   │
│  ═════════════════════════         ═══════════════════════════   │
│  Layer: Input (how values enter)   Layer: Output (K8s behavior)  │
│  Scope: #config fields             Scope: #SecretSchema entries  │
│  Controls: redaction, encryption,  Controls: hash suffix,        │
│    dispatch to ESC                   K8s immutable flag,         │
│                                      rolling updates, GC         │
└──────────────────────────────────────────────────────────────────┘
```

They compose as follows:

| `#Secret` Variant   | Immutable Applicable? | Hash Input               | Notes                              |
|----------------------|-----------------------|--------------------------|------------------------------------|
| `#SecretLiteral`     | Yes                   | data values              | Standard case — hash the literal   |
| `#SecretRef` (esc)   | Yes                   | source + path + remoteKey | Hash the reference, not the runtime value. OPM doesn't know the actual secret. ESC handles rotation. |
| `#SecretRef` (k8s)   | No                    | N/A                      | OPM doesn't own the resource. Immutable is inapplicable. |

The hash input varies by source type because OPM's visibility into the data
varies. For `#SecretLiteral`, OPM has the actual values. For `#SecretRef`, OPM
only has the reference metadata. The hash captures what OPM knows — changes to
the reference trigger recreation, while runtime rotation by the external store
does not.

### Release Inventory (RFC-0001)

Required dependency. Immutable config is a special case of the rename problem
that motivated RFC-0001. Both should ship together as a cohesive feature:

- Inventory provides the GC mechanism for old hash-suffixed resources.
- Immutable config is a major use case that validates the inventory design.
- The inventory's stale set computation handles immutable resources with zero
  additional logic.

### K8s Coverage Gap Analysis (env valueFrom)

The env valueFrom gap (item 1.1) is the key enabler for the full rolling update
story. This RFC is designed to work with future env wiring but does not require
it:

```text
┌──────────────────────────────────────────────────────────────────────────┐
│  PHASED IMPLEMENTATION                                                   │
│                                                                          │
│  Phase 1 (this RFC + RFC-0001):                                          │
│    Schema: immutable field on #SecretSchema / #ConfigMapSchema           │
│    Transformer: hash suffix + K8s immutable flag                         │
│    Inventory: GC of old resources                                        │
│    Result: Immutable resources created and pruned correctly              │
│            Workloads don't auto-reference them yet                       │
│            `opm mod build` output shows correctly named resources        │
│                                                                          │
│  Phase 2 (env valueFrom implementation — RFC-0005):                      │
│    Schema: env.valueFrom, envFrom on #ContainerSchema                    │
│    Transformer: workload transformers use #ImmutableName to resolve      │
│                 secret/configmap references in env vars and volumes      │
│    Result: Config change → new hash → new name in pod template           │
│            → rolling update triggered automatically                      │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### Experiment 001-config-sources

This RFC supersedes the `001-config-sources` experiment. The experiment
validated the need for sensitivity tagging, env wiring, and transformer
dispatch. This RFC takes a different approach (per-entry `immutable` field
vs. unified `#ConfigSourceSchema`), but the experiment's learnings informed the
design — particularly around naming conventions and transformer output
structure.

## Scenarios

### Scenario A: Basic Immutable Secret

```text
Module:
  #config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
  values:  db: password: { value: "abc" }

  secrets: {
      "db-creds": {
          immutable: true
          data: { password: #config.db.password }
      }
  }

Apply v1:
  #SecretContentHash normalizes: password="abc"
  Rendered: Secret/db-creds-a3f8b2c1e9 (immutable: true, data: {password: base64("abc")})
  Applied OK → inventory written

Apply v2 (data changed):
  values: db: password: { value: "xyz" }
  #SecretContentHash normalizes: password="xyz"

  Rendered: Secret/db-creds-7c2d91f0b4 (new hash!)
  previous_inventory has Secret/db-creds-a3f8b2c1e9
  current_inventory has Secret/db-creds-7c2d91f0b4
  stale = {Secret/db-creds-a3f8b2c1e9}

  Apply Secret/db-creds-7c2d91f0b4 OK → Prune Secret/db-creds-a3f8b2c1e9 → write inventory
  Result: Clean. [x]
```

### Scenario B: Mixed Mutable and Immutable

```text
Module:
  #config: {
      db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
      feature: beta: #Secret & { $secretName: "feature-flags", $dataKey: "enable_beta" }
  }
  values: {
      db: password: { value: "abc" }
      feature: beta: { value: "true" }
  }

  secrets: {
    "db-creds":      { immutable: true, data: { password: #config.db.password } }
    "feature-flags": { data: { enable_beta: #config.feature.beta } }  // mutable (default)
  }

Apply:
  Rendered: Secret/db-creds-a3f8b2c1e9 (immutable: true)
  Rendered: Secret/feature-flags        (no suffix, no immutable flag)

  Both coexist. Only db-creds gets the hash treatment.
  Result: Clean. [x]
```

### Scenario C: Data Unchanged — Idempotent

```text
Apply v1:
  #config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
  values:  db: password: { value: "abc" }
  secrets: { "db-creds": { immutable: true, data: { pass: #config.db.password } } }
  Rendered: Secret/db-creds-a3f8b2c1e9

Apply v2 (same data):
  Same data → same hash → same name → SSA detects no change → unchanged
  No new resource created. No pruning needed.
  Result: Clean. [x]
```

### Scenario D: Transition from Mutable to Immutable

```text
Apply v1 (mutable):
  #config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
  values:  db: password: { value: "abc" }
  secrets: { "db-creds": { data: { pass: #config.db.password } } }
  Rendered: Secret/db-creds
  Inventory: [Secret/db-creds]

Apply v2 (now immutable):
  secrets: { "db-creds": { immutable: true, data: { pass: #config.db.password } } }
  Rendered: Secret/db-creds-a3f8b2c1e9
  stale = {Secret/db-creds}

  Apply Secret/db-creds-a3f8b2c1e9 OK → Prune Secret/db-creds → write inventory
  Result: Clean transition. [x]
```

### Scenario E: Transition from Immutable to Mutable

```text
Apply v1 (immutable):
  #config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
  values:  db: password: { value: "abc" }
  secrets: { "db-creds": { immutable: true, data: { pass: #config.db.password } } }
  Rendered: Secret/db-creds-a3f8b2c1e9
  Inventory: [Secret/db-creds-a3f8b2c1e9]

Apply v2 (now mutable):
  secrets: { "db-creds": { data: { pass: #config.db.password } } }
  Rendered: Secret/db-creds
  stale = {Secret/db-creds-a3f8b2c1e9}

  Apply Secret/db-creds OK → Prune Secret/db-creds-a3f8b2c1e9 → write inventory
  Result: Clean transition. [x]
```

### Scenario F: Rollback to Previous Data

```text
  #config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }

Apply v1: values: db: password: { value: "abc" }
          → Secret/db-creds-a3f8b2c1e9
Apply v2: values: db: password: { value: "xyz" }
          → Secret/db-creds-7c2d91f0b4  (hash1 pruned)
Apply v3: values: db: password: { value: "abc" }
          → Secret/db-creds-a3f8b2c1e9  (hash2 pruned)

The v1 resource was pruned in v2. v3 recreates it fresh.
Same hash → same name → new resource (no conflict, old one was deleted).
Result: Clean. [x]
```

### Scenario G: Partial Failure

```text
Apply v1: Secret/db-creds-a3f8b2c1e9 in inventory

Apply v2 (data changed):
  Rendered: Secret/db-creds-7c2d91f0b4, Deployment/app
  Apply Secret/db-creds-7c2d91f0b4 OK
  Apply Deployment/app FAIL

  Failure → no prune, no inventory write (RFC-0001 rule)
  Secret/db-creds-7c2d91f0b4 is a ghost resource
  Secret/db-creds-a3f8b2c1e9 still exists (not pruned)

  User fixes and retries:
  previous_inventory still = v1 (not updated)
  stale still = {Secret/db-creds-a3f8b2c1e9}
  Apply all OK → prune old → write inventory
  Result: Clean on retry. [x]
```

### Scenario H: First Apply with Immutable — No Inventory

```text
No previous inventory Secret exists.
previous_inventory = empty set
stale = empty set

Rendered: Secret/db-creds-a3f8b2c1e9
Applied OK → nothing to prune → inventory Secret created
Result: Clean. [x]
```

### Scenario I: Multiple Immutable Resources in One Component

```text
#config: {
    db: {
        password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
    }
    api: {
        key: #Secret & { $secretName: "api-key", $dataKey: "key" }
    }
}
values: {
    db: password: { value: "abc" }
    api: key: { value: "sk_live_xyz" }
}

secrets: {
    "db-creds": { immutable: true, data: { password: #config.db.password } }
    "api-key":  { immutable: true, data: { key: #config.api.key } }
}
configMaps: {
    "app-settings": { immutable: true, data: { level: "info" } }
}

Each gets its own independent hash:
  Secret/db-creds-a3f8b2c1e9
  Secret/api-key-e7d1b4f8a2
  ConfigMap/app-settings-c5f9a2d7b1

Changing one does not affect the others.
Result: Clean. [x]
```

## Open Questions

### Q1: Hash Stability Across CUE Versions

**Question**: Is the SHA256 output from CUE's `crypto/sha256` guaranteed to be
identical across CUE SDK versions?

**Assessment**: SHA256 is a standardized algorithm (FIPS 180-4). CUE delegates
to Go's `crypto/sha256` package. The output will not change across versions.
The risk is in the *input* string construction, not the hash function. The
design mitigates this by using explicit key sorting (`list.SortStrings`) and a
simple `key=value\n` concatenation format with no CUE-version-dependent
behavior.

**Risk**: Low.

### Q2: Binary Data in Secrets

**Question**: `#SecretSchema` data now holds `#Secret` entries. If `binaryData`
support is needed (for certificates, etc.), it could be added as an additional
field on `#SecretLiteral` (e.g., `binaryValue?: bytes`) or as a separate
`#SecretBinaryLiteral` variant.

**Mitigation**: The `#SecretContentHash` normalizes `#Secret` entries to strings
for hashing. A `binaryValue` field would be included via its base64 encoding.
The hash interface (`#SecretContentHash`) would need a normalization rule for
the new variant, but `#ContentHash` itself remains unchanged.

### Q3: Maximum Resource Name Length

**Question**: Kubernetes resource names are limited to 253 characters (DNS
subdomain). Adding a 10-character hash suffix plus a `-` separator adds 11
characters. Could this exceed the limit?

**Assessment**: OPM resource names are constrained by `#NameType` to 63
characters. The component name is also constrained to 63 characters. The
transformer combines them as `<entry-name>-<hash>` (max 63 + 1 + 10 = 74
characters) or as `<component-name>-<entry-name>-<hash>` depending on the
naming convention. All are well within the 253-character limit and the
63-character label value limit.

**Risk**: Negligible.

## Deferred Work

### Workload Env Wiring (env valueFrom / envFrom)

The workload transformer side of the cross-transformer reference pattern.
When implemented, workload transformers will use `#ImmutableName` /
`#SecretImmutableName` to resolve Secret and ConfigMap references in `envFrom`,
`env.valueFrom.secretKeyRef`, `env.valueFrom.configMapKeyRef`, and
`volumes[].secret`/`volumes[].configMap`.
See [RFC-0005](0005-env-config-wiring.md) for the full env wiring design.

### ExternalSecret Immutability

For `#SecretRef` sources that produce ExternalSecret CRs (esc), the
`immutable` field could be propagated to the ExternalSecret's `target` spec.
This is a provider-level concern deferred to the ESC handler implementation.

## References

- [Timoni Immutable ConfigMaps and Secrets](https://timoni.sh/cue/module/immutable-config/) — Timoni's CUE-based immutable config pattern
- [Kustomize configMapGenerator](https://kubectl.docs.kubernetes.io/references/kustomize/kustomization/configmapgenerator/) — Kustomize's hash-suffixed config generation
- [KEP-1412: Immutable Secrets and ConfigMaps](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1412-immutable-secrets-and-configmaps) — Kubernetes native immutability
- [Helm: Automatically Roll Deployments](https://helm.sh/docs/howto/charts_tips_and_tricks/#automatically-roll-deployments) — Helm's checksum annotation pattern
- [RFC-0001: Release Inventory](0001-release-inventory.md) — OPM's inventory-based pruning system
- [RFC-0002: Sensitive Data Model](0002-sensitive-data-model.md) — First-class #Secret type for OPM
- [RFC-0005: Environment & Config Wiring](0005-env-config-wiring.md) — Env var wiring, envFrom, volume mounts
