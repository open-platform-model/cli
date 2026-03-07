# OPM Authoring Guide

This guide documents the patterns, conventions, and pitfalls for writing OPM
artifacts as learned from the factory experiment. It covers all four artifact
types: Module, ModuleRelease, Bundle, and BundleRelease.

---

## Personas

| Artifact       | Written by            | Consumed by                         |
|----------------|-----------------------|-------------------------------------|
| Module         | Module Author         | Platform Operators, End-users       |
| ModuleRelease  | End-user              | OPM renderer                        |
| Bundle         | Platform Operator     | End-users                           |
| BundleRelease  | End-user              | OPM renderer                        |

---

## 1. Writing a Module

A Module is a portable application blueprint. It defines a config schema
(`#config`) and a set of components (`#components`). The schema is the
consumer-facing API contract; components describe how to deploy the application.

### File Structure

A module lives in its own CUE package, typically two files:

```
modules/my-app/
├── module.cue      # metadata + #config schema
└── components.cue  # #components definitions
```

Both files share the same package name (e.g. `package myapp`). CUE loads them
together as a single package, so `#config` defined in `module.cue` is directly
accessible in `components.cue`.

### module.cue — Metadata and Schema

```cue
package myapp

import (
    m       "opmodel.dev/core/module@v1"
    schemas "opmodel.dev/schemas@v1"
)

// Apply the Module constraint at package scope.
m.#Module

metadata: {
    modulePath:       "example.com/modules"  // registry path, no version
    name:             "my-app"               // kebab-case, RFC 1123
    version:          "0.1.0"               // semver 2.0
    description:      "A short description"
    defaultNamespace: "default"             // optional deployment default
}

#config: {
    // Your schema here — see "Defining #config" below.
}
```

The `m.#Module` embed at package scope (not inside a named field) is
required. It applies the `#Module` constraint to the entire package, computing
the `metadata.fqn` and `metadata.uuid` from the metadata fields.

### Defining `#config`

`#config` is the OpenAPI-like schema that consumers fill in when creating a
ModuleRelease. Define constraints and defaults here — not concrete values.

```cue
#config: {
    // String with default
    motd: string | *"My Server"

    // Bounded integer with default
    maxPlayers: uint & >0 & <=1000 | *20

    // Boolean with default
    pvp: bool | *true

    // Optional field (absent unless consumer provides it)
    seed?: string

    // Enum
    difficulty: "peaceful" | "easy" | *"normal" | "hard"

    // Nested struct
    server: {
        port: uint & >0 & <=65535 | *25565
        motd: string | *"Welcome"
    }

    // Shared schema type
    image: schemas.#Image & {
        repository: string | *"my-image"
        tag:        string | *"latest"
        digest:     string | *""
    }
}
```

**`#config` must be OpenAPIv3 compliant.** Do not use CUE-specific constructs
(`for`, `if`, comprehensions) inside `#config` — the schema must be expressible
as a plain OpenAPI schema for tooling compatibility.

#### Mutual Exclusion with `matchN`

Use `matchN` to model mutually exclusive options:

```cue
#config: {
    // At most one server type
    vanilla?: {}
    paper?:   {}
    forge?:   { version: string }

    matchN(<=1, [{vanilla!: _}, {paper!: _}, {forge!: _}])
}
```

`matchN(1, [...])` requires exactly one. `matchN(<=1, [...])` allows zero or
one. The constraint is enforced at CUE evaluation time — no extra code needed.

#### Declaring Secrets in `#config`

For any sensitive value, use `schemas.#Secret` instead of `string`. This
unlocks automatic secret discovery, K8s Secret emission, and consumer choice of
fulfillment variant.

```cue
#config: {
    rcon: {
        password: schemas.#Secret & {
            $secretName: "server-secrets"  // groups into this K8s Secret
            $dataKey:    "rcon-password"   // key within the K8s Secret
        }
    }
}
```

- `$secretName` — the name of the K8s `Secret` resource OPM will create.
  Multiple `#Secret` fields sharing the same `$secretName` are grouped into one
  K8s Secret resource.
- `$dataKey` — the key within that Secret's `data` map.
- `$description` — optional human-readable description.

**Module authors set `$secretName` and `$dataKey`.** Consumers never touch them
— routing metadata flows through CUE unification automatically.

The `$opm: "secret"` discriminator (present on all `#Secret` variants) enables
automatic discovery by the `_autoSecrets` pipeline at release time. See
[Secret Auto-Discovery](#secret-auto-discovery) below.

### components.cue — Component Definitions

Components describe the deployable units. They reference `#config` directly —
those references remain unresolved at module definition time and are resolved
when a ModuleRelease unifies the module with concrete values.

```cue
package myapp

import (
    resources_workload "opmodel.dev/resources/workload@v1"
    traits_network     "opmodel.dev/traits/network@v1"
    traits_workload    "opmodel.dev/traits/workload@v1"
    traits_security    "opmodel.dev/traits/security@v1"
)

#components: {
    server: {
        // Compose by embedding resource and trait mixins.
        resources_workload.#Container
        traits_workload.#Scaling
        traits_network.#Expose
        traits_security.#SecurityContext

        // Required: workload type label (stateless | stateful | daemon | task | scheduled-task)
        metadata: labels: "core.opmodel.dev/workload-type": "stateless"

        spec: {
            scaling: count: 1

            container: {
                name:  "server"
                image: #config.image  // resolved at release time

                env: {
                    // Literal value
                    MODE: {
                        name:  "MODE"
                        value: "production"
                    }

                    // Value from config (string interpolation for non-strings)
                    MAX_PLAYERS: {
                        name:  "MAX_PLAYERS"
                        value: "\(#config.maxPlayers)"
                    }

                    // Value from config (string field — no interpolation needed)
                    MOTD: {
                        name:  "MOTD"
                        value: #config.motd
                    }

                    // Secret reference — OPM wires a secretKeyRef automatically
                    DB_PASSWORD: {
                        name: "DB_PASSWORD"
                        from: #config.db.password  // #Secret field
                    }
                }
            }

            // Expose the service
            expose: {
                ports: http: {
                    targetPort: 8080
                    protocol:   "TCP"
                }
            }
        }
    }
}
```

#### Composing Resources and Traits

Resources and traits each export a `#Xxx` mixin type that pre-wires itself into
`#Component`. Embed them directly inside the component:

```cue
server: {
    resources_workload.#Container      // adds spec.container
    resources_storage.#Volumes         // adds spec.volumes
    traits_workload.#Scaling           // adds spec.scaling
    traits_workload.#SidecarContainers // adds spec.sidecars
    traits_workload.#RestartPolicy     // adds spec.restartPolicy
    traits_workload.#UpdateStrategy    // adds spec.updateStrategy
    traits_workload.#GracefulShutdown  // adds spec.gracefulShutdown
    traits_network.#Expose             // adds spec.expose
    traits_security.#SecurityContext   // adds spec.securityContext
    ...
}
```

Each mixin adds its own `spec` sub-field. The `#Component` base type merges all
of them into a single closed `spec` struct via `_allFields` + `close({...})`.

#### Conditional Fields

Use CUE's `if` guard to include fields conditionally based on `#config`:

```cue
// Include sidecar only when backup is enabled
if #config.backup.enabled {
    traits_workload.#SidecarContainers
}

// Inject optional env var only when the config field is present
if #config.server.ops != _|_ {
    OPS: {
        name:  "OPS"
        value: strings.Join(#config.server.ops, ",")
    }
}

// CUE-idiomatic: check for bottom (_|_) to test field presence
if #config.seed != _|_ {
    SEED: { name: "SEED", value: #config.seed }
}
```

Use `!= _|_` (not bottom) to test if an optional field was provided. Use
`== _|_` to provide a fallback when a field is absent.

#### Env Var Patterns

```cue
env: {
    // 1. Literal constant
    ENV: { name: "ENV", value: "production" }

    // 2. Direct string field from config
    MOTD: { name: "MOTD", value: #config.motd }

    // 3. Non-string field — must interpolate to string
    MAX_PLAYERS: { name: "MAX_PLAYERS", value: "\(#config.maxPlayers)" }
    EULA:        { name: "EULA",        value: "\(#config.eula)" }

    // 4. Secret — use `from:` instead of `value:`
    // OPM automatically wires a secretKeyRef using the #Secret routing metadata.
    DB_PASSWORD: { name: "DB_PASSWORD", from: #config.db.password }
}
```

The `from:` field takes a `#Secret`-typed config field. At release time,
OPM wires the correct `secretKeyRef` (or equivalent) based on the resolved
secret variant.

### Secret Auto-Discovery

When a ModuleRelease is rendered, OPM automatically:

1. Walks `#config` (up to 3 levels deep) looking for fields with `$opm: "secret"`
2. Groups discovered secrets by `$secretName` / `$dataKey`
3. Injects an `opm-secrets` component with those grouped secrets
4. The `SecretTransformer` emits a K8s `Secret` resource for each group

**Module authors do not wire secrets manually.** Declaring `schemas.#Secret` on a
config field is sufficient.

---

## 2. Writing a ModuleRelease

A ModuleRelease binds a Module to concrete values and a target namespace.

### File Structure

```
releases/my-app/
├── release.cue   # binding: which module, name, namespace
└── values.cue    # concrete values satisfying #config
```

### release.cue

```cue
package myapp

import (
    mr  "opmodel.dev/core/modulerelease@v1"
    app "opmodel.dev/examples/modules/my-app@v1"
)

// Apply the ModuleRelease constraint at package scope.
mr.#ModuleRelease

metadata: {
    name:      "my-app-production"
    namespace: "production"
}

// Reference the module to deploy.
#module: app
```

`#module` is a hidden field (not emitted in output). The `metadata.name` and
`metadata.namespace` together uniquely identify this release — a UUID is
computed from these.

### values.cue

```cue
package myapp

// Concrete values — must satisfy the module's #config schema.
values: {
    motd:       "Production Server"
    maxPlayers: 100
    pvp:        false

    image: {
        repository: "my-image"
        tag:        "v2.1.0"
        digest:     ""
    }

    // Secret — choose a fulfillment variant:

    // Option A: Literal (OPM creates a K8s Secret)
    db: password: value: "my-db-password"

    // Option B: Reference an existing K8s Secret (OPM emits no Secret resource)
    db: password: secretName: "existing-db-creds", remoteKey: "password"

    // Option C: External secret store via ESO (OPM emits an ExternalSecret CR)
    db: password: externalPath: "/prod/db", remoteKey: "password"
}
```

Only one secret variant is needed for each `#Secret` field. CUE unification
resolves the disjunction when `values` is merged with `#config`.

### Multiple Values Profiles

A single release directory can contain multiple values files for different
deployment configurations:

```
releases/minecraft/
├── release.cue
├── values.cue              # default (Paper, minimal resources)
├── values_forge.cue        # Forge modpack profile
├── values_paper_restic.cue # Paper + Restic backup profile
└── values_fabric_modrinth.cue
```

The OPM loader accepts an explicit `--values` flag to select which file to use,
defaulting to `values.cue`. Each values file must be in the same package and
must not conflict with other files loaded alongside it.

### What Happens at Render Time

When OPM renders a ModuleRelease:

1. `unifiedModule = #module & {#config: values}` — merges schema with values
2. `_autoSecrets` walks `unifiedModule.#config` discovering `#Secret` fields
3. If secrets found: injects `opm-secrets` component with grouped secret data
4. `components` map is finalised and passed to the provider's transformers

---

## 3. Writing a Bundle

A Bundle is a curated composition of modules. Platform Operators write Bundles
to expose a simplified config surface that hides per-module complexity.

### File Structure

```
bundles/my-stack/
└── bundle.cue   # all bundle definition in one file
```

### bundle.cue

```cue
package mystack

import (
    bundle  "opmodel.dev/core/bundle@v1"
    schemas "opmodel.dev/schemas@v1"
    appA    "opmodel.dev/examples/modules/app-a@v1"
    appB    "opmodel.dev/examples/modules/app-b@v1"
)

// Apply the Bundle constraint at package scope.
bundle.#Bundle

metadata: {
    modulePath:  "example.com/bundles"
    name:        "my-stack"
    version:     "v1"
    description: "App A + App B bundled for easy deployment"
}
```

### Defining `#config` — The Bundle-Level Schema

The bundle `#config` is the consumer-facing surface. Keep it simpler than the
individual modules:

```cue
// Use the C= alias — see "The C= alias pitfall" in the CUE Pitfalls section.
C=#config: {
    namespace:  string | *"my-stack"
    maxPlayers: uint & >0 & <=10000 | *20
    motd:       string | *"A Game Server"

    // Secret passthrough — consumer chooses fulfillment variant
    dbPassword: schemas.#Secret & appA.#config.db.password
}
```

The `C=` alias is required. Inside `#instances` blocks, the name `#config`
would resolve to the *module's* `#config` (nearest scope), not the bundle's.
The `C=` alias at package scope avoids this collision.

### Defining `#instances`

Each entry in `#instances` is a named module deployment with explicit values
wiring:

```cue
#instances: {
    frontend: {
        module:           appA
        metadata: namespace: C.namespace
        values: {
            // Wire from bundle config
            maxPlayers: C.maxPlayers
            motd:       C.motd
            db: password: C.dbPassword
        }
    }

    backend: {
        module:           appB
        metadata: namespace: C.namespace
        values: {
            // Hardcoded value — consumer cannot override
            port: 8080
            // Wired from bundle config
            maxPlayers: C.maxPlayers
        }
    }
}
```

#### The Three Wiring Patterns

```cue
// 1. Hardcode — value is fixed, consumer cannot change it
values: port: 8080

// 2. Wire — consumer sets the bundle-level field, it maps into the module
values: maxPlayers: C.maxPlayers

// 3. Passthrough — expose the full module schema to the consumer
//    (rarely needed; use when the bundle shouldn't restrict the module's surface)
values: C.serverConfig   // where serverConfig: appA.#config.server (full sub-tree)
```

#### Secret Passthrough

To expose a module's `#Secret` field at the bundle level, embed the module's
routing metadata using CUE unification:

```cue
C=#config: {
    // Inherits $secretName and $dataKey from appA's declaration.
    // Consumer provides the fulfillment variant; routing metadata comes from the module.
    dbPassword: schemas.#Secret & appA.#config.db.password
}
```

This makes the module the single source of truth for secret routing. The bundle
author never duplicates `$secretName`/`$dataKey` — they flow in from the module
definition via CUE unification.

Then wire it into the module instance values normally:

```cue
values: {
    db: password: C.dbPassword
}
```

---

## 3b. Dynamic Instance Generation

Sometimes a bundle should deploy a variable number of module instances determined
by the consumer at release time — for example, a game server fleet where the
operator names and configures each server independently.

This section documents the **dynamic instance generation** pattern used in the
`gamestack` bundle.

### When to use this pattern

Use dynamic instance generation when:

- The number of instances is not known at bundle authoring time
- Each instance is homogeneous in type (same module) but heterogeneous in config
- Infrastructure modules (router, monitor, etc.) need to auto-discover all instances

### The `#config` map field

Declare a map field in the bundle `#config` for the dynamic instances. Use a
private helper definition (`_#`) to constrain the per-entry shape:

```cue
C=#config: {
    // Consumer adds named entries here; each entry becomes a module instance.
    servers?: [string]: _#serverConfig
}

// Private: per-entry config shape — not directly consumer-facing
_#serverConfig: {
    motd?:      string
    maxPlayers: uint & >0 & <=1000 | *20
    port:       uint & >0 & <=65535 | *25565
    // ... other per-instance fields
}
```

`_#` prefixed definitions are private to the package and not exported. This keeps
the per-entry schema clean without polluting the module's exported API surface.

### Iterating the map in `#instances`

Use a `for` comprehension inside `#instances` to generate one `#BundleInstance`
per map entry. Always apply three rules:

**Rule 1 — `*field | {}` fallback for optional maps**

CUE cannot iterate over `_|_` (bottom), and a plain `field | {}` disjunction
stays unresolved when the field IS present (both branches satisfy the open struct
`{}`). The `*` default marker disambiguates: it picks the concrete value when
present and falls back to `{}` when the field is absent (`_|_`):

```cue
for _name, _cfg in (*C.servers | {}) { ... }  // correct: * resolves the disjunction
for _name, _cfg in (C.servers | {})  { ... }  // wrong: unresolved disjunction error
for _name, _cfg in C.servers         { ... }  // error if servers is absent
```

**Rule 2 — `let`-capture comprehension variables**

CUE comprehension variables are lazy references. If passed directly into a
definition unification, the definition's type constraints dominate and concrete
values may be stripped. Capture with `let` at each loop iteration:

```cue
for _name, _cfg in (C.servers | {}) {
    let _c = _cfg                // <-- capture before crossing the definition boundary
    "\(_name)": {
        module: mc
        values: { maxPlayers: _c.maxPlayers }  // concrete value preserved
    }
}
```

Without `let _c = _cfg`, the transformer pipeline may receive empty `{}` for
struct fields that should contain concrete values.

**Rule 3 — Pre-compute shared `let` bindings outside comprehensions**

`let` bindings declared in a struct scope are visible to all comprehensions in
that struct. Pre-compute any values used across multiple comprehensions once,
at the top of `#instances`:

```cue
#instances: {
    let _ns      = C.namespace    // resolved once, used in every instance
    let _relName = C.releaseName

    for _name, _cfg in (C.servers | {}) {
        let _c = _cfg
        "\(_name)": { metadata: namespace: _ns }
    }

    router: { metadata: namespace: _ns }  // same binding, no duplication
}
```

This also avoids carrying `#NameType` constraints from `C.namespace` (a
constrained type) into string interpolations — the `let` resolves the concrete
value in the current scope.

### Auto-wiring infrastructure modules

The key benefit of the dynamic map is that infrastructure modules (mc-router,
mc-monitor, etc.) can derive their configuration directly from the same map.

**Pattern: build lists via comprehension, then wire**

```cue
#instances: {
    let _domain  = C.domain
    let _relName = C.releaseName
    let _ns      = C.namespace

    // Dynamic server instances
    for _name, _cfg in (C.servers | {}) {
        let _c = _cfg
        "\(_name)": { module: mc, ... }
    }

    // mc-router: one mapping per server, auto-built
    let _mappings = [ for _name, _cfg in (*C.servers | {}) {
        {
            externalHostname: "\(_name).\(_domain)"
            host:             "\(_relName)-\(_name).\(_ns).svc"
            port:             _cfg.port
        }
    }]

    router: {
        module: mcRouter
        values: { router: mappings: _mappings }
    }

    // mc-monitor: one javaServer target per server, auto-built
    let _targets = [ for _name, _cfg in (*C.servers | {}) {
        {
            host: "\(_relName)-\(_name).\(_ns).svc"
            port: _cfg.port
        }
    }]

    monitor: {
        module: mcMonitor
        values: { javaServers: _targets, prometheus: {} }
    }
}
```

### Conditional list elements for optional maps

When you need a list that contains one element only if a condition is true (and
is otherwise empty), use the conditional list element syntax:

```cue
// [{...}] when condition is true, [] otherwise
let _proxyMapping = [
    if C.network != _|_ {
        {
            externalHostname: "\(C.network.hostname).\(_domain)"
            host:             "\(_relName)-proxy.\(_ns).svc"
            port:             25577
        }
    },
]
```

This is the idiomatic way to produce a "maybe one" list in CUE.

### Merging dynamic and static lists

Lists from multiple comprehensions can be concatenated with `+`:

```cue
let _standaloneTargets = [ for _name, _cfg in (C.servers | {}) { ... }]
let _networkTargets    = [ for _name, _cfg in (C.network.servers | {}) { ... }]

monitor: {
    values: { javaServers: _standaloneTargets + _networkTargets }
}
```

`*C.network.servers | {}` safely returns `{}` (iterable empty) when `C.network`
is absent, because an absent optional field evaluates to `_|_` and
`*_|_ | {}` resolves to `{}`.

### The `releaseName` convention

When instances need to discover each other via K8s service DNS, the bundle needs
to know the release name to construct service hostnames like:

```
{releaseName}-{instanceName}.{namespace}.svc
```

The BundleRelease produces ModuleRelease names of the form `{releaseName}-{instanceName}`
(see `core/bundlerelease/bundle_release.cue`). Expose `releaseName` as a required
config field and document that it must match `BundleRelease.metadata.name`:

```cue
C=#config: {
    // Must match the BundleRelease metadata.name exactly.
    // Used to compute K8s service DNS names for cross-instance references.
    releaseName: string
    ...
}
```

### Multiple concurrent modes (optional + required instances)

A bundle can mix dynamic and static instances in the same `#instances` block,
and can support multiple optional modes simultaneously:

```cue
#instances: {
    // Dynamic: standalone servers (optional map)
    for _name, _cfg in (C.servers | {}) { ... }

    // Dynamic + conditional: proxied backend servers (only when network is set)
    if C.network != _|_ {
        for _name, _cfg in C.network.servers { ... }
        proxy: { module: vel, ... }  // static instance inside conditional block
    }

    // Static: always present regardless of mode
    router:  { module: rtr, ... }
    monitor: { module: mon, ... }
}
```

CUE merges comprehensions, conditional blocks, and literal fields in a struct
without ordering constraints. The resulting `#instances` map is the union of all
generated entries.

### Instance name collision

Dynamic comprehensions and static instance names share the same `#instances` map
key space. A consumer using a reserved name (e.g. `servers: { router: {...} }`)
will cause a CUE unification conflict. Document reserved names in your bundle:

```cue
C=#config: {
    // Reserved instance names: router, monitor, proxy
    // Do not use these as server names.
    servers?: [string]: _#serverConfig
}
```

### Gamestack example

The `examples/bundles/gamestack/bundle.cue` demonstrates all of these patterns
in a production-representative scenario: 2 deployment modes, 4 module types, and
auto-wired infrastructure across a variable-size server fleet.

---

## 4. Writing a BundleRelease

A BundleRelease binds a Bundle to concrete values.

### File Structure

```
releases/my-stack/
├── release.cue   # binding: which bundle, name
└── values.cue    # concrete values satisfying #bundle.#config
```

### release.cue

```cue
package mystack

import (
    br    "opmodel.dev/core/bundlerelease@v1"
    stack "opmodel.dev/examples/bundles/my-stack@v1"
)

// Apply the BundleRelease constraint at package scope.
br.#BundleRelease

metadata: {
    name: "my-stack-production"
}

// Reference the bundle to deploy.
#bundle: stack
```

Note: `BundleRelease.metadata` has only `name` — no `namespace`. Each module
instance in the bundle carries its own `namespace` (set in the bundle's
`#instances`).

### values.cue

```cue
package mystack

// Concrete values satisfying the bundle's #config schema.
values: {
    namespace:  "production"
    maxPlayers: 50
    motd:       "Welcome to Production!"

    // Secret fulfillment — same variants as ModuleRelease
    dbPassword: value: "my-secret-password"
}
```

### What Happens at Render Time

When OPM renders a BundleRelease:

1. `unifiedBundle = #bundle & {#config: values}`
2. For each instance in `unifiedBundle.#instances`:
   - A `#ModuleRelease` is constructed with name `"{bundleReleaseName}-{instanceName}"`
   - The instance's wired `values` flow in as the module release values
3. Each module release is rendered independently through the full pipeline

---

## 5. CUE Patterns and Pitfalls

These patterns emerged from building the factory experiment. Understanding them
prevents hours of debugging non-obvious CUE behaviour.

---

### Comprehension Variables Lose Concrete Values in Definition Unification

**This is the most important pitfall in this codebase.**

CUE comprehension variables (`for k, v in expr`) are lazy references. When
passed directly into a definition unification, the definition's type constraint
dominates and concrete field values are lost.

```cue
// WRONG: v is a lazy reference — concrete struct fields may be stripped
for k, v in someMap {
    (k): #SomeDefinition & {field: v}
}

// RIGHT: let-capture forces eager evaluation in the local scope
for k, v in someMap {
    let _v = v
    (k): #SomeDefinition & {field: _v}
}
```

**Symptom:** Fields that should have concrete values (e.g. `data: {"key": "value"}`)
appear empty after `cue export` (e.g. `data: {}`), even though `cue eval -A`
shows the values as present.

**When this applies:**

- The map values are structs with concrete scalar fields
- The definition being unified with has a typed field (e.g. `[string]: #Secret`)
- The output must survive `cue export` / `Syntax(cue.Final())`

**All sites where this pattern is used in the codebase:**

| File | Context |
|------|---------|
| `core/helpers/autosecrets.cue` | `let _d = data` before `#SecretSchema & {data: _d}` |
| `schemas/config.cue` (`#ImmutableName`) | `let _d = data` before `#ContentHash & {data: _d}` |
| `schemas/config.cue` (`#SecretImmutableName`) | `let _d = data` before `#SecretContentHash & {data: _d}` |

See `docs/secret-discovery-concrete-values.md` for a full analysis.

---

### Definition Fields (`#`) Carry Constraints, Not Concrete Values

In CUE, `#`-prefixed fields are *definition fields*. They carry schema
constraints — the type lattice, defaults, and validation rules — but they do
**not** forward concrete values when accessed from outside the scope where
concrete unification occurred.

```cue
let unified = #module & {#config: values}

// WRONG: accessing through a # field loses concrete values
_result: (schemas.#AutoSecrets & {#in: unified.#config}).out

// RIGHT: use a regular (non-#) field to carry concrete values out
// Add `resolvedConfig: #config` as a regular field on #Module, then:
_result: (schemas.#AutoSecrets & {#in: unified.resolvedConfig}).out
```

**However:** In practice, `_autoSecrets` in `#ModuleRelease` accesses
`unifiedModule.#config` and it *does* produce concrete output. The pitfall
only manifests at the *next* step, when the discovered secrets are passed
through another definition boundary in `#OpmSecretsComponent`. This is the
comprehension variable pitfall above — both patterns interact.

**Rule of thumb:** When a helper definition needs to iterate concrete struct
values, pass it through a regular field or `let`-capture at each boundary.

---

### `let` Capture for String Interpolation with Constrained Types

Inside comprehensions, CUE carries type constraints along with values. String
interpolation (`"\(x)"`) requires a *concrete* value, not a constrained type.
Capture the value with `let` **outside** the comprehension:

```cue
// WRONG: #NameType constraints flow into the comprehension, breaking interpolation
for instName, inst in unifiedBundle.#instances {
    (instName): mr.#ModuleRelease & {
        metadata: name: "\(metadata.name)-\(instName)"  // error: not concrete
    }
}

// RIGHT: let-capture outside the comprehension resolves the concrete value first
let _name = metadata.name
for instName, inst in unifiedBundle.#instances {
    (instName): mr.#ModuleRelease & {
        metadata: name: "\(_name)-\(instName)"  // works
    }
}
```

Similarly for any constrained value used in interpolation within a comprehension.

---

### The `C=` Alias for Bundle `#config` Scope Collision

Inside a `#instances` block, the identifier `#config` resolves to the **nearest
enclosing** `#config` — which is the *module's* `#config`, not the bundle's.

```cue
// WRONG: inside #instances, #config resolves to the module's #config
#instances: {
    server: {
        module: mc
        values: {
            maxPlayers: #config.maxPlayers  // resolves to mc.#config.maxPlayers — wrong!
        }
    }
}

// RIGHT: alias at package scope captures the bundle's #config before instances
C=#config: {
    maxPlayers: uint | *20
    ...
}

#instances: {
    server: {
        module: mc
        values: {
            maxPlayers: C.maxPlayers  // unambiguous reference to bundle config
        }
    }
}
```

---

### Guard `!= _|_` Before Forwarding Optional Fields

Passing CUE's bottom value (`_|_`) into a definition unification poisons the
entire result. Always guard optional field forwarding:

```cue
// WRONG: if inst.values is absent, this injects _|_ into #ModuleRelease
values: _inst.values

// RIGHT: only forward when present
if _inst.values != _|_ {
    values: _inst.values
}
```

This applies anywhere you forward a potentially-absent field into a struct that
expects a concrete value.

---

### `*field | {}` Default Fallback for Optional Fields in Comprehensions

CUE cannot `for`-iterate over bottom (`_|_`). When a field is optional, use
`*field | {}` to fall back to an empty struct when the field is absent:

```cue
// WRONG: fails if labels? is absent
let _labelPairs = [for k, v in comp.metadata.labels {...}]

// RIGHT: safe empty-struct fallback
let _labelPairs = [for k, v in (*comp.metadata.labels | {}) {...}]
```

This is the idiomatic CUE pattern for "iterate this map if it exists, otherwise
do nothing."

---

### `#in: {...}` vs `#in: _` — Top Is Not Iterable

When a helper definition iterates its input with `for`, the input constraint
must be `{...}` (open struct), not `_` (top). Top is not a struct and CUE
refuses to range over it:

```cue
// WRONG: _ (top) is not iterable
#MyHelper: {
    X=#in: _
    out: { for k, v in X {...} }  // error: cannot range over _ (incomplete type _)
}

// RIGHT: {...} constrains to open struct — iterable and accepts any struct value
#MyHelper: {
    X=#in: {...}
    out: { for k, v in X {...} }
}
```

The `{...}` constraint serves two roles: it accepts any open struct (including
definition values), and it ensures the value is iterable.

---

### `close()` + `_allFields` for Controlled Spec Composition

`#Component` merges all resource and trait specs into a single closed struct.
The pattern prevents arbitrary extra fields from slipping through:

```cue
_allFields: {
    for _, resource in #resources {
        if resource.spec != _|_ { resource.spec }
    }
    for _, trait in (*#traits | {}) {
        if trait.spec != _|_ { trait.spec }
    }
}

spec: close({ _allFields })
```

`close()` makes the struct exhaustive — any field not declared by a resource or
trait will cause a CUE error. This catches typos and field drift at definition
time rather than at runtime.

Every resource and trait definition also uses `close()` on its own `spec`:

```cue
spec: close({ scaling: schemas.#ScalingSchema })
```

This ensures that when the trait is composed into a component, only the fields
declared in the trait's schema can exist under its key.

---

### The `$opm` Discriminator — Using `(v & {...}) != _|_` as a Struct Test

CUE has no `typeof` operator. To test whether a value is a struct you can
recurse into, unify with `{...}` (open struct) and check for bottom:

```cue
// "Is v a struct?" in CUE
if (v & {...}) != _|_ {
    // v is a struct — safe to iterate
}
```

Scalars (strings, numbers, booleans) and closed structs that don't unify with
`{...}` produce `_|_`, correctly skipping them.

The `$opm` discriminator (`$opm: "secret"`) uses this pattern for discovery:

```cue
// "Is v a #Secret?" — check for the discriminator field
if v.$opm != _|_ {
    // v has $opm set — it's a #Secret
}

// "Is v a non-secret struct we can recurse into?"
if v.$opm == _|_
if (v & {...}) != _|_ {
    // recurse
}
```

---

### Import Alias Workaround for Definition-Only Usage

CUE's import tracker may not register an import as used when it appears only
inside `#`-prefixed definition fields. Creating a package-scope alias forces
the import to be tracked:

```cue
import (
    module "opmodel.dev/core/module@v1"
)

// WRONG: import may be dropped if only used inside # fields
#SomeDefinition: {
    x: module.#Module  // CUE may not see this import as used
}

// RIGHT: declare a package-scope alias to force import tracking
#Module: module.#Module

#SomeDefinition: {
    x: #Module  // now the import is tracked correctly
}
```

---

### Regular Fields for Cross-Package `if`-Guards

Hidden or definition fields can cause `if`-guards to fail when a definition is
unified across package boundaries. Use regular (non-`#`) fields for inputs to
helper definitions that use `if`:

```cue
// WRONG: hidden field input — if-guards may not evaluate correctly cross-package
#NormalizeCPU: {
    #in: string
    out: ...
    if #in == "1000m" { ... }  // may fail cross-package
}

// RIGHT: regular field input
#NormalizeCPU: {
    in: string  // regular field
    out: ...
    if in == "1000m" { ... }  // evaluates correctly
}
```

---

### `#config` Must Be OpenAPIv3 Compliant

The `#config` schema comment in `core/module/module.cue` carries an explicit
constraint:

```cue
// MUST be OpenAPIv3 compliant (no CUE templating - for/if statements)
#config: _
```

This means:

- No `for` comprehensions inside `#config`
- No `if` guards inside `#config`
- No `let` bindings inside `#config`
- No references to external CUE values inside `#config` (except type references from `schemas`)

`matchN` is acceptable — it maps to `oneOf`/`anyOf` in OpenAPI. Disjunctions
(`A | B`) map to `oneOf`. Optional fields (`field?:`) map to non-required
properties. Default values (`*"x" | string`) map to `default`.

---

### `debugValues` — Required for Full Evaluation

Every module and bundle **must** declare concrete `debugValues`. Without them,
dependent packages (bundles importing the module, releases importing the bundle)
cannot fully evaluate through the component and transformer chain.

`debugValues` is typed as `_` (top) in `#Module` and `#Bundle`. Provide a value
that satisfies the full `#config` schema with concrete, representative data:

```cue
// module.cue — satisfies every required field and exercises optional ones
debugValues: #config & {
    motd:       "Debug Server"
    maxPlayers: 10
    port:       25565
    // optional fields shown to exercise conditional component branches:
    rcon: { enabled: true, port: 25575 }
}
```

For bundles, use the `C=` alias so the value is constrained by the bundle schema:

```cue
// bundle.cue
debugValues: C & {
    namespace:   "debug"
    releaseName: "debug-stack"
    // ... all required fields
}
```

**Why it matters:** When a `#BundleRelease` iterates `unifiedBundle.#instances`,
each instance's `values` field is typed as `module.#config`. Without concrete
`debugValues` on the imported module, `module.#config` stays as `_` (top) and
`values` evaluates to `_` in the release — meaning the values are never wired
through and the module runs entirely on defaults. This produces no error during
`cue vet` but silently breaks the release.

**Symptom:** `releases.<instance>.values: _` in `cue eval -A` output, and the
rendered components use module defaults instead of your configured values.

---

## Quick Reference

### Module Author Checklist

- [ ] `m.#Module` applied at package scope in `module.cue`
- [ ] `metadata.modulePath`, `name`, `version` filled in
- [ ] `#config` uses only OpenAPIv3-compatible constructs
- [ ] Sensitive fields use `schemas.#Secret` with `$secretName`/`$dataKey`
- [ ] Components embed resource and trait mixins (not declared from scratch)
- [ ] Workload type label set: `"core.opmodel.dev/workload-type": "stateful" | ...`
- [ ] Optional config fields guarded with `!= _|_` in component conditionals
- [ ] Secret env vars use `from:` not `value:`
- [ ] `debugValues: #config & { ... }` with concrete values for all required fields

### Platform Operator (Bundle) Checklist

- [ ] `bundle.#Bundle` applied at package scope
- [ ] Bundle `#config` uses `C=#config` alias (not bare `#config`)
- [ ] Secret passthrough uses `schemas.#Secret & module.#config.field`
- [ ] Each `#instances` entry has `module:`, `metadata.namespace`, and `values:`
- [ ] Values wiring is explicit — no implicit inheritance between instances
- [ ] `debugValues: C & { ... }` with concrete values for all required fields

### End-user (Release) Checklist

- [ ] `mr.#ModuleRelease` or `br.#BundleRelease` at package scope
- [ ] `metadata.name` and `metadata.namespace` set (ModuleRelease)
- [ ] `metadata.name` set (BundleRelease — namespace comes from bundle)
- [ ] `#module` or `#bundle` set to the imported package
- [ ] All required `#config` fields satisfied in `values.cue`
- [ ] Secret values use one of: `value:`, `secretName:+remoteKey:`, or `externalPath:+remoteKey:`
