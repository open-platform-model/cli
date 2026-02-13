# RFC-0002: Sensitive Data Model

| Field       | Value              |
|-------------|--------------------|
| **Status**  | Draft              |
| **Created** | 2026-02-09         |
| **Authors** | OPM Contributors   |

## Summary

Introduce a `#Secret` type that makes sensitive data a first-class concept in
OPM. Today, all values flow through `#config` → `values` → transformer
identically — `db.host` and `db.password` are both plain strings. Passwords end
up as plaintext in CUE files, git repositories, and rendered manifests.

`#Secret` tags a field as sensitive at the schema level. This single annotation
propagates through every layer — module definition, release fulfillment,
transformer output — enabling the toolchain to redact, encrypt, and dispatch
secrets to platform-appropriate resources (K8s Secrets, ExternalSecrets, CSI
volumes) without the module author managing any of that machinery.

The design supports two input paths (literal values, external references) plus
CLI `@` tag injection, and two output targets (environment variables and volume
mounts), while remaining backward compatible with existing modules.

## Motivation

### The Problem

OPM has no concept of "sensitive." Every value that passes through `#config` →
`values` → transformer is a plain string:

```text
┌─────────────────────────────────────────────────────────────────┐
│  Module #config                                                 │
│                                                                 │
│  db: {                                                          │
│      host:     string     <- not sensitive                      │
│      password: string     <- sensitive, but OPM can't tell      │
│  }                                                              │
│                                                                 │
│  Both fields flow identically:                                  │
│    CUE file -> git -> rendered YAML -> kubectl apply            │
│    Password is plaintext at every stage.                        │
└─────────────────────────────────────────────────────────────────┘
```

This creates four concrete problems:

1. **No redaction** — `cue export` prints passwords alongside hostnames. Logs,
   CI output, and debugging sessions expose secrets.
2. **No encryption** — Stored CUE artifacts contain plaintext secrets.
3. **No external references** — No way to say "this value lives in Vault" or
   "use the existing K8s Secret called `db-creds`."
4. **No platform integration** — Transformers emit the same resource structure
   for `host` and `password`. No dispatch to ExternalSecrets Operator, CSI
   drivers, or other secret management infrastructure.

### The Opportunity

If OPM knows which fields are sensitive, the entire toolchain can act on it:

- **Authors** mark fields as `#Secret` and wire them to containers — done.
- **Users** choose how to provide secrets (literal, external ref, `@` tag).
- **Tooling** redacts secrets in output, encrypts in storage, validates
  fulfillment before deploy.
- **Transformers** dispatch to the correct platform mechanism based on how the
  secret was provided.

### Why Now

The [RFC-0004: Interface Architecture](0004-interface-architecture.md) introduces
`provides`/`requires` with typed shapes. Those shapes include fields like
`#Postgres.password` — currently typed as `string`. Without a sensitive data
model, the Interface system has a blind spot: it can type-check that a password
field exists, but cannot ensure it is handled securely.

## Prior Art

The `config-sources` experiment (`experiments/001-config-sources/`) prototyped
an earlier version of this design. This section documents what was validated and
what changed.

### What the Experiment Validated

| Experiment Feature | Finding |
|---|---|
| `#ConfigSourceSchema` with `type: "config" \| "secret"` discriminator | Sensitivity tagging is needed |
| `env.from: { source, key }` wiring | Env vars need reference syntax beyond plain `value:` |
| Transformer dispatch (ConfigMap vs Secret based on type) | Output must differ based on sensitivity |
| External refs (`externalRef.name`) emitting nothing | The "pre-existing resource" pattern works |
| K8s resource naming `{component}-{source}` | Predictable naming is essential |

### What This RFC Changes

```text
┌────────────────────────────────────┬────────────────────────────────────────┐
│ Experiment Approach                │ This RFC                               │
├────────────────────────────────────┼────────────────────────────────────────┤
│ configSources as a separate        │ #Secret as a type in #config.          │
│ component resource.                │ Secrets belong at the schema level.    │
│                                    │                                        │
│ env.from: { source: "app-settings" │ env.from: #config.db.password           │
│            key: "LOG_LEVEL" }      │ Direct CUE refs. Secrets only.         │
│                                    │                                        │
│ data + externalRef mutual          │ #SecretLiteral | #SecretRef union.     │
│ exclusivity.                       │ Cleaner CUE.                           │
│                                    │                                        │
│ Config and secrets unified in      │ Config is string, secrets are #Secret. │
│ configSources.                     │ value: for config, from: for secrets.  │
└────────────────────────────────────┴────────────────────────────────────────┘
```

### What Carries Forward

- Transformer dispatch pattern (literal -> K8s Secret, external -> nothing)
- Consistent env var output (`valueFrom.secretKeyRef` for all secret variants)
- Provider handler interface concept
- Naming convention for generated K8s resources

## Design

### The #Secret Type

`#Secret` is a struct-only union type with two variants. Every `#Secret` value
is always a struct — plain `string` is not accepted. This ensures the
transformer can always distinguish sensitive fields by shape, and CUE error
messages remain clear (no `string | struct` disjunction ambiguity):

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
    source!:      *"k8s" | "k8s-eso"
    path!:        string       // K8s Secret name (k8s) or external path (eso)
    remoteKey!:   string       // key within the referenced secret
}
```

**Variant 1: Literal.** The user provides the actual value. Backward compatible
with how OPM works today. Flows through the system into a K8s Secret resource.

```cue
#SecretLiteral: {
    $opm:         "secret"
    $secretName!: #NameType
    $dataKey!:    string
    description?: string
    value!:       string
}
```

**Variant 2: Reference.** Points to an external secret source or pre-existing
K8s Secret. The value never enters OPM — the platform resolves it at deploy
time.

```cue
#SecretRef: {
    $opm:         "secret"
    $secretName!: #NameType
    $dataKey!:    string
    description?: string
    source!:      *"k8s" | "k8s-eso"
    path!:        string       // K8s Secret name (k8s) or external path (eso)
    remoteKey!:   string       // key within the referenced secret
}
```

Design rationale:

- **`$opm: "secret"` discriminator.** A concrete value present on every
  `#Secret` variant. Enables CUE-native auto-discovery via the negation test
  (see [RFC-0005](0005-env-config-wiring.md)). No tags or external tooling
  needed.

- **`$secretName` replaces the old `owner` concept.** Clearer: it IS the K8s
  Secret resource name. The `$` prefix distinguishes author-set routing fields
  from user-set fulfillment fields. Multiple `#config` fields sharing the same
  `$secretName` are grouped into one K8s Secret with multiple data keys.

- **`$dataKey` replaces the old `key` concept.** Avoids ambiguity. The old
  `key` field was overloaded (both the data key AND the external lookup key on
  references). Now `$dataKey` is always the output data key; `remoteKey` is the
  external lookup key.

- **`remoteKey` on `#SecretRef`.** Separate from `$dataKey`. The external
  secret's key (in Vault/ESO/another K8s Secret) may differ from the logical
  data key that the module uses.

- **`source: *"k8s" | "k8s-eso"`.** Simplified from individual provider
  enumeration. ESO (External Secrets Operator) abstracts over external
  providers (Vault, AWS SM, GCP SM, etc.), so `"k8s-eso"` covers all external
  sources.

- **No `#SecretDeferred`.** Unfulfilled `#Secret` fields are CUE incompleteness
  errors, caught at evaluation time. If a user omits a required secret value,
  CUE itself reports the error — no special deferred variant needed.

- **No `#SecretBase`.** Each variant carries its own fields inline. With only
  two variants and the `$opm` discriminator on both, a shared base adds no
  value.

- **`$`-prefixed fields.** Regular CUE fields (visible in iteration), not
  hidden. The `$` prefix is a naming convention to visually distinguish
  author-set routing fields from user-set fulfillment fields.

- **Users never set `$secretName`/`$dataKey`.** CUE unification propagates the
  author's values through. Users only provide `value` (for literals) or
  `source`/`path`/`remoteKey` (for refs).

Both `$secretName` and `$dataKey` are set by the module author in the `#config`
schema. See [RFC-0005](0005-env-config-wiring.md) for the three-layer model
that connects `#config` declarations to `spec.secrets` resource creation and
container wiring.

The critical property is not which variant is used — it is that **the field is
typed as `#Secret` at all**:

```text
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│  log_level: string      -> non-sensitive                     │
│                            Appears in logs, exports, output  │
│                            Emitted as ConfigMap or plain env │
│                                                              │
│  db_password: #Secret   -> sensitive                         │
│                            Redacted in logs and exports      │
│                            Emitted as K8s Secret             │
│                            Encrypted in stored artifacts     │
│                            Platform may resolve externally   │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Value Constraints

Because `#Secret` is a CUE definition, authors can constrain the `value` field
using standard CUE expressions. No OPM-specific mechanism needed — this is CUE
unification.

Constraints fire when the resolved `#Secret` is a `#SecretLiteral` (written
directly or injected via `@` tag). For `#SecretRef`, the `value` field is
absent, so constraints are inert — they serve as machine-readable documentation.

```cue
// Minimum length
#config: {
    db: password: #Secret & {
        $secretName: "db-creds"
        $dataKey:    "password"
        value?: string & strings.MinRunes(12)
    }
}

// Prefix match
#config: {
    stripe_key: #Secret & {
        $secretName: "stripe-key"
        $dataKey:    "token"
        value?: string & =~"^sk_(test|live)_"
    }
}
```

Authors can define reusable constraint patterns:

```cue
#PEMSecret: #Secret & {
    value?: string & =~"^-----BEGIN [A-Z ]+-----\n" & =~"\n-----END [A-Z ]+-----\n?$"
}

#StrongPasswordSecret: #Secret & {
    value?: string & =~"^.{16,}$" & =~".*[A-Z].*" & =~".*[a-z].*" & =~".*[0-9].*"
}

#APIKeySecret: {
    _prefix: string
    #Secret & { value?: string & =~"^\(_prefix)" }
}

#config: {
    tls_key:      #PEMSecret
    admin_pass:   #StrongPasswordSecret
    stripe_key:   #APIKeySecret & { _prefix: "sk_(test|live)_" }
}
```

#### When Do Constraints Apply?

```text
┌───────────────────────────────────────┬──────────┬────────────────────────────┐
│ Input Path                            │ Checked? │ Reason                     │
├───────────────────────────────────────┼──────────┼────────────────────────────┤
│ #SecretLiteral { value: "..." }       │  [x]     │ value set, CUE evaluates   │
│ @ tag (CLI resolves to literal)       │  [x]     │ CLI injects { value: ... } │
│ #SecretRef                            │  [ ]     │ value not set, inert       │
└───────────────────────────────────────┴──────────┴────────────────────────────┘
```

For `#SecretRef`, constraints remain in the schema as machine-readable
declarations. While the CLI cannot validate these at eval time (the value is
not yet resolved), a future OPM controller that fetches secrets at
reconciliation time can evaluate constraints post-fetch and surface violations
as status conditions. See [Deferred Work](#deferred-work).

### Input Paths

Two input paths for providing secret values, plus CLI `@` tag injection. All
three can coexist within a single module release.

```text
┌────────────────────────────────────────────────────────────────────────┐
│                     HOW SECRETS ENTER OPM                              │
│                                                                        │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────┐  │
│  │  Path 1: Literal │  │  Path 2: Ref     │  │  Path 3: @ Tag       │  │
│  │                  │  │                  │  │                      │  │
│  │  { value: "..." }│  │  { source: ".."  │  │  @secret(eso,...)    │  │
│  │                  │  │    path: "..."   │  │                      │  │
│  │  User provides   │  │    remoteKey:    │  │  CLI resolves to     │  │
│  │  the value.      │  │    "..." }       │  │  { value: "..." }    │  │
│  │                  │  │                  │  │  before CUE eval.    │  │
│  │                  │  │  Platform        │  │                      │  │
│  │                  │  │  resolves at     │  │                      │  │
│  │                  │  │  deploy time.    │  │                      │  │
│  └────────┬─────────┘  └────────┬─────────┘  └──────────┬───────────┘  │
│           └─────────────────────┼───────────────────────┘              │
│                                 ▼                                      │
│                     ┌───────────────────┐                              │
│                     │  #Secret field    │                              │
│                     │  in #config       │                              │
│                     └───────────────────┘                              │
└────────────────────────────────────────────────────────────────────────┘
```

**Path 1: Literal.** User writes the value directly in `values`:

```cue
// Module definition
#config: db: password: #Secret

// Module release
values: db: password: { value: "my-secret-password" }
```

**Path 2: External Reference.** User points to a secret store:

```cue
values: db: password: {
    source:    "k8s-eso"
    path:      "secret/data/prod/db"
    remoteKey: "password"
}
```

**Path 3: `@` Tag Injection.** CUE attributes resolved by the CLI before
evaluation:

```cue
values: db: password: _ @secret(source=eso, path="secret/data/prod/db", key=password)
```

The CLI transforms this to `{ value: "the-fetched-value" }` before CUE eval.
The `@secret(...)` tag is CLI sugar — not part of the OPM schema. Module
authors do not need to know about it. Future tags (e.g., `@env("DB_PASSWORD")`,
`@file("/run/secrets/db")`) can be added without schema changes.

### Output Dispatch

The transformer inspects the resolved `#Secret` variant and produces different
K8s resources:

```text
┌───────────────────────────────────────┬──────────────────────────┬──────────────────────────────────┐
│ Input Variant                         │ K8s Resource Emitted     │ Env Var Wiring                   │
├───────────────────────────────────────┼──────────────────────────┼──────────────────────────────────┤
│ #SecretLiteral { value: "..." }       │ Secret (base64 data)     │ secretKeyRef -> $secretName      │
│ #SecretRef source: "k8s"              │ Nothing (already exists) │ secretKeyRef -> path             │
│ #SecretRef source: "k8s-eso"          │ ExternalSecret CR        │ secretKeyRef -> $secretName      │
└───────────────────────────────────────┴──────────────────────────┴──────────────────────────────────┘
```

**Resource naming.** The K8s Secret name is the `$secretName` field. The module
author sets `$secretName` in the `#config` schema. For `source: "k8s"`, the
`path` field IS the K8s Secret name — `$secretName` is irrelevant because OPM
does not manage the resource.

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
-> K8s Secret name: "db-creds"
-> K8s Secret key:  "password"
```

**Consistent env var output.** Regardless of variant, the container wiring is
always `valueFrom.secretKeyRef`. Only the `name` and `key` change:

```text
┌─────────────────────┐      ┌──────────────────────────────────┐
│  #SecretLiteral     │---->>│  name: "db-creds"                │  ($secretName)
│  #SecretRef (eso)   │---->>│  name: "db-creds"                │  ($secretName, ESO target)
│  #SecretRef (k8s)   │---->>│  name: "existing-db-secret"      │  (path)
└─────────────────────┘      └──────────────────────────────────┘
                                        │
                                        ▼
                             env:
                               - name: DB_PASSWORD
                                 valueFrom:
                                   secretKeyRef:
                                     name: <$secretName or path>
                                     key: <$dataKey or remoteKey>
```

### Wiring Model

Developers wire config and secrets to container env vars using two fields:
`value` for non-sensitive data (plain strings) and `from` for sensitive data
(`#Secret` references). The `from` field accepts only `#Secret` — non-sensitive
config always uses `value`:

```cue
#config: {
    log_level: string
    db: {
        host:     string
        password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
    }
}

spec: container: env: {
    LOG_LEVEL:   { value: #config.log_level }
    DB_HOST:     { value: #config.db.host }
    DB_PASSWORD: { from:  #config.db.password }
}
```

The two fields have distinct semantics:

```text
value: #config.log_level
  -> resolves to string "info"
  -> emit: { value: "info" }

from: #config.db.password
  -> resolves to #Secret
  -> emit: { valueFrom: { secretKeyRef: { ... } } }
```

`value` handles non-sensitive config via direct CUE references that resolve to
strings. `from` handles secrets via direct CUE references that resolve to
`#Secret`. Both are type-safe (CUE validates at definition time) and
self-documenting.

**EnvVar schema:**

```cue
#EnvVarSchema: {
    name!:  string
    value?: string    // inline literal (non-sensitive config)
    from?:  #Secret   // reference to a #Secret in #config
}
```

`value` and `from` are mutually exclusive. An env var is either a non-sensitive
literal (`value`) or a reference to a `#Secret` field (`from`). If both are
set, CUE evaluation errors. If neither is set, the env var declaration is
incomplete. See [RFC-0005](0005-env-config-wiring.md) for the full
`#EnvVarSchema` including `fieldRef` and `resourceFieldRef`.

**Why two fields, not one.** A single `value: string | #Secret` field was
considered. It eliminates the mutual-exclusivity question, but creates a
`value: { value: "..." }` stutter when a `#SecretLiteral` is resolved — the
outer `value` (env var field) wraps the inner `value` (`#SecretLiteral.value`).
Two fields give each case its natural keyword: `value` for non-sensitive
literals, `from` for secret references. The separation also makes transformer
logic straightforward — `value` is always a plain string, `from` is always a
`#Secret`.

### Provider Handlers

The K8s provider registers handlers for each secret source type. A handler
takes a `#SecretRef` and produces K8s resources plus a `secretKeyRef`:

```cue
#SecretSourceHandler: {
    #resolve: {
        #ref:       #SecretRef
        #component: _
        #context:   #TransformerContext

        resources:    [string]: {...}     // K8s resources to emit (may be empty)
        secretKeyRef: { name!: string, key!: string }
    }
}
```

**Registered handlers:**

```cue
#SecretSourceHandlers: {
    "k8s": #K8sSecretHandler
    "k8s-eso": #ExternalSecretHandler
}
```

Since providers are simplified to `"k8s"` and `"k8s-eso"`, the handler map is
minimal. ESO (External Secrets Operator) abstracts over external providers
(Vault, AWS SM, GCP SM, etc.), so backend `ClusterSecretStore` configuration
is a platform-level concern — deployed and configured outside OPM module scope.

**ExternalSecret handler (ESO)** — emits an `ExternalSecret` CR:

```cue
#ExternalSecretHandler: #SecretSourceHandler & {
    #resolve: {
        let _name = #ref.$secretName

        resources: "\(_name)": {
            apiVersion: "external-secrets.io/v1beta1"
            kind:       "ExternalSecret"
            metadata: name: _name
            spec: {
                refreshInterval: "1h"
                secretStoreRef: { kind: "ClusterSecretStore" }
                target: name: _name
                data: [{ secretKey: #ref.$dataKey, remoteRef: { key: #ref.path, property: #ref.remoteKey } }]
            }
        }
        secretKeyRef: { name: _name, key: #ref.$dataKey }
    }
}
```

The `ClusterSecretStore` name is resolved at the platform level. Module authors
and users do not specify it — it is an infrastructure concern.

### Volume-Mounted Secrets

Not all secrets are environment variables. TLS certificates, service account
keys, and credential files are mounted as volumes. The same `#Secret` type
handles both — what differs is the wiring target.

```cue
// Env var wiring
env: DB_PASSWORD: { name: "DB_PASSWORD", from: #config.db.password }

// Volume mount wiring
volumeMounts: "tls-cert": { mountPath: "/etc/tls", from: #config.tls }
```

When `from` resolves to a `#Secret` in a volume mount context, the transformer
emits a K8s Secret (or ExternalSecret), a `volume` entry referencing it, and a
`volumeMount` on the container.

Multi-key secrets become multiple files in the mounted volume:

```cue
#config: tls: #Secret & { $secretName: "tls-cert", $dataKey: "tls.crt" }

volumeMounts: "tls": {
    mountPath: "/etc/tls"
    from:      #config.tls
    // -> /etc/tls/tls.crt
    // -> /etc/tls/tls.key
}
```

### Unified Config Pattern

Config (non-sensitive) and secrets (sensitive) use different fields but the same
referencing pattern — direct CUE references to `#config`:

```cue
#config: {
    log_level:   string                                                    // non-sensitive
    app_port:    int                                                       // non-sensitive
    db_password: #Secret & { $secretName: "db-creds", $dataKey: "password" }  // sensitive
    api_key:     #Secret & { $secretName: "api-key", $dataKey: "token" }      // sensitive
    tls:         #Secret & { $secretName: "tls-cert", $dataKey: "tls.crt" }   // sensitive (volume target)
}

spec: container: {
    env: {
        LOG_LEVEL:   { value: #config.log_level }
        APP_PORT:    { value: "\(#config.app_port)" }
        DB_PASSWORD: { from:  #config.db_password }
        API_KEY:     { from:  #config.api_key }
    }
    volumeMounts: "tls": { mountPath: "/etc/tls", from: #config.tls }
}
```

```text
value: #config.log_level   type: string   -> { value: "info" }
from:  #config.db_password type: #Secret  -> K8s Secret + secretKeyRef
from:  #config.tls         type: #Secret  -> K8s Secret + volume + volumeMount
```

Non-sensitive fields resolve to plain strings via `value:`. The transformer
emits inline `{ value: "..." }` on the env var. No ConfigMap is generated from
`value:` references — if ConfigMap-backed config is needed, use `spec.configMaps`
directly. See [RFC-0005](0005-env-config-wiring.md) for the full env wiring
model.

## Interface Integration

The [RFC-0004: Interface Architecture](0004-interface-architecture.md) defines
`provides`/`requires` with typed shapes. This RFC upgrades sensitive fields from
`string` to `#Secret`:

```cue
// Before
#PostgresInterface: #Interface & {
    #shape: {
        host!:     string
        port:      uint | *5432
        dbName!:   string
        username!: string
        password!: string         // no sensitivity distinction
    }
}

// After
#PostgresInterface: #Interface & {
    #shape: {
        host!:     string
        port:      uint | *5432
        dbName!:   string
        username!: string
        password!: #Secret        // now typed as sensitive
    }
}
```

Platform fulfillment must now provide a `#Secret` for `password`. The module
author uses `value:` for non-sensitive fields and `from:` for secrets:

```cue
requires: "db": #Postgres

spec: container: env: {
    DB_HOST:     { value: requires.db.host }                            // string -> plain
    DB_PASSWORD: { from:  requires.db.password }                        // #Secret -> secretKeyRef
}
```

The module author does not know or care whether `password` was fulfilled with a
literal, an ESO ref, or a K8s Secret ref.

## Backward Compatibility

### Migration: string to #Secret

When a field is upgraded from `string` to `#Secret`, existing literal values
must be wrapped in `#SecretLiteral`. This is a one-time mechanical change:

```cue
// Before (plain string)
values: db: password: "my-secret-password"

// After (#Secret requires struct)
values: db: password: { value: "my-secret-password" }
```

`#Secret` does not accept bare `string`. This is deliberate — it ensures that
sensitivity is always explicit and the transformer can distinguish sensitive
fields by shape without inspecting the schema type.

### Warnings

The OPM CLI emits warnings when `#SecretLiteral` is used in production
contexts:

```text
WARNING: db.password is a #Secret field with a literal value.
  Consider using a #SecretRef or @secret() tag for production.
```

This is a CLI behavior, not a CUE validation error.

### Strict Mode (Optional)

Organizations can enable strict mode via platform policy:

```cue
#NoLiteralSecrets: #PolicyRule & {
    // Validates that no #Secret field in the release is a #SecretLiteral
}
```

Opt-in. The default always permits literals.

## Scenarios

### Scenario A: Dev Environment with Literal [x]

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
values:  db: password: { value: "dev-password-123" }

Transformer emits:
  Secret/db-creds  { data: { password: base64("dev-password-123") } }
  env: DB_PASSWORD -> secretKeyRef -> db-creds / password

Result: Works like today, but value is in a K8s Secret instead of plaintext. [x]
```

### Scenario B: Production with ESO Ref [x]

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
values:  db: password: { source: "k8s-eso", path: "secret/data/prod/db", remoteKey: "password" }

Transformer emits:
  ExternalSecret/db-creds  (references ClusterSecretStore)
  env: DB_PASSWORD -> secretKeyRef -> db-creds / password

No plaintext value in CUE files or rendered manifests. [x]
```

### Scenario C: Existing K8s Secret [x]

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
values:  db: password: { source: "k8s", path: "existing-db-secret", remoteKey: "password" }

Transformer emits:
  Nothing (Secret already exists in cluster)
  env: DB_PASSWORD -> secretKeyRef -> existing-db-secret / password

Zero new resources. $secretName is ignored for source: "k8s". [x]
```

### Scenario D: Unfulfilled Secret (CUE Incompleteness Error) [x]

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
values:  db: {}   // password not provided

CUE reports incompleteness error:
  db.password.value: incomplete value string (and target type #Secret)

No explicit deferred variant needed. CUE catches the missing value naturally. [x]
```

### Scenario E: CLI @ Tag Injection [x]

```text
values: db: password: _ @secret(source=eso, path="secret/data/prod/db", key=password)

CLI resolves @secret tag:
  1. Fetches from external store: secret/data/prod/db -> key "password"
  2. Injects: db: password: { value: "the-fetched-value" }
  3. CUE evaluates with the injected literal

From module's perspective, identical to Scenario A. [x]
```

### Scenario F: Interface Fulfillment with Secret [x]

```text
Module:   requires: "db": #Postgres   (password!: #Secret)
Platform: bindings: "user-api": requires: "db": {
              host: "db.prod.internal"
              password: { source: "k8s-eso", path: "secret/data/prod/db", remoteKey: "password" }
          }

Module author wiring:
  env: DB_PASSWORD: { from: requires.db.password }

Result: Module author wrote no secret-handling code.
        Platform provided ESO ref. Transformer emits ExternalSecret. [x]
```

## Trade-offs

**Advantages:**

- Type-safe sensitivity — `#Secret` is checked by CUE at definition time.
- Developer simplicity — declare `#Secret`, wire with `from:`, done.
- User flexibility — two input paths plus `@` tag cover dev through production.
- Toolchain awareness — every tool in the pipeline can distinguish sensitive
  from non-sensitive.
- Platform portability — `#Secret` dispatches to K8s Secrets, ExternalSecrets,
  or future provider equivalents.
- Clear wiring — `value:` for config, `from:` for secrets. Same CUE reference pattern.
- CUE-native validation — unfulfilled secrets produce CUE incompleteness errors
  with no custom deferred variant needed.

**Disadvantages:**

- New core type — developers must learn `#Secret` alongside Resource, Trait,
  Blueprint, Interface.
- Provider handler implementation — `"k8s"` and `"k8s-eso"` handlers must be
  implemented and maintained.
- `@` tag requires CLI support — useless without OPM CLI implementation.
- Migration cost — existing modules must wrap literal values in
  `{ value: "..." }` when upgrading fields from `string` to `#Secret`.

**Risks:**

```text
┌──────────────────────────────────────────┬──────────┬────────────┬────────────────────────────────┐
│ Risk                                     │ Severity │ Likelihood │ Mitigation                     │
├──────────────────────────────────────────┼──────────┼────────────┼────────────────────────────────┤
│ @ tag resolution complexity (multi-      │ Medium   │ High       │ Start with @secret(k8s, ...)   │
│ provider auth)                           │          │            │ only. Add eso later.           │
│                                          │          │            │                                │
│ ExternalSecret CRD version drift         │ Low      │ Medium     │ Pin to ESO v1beta1. Abstract   │
│                                          │          │            │ behind handler interface.      │
│                                          │          │            │                                │
│ Developers ignore #Secret and use        │ Medium   │ Medium     │ Lint rules in CI. Policy       │
│ string everywhere                        │          │            │ rules for production.          │
└──────────────────────────────────────────┴──────────┴────────────┴────────────────────────────────┘
```

## Open Questions

### ~~Q1: CUE Representation of #Secret~~ (Decided)

**Decision:** Option C — always-struct. `#Secret` does not accept bare `string`.
Both variants (`#SecretLiteral`, `#SecretRef`) are structs. This eliminates the
`string | struct` disjunction problem in CUE, produces clear error messages,
and ensures the transformer can always distinguish sensitive fields by shape.

### Q2: @ Tag Syntax

CUE attributes have the form `@attr(arg1, arg2, ...)`.

Options: (A) Positional: `@secret(eso, "secret/data/db", "password")`.
(B) Named: `@secret(source=eso, path="secret/data/db", key=password)`.
(C) URI-style: `@secret("eso:secret/data/db#password")`.

**Recommendation:** Option B. Named parameters are self-documenting and
extensible.

### ~~Q3: Config Aggregation Strategy~~ (Superseded)

**Decision:** Not applicable. The `from:` field now accepts only `#Secret`, not
`string`. Non-sensitive config uses `value:` which emits inline `{ value: "..." }`
on the env var — no ConfigMap is generated. If ConfigMap-backed config is needed,
use `spec.configMaps` and `envFrom` directly. See [RFC-0005](0005-env-config-wiring.md).

### Q4: SecretStore Configuration

Where does the user specify which `ClusterSecretStore` to use for a given
`source` type? Options: (A) Provider-level. (B) Release-level. (C) Policy-level.

**Recommendation:** Option C. Secret store config is an environment concern.
Aligns with the Interface RFC's platform fulfillment model. ESO simplifies this
further — the platform operator configures `ClusterSecretStore` resources once
and all `source: "k8s-eso"` refs resolve through them.

### Q5: Interaction with Volume Resources

How does `#Secret`-based volume mounting interact with the existing
`#VolumeSchema`? Options: (A) Separate concepts. (B) Unified `from:` resolves
to either. (C) New volume type in existing system.

**Recommendation:** Option A for now. PVC and secret volumes have different
lifecycle and security properties.

### Q6: @opm() Attribute as Field Decorator

Could CUE attributes serve as a decorator-like annotation — similar to Python
decorators — that replaces the verbose struct syntax for `#Secret` declarations?

Today, marking a field as sensitive requires the module author to change the
field's type from `string` to `#Secret` and attach routing metadata as struct
fields:

```cue
// Current: field type changes from string to #Secret, routing metadata inline
#config: {
    db: {
        host:     string
        password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
    }
}
```

An alternative approach uses CUE's `@attr()` syntax as a decorator on the
field, keeping the field type as `string` and expressing sensitivity plus
routing metadata entirely through annotation:

```cue
// Proposed: @opm() decorator, field stays string
#config: {
    db: {
        host: string
        @opm(secret, desc="Database Password", secretName="db-creds", secretKey="password")
        password: string
    }
}
```

This pattern is familiar to Python developers where decorators annotate behavior
without changing the underlying type signature. The `@opm()` attribute would
carry the sensitivity tag (`secret`), a human-readable description, and all
routing metadata in a single annotation.

**Potential advantages:**

- **Simpler DX.** Fields remain `string`. Users provide plain values without
  wrapping in `{ value: "..." }`. The `string` -> `#Secret` migration cost
  disappears.
- **Familiar pattern.** Developers who know Python decorators, Java annotations,
  or TypeScript decorators recognize this immediately.
- **Clean schema.** Routing metadata (`secretName`, `secretKey`) lives in
  annotation space, not in the value graph. No `$`-prefixed fields polluting
  struct output.
- **Solves `$opm` visibility.** The `$opm: "secret"` discriminator field
  visibility question (see [RFC-0005 Q2](0005-env-config-wiring.md)) becomes
  moot — attributes are invisible in the value graph by definition.

**Potential disadvantages:**

- **No CUE-native type safety.** The field is still `string`, so CUE evaluation
  alone cannot distinguish sensitive from non-sensitive fields. All sensitivity
  detection moves to the Go SDK via the attribute API.
- **Breaks negation test discovery.** The CUE-native secret discovery pattern
  `(v & {$opm: !="secret", ...}) == _|_` (used in
  [RFC-0005](0005-env-config-wiring.md)) relies on `$opm` being a concrete
  field in the value graph. Attributes are not part of the value graph, so
  discovery must be entirely Go-side.
- **Tooling dependency.** Every tool that needs to know about sensitivity must
  use the Go SDK attribute API. Pure CUE evaluation (e.g., `cue export`) cannot
  distinguish `password: string` from `password: string` with `@opm(secret)`.
- **CUE attribute limitations.** Attributes are metadata — they cannot affect
  unification, defaults, or constraints. Value constraints like
  `strings.MinRunes(12)` would still need to be expressed separately on the
  field.

**Open sub-questions:**

1. Can `@opm()` and `#Secret` coexist? Could `@opm(secret)` be syntactic sugar
   that the CLI expands into the `#Secret` struct before CUE evaluation — giving
   authors the clean decorator syntax while preserving CUE-native type safety
   internally?
2. Should the scope expand beyond secrets? If `@opm()` works as a general
   decorator, it could carry other annotations too (e.g.,
   `@opm(immutable)`, `@opm(deprecated)`).
3. How does this interact with Interface shapes
   ([RFC-0004](0004-interface-architecture.md))? Interface `#shape` fields typed
   as `#Secret` enforce sensitivity at the type level. An attribute-only
   approach would lose that enforcement.

This question expands the scope of the existing
[Deferred Work: @opm() Attribute](#opm-attribute) from "replace the `$opm`
discriminator" to "replace the entire `#Secret` struct pattern with a
decorator-based approach."

## Deferred Work

### @ Tag CLI Implementation

The `@secret(...)` tag resolution requires CLI-side provider authentication
(tokens, credentials, etc.). Start with `@secret(k8s, ...)` and expand
incrementally.

### Additional @ Tags

Future tags: `@env("DB_PASSWORD")` (read from environment), `@file("/run/
secrets/db")` (read from file). These are CLI features, not schema changes.

### Policy Engine Integration

Enforcement rules like `#NoLiteralSecrets` require a policy evaluation pass.
Design deferred until the policy system is specified.

### Controller-Time Secret Constraint Validation

When OPM implements a controller, `#SecretRef` values can be validated against
schema constraints after the controller fetches the actual secret from the
external store. An externally-sourced password that fails a `MinRunes(12)`
constraint would surface as a status condition on the ModuleRelease resource,
rather than silently deploying a non-conforming value. This closes the
validation gap for externally-resolved secrets that the CLI cannot check at
eval time.

### CSI Volume Driver Support

The current design covers K8s Secrets and ExternalSecrets. CSI secret store
drivers (e.g., `secrets-store.csi.k8s.io`) could be added as an additional
provider handler.

### @opm() Attribute

A potential future replacement for the `$opm` discriminator field. CUE
attributes (e.g., `@opm(secret)`) are metadata that don't affect the value
graph. This could provide a cleaner discovery mechanism via the Go SDK's
attribute API, avoiding the `$opm` field visibility question entirely.
See [Q6](#q6-opm-attribute-as-field-decorator) for the broader question of
whether `@opm()` could replace the entire `#Secret` struct pattern with a
decorator-based approach. Deferred to RFC-0005's analysis of the `$opm` field
visibility trade-offs.

## References

- [RFC-0004: Interface Architecture](0004-interface-architecture.md) — `provides`/`requires` typed shapes
- [RFC-0005: Environment & Config Wiring](0005-env-config-wiring.md) — Env var wiring, envFrom, volume mounts
- [External Secrets Operator](https://external-secrets.io/) — ExternalSecret CRD and ClusterSecretStore
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) — Native K8s Secret resources
- [CUE Attributes](https://cuelang.org/docs/reference/spec/#attributes) — `@attr(...)` syntax specification
- [CSI Secrets Store Driver](https://secrets-store-csi-driver.sigs.k8s.io/) — Volume-mounted secrets via CSI
