# RFC-0002: Sensitive Data Model

| Field       | Value              |
|-------------|--------------------|
| **Status**  | Draft              |
| **Created** | 2026-02-09         |
| **Authors** | OPM Contributors   |

## Summary

Introduce a `#Secret` type that makes sensitive data a first-class concept in OPM. Today, all values flow through `#config` → `values` → transformer identically — `db.host` and `db.password` are both plain strings. Passwords end up as plaintext in CUE files, git repositories, and rendered manifests.

`#Secret` tags a field as sensitive at the schema level. This single annotation propagates through every layer — module definition, release fulfillment, transformer output — enabling the toolchain to redact, encrypt, and dispatch secrets to platform-appropriate resources (K8s Secrets, ExternalSecrets, CSI volumes) without the module author managing any of that machinery.

`#Secret` is a three-variant disjunction (`#SecretLiteral | #SecretK8sRef | #SecretEsoRef`). Each variant carries a `$opm: "secret"` discriminator that enables **auto-discovery** — CUE comprehensions walk resolved `#config` values, detect `#Secret` fields via a negation test, and group them by `$secretName` to generate the K8s Secret resource layout automatically. Module authors declare secrets once in `#config` and wire them in env vars — no manual bridging layer required.

The design supports three input paths (literal values, K8s Secret references, ESO external references) plus CLI `@` tag injection, while remaining backward compatible with existing modules.

## Motivation

### The Problem

OPM has no concept of "sensitive." Every value that passes through `#config` → `values` → transformer is a plain string:

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

1. **No redaction** — `cue export` prints passwords alongside hostnames. Logs,    CI output, and debugging sessions expose secrets.
2. **No encryption** — Stored CUE artifacts contain plaintext secrets.
3. **No external references** — No way to say "this value lives in Vault" or    "use the existing K8s Secret called `db-creds`."
4. **No platform integration** — Transformers emit the same resource structure for `host` and `password`. No dispatch to ExternalSecrets Operator, CSI drivers, or other secret management infrastructure.

### The Opportunity

If OPM knows which fields are sensitive, the entire toolchain can act on it:

- **Authors** mark fields as `#Secret` and wire them to containers — done.
- **Users** choose how to provide secrets (literal, external ref, `@` tag).
- **Tooling** redacts secrets in output, encrypts in storage, validates fulfillment before deploy.
- **Transformers** dispatch to the correct platform mechanism based on how the secret was provided.

### Why Now

The [RFC-0004: Interface Architecture](0004-interface-architecture.md) introduces `provides`/`requires` with typed shapes. Those shapes include fields like `#Postgres.password` — currently typed as `string`. Without a sensitive data model, the Interface system has a blind spot: it can type-check that a password field exists, but cannot ensure it is handled securely.

## Prior Art

The `config-sources` experiment (`experiments/001-config-sources/`) prototyped an earlier version of this design. This section documents what was validated and what changed.

### What the Experiment Validated

| Experiment Feature                                                    | Finding                                              |
|-----------------------------------------------------------------------|------------------------------------------------------|
| `#ConfigSourceSchema` with `type: "config" \| "secret"` discriminator | Sensitivity tagging is needed                        |
| `env.from: { source, key }` wiring                                    | Env vars need reference syntax beyond plain `value:` |
| Transformer dispatch (ConfigMap vs Secret based on type)              | Output must differ based on sensitivity              |
| External refs (`externalRef.name`) emitting nothing                   | The "pre-existing resource" pattern works            |
| K8s resource naming `{component}-{source}`                            | Predictable naming is essential                      |

### What This RFC Changes

```text
┌────────────────────────────────────┬────────────────────────────────────────┐
│ Experiment Approach                │ This RFC                               │
├────────────────────────────────────┼────────────────────────────────────────┤
│ configSources as a separate        │ #Secret as a type in #config.          │
│ component resource.                │ Secrets belong at the schema level.    │
│                                    │                                        │
│ env.from: { source: "app-settings" │ env.from: values.db.password           │
│            key: "LOG_LEVEL" }      │ Direct CUE refs. Secrets only.         │
│                                    │                                        │
│ data + externalRef mutual          │ #SecretLiteral | #SecretK8sRef |       │
│ exclusivity.                       │ #SecretEsoRef union. Cleaner CUE.      │
│                                    │                                        │
│ Config and secrets unified in      │ Config is string, secrets are #Secret. │
│ configSources.                     │ value: for config, from: for secrets.  │
└────────────────────────────────────┴────────────────────────────────────────┘
```

### What Carries Forward From Experiment 001

- Transformer dispatch pattern (literal -> K8s Secret, external -> nothing)
- Consistent env var output (`valueFrom.secretKeyRef` for all secret variants)
- Provider handler interface concept
- Naming convention for generated K8s resources

### Experiment 002: Secret Discovery & Auto-Grouping

The `secret-discovery` experiment (`experiments/002-secret-discovery/`) built on this RFC's `#Secret` type to prototype auto-discovery and auto-grouping of secrets from resolved `#config` values. This section documents what was validated and what changed.

#### What Experiment 002 Validated

| Experiment Feature                                   | Finding                                                                        |
|------------------------------------------------------|--------------------------------------------------------------------------------|
| `$opm: "secret"` discriminator on all variants       | Negation test `(v & {$opm: !="secret", ...}) == _|_` reliably detects secrets  |
| Three-level traversal (unrolled comprehensions)      | Covers practical nesting: flat, nested, deeply nested                          |
| Auto-grouping by `$secretName` / `$dataKey`          | Produces K8s Secret resource layout without manual bridging                    |
| Mixed variants in same group (literal + ref)         | Transformer dispatches per-entry within a group                                |
| False-positive rejection                             | Anonymous open structs and scalars correctly skipped                           |
| Three-variant `#Secret` disjunction                  | `#SecretLiteral \| #SecretK8sRef \| #SecretEsoRef` — type is the discriminator |
| Env wiring: `from:` for secrets, `value:` for config | Clean separation without type ambiguity                                        |

#### What This RFC Incorporates From Experiment 002

```text
┌────────────────────────────────────┬────────────────────────────────────────┐
│ Before (RFC-0002 + RFC-0005)       │ After (this RFC)                       │
├────────────────────────────────────┼────────────────────────────────────────┤
│ Single #SecretRef with             │ Split into #SecretK8sRef and           │
│ source: "k8s" | "k8s-eso"          │ #SecretEsoRef. Type is discriminator.  │
│ discriminator field.               │                                        │
│                                    │                                        │
│ Manual spec.secrets bridging       │ Auto-discovery via negation test.      │
│ layer (RFC-0005 Layer 2).          │ No bridging layer needed.              │
│ Three steps per secret.            │ One declaration per secret.            │
│                                    │                                        │
│ path field (dual-purpose:          │ secretName on #SecretK8sRef,           │
│ K8s Secret name or external path). │ externalPath on #SecretEsoRef.         │
│                                    │                                        │
│ description?: string               │ $description?: string                  │
│ (no $ prefix).                     │ ($-prefix for author-set metadata).    │
└────────────────────────────────────┴────────────────────────────────────────┘
```

#### What Carries Forward From Experiment 002

- `$opm: "secret"` discriminator on every variant — enables CUE-native discovery
- `$secretName` / `$dataKey` as author-set routing fields propagated by CUE unification
- Negation test pattern for secret detection
- Auto-grouping comprehension pattern
- `from:` / `value:` env var wiring distinction

## Design

### The #Secret Type

`#Secret` is a struct-only union type with three variants. Every `#Secret` value is always a struct — plain `string` is not accepted. This ensures the transformer can always distinguish sensitive fields by shape, and CUE error messages remain clear (no `string | struct` disjunction ambiguity):

```cue
#Secret: #SecretLiteral | #SecretK8sRef | #SecretEsoRef

#SecretLiteral: {
    $opm:          "secret"
    $secretName!:  #NameType    // K8s Secret resource name (grouping key)
    $dataKey!:     string       // data key within that K8s Secret
    $description?: string
    value!:        string
}

#SecretK8sRef: {
    $opm:          "secret"
    $secretName!:  #NameType
    $dataKey!:     string
    $description?: string
    secretName!:   string       // pre-existing K8s Secret name
    remoteKey!:    string       // key within that K8s Secret
}

#SecretEsoRef: {
    $opm:          "secret"
    $secretName!:  #NameType
    $dataKey!:     string
    $description?: string
    externalPath:  string       // path in the external secret store
    remoteKey:     string       // key within the external secret
}
```

**Variant 1: Literal.** The user provides the actual value. Backward compatible with how OPM works today. Flows through the system into a K8s Secret resource.

```cue
#SecretLiteral: {
    $opm:          "secret"
    $secretName!:  #NameType
    $dataKey!:     string
    $description?: string
    value!:        string
}
```

**Variant 2: K8s Secret Reference.** Points to a pre-existing K8s Secret in the cluster. The value never enters OPM — the Secret already exists. OPM emits no resource, only wires the `secretKeyRef`.

```cue
#SecretK8sRef: {
    $opm:          "secret"
    $secretName!:  #NameType
    $dataKey!:     string
    $description?: string
    secretName!:   string       // pre-existing K8s Secret name
    remoteKey!:    string       // key within that K8s Secret
}
```

**Variant 3: ESO Reference.** Points to an external secret store via External Secrets Operator. OPM emits an `ExternalSecret` CR that creates a K8s Secret at deploy time.

```cue
#SecretEsoRef: {
    $opm:          "secret"
    $secretName!:  #NameType
    $dataKey!:     string
    $description?: string
    externalPath:  string       // path in the external secret store
    remoteKey:     string       // key within the external secret
}
```

Design rationale:

- **`$opm: "secret"` discriminator.** A concrete value present on every `#Secret` variant. Enables CUE-native auto-discovery via the negation test (see [Discovery & Auto-Grouping](#discovery--auto-grouping)). No tags or external tooling needed.

- **Three separate variants instead of `source` discriminator.** The previous design used a single `#SecretRef` with `source: *"k8s" | "k8s-eso"`. Splitting into `#SecretK8sRef` and SecretEsoRef` makes the type itself the discriminator. Each variant carries only the fields relevant to its source — no overloaded `path` field, no dead fields. CUE disjunction handles dispatch.

- **`$secretName` replaces the old `owner` concept.** Clearer: it IS the K8s Secret resource name. The `$` prefix distinguishes author-set routing fields from user-set fulfillment fields. Multiple `#config` fields sharing the same `$secretName` are grouped into one K8s Secret with multiple data keys.

- **`$dataKey` replaces the old `key` concept.** Avoids ambiguity. The old `key` field was overloaded (both the data key AND the external lookup key on references). Now `$dataKey` is always the output data key; `remoteKey` is the external lookup key.

- **`secretName` on `#SecretK8sRef`.** The name of the pre-existing K8s Secret to reference. Replaces the overloaded `path` field from the previous design. Distinct from `$secretName` — `$secretName` is the author's logical grouping key (ignored for K8s refs since OPM does not manage the resource); `secretName` is the actual K8s Secret name in the cluster.

- **`externalPath` on `#SecretEsoRef`.** The path in the external secret store (Vault, AWS SM, GCP SM, etc.). Replaces the overloaded `path` field.

- **`remoteKey` on both ref variants.** Separate from `$dataKey`. The external secret's key (in another K8s Secret or in an external store) may differ from the logical data key that the module uses.

- **`$description` with `$` prefix.** Follows the convention that author-set metadata fields use the `$` prefix to distinguish them from user-set fulfillment fields. Human-readable description of the secret's purpose.

- **No `#SecretDeferred`.** Unfulfilled `#Secret` fields are CUE incompleteness errors, caught at evaluation time. If a user omits a required secret value, CUE itself reports the error — no special deferred variant needed.

- **No `#SecretBase`.** Each variant carries its own fields inline. With three variants and the `$opm` discriminator on all, a shared base adds no value.

- **`$`-prefixed fields.** Regular CUE fields (visible in iteration), not hidden. The `$` prefix is a naming convention to visually distinguish author-set routing fields from user-set fulfillment fields.

- **Users never set `$secretName`/`$dataKey`/`$description`.** CUE unification propagates the author's values through. Users only provide `value` (for literals), `secretName`/`remoteKey` (for K8s refs), or `externalPath`/`remoteKey` (for ESO refs).

Both `$secretName` and `$dataKey` are set by the module author in the `#config` schema. The auto-discovery mechanism (see [Discovery & Auto-Grouping](#discovery--auto-grouping)) walks resolved values, detects `#Secret` fields, and groups them by `$secretName`/`$dataKey` to produce the K8s Secret resource layout — eliminating the manual bridging layer that previously connected `#config` declarations to `spec.secrets`.

The critical property is not which variant is used — it is that **the field is typed as `#Secret` at all**:

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

Because `#Secret` is a CUE definition, authors can constrain the `value` field using standard CUE expressions. No OPM-specific mechanism needed — this is CUE unification.

Constraints fire when the resolved `#Secret` is a `#SecretLiteral` (written directly or injected via `@` tag). For `#SecretK8sRef` and `#SecretEsoRef`, the `value` field is absent, so constraints are inert — they serve as machine-readable documentation.

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
│ #SecretK8sRef / #SecretEsoRef         │  [ ]     │ value not set, inert       │
└───────────────────────────────────────┴──────────┴────────────────────────────┘
```

For `#SecretK8sRef` and `#SecretEsoRef`, constraints remain in the schema as machine-readable declarations. While the CLI cannot validate these at eval time (the value is not yet resolved), a future OPM controller that fetches secrets at reconciliation time can evaluate constraints post-fetch and surface violations as status conditions. See [Deferred Work](#deferred-work).

### Input Paths

Three input paths for providing secret values, plus CLI `@` tag injection. All can coexist within a single module release.

```text
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                          HOW SECRETS ENTER OPM                                      │
│                                                                                     │
│  ┌───────────────┐  ┌────────────────┐  ┌────────────────┐  ┌────────────────────┐  │
│  │ Path 1:       │  │ Path 2:        │  │ Path 3:        │  │ Path 4:            │  │
│  │ Literal       │  │ K8s Ref        │  │ ESO Ref        │  │ @ Tag              │  │
│  │               │  │                │  │                │  │                    │  │
│  │ { value: ..}  │  │ { secretName:  │  │ { externalPath │  │ @secret(eso,...)   │  │
│  │               │  │   remoteKey: } │  │   remoteKey: } │  │                    │  │
│  │ User provides │  │ Pre-existing   │  │ ESO resolves   │  │ CLI resolves to    │  │
│  │ the value.    │  │ K8s Secret.    │  │ at deploy.     │  │ { value: "..." }   │  │
│  └──────┬────────┘  └──────┬─────────┘  └──────┬─────────┘  └──────┬─────────────┘  │
│         └──────────────────┼──────────────────┼────────────────────┘                │
│                            ▼                  ▼                                     │
│                  ┌───────────────────┐                                              │
│                  │  #Secret field    │                                              │
│                  │  in #config       │                                              │
│                  └───────────────────┘                                              │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

**Path 1: Literal.** User writes the value directly in `values`:

```cue
// Module definition
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }

// Module release — resolves to #SecretLiteral
values: db: password: { value: "my-secret-password" }
```

**Path 2: K8s Secret Reference.** User points to a pre-existing K8s Secret:

```cue
// Resolves to #SecretK8sRef
values: db: password: {
    secretName: "existing-db-secret"
    remoteKey:  "password"
}
```

**Path 3: ESO Reference.** User points to an external secret store via ESO:

```cue
// Resolves to #SecretEsoRef
values: db: password: {
    externalPath: "secret/data/prod/db"
    remoteKey:    "password"
}
```

**Path 4: `@` Tag Injection.** CUE attributes resolved by the CLI before evaluation:

```cue
values: db: password: _ @secret(source=eso, path="secret/data/prod/db", key=password)
```

The CLI transforms this to `{ value: "the-fetched-value" }` before CUE eval. The `@secret(...)` tag is CLI sugar — not part of the OPM schema. Module authors do not need to know about it. Future tags (e.g., `@env("DB_PASSWORD")`, `@file("/run/secrets/db")`) can be added without schema changes.

### Discovery & Auto-Grouping

The `$opm: "secret"` discriminator on every `#Secret` variant enables **automatic discovery** of secret fields from resolved `#config` values. This eliminates the manual bridging layer that previously required module authors to declare each secret in `#config`, repeat the grouping in `spec.secrets`, and wire it in env vars — three steps per secret.

With auto-discovery, module authors declare secrets once in `#config` and wire them in env vars. The system discovers and groups secrets automatically.

#### Negation-Based Detection

To detect whether a resolved value is a `#Secret`, the system uses a CUE negation test:

```cue
(v & {$opm: !="secret", ...}) == _|_
```

This expression produces bottom (true) **only** when `$opm` is already `"secret"` on the value. For any other value:

```text
┌──────────────────────────────────────┬────────────┬────────────────────────────────┐
│ Value Shape                          │ Result     │ Reason                         │
├──────────────────────────────────────┼────────────┼────────────────────────────────┤
│ #Secret variant ($opm: "secret")     │ _|_ (true) │ "secret" != "secret" conflicts │
│ Scalar (string, int, bool)           │ not _|_    │ Fails struct unification       │
│ Anonymous open struct (no $opm)      │ not _|_    │ $opm added without conflict    │
│ Closed definition struct             │ not _|_    │ $opm rejected as disallowed    │
└──────────────────────────────────────┴────────────┴────────────────────────────────┘
```

No false positives regardless of struct closedness. The second guard `(v & {...}) != _|_` filters out scalars before the negation test runs.

#### Three-Level Traversal

CUE has no recursion. The discovery comprehension manually traverses up to three levels deep, which covers the practical nesting patterns in module configs:

```cue
_#discoverSecrets: {
    #in: {...}
    out: {
        // Level 1: direct fields (e.g., #config.dbPassword)
        for k1, v1 in #in
        if ((v1 & {$opm: !="secret", ...}) == _|_)
        if ((v1 & {...}) != _|_) {
            (k1): v1
        }

        // Level 2: one level of nesting (e.g., #config.cache.password)
        for k1, v1 in #in
        if ((v1 & {$opm: !="secret", ...}) != _|_)
        if ((v1 & {...}) != _|_) {
            for k2, v2 in v1
            if ((v2 & {$opm: !="secret", ...}) == _|_)
            if ((v2 & {...}) != _|_) {
                ("\(k1)/\(k2)"): v2
            }
        }

        // Level 3: two levels of nesting (e.g., #config.integrations.payments.stripeKey)
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

The result is a flat map of all discovered secrets keyed by their config path (e.g., `"dbUser"`, `"cache/password"`, `"integrations/payments/stripeKey"`). The path keys are internal identifiers — grouping uses `$secretName`/`$dataKey`.

#### Auto-Grouping

Discovered secrets are grouped by `$secretName`, keyed by `$dataKey`. This produces the K8s Secret resource layout automatically:

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

Multiple `#config` fields sharing the same `$secretName` are grouped into one K8s Secret with multiple data keys:

```text
#config: {
    dbUser:     #Secret & { $secretName: "db-credentials", $dataKey: "username" }
    dbPassword: #Secret & { $secretName: "db-credentials", $dataKey: "password" }
}

// Auto-grouped result:
spec: secrets: {
    "db-credentials": {
        "username": { ... }    // from dbUser
        "password": { ... }    // from dbPassword
    }
}
```

Mixed variants (literal + ref) within the same group are handled per-entry by the transformer. The auto-discovery and auto-grouping run as part of CUE evaluation — no Go code or external tooling required at this stage.

#### Complete Flow

```text
┌──────────────────────────────────────────────────────────────┐
│  1. Author declares #Secret fields in #config                │
│     dbUser: #Secret & { $secretName: "db-creds", ... }       │
│                                                              │
│  2. User fulfills with a variant in values                   │
│     values: dbUser: { value: "admin" }                       │
│                                                              │
│  3. Auto-discovery walks resolved values                     │
│     _discovered: { dbUser: { $opm: "secret", ... } }         │
│                                                              │
│  4. Auto-grouping produces spec.secrets                      │
│     spec: secrets: { "db-creds": { "username": ... } }       │
│                                                              │
│  5. Transformer emits K8s resources from spec.secrets        │
│     Secret/db-creds, ExternalSecret/..., or nothing          │
│                                                              │
│  6. Author wires env vars with from:/value:                  │
│     env: DB_USER: { from: values.dbUser }                    │
└──────────────────────────────────────────────────────────────┘
```

#### CUE-Native vs Go-Based Discovery

The discovery mechanism can be implemented either in pure CUE (comprehensions with the negation test) or in Go (walking the CUE value tree via the `cuelang.org/go` API). Both produce the same output — a flat map of discovered secrets and a grouped `spec.secrets` layout. The trade-offs are about where the logic lives and what constraints each approach imposes.

```text
┌──────────────────────┬─────────────────────────────────┬──────────────────────────────────┐
│ Dimension            │ CUE-Native                      │ Go-Based                         │
├──────────────────────┼─────────────────────────────────┼──────────────────────────────────┤
│ Discovery mechanism  │ Negation test in CUE            │ Walk the CUE value tree via      │
│                      │ comprehensions. Relies on $opm  │ cuelang.org/go API. Test each    │
│                      │ as a concrete field in the      │ field for $opm == "secret" or    │
│                      │ value graph.                    │ use Value.Unify() with #Secret.  │
│                      │                                 │ Could also use attribute API if  │
│                      │                                 │ @opm(secret) were adopted.       │
│                      │                                 │                                  │
│ Auto-grouping        │ CUE comprehension iterates the  │ Go code builds a                 │
│                      │ discovered map, groups by       │ map[string]map[string]Secret in  │
│                      │ $secretName/$dataKey. Result is │ memory. Result injected back     │
│                      │ part of CUE evaluation output.  │ into CUE or emitted directly.    │
│                      │                                 │                                  │
│ Depth limitations    │ Fixed at 3 levels (manually     │ Unlimited recursion. Arbitrary   │
│                      │ unrolled). CUE lacks recursion. │ depth with a standard recursive  │
│                      │ Level 4+ requires extending     │ function.                        │
│                      │ the comprehension.              │                                  │
│                      │                                 │                                  │
│ Tooling independence │ Works with plain cue eval,      │ Requires the OPM CLI or Go SDK.  │
│                      │ cue vet, cue export. No OPM     │ cue eval alone cannot run the    │
│                      │ CLI required. Authors test      │ discovery -- Go logic runs       │
│                      │ discovery with standard CUE     │ separately.                      │
│                      │ tooling.                        │                                  │
│                      │                                 │                                  │
│ Extensibility        │ New discriminators work if the  │ New discriminators, variant      │
│                      │ comprehension is generalized.   │ types, and arbitrary depth are   │
│                      │ New variants work if they carry │ trivial to add. Can discover     │
│                      │ $opm. Deeper nesting requires   │ based on attributes, type        │
│                      │ manual unrolling.               │ assertions, or structural        │
│                      │                                 │ patterns.                        │
│                      │                                 │                                  │
│ Error reporting      │ CUE evaluation errors if the    │ Rich diagnostics: report         │
│                      │ negation test is malformed.     │ discovered count per depth,      │
│                      │ False negatives are silent.     │ warn about unusual patterns,     │
│                      │ No diagnostic output for        │ explain skipped fields with      │
│                      │ skipped fields.                 │ reasons.                         │
└──────────────────────┴─────────────────────────────────┴──────────────────────────────────┘
```

**Discovery mechanism.** Both approaches detect secrets reliably. CUE-native is elegant but opaque — the negation test is non-obvious to developers unfamiliar with CUE's evaluation model. Go-based is more readable but couples discovery to the OPM CLI.

**Auto-grouping.** Functionally equivalent. CUE-native has the advantage that the grouped result is visible in `cue eval` output — module authors can inspect `spec.secrets` directly. Go-based grouping is internal unless explicitly surfaced.

**Depth limitations.** The most significant practical difference. Three levels covers observed patterns but is a hard limit. Go has no such constraint. If CUE adds recursion support in the future, this gap closes.

**Tooling independence.** The strongest argument for CUE-native. Module authors can validate discovery with `cue eval main.cue -e _discovered --all` without installing the OPM CLI. This aligns with OPM's Principle I: CUE-native validation at definition time.

**Extensibility.** Go is more extensible in every dimension. CUE-native extensibility is constrained by CUE's language features. However, the current design (one discriminator, three variants, three levels) is sufficient for the foreseeable scope.

**Error reporting.** Go wins clearly. CUE comprehensions are silent about what they skip — correct behavior, but poor for debugging. A Go-based approach can emit warnings ("field X looks like a secret but lacks `$opm`") that CUE cannot.

**Decision.** The CUE-native approach is adopted as the primary discovery mechanism. Tooling independence — module authors testing discovery with standard CUE tools — outweighs the depth limitation and error reporting gaps. However, the OPM CLI's Go-based transformer independently walks the value tree to produce K8s resources, serving as a second pass that applies Go-based validation, diagnostics, and handles edge cases that CUE comprehensions cannot express. This gives both: CUE-native discovery for author-time validation and Go-based processing for build-time resource generation.

### Output Dispatch

The transformer inspects the resolved `#Secret` variant and produces different K8s resources. With three separate types, dispatch is based on variant type — no `source` field inspection needed:

```text
┌───────────────────────────────────────┬──────────────────────────┬──────────────────────────────────┐
│ Input Variant                         │ K8s Resource Emitted     │ Env Var Wiring                   │
├───────────────────────────────────────┼──────────────────────────┼──────────────────────────────────┤
│ #SecretLiteral { value: "..." }       │ Secret (base64 data)     │ secretKeyRef -> $secretName      │
│ #SecretK8sRef                         │ Nothing (already exists) │ secretKeyRef -> secretName       │
│ #SecretEsoRef                         │ ExternalSecret CR        │ secretKeyRef -> $secretName      │
└───────────────────────────────────────┴──────────────────────────┴──────────────────────────────────┘
```

**Resource naming.** The K8s Secret name is the `$secretName` field. The module author sets `$secretName` in the `#config` schema. For `#SecretK8sRef`, the `secretName` field IS the pre-existing K8s Secret name — `$secretName` is irrelevant because OPM does not manage the resource.

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
-> K8s Secret name: "db-creds"
-> K8s Secret key:  "password"
```

**Consistent env var output.** Regardless of variant, the container wiring is always `valueFrom.secretKeyRef`. Only the `name` and `key` change:

```text
┌─────────────────────┐      ┌──────────────────────────────────┐
│  #SecretLiteral     │---->>│  name: "db-creds"                │  ($secretName)
│  #SecretEsoRef      │---->>│  name: "db-creds"                │  ($secretName, ESO target)
│  #SecretK8sRef      │---->>│  name: "existing-db-secret"      │  (secretName)
└─────────────────────┘      └──────────────────────────────────┘
                                        │
                                        ▼
                             env:
                               - name: DB_PASSWORD
                                 valueFrom:
                                   secretKeyRef:
                                     name: <$secretName or secretName>
                                     key: <$dataKey or remoteKey>
```

### Wiring Model

Developers wire config and secrets to container env vars using two fields: `value` for non-sensitive data (plain strings) and `from` for sensitive data (`#Secret` references). The `from` field accepts only `#Secret` — non-sensitive config always uses `value`. Both reference resolved `values` (the unified result of `#config` schema + user fulfillment):

```cue
#config: {
    log_level: string
    db: {
        host:     string
        password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
    }
}

values: #config & {
    log_level: "info"
    db: {
        host:     "db.prod.internal"
        password: { value: "my-secret" }
    }
}

spec: container: env: {
    LOG_LEVEL:   { value: values.log_level }
    DB_HOST:     { value: values.db.host }
    DB_PASSWORD: { from:  values.db.password }
}
```

The two fields have distinct semantics:

```text
value: values.log_level
  -> resolves to string "info"
  -> emit: { value: "info" }

from: values.db.password
  -> resolves to #Secret (carries $opm, $secretName, $dataKey, ...)
  -> transformer dispatches by variant type:
     #SecretLiteral → { valueFrom: { secretKeyRef: { name: $secretName, key: $dataKey } } }
     #SecretK8sRef  → { valueFrom: { secretKeyRef: { name: secretName, key: remoteKey } } }
     #SecretEsoRef  → { valueFrom: { secretKeyRef: { name: $secretName, key: $dataKey } } }
```

`value` handles non-sensitive config via direct CUE references that resolve to strings. `from` handles secrets via direct CUE references that resolve to `#Secret`. Both are type-safe (CUE validates at definition time) and self-documenting. The `from` field carries the full resolved `#Secret` struct including routing info — the transformer reads `$secretName`, `$dataKey`, and the variant-specific fields to produce the correct `secretKeyRef`.

**EnvVar schema:**

```cue
#EnvVarSchema: {
    name!:  string
    value?: string    // inline literal (non-sensitive config)
    from?:  #Secret   // reference to a #Secret in values
}
```

`value` and `from` are mutually exclusive. An env var is either a non-sensitive literal (`value`) or a reference to a `#Secret` field (`from`). If both are set, CUE evaluation errors. If neither is set, the env var declaration is incomplete. See [RFC-0005](0005-env-config-wiring.md) for the full `#EnvVarSchema` including `fieldRef` and `resourceFieldRef`.

**Why two fields, not one.** A single `value: string | #Secret` field was considered. It eliminates the mutual-exclusivity question, but creates a `value: { value: "..." }` stutter when a `#SecretLiteral` is resolved — the outer `value` (env var field) wraps the inner `value` (`#SecretLiteral.value`). Two fields give each case its natural keyword: `value` for non-sensitive literals, `from` for secret references. The separation also makes transformer logic straightforward — `value` is always a plain string, `from` is always a `#Secret`.

### Provider Handlers

The K8s provider dispatches on variant type. Each variant type has a corresponding handler that produces K8s resources and a `secretKeyRef`:

```cue
#SecretSourceHandler: {
    #resolve: {
        #secret:    #Secret
        #component: _
        #context:   #TransformerContext

        resources:    [string]: {...}     // K8s resources to emit (may be empty)
        secretKeyRef: { name!: string, key!: string }
    }
}
```

**Dispatch by variant type:**

```text
┌─────────────────┬─────────────────────────────────────────────────────┐
│ Variant Type    │ Handler Behavior                                    │
├─────────────────┼─────────────────────────────────────────────────────┤
│ #SecretLiteral  │ Emit K8s Secret with base64 data entry.             │
│                 │ secretKeyRef: { name: $secretName, key: $dataKey }  │
│                 │                                                     │
│ #SecretK8sRef   │ Emit nothing (Secret already exists in cluster).    │
│                 │ secretKeyRef: { name: secretName, key: remoteKey }  │
│                 │                                                     │
│ #SecretEsoRef   │ Emit ExternalSecret CR (ESO creates target Secret). │
│                 │ secretKeyRef: { name: $secretName, key: $dataKey }  │
└─────────────────┴─────────────────────────────────────────────────────┘
```

Since variant types are separate CUE definitions, dispatch is type-based — no `source` field inspection needed. ESO (External Secrets Operator) abstracts over external providers (Vault, AWS SM, GCP SM, etc.), so backend `ClusterSecretStore` configuration is a platform-level concern — deployed and configured outside OPM module scope.

**ExternalSecret handler (ESO)** — emits an `ExternalSecret` CR:

```cue
#ExternalSecretHandler: #SecretSourceHandler & {
    #resolve: {
        let _name = #secret.$secretName

        resources: "\(_name)": {
            apiVersion: "external-secrets.io/v1beta1"
            kind:       "ExternalSecret"
            metadata: name: _name
            spec: {
                refreshInterval: "1h"
                secretStoreRef: { kind: "ClusterSecretStore" }
                target: name: _name
                data: [{
                    secretKey: #secret.$dataKey
                    remoteRef: {
                        key:      #secret.externalPath
                        property: #secret.remoteKey
                    }
                }]
            }
        }
        secretKeyRef: { name: _name, key: #secret.$dataKey }
    }
}
```

The `ClusterSecretStore` name is resolved at the platform level. Module authors and users do not specify it — it is an infrastructure concern.

### Volume-Mounted Secrets

Not all secrets are environment variables. TLS certificates, service account keys, and credential files are mounted as volumes. The same `#Secret` type handles both — what differs is the wiring target.

```cue
// Env var wiring
env: DB_PASSWORD: { name: "DB_PASSWORD", from: values.db.password }

// Volume mount wiring
volumeMounts: "tls-cert": { mountPath: "/etc/tls", from: values.tls }
```

When `from` resolves to a `#Secret` in a volume mount context, the transformer emits a K8s Secret (or ExternalSecret), a `volume` entry referencing it, and a `volumeMount` on the container.

Multi-key secrets become multiple files in the mounted volume:

```cue
#config: tls: #Secret & { $secretName: "tls-cert", $dataKey: "tls.crt" }

volumeMounts: "tls": {
    mountPath: "/etc/tls"
    from:      values.tls
    // -> /etc/tls/tls.crt
    // -> /etc/tls/tls.key
}
```

### Unified Config Pattern

Config (non-sensitive) and secrets (sensitive) use different fields but the same referencing pattern — direct CUE references to resolved `values`:

```cue
#config: {
    log_level:   string                                                    // non-sensitive
    app_port:    int                                                       // non-sensitive
    db_password: #Secret & { $secretName: "db-creds", $dataKey: "password" }  // sensitive
    api_key:     #Secret & { $secretName: "api-key", $dataKey: "token" }      // sensitive
    tls:         #Secret & { $secretName: "tls-cert", $dataKey: "tls.crt" }   // sensitive (volume target)
}

values: #config & { /* user fulfillment */ }

spec: container: {
    env: {
        LOG_LEVEL:   { value: values.log_level }
        APP_PORT:    { value: "\(values.app_port)" }
        DB_PASSWORD: { from:  values.db_password }
        API_KEY:     { from:  values.api_key }
    }
    volumeMounts: "tls": { mountPath: "/etc/tls", from: values.tls }
}
```

```text
value: values.log_level   type: string   -> { value: "info" }
from:  values.db_password type: #Secret  -> K8s Secret + secretKeyRef
from:  values.tls         type: #Secret  -> K8s Secret + volume + volumeMount
```

Non-sensitive fields resolve to plain strings via `value:`. The transformer emits inline `{ value: "..." }` on the env var. No ConfigMap is generated from `value:` references — if ConfigMap-backed config is needed, use `spec.configMaps` directly. See [RFC-0005](0005-env-config-wiring.md) for the full env wiring model.

Secret fields are auto-discovered from `values` and auto-grouped into `spec.secrets` (see [Discovery & Auto-Grouping](#discovery--auto-grouping)). The module author does not need to manually declare `spec.secrets`.

## Interface Integration

The [RFC-0004: Interface Architecture](0004-interface-architecture.md) defines `provides`/`requires` with typed shapes. This RFC upgrades sensitive fields from `string` to `#Secret`:

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

Platform fulfillment must now provide a `#Secret` for `password`. The module author uses `value:` for non-sensitive fields and `from:` for secrets:

```cue
requires: "db": #Postgres

spec: container: env: {
    DB_HOST:     { value: requires.db.host }                            // string -> plain
    DB_PASSWORD: { from:  requires.db.password }                        // #Secret -> secretKeyRef
}
```

The module author does not know or care whether `password` was fulfilled with a `#SecretLiteral`, a `#SecretK8sRef`, or a `#SecretEsoRef`.

## Backward Compatibility

### Migration: string to #Secret

When a field is upgraded from `string` to `#Secret`, existing literal values must be wrapped in `#SecretLiteral`. This is a one-time mechanical change:

```cue
// Before (plain string)
values: db: password: "my-secret-password"

// After (#Secret requires struct)
values: db: password: { value: "my-secret-password" }
```

`#Secret` does not accept bare `string`. This is deliberate — it ensures that sensitivity is always explicit and the transformer can distinguish sensitive fields by shape without inspecting the schema type.

### Warnings

The OPM CLI emits warnings when `#SecretLiteral` is used in production contexts:

```text
WARNING: db.password is a #Secret field with a literal value. Consider using a #SecretK8sRef, #SecretEsoRef, or @secret() tag for production.
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
values:  db: password: { externalPath: "secret/data/prod/db", remoteKey: "password" }

Resolves to: #SecretEsoRef

Transformer emits:
  ExternalSecret/db-creds  (references ClusterSecretStore)
  env: DB_PASSWORD -> secretKeyRef -> db-creds / password

No plaintext value in CUE files or rendered manifests. [x]
```

### Scenario C: Existing K8s Secret [x]

```text
#config: db: password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
values:  db: password: { secretName: "existing-db-secret", remoteKey: "password" }

Resolves to: #SecretK8sRef

Transformer emits:
  Nothing (Secret already exists in cluster)
  env: DB_PASSWORD -> secretKeyRef -> existing-db-secret / password

Zero new resources. $secretName is ignored for #SecretK8sRef. [x]
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
              password: { externalPath: "secret/data/prod/db", remoteKey: "password" }
          }

Module author wiring:
  env: DB_PASSWORD: { from: requires.db.password }

Result: Module author wrote no secret-handling code.
        Platform provided #SecretEsoRef. Transformer emits ExternalSecret. [x]
```

### Scenario G: Auto-Discovery with Mixed Variants [x]

```text
#config: {
    dbUser:     #Secret & { $secretName: "db-credentials", $dataKey: "username" }
    dbPassword: #Secret & { $secretName: "db-credentials", $dataKey: "password" }
    apiKey:     #Secret & { $secretName: "api-credentials", $dataKey: "api-key" }
    logLevel:   string | *"info"
    cache: {
        password: #Secret & { $secretName: "cache-credentials", $dataKey: "password" }
        host:     string
    }
}

values: #config & {
    dbUser:     { value: "admin" }                                            // #SecretLiteral
    dbPassword: { secretName: "myapp-db-secrets", remoteKey: "db-password" }  // #SecretK8sRef
    apiKey:     { value: "sk_live_abc123" }                                   // #SecretLiteral
    logLevel:   "debug"
    cache: {
        password: { externalPath: "secret/cache", remoteKey: "password" }     // #SecretEsoRef
        host:     "redis://cache.prod.internal"
    }
}

Auto-discovery finds: dbUser, dbPassword, apiKey, cache/password
  (logLevel: string -- skipped. cache/host: string -- skipped.)

Auto-grouping produces:
  "db-credentials":    { username: #SecretLiteral, password: #SecretK8sRef }
  "api-credentials":   { api-key: #SecretLiteral }
  "cache-credentials": { password: #SecretEsoRef }

Transformer emits:
  Secret/db-credentials     { data: { username: base64("admin") } }
  Secret/api-credentials    { data: { api-key: base64("sk_live_abc123") } }
  ExternalSecret/cache-credentials
  Nothing for dbPassword    (pre-existing K8s Secret)

Mixed variants in "db-credentials" group handled per-entry. [x]
```

## Trade-offs

**Advantages:**

- Type-safe sensitivity — `#Secret` is checked by CUE at definition time.
- Developer simplicity — declare `#Secret`, wire with `from:`, done. No manual bridging layer.
- Auto-discovery — secrets are detected from resolved values via CUE-native negation test. No external tooling or Go code needed at the CUE evaluation stage.
- Auto-grouping — discovered secrets are grouped by `$secretName`/`$dataKey` into the K8s Secret resource layout automatically. One declaration per secret instead of three.
- User flexibility — three input paths (`#SecretLiteral`, `#SecretK8sRef`, `#SecretEsoRef`) plus `@` tag cover dev through production.
- Toolchain awareness — every tool in the pipeline can distinguish sensitive from non-sensitive.
- Platform portability — variant types dispatch to K8s Secrets, ExternalSecrets, or future provider equivalents.
- Clear wiring — `value:` for config, `from:` for secrets. Same CUE reference pattern.
- CUE-native validation — unfulfilled secrets produce CUE incompleteness errors with no custom deferred variant needed.
- No overloaded fields — each variant carries only the fields relevant to its source. No dual-purpose `path` or `source` discriminator field.

**Disadvantages:**

- New core type — developers must learn `#Secret` alongside Resource, Trait, Blueprint, Interface.
- Three variant types — more types to understand than a single `#SecretRef` wi  handlers must be implemented and maintained.
- `@` tag requires CLI support — useless without OPM CLI implementation.
- Migration cost — existing modules must wrap literal values in `{ value: "..." }` when upgrading fields from `string` to `#Secret`.
- Three-level depth limit — the auto-discovery traversal covers up to three levels of nesting. Deeper nesting requires extending the comprehension.

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
│                                          │          │            │                                │
│ Three-level depth limit insufficient     │ Low      │ Low        │ 3 levels covers practical      │
│ for deeply nested configs                │          │            │ patterns. Extend if needed.    │
└──────────────────────────────────────────┴──────────┴────────────┴────────────────────────────────┘
```

## Open Questions

### ~~Q1: CUE Representation of #Secret~~ (Decided)

**Decision:** Option C — always-struct. `#Secret` does not accept bare `string`. All three variants (`#SecretLiteral`, `#SecretK8sRef`, `#SecretEsoRef`) are structs. This eliminates the `string | struct` disjunction problem in CUE, produces clear error messages, and ensures the transformer can always distinguish sensitive fields by shape. Additionally, experiment 002 validated that the struct-based approach enables CUE-native auto-discovery via the `$opm: "secret"` discriminator and negation test.

### Q2: @ Tag Syntax

CUE attributes have the form `@attr(arg1, arg2, ...)`.

Options: (A) Positional: `@secret(eso, "secret/data/db", "password")`.
(B) Named: `@secret(source=eso, path="secret/data/db", key=password)`.
(C) URI-style: `@secret("eso:secret/data/db#password")`.

**Recommendation:** Option B. Named parameters are self-documenting and
extensible.

### ~~Q3: Config Aggregation Strategy~~ (Superseded)

**Decision:** Not applicable. The `from:` field now accepts only `#Secret`, not `string`. Non-sensitive config uses `value:` which emits inline `{ value: "..." }` on the env var — no ConfigMap is generated. If ConfigMap-backed config is needed, use `spec.configMaps` and `envFrom` directly. See [RFC-0005](0005-env-config-wiring.md).

### Q4: SecretStore Configuration

Where does the user specify which `ClusterSecretStore` to use for a given `#SecretEsoRef`? Options: (A) Provider-level. (B) Release-level. (C) Policy-level.

**Recommendation:** Option C. Secret store config is an environment concern. Aligns with the Interface RFC's platform fulfillment model. ESO simplifies this further — the platform operator configures `ClusterSecretStore` resources once and all `#SecretEsoRef` variants resolve through them.

### Q5: Interaction with Volume Resources

How does `#Secret`-based volume mounting interact with the existing `#VolumeSchema`? Options: (A) Separate concepts. (B) Unified `from:` resolves to either. (C) New volume type in existing system.

**Recommendation:** Option A for now. PVC and secret volumes have different lifecycle and security properties.

### Q6: @opm() Attribute as Field Decorator

Could CUE attributes serve as a decorator-like annotation — similar to Python decorators — that replaces the verbose struct syntax for `#Secret` declarations?

Today, marking a field as sensitive requires the module author to change the field's type from `string` to `#Secret` and attach routing metadata as struct fields:

```cue
// Current: field type changes from string to #Secret, routing metadata inline
#config: {
    db: {
        host:     string
        password: #Secret & { $secretName: "db-creds", $dataKey: "password" }
    }
}
```

An alternative approach uses CUE's `@attr()` syntax as a decorator on the field, keeping the field type as `string` and expressing sensitivity plus routing metadata entirely through annotation:

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

This pattern is familiar to Python developers where decorators annotate behavior without changing the underlying type signature. The `@opm()` attribute would carry the sensitivity tag (`secret`), a human-readable description, and all routing metadata in a single annotation.

**Potential advantages:**

- **Simpler DX.** Fields remain `string`. Users provide plain values without wrapping in `{ value: "..." }`. The `string` -> `#Secret` migration cost disappears.
- **Familiar pattern.** Developers who know Python decorators, Java annotations, or TypeScript decorators recognize this immediately.
- **Clean schema.** Routing metadata (`secretName`, `secretKey`) lives in annotation space, not in the value graph. No `$`-prefixed fields polluting  struct output.
- **Solves `$opm` visibility.** The `$opm: "secret"` discriminator field visibility question (see [RFC-0005 Q2](0005-env-config-wiring.md)) becomes moot — attributes are invisible in the value graph by definition.

**Potential disadvantages:**

- **No CUE-native type safety.** The field is still `string`, so CUE evaluation alone cannot distinguish sensitive from non-sensitive fields. All sensitivity detection moves to the Go SDK via the attribute API.
- **Breaks auto-discovery.** The CUE-native secret discovery pattern `(v & {$opm: !="secret", ...}) == _|_` — validated in experiment 002 and adopted in this RFC (see [Discovery & Auto-Grouping](#discovery--auto-grouping)) — relies on `$opm` being a concrete field in the value graph. Attributes are not part of the value graph, so discovery and auto-grouping would move entirely to Go code, losing the CUE-native evaluation advantage.
- **Tooling dependency.** Every tool that needs to know about sensitivity must use the Go SDK attribute API. Pure CUE evaluation (e.g., `cue export`) cannot distinguish `password: string` from `password: string` with `@opm(secret)`.
- **CUE attribute limitations.** Attributes are metadata — they cannot affect unification, defaults, or constraints. Value constraints like `strings.MinRunes(12)` would still need to be expressed separately on the field.

**Assessment after experiment 002:** The auto-discovery mechanism validated in experiment 002 is a strong argument against the decorator approach. The negation test and auto-grouping are pure CUE — no Go code needed. Moving to attributes would sacrifice this property. The `$opm: "secret"` discriminator, while visible in the value graph, is what makes auto-discovery possible.

**Remaining sub-questions:**

1. Could `@opm(secret)` be syntactic sugar that the CLI expands into the `#Secret` struct before CUE evaluation — preserving auto-discovery while giving authors cleaner syntax?
2. Should `@opm()` be explored for non-secret annotations (e.g., `@opm(immutable)`, `@opm(deprecated)`) where auto-discovery is not needed?

This question expands the scope of the existing [Deferred Work: @opm() Attribute](#opm-attribute) from "replace the `$opm` discriminator" to "replace the entire `#Secret` struct pattern with a decorator-based approach."

## Deferred Work

### @ Tag CLI Implementation

The `@secret(...)` tag resolution requires CLI-side provider authentication (tokens, credentials, etc.). Start with `@secret(k8s, ...)` and expand incrementally.

### Additional @ Tags

Future tags: `@env("DB_PASSWORD")` (read from environment), `@file("/run/secrets/db")` (read from file). These are CLI features, not schema changes.

### Policy Engine Integration

Enforcement rules like `#NoLiteralSecrets` require a policy evaluation pass. Design deferred until the policy system is specified.

### Controller-Time Secret Constraint Validation

When OPM implements a controller, `#SecretK8sRef` and `#SecretEsoRef` values can be validated against schema constraints after the controller fetches the actual secret from the external store. An externally-sourced password that fails a `MinRunes(12)` constraint would surface as a status condition on the ModuleRelease resource, rather than silently deploying a non-conforming value. This closes the validation gap for externally-resolved secrets that the CLI cannot check at eval time.

### CSI Volume Driver Support

The current design covers K8s Secrets and ExternalSecrets via three variant types. CSI secret store drivers (e.g., `secrets-store.csi.k8s.io`) could be added as an additional variant type (e.g., `#SecretCsiRef`).

### Recursive Discovery

The current auto-discovery traversal is limited to three levels of nesting (unrolled comprehensions). If CUE gains recursion support in the future, the traversal could be generalized to arbitrary depth. For now, three levels covers the practical nesting patterns observed in module configs.

### @opm() Attribute

A potential future replacement for the `$opm` discriminator field. CUE attributes (e.g., `@opm(secret)`) are metadata that don't affect the value graph. However, experiment 002 validated that the `$opm: "secret"` discriminator in the value graph is what enables CUE-native auto-discovery and auto-grouping. Replacing it with an attribute would move discovery entirely to Go code. See [Q6](#q6-opm-attribute-as-field-decorator) for the full analysis. A potential middle ground: `@opm(secret)` as CLI sugar that expands into the `#Secret` struct before CUE evaluation.

## References

- [RFC-0004: Interface Architecture](0004-interface-architecture.md) — `provides`/`requires` typed shapes
- [RFC-0005: Environment & Config Wiring](0005-env-config-wiring.md) — Env var wiring, envFrom, volume mounts
- [Experiment 001: Config Sources](../../../catalog/experiments/001-config-sources/) — Initial config/secret prototyping
- [Experiment 002: Secret Discovery](../../../catalog/experiments/002-secret-discovery/) — Auto-discovery and auto-grouping validation
- [External Secrets Operator](https://external-secrets.io/) — ExternalSecret CRD and ClusterSecretStore
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) — Native K8s Secret resources
- [CUE Attributes](https://cuelang.org/docs/reference/spec/#attributes) — `@attr(...)` syntax specification
- [CSI Secrets Store Driver](https://secrets-store-csi-driver.sigs.k8s.io/) — Volume-mounted secrets via CSI
