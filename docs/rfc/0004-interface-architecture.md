# RFC-0004: Interface Architecture

| Field       | Value                                                                      |
|-------------|----------------------------------------------------------------------------|
| **Status**  | Draft                                                                      |
| **Created** | 2026-02-06                                                                 |
| **Authors** | OPM Contributors                                                           |
| **Related** | RFC-0002 (Sensitive Data Model)                                            |

## Summary

Introduce **Interfaces** as a new first-class definition type in the Open Platform Model. Interfaces add a `provides`/`requires` model that allows module authors to declare what their components offer and depend on using well-known, typed contracts. The platform is responsible for fulfilling these contracts at deployment time. This transforms OPM from a deployment configuration system into an application description language — one where service communication, data dependencies, and infrastructure requirements are expressed as typed contracts rather than configuration details.

## Motivation

### The Problem

Today, OPM components are isolated islands. A module author defines a web service with a container, some traits, and maybe an Expose/Route for networking. But the critical question — **what does this component talk to, and what talks to it?** — is answered outside OPM, in ad-hoc configuration, environment variables, and tribal knowledge.

Consider a typical microservice:

```text
┌─────────────────────────────────────────────────────────────────┐
│  user-service                                                   │
│                                                                 │
│  What OPM knows today:                                          │
│    - Container image and resource sizing                        │
│    - Scaling configuration                                      │
│    - Health checks                                              │
│    - It's exposed on port 8080                                  │
│                                                                 │
│  What OPM does NOT know:                                        │
│    - It needs a PostgreSQL database                             │
│    - It needs a Redis cache                                     │
│    - It produces events to a Kafka topic                        │
│    - It provides a gRPC API consumed by 3 other services        │
│    - The connection strings for all of the above                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Without this information, the platform cannot:

- Validate that all dependencies are satisfied before deployment
- Auto-wire connections between components
- Provision managed services (DaaS, CaaS) to fulfill requirements
- Build a service dependency graph
- Reason about the application's architecture

### The Opportunity

If OPM knows the **communication contracts** of every component, it becomes more than a deployment tool. It becomes a **language for describing applications** — their structure, their dependencies, their interfaces with the world.

### Why Now

The OPM trait system already models some communication patterns (Expose, Route traits). But these are protocol-specific plumbing, not application-level contracts. As the catalog grows with more traits (HttpRoute, GrpcRoute, TcpRoute), the pattern is clear: what module authors actually want to express is not "create an HTTPRoute with these match rules" but "I provide an HTTP API" and "I need a database."

## Prior Art

### Industry Approaches

| System                 | Model                                     | Key Difference from OPM Interfaces |
|------------------------|-------------------------------------------|--------------------------------------------------------------------------------|
| **Kubernetes Service** | Label selector matching                   | No typed contract. Consumer must know port/protocol by convention. |
| **K8s Gateway API**    | Route resources reference Services        | Protocol-aware routing but no dependency declaration or auto-wiring. |
| **Docker Compose**     | `depends_on` + env vars                   | Startup ordering only. No typed contract. Connection details are manual. |
| **Terraform**          | Provider model + outputs/inputs           | Similar concept (outputs = provides, inputs = requires). But HCL, not a type system. No runtime portability. |
| **Crossplane**         | Claims + Compositions                     | Very similar model. Claims ≈ requires, Compositions ≈ fulfillment. But Crossplane is K8s-only and resource-centric, not application-centric. |
| **Dapr**               | Building blocks (state, pubsub, bindings) | Capability-based (closer to Path C / Capabilities). Runtime sidecars, not compile-time contracts. |
| **Score**              | Workload spec with resources              | Has `resources` as abstract dependencies. Similar to requires. But no well-known typed shapes. |
| **Acorn**              | Services + secrets linking                | Service discovery with secret injection. Closer to traditional injection than typed contracts. |
| **Radius**             | Recipes + connections                     | Very similar philosophy. Recipes ≈ platform fulfillment. Connections ≈ requires. Radius is Azure-centric. |

OPM's differentiator: **compile-time type safety via CUE + provider-agnostic fulfillment + well-known typed interface catalog**.

### What OPM Already Models

The trait system (Expose, HttpRoute, GrpcRoute, TcpRoute) covers protocol-level networking. These are explicit plumbing — the module author controls every rule and port. Interfaces operate at a higher level: the module author declares intent ("I provide an HTTP API", "I need a database") and the platform decides how to fulfill it. The two systems are peers with independent rendering pipelines, not layers.

## Design

### Core Concept

**An Interface is a typed contract that describes a communication endpoint.**

- Module authors declare which interfaces their components `provide` (offer to
  others) and `require` (depend on).
- OPM publishes a catalog of **well-known interfaces** — standard types like
  `#HttpServer`, `#Postgres`, `#Redis`, `#KafkaTopic`.
- Because interfaces are well-known, their **shape is known at author time**.
  Module authors can reference interface fields directly in their definitions   (e.g., `requires.db.host`).
- The **platform fulfills** `requires` at deployment time — by wiring to another
  component, provisioning a managed service, or binding to an external endpoint.
- The **module definition is unchanged** regardless of how the platform fulfills
  the requirement.

This model is analogous to interfaces in programming languages: you code against the interface, and the runtime provides the implementation.

### System Overview

```text
┌─────────────────────────────────────────────────────────────────────┐
│                       OPM ARCHITECTURE                              │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  INTERFACE CATALOG (well-known types)                         │  │
│  │                                                               │  │
│  │  Network:  #HttpServer  #GrpcServer  #TcpServer  #UdpServer   │  │
│  │  Data:     #Postgres    #Mysql    #Redis    #Mongodb   #S3    │  │
│  │  Messaging:#KafkaTopic  #NatsStream  #Amqp                    │  │
│  │  Identity: #OidcProvider                                      │  │
│  │  ...extensible by platform operators...                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│         │                                        │                  │
│         │ provides                               │ requires         │
│         ▼                                        ▼                  │
│  ┌──────────────────┐                    ┌──────────────────┐       │
│  │   Component A    │                    │   Component B    │       │
│  │                  │                    │                  │       │
│  │  provides:       │                    │  provides:       │       │
│  │    api: #Http    │◄───────────────────│    ...           │       │
│  │                  │    requires:       │                  │       │
│  │  requires:       │      api: #Http    │  requires:       │       │
│  │    db: #Postgres │                    │    db: #Postgres │       │
│  └────────┬─────────┘                    └────────┬─────────┘       │
│           │                                       │                 │
│           │ requires                              │ requires        │
│           ▼                                       ▼                 │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                    PLATFORM FULFILLMENT                      │   │
│  │                                                              │   │
│  │  Option A: Another component in scope provides #Postgres     │   │
│  │  Option B: Platform provisions managed DB (DaaS)             │   │
│  │  Option C: Platform binds to external service                │   │
│  │                                                              │   │
│  │  In all cases → concrete values injected into interface      │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Data Flow

```text
┌───────────────┐      ┌───────────────┐      ┌───────────────┐      ┌───────────────┐
│   Author      │      │   Platform    │      │   Render      │      │   Deploy      │
│   Time        │────▶│   Binding     │────▶│   Pipeline    │────▶│   Time        │
│               │      │               │      │               │      │               │
│ Define        │      │ Resolve       │      │ Each path has │      │ K8s resources │
│ provides &    │      │ requires to   │      │ its own       │      │ are created   │
│ requires with │      │ concrete      │      │ rendering:    │      │ with concrete │
│ well-known    │      │ providers     │      │ traits →      │      │ values        │
│ types         │      │ (in-scope,    │      │   transformers│      │               │
│               │      │  DaaS, or     │      │ interfaces →  │      │               │
│               │      │  external)    │      │   resolvers   │      │               │
└───────────────┘      └───────────────┘      └───────────────┘      └───────────────┘
```

### The Interface Definition Type

#### Position in the Definition Type System

Interface joins the existing definition types as a new first-class concept:

| Type | Question It Answers | Level |
|------|---------------------|-------|
| **Resource** | "What exists?" | Component |
| **Trait** | "How does it behave?" | Component |
| **PolicyRule** | "What must be true?" | Policy |
| **Blueprint** | "What is the pattern?" | Component |
| **Interface** | "What does it communicate?" | Component |
| **Lifecycle** | "What happens on transitions?" | Component/Module |
| **Status** | "What is computed state?" | Module |
| **Test** | "Does the lifecycle work?" | Separate artifact |

#### What Interface Infers

- "This component **communicates** via this contract"
- "This contract has a **known shape** with typed fields"
- "The **platform** is responsible for fulfilling required interfaces"
- "The module author can **reference** interface fields at definition time"

#### When to Use Interface

Ask yourself:

- Does this component communicate with other services or infrastructure?
- Is the communication pattern standardized (HTTP, gRPC, database protocol)?
- Should the platform be able to provision or wire this dependency?
- Do you want type-safe references to connection details?

#### Core Definition

```cue
#Interface: {
    apiVersion: "opmodel.dev/core/v0"
    kind:       "Interface"

    metadata: {
        apiVersion!:  #APIVersionType
        name!:        #NameType
        _definitionName: (#KebabToPascal & {"in": name}).out
        fqn: #FQNType & "\(apiVersion)#\(_definitionName)"

        description?: string
        labels?:      #LabelsAnnotationsType
        annotations?: #LabelsAnnotationsType
    }

    // The contract — typed fields this interface exposes.
    // When used in `provides`, the module author fills these with concrete values.
    // When used in `requires`, the platform fills these at deployment time.
    #shape!: {...}

    // Sensible defaults for the shape fields.
    #defaults: #shape
}
```

#### Key Design Decision: Shape as Contract

The `#shape` field is what makes interfaces powerful. It defines a **typed contract** — a set of fields with types and constraints. For example, the Postgres interface shape includes `host`, `port`, `dbName`, `username`, `password`. These fields are:

- **Known at author time** — the module author can reference `requires.db.host`
  in their container env vars.
- **Typed by CUE** — invalid references (e.g., `requires.db.hostname`) fail at
  validation time.
- **Concrete at deploy time** — the platform fills in actual values when
  fulfilling the interface.

### Well-Known Interface Library

OPM provides a catalog of standard interface types. These are published as CUE definitions in the `interfaces` module, organized by category.

#### Network Interfaces

```cue
// interfaces/network/http_server.cue
#HttpServerInterface: #Interface & {
    metadata: {
        apiVersion:  "opmodel.dev/interfaces/network@v0"
        name:        "http-server"
        description: "An HTTP server endpoint"
    }

    #shape: {
        port!:       uint & >=1 & <=65535
        paths?: [...{
            path!:     string
            pathType?: "Prefix" | "Exact" | *"Prefix"
        }]
        hostnames?:  [...string]
        visibility:  "public" | "internal" | *"internal"
    }
}

// interfaces/network/grpc_server.cue
#GrpcServerInterface: #Interface & {
    metadata: {
        apiVersion:  "opmodel.dev/interfaces/network@v0"
        name:        "grpc-server"
        description: "A gRPC server endpoint"
    }

    #shape: {
        port!:       uint & >=1 & <=65535
        services?:   [...string]   // fully-qualified gRPC service names
        hostnames?:  [...string]
        visibility:  "public" | "internal" | *"internal"
    }
}

// Similarly: #TcpServerInterface, #UdpServerInterface, #WebSocketServerInterface
```

#### Data Interfaces

```cue
// interfaces/data/postgres.cue
#PostgresInterface: #Interface & {
    metadata: {
        apiVersion:  "opmodel.dev/interfaces/data@v0"
        name:        "postgres"
        description: "A PostgreSQL database connection"
    }

    #shape: {
        host!:     string
        port:      uint | *5432
        dbName!:   string
        username!: string
        password!: string
        sslMode?:  "disable" | "require" | "verify-ca" | "verify-full" | *"disable"
    }
}

// interfaces/data/redis.cue
#RedisInterface: #Interface & {
    metadata: {
        apiVersion:  "opmodel.dev/interfaces/data@v0"
        name:        "redis"
        description: "A Redis connection"
    }

    #shape: {
        host!:     string
        port:      uint | *6379
        password?: string
        db:        uint | *0
    }
}

// Similarly: #MysqlInterface, #MongodbInterface, #S3Interface
```

#### Messaging Interfaces

```cue
// interfaces/messaging/kafka_topic.cue
#KafkaTopicInterface: #Interface & {
    metadata: {
        apiVersion:  "opmodel.dev/interfaces/messaging@v0"
        name:        "kafka-topic"
        description: "A Kafka topic for producing or consuming messages"
    }

    #shape: {
        brokers!:  [...string]
        topic!:    string
        groupId?:  string
        auth?: {
            mechanism?: "PLAIN" | "SCRAM-SHA-256" | "SCRAM-SHA-512"
            username?:  string
            password?:  string
        }
    }
}

// Similarly: #NatsStreamInterface, #AmqpInterface
```

#### Extensibility

Platform operators can define custom interfaces for their organization:

```cue
// Custom interface defined by a platform team
#InternalAuthInterface: #Interface & {
    metadata: {
        apiVersion:  "acme.com/interfaces/identity@v0"
        name:        "internal-auth"
        description: "ACME internal authentication service"
    }

    #shape: {
        endpoint!:    string
        clientId!:    string
        clientSecret!: string
        realm:        string | *"acme"
    }
}
```

### Provides and Requires on Components

#### Component-Level Fields

`provides` and `requires` are first-class fields on `#Component`:

```cue
#Component: {
    // ... existing fields (metadata, #resources, #traits, #policies, spec) ...

    // Interfaces this component implements / offers to others
    provides?: [string]: #Interface

    // Interfaces this component depends on
    requires?: [string]: #Interface
}
```

Both are maps keyed by a **local name** — an alias the module author uses to reference the interface within the component (e.g., `"db"`, `"cache"`, `"api"`).

#### Provides: What a Component Offers

When a component declares `provides`, it makes an interface available for other components (or external consumers) to depend on. The module author fills in the shape with concrete configuration values:

```cue
userService: #Component & {
    provides: {
        "user-api": interfaces.#HttpServer & {
            port:       8080
            paths:      [{ path: "/api/v1/users" }]
            visibility: "public"
        }
        "user-grpc": interfaces.#GrpcServer & {
            port:     9090
            services: ["user.v1.UserService"]
        }
    }
}
```

#### Requires: What a Component Needs

When a component declares `requires`, it states a dependency on an interface that the platform must fulfill. The module author references the shape's fields but does not provide values — those come from the platform:

```cue
userService: #Component & {
    requires: {
        "db":    interfaces.#Postgres
        "cache": interfaces.#Redis
    }

    spec: container: env: {
        // These references are valid because the shape is known
        DATABASE_HOST: { name: "DATABASE_HOST", value: requires.db.host }
        DATABASE_PORT: { name: "DATABASE_PORT", value: requires.db.port }
        DATABASE_NAME: { name: "DATABASE_NAME", value: requires.db.dbName }
        REDIS_URL:     { name: "REDIS_URL",     value: requires.cache.host }
    }
}
```

#### The "No Injection" Model

Traditional platforms inject connection details via opaque mechanisms (environment variables from Secrets, mounted config files). OPM takes a fundamentally different approach:

**Because interfaces are well-known, the module author references their fields
directly in the definition.** The interface shape acts as a typed API between the module and the platform. The platform's job is to make those references resolve to concrete values.

```text
TRADITIONAL                           OPM
──────────                            ───

Module author:                        Module author:
  "Put DB_HOST somewhere              "I require #Postgres.
   I can read it"                      I reference requires.db.host"

Platform:                             Platform:
  "Here's a Secret with               "Here's the #Postgres contract
   DB_HOST=pg.svc"                     fulfilled: {host: 'pg.svc', ...}"

Problem:                              Advantage:
  No type safety.                      CUE validates the reference.
  No validation that                   Shape mismatch caught at
  DB_HOST exists or is                 definition time.
  the right type.                      Platform can verify fulfillment.
```

#### When NOT to Use Interfaces: Direct Component References

Interfaces solve a specific problem: communicating with something **the module author does not control**. When a module author brings their own database component, they already know its name and ports. Using a typed interface contract for this adds ceremony without benefit.

**The rule is simple: use interfaces for external dependencies, use direct
references for internal ones.**

```text
┌─────────────────────────────────────────────────────────────────────┐
│  WITHIN A MODULE: Direct references                                 │
│                                                                     │
│  The module author controls both components. They know the name,    │
│  the port, the configuration. Just reference it directly.           │
│                                                                     │
│  ┌─ Module: my-app ───────────────────────────────────────────┐     │
│  │                                                            │     │
│  │  api-server                    database                    │     │
│  │  spec: container: env:         spec: container:            │     │
│  │    DB_HOST: "database"  ◄──────  ports:                    │     │
│  │    DB_PORT: "5432"               postgres: 5432            │     │
│  │                                                            │     │
│  │  No interface needed. The module author knows both sides.  │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                     │
│  ACROSS MODULES / TO PLATFORM: Interfaces                           │
│                                                                     │
│  The module author does NOT control the other side.                 │
│  They don't know who provides it, where it runs, or how it's        │
│  configured. They need a typed contract.                            │
│                                                                     │
│  ┌─ Module: my-app ─────────┐                                       │
│  │                          │     ┌─ ??? ──────────────────────┐    │
│  │  api-server              │     │                            │    │
│  │  requires:               │     │  Could be another module   │    │
│  │    "db": #Postgres  ◄────┼─────│  Could be platform DaaS    │    │
│  │                          │     │  Could be external service │    │
│  └──────────────────────────┘     └────────────────────────────┘    │
│                                                                     │
│  Interface needed. The provider is unknown at author time.          │
└─────────────────────────────────────────────────────────────────────┘
```

The decision matrix:

| Scenario | Approach | Why |
|----------|----------|-----|
| Component talks to sibling in same module | Direct reference (name + port) | Author controls both sides. Name and ports are known constants. |
| Component depends on another module's service | `requires: #GrpcServer` | Author doesn't control the provider. Needs a typed contract. |
| Component depends on platform infrastructure | `requires: #Postgres` | Provider is the platform itself (DaaS, managed service). Needs a contract. |
| Component depends on external service | `requires: #Postgres` | Provider is outside the system entirely. Needs a contract. |

### The Three Paths

OPM offers three independent design patterns for describing component communication and behavior. Each path has its own rendering pipeline and serves different use cases. They are peers, not layers — none compiles down to another.

```text
┌─────────────────────────────────────────────────────────────────────┐
│  PATH C: CAPABILITIES (abstract, infrastructure-like)               │
│                                                                     │
│  For infrastructure components that ARE the interface.              │
│  "I am a data-store"  "I am an event-broker"                        │
│  Platform resolves to infrastructure provisioning.                  │
│                                                                     │
│  Usage: SimpleDatabase blueprint, managed service proxies           │
│  Own rendering pipeline: capability resolvers.                      │
├─────────────────────────────────────────────────────────────────────┤
│  PATH B: INTERFACES (provides/requires, contract-driven)            │
│                                                                     │
│  For application components that CONSUME and PROVIDE interfaces.    │
│  "I provide an HTTP API"   "I require a database connection"        │
│  Platform wires dependencies, generates provider resources.         │
│                                                                     │
│  Usage: Microservices, APIs, workers, gateways                      │
│  Own rendering pipeline: interface resolvers.                       │
├─────────────────────────────────────────────────────────────────────┤
│  PATH A: TRAITS (protocol-specific, explicit)                       │
│                                                                     │
│  Direct control over networking and workload primitives.            │
│  Expose + HttpRoute + GrpcRoute + TcpRoute                          │
│  No abstraction, maximum control.                                   │
│                                                                     │
│  Usage: Fine-grained control, simple cases                          │
│  Own rendering pipeline: trait transformers.                        │
└─────────────────────────────────────────────────────────────────────┘

Each path is independent. They do NOT compile down to each other.
A component uses ONE path for a given concern. Mixing is possible
across concerns (e.g., traits for networking + interfaces for data
dependencies) but a single concern should not span multiple paths.
```

#### Path Independence

Each path has its **own rendering pipeline** that produces provider-specific resources directly:

- **Path A (Traits)**: Trait transformers convert trait specs into K8s resources
  (Services, HTTPRoutes, Deployments, etc.) — the existing pipeline.
- **Path B (Interfaces)**: Interface resolvers fulfill `requires` contracts,
  resolve `provides` declarations, and generate provider resources directly. No   trait intermediary.
- **Path C (Capabilities)**: Capability resolvers provision or bind
  infrastructure and inject connection details.

This independence means:

- Interfaces do not depend on traits. They generate their own output.
- Traits do not know about interfaces. They are self-contained.
- A component using `provides: #HttpServer` does NOT implicitly create an Expose
  or HttpRoute trait.
- Each path can evolve independently without cascading changes.

#### When to Use Each Path

| Scenario | Recommended Path |
|----------|-----------------|
| Simple service with one port, no dependencies | Path A (Expose trait) |
| Service with HTTP routing rules | Path A (Expose + HttpRoute traits) |
| Microservice with database and cache dependencies | Path B (Interfaces) |
| Complex app with multiple APIs, dependencies, events | Path B (Interfaces) |
| Database component (IS the infrastructure) | Path C (Capabilities) |
| Managed service proxy | Path C (Capabilities) |
| Fine-grained networking control alongside interface dependencies | Path A for networking + Path B for data deps |

#### Choosing Between Traits and Interfaces

Components should use the path that best fits their needs. Traits and interfaces address different concerns:

```cue
// Path A: Traits — explicit protocol-level control
myService: #Component & {
    workload_traits.#Expose & { spec: expose: { type: "ClusterIP", ports: { http: { targetPort: 8080 }}}}
    network_traits.#HttpRoute & { spec: httpRoute: { rules: [{ ... }] }}
}
```

```cue
// Path B: Interfaces — contract-driven, platform-resolved
myService: #Component & {
    provides: {
        "api": #HttpServer & { port: 8080, paths: [{ path: "/api" }], visibility: "public" }
    }
}
```

These are **different approaches**, not equivalent representations. Path A gives direct control over the networking primitives. Path B declares intent and lets the platform decide how to fulfill it. The output may differ depending on the platform's interface resolver implementation.

### Platform Fulfillment

The platform is responsible for **fulfilling** all `requires` declarations. Fulfillment means providing concrete values for every field in the interface's `#shape`. The platform has multiple strategies:

#### Strategy 1: Cross-Module Matching

When a component in one module `requires` an interface that a component in another module `provides`, and both are deployed in the same Policy, the platform can auto-wire them. This is the primary use case for interfaces — connecting modules that don't know about each other.

```text
┌─ Policy: production ─────────────────────────────────────────────┐
│                                                                  │
│  ┌─ Module: app ───────────┐   ┌─ Module: data-tier ──────────┐  │
│  │                         │   │                              │  │
│  │  user-service           │   │  postgres-primary            │  │
│  │  ┌───────────────────┐  │   │  ┌───────────────────────┐   │  │
│  │  │ requires:         │  │   │  │ provides:             │   │  │
│  │  │   "db": #Postgres │◄─┼───┼──│  "primary": #Postgres │   │  │
│  │  └───────────────────┘  │   │  │  {                    │   │  │
│  │                         │   │  │    host: "pg.svc"     │   │  │
│  └─────────────────────────┘   │  │    port: 5432         │   │  │
│                                │  │    ...                │   │  │
│  Neither module knows the      │  │  }                    │   │  │
│  other. The platform matches   │  └───────────────────────┘   │  │
│  requires to provides across   └──────────────────────────────┘  │
│  module boundaries.                                              │
└──────────────────────────────────────────────────────────────────┘
```

The platform sees that `user-service` requires `#Postgres` and `postgres-primary` provides `#Postgres`. It unifies the provider's concrete values into the consumer's `requires.db` field.

Note: this is specifically for **cross-module** dependencies. Within a single module, the author controls both components and should use direct references (component name + port) instead of interfaces.

#### Strategy 2: Platform-Provisioned Service (DaaS, CaaS, etc.)

When no in-scope component provides the required interface, the platform can provision infrastructure:

```text
┌─ Policy: production ──────────────────────────────────────────────┐
│                                                                   │
│  user-service                                                     │
│  ┌─────────────────────┐                                          │
│  │ requires:           │                                          │
│  │   "db": #Postgres   │◄────── No provider in scope              │
│  └─────────────────────┘                                          │
│           │                                                       │
│           │ Platform: "I can fulfill #Postgres"                   │
│           ▼                                                       │
│  ┌──────────────────────────────────────────┐                     │
│  │  Platform provisions:                    │                     │
│  │    AWS RDS instance                      │                     │
│  │    OR Google Cloud SQL                   │                     │
│  │    OR self-hosted StatefulSet            │                     │
│  │                                          │                     │
│  │  Fulfills with:                          │                     │
│  │    host: "prod-db.rds.amazonaws.com"     │                     │
│  │    port: 5432                            │                     │
│  │    dbName: "users"                       │                     │
│  │    username: "app"                       │                     │
│  │    password: <from Secret>               │                     │
│  └──────────────────────────────────────────┘                     │
└───────────────────────────────────────────────────────────────────┘
```

This is the **DaaS (Database as a Service)** model. The platform advertises which interfaces it can fulfill. The module author simply declares `requires: { "db": #Postgres }` and the platform handles the rest.

#### Strategy 3: External Service Binding

The platform binds the requirement to an external, pre-existing service:

```cue
// Platform binding configuration (Policy-level or Bundle-level):

bindings: {
    "user-service": {
        requires: {
            "db": {
                host:     "prod-db.example.com"
                port:     5432
                dbName:   "users"
                username: "readonly"
                password: <from external secret manager>
            }
        }
    }
}
```

#### Fulfillment Validation

At deployment time, the platform MUST validate:

1. **Completeness**: Every `requires` on every component is fulfilled.
2. **Type compatibility**: The fulfilled values match the interface's `#shape`
   constraints.
3. **No dangling references**: Every `requires.X.field` reference in the
   component resolves.

If any validation fails, deployment is blocked. This provides a safety net that catches misconfiguration before anything runs.

### Type Safety and Validation

The interface system provides type safety at three stages, each catching different classes of errors:

```text
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│  Author Time  │     │  Module Time  │     │  Deploy Time  │
│               │     │               │     │               │
│  CUE validates│     │  All requires │     │  All requires │
│  field refs:  │     │  declared:    │     │  fulfilled:   │
│               │     │               │     │               │
│  requires.db  │     │  #Postgres is │     │  host, port,  │
│    .host [x]  │     │  a known      │     │  dbName are   │
│    .hostname  │     │  interface [x]│     │  concrete [x] │
│           [ ] │     │               │     │               │
│    .port [x]  │     │  All provides │     │  Type         │
│    .sslMode   │     │  have concrete│     │  constraints  │
│           [x] │     │  values for   │     │  satisfied [x]│
│               │     │  shape [x]    │     │               │
│  Field exists │     │               │     │  No unfulfilled│
│  on #Postgres │     │               │     │  requires [x] │
│  shape? [x]/  │     │               │     │               │
│         [ ]   │     │               │     │               │
└───────────────┘     └───────────────┘     └───────────────┘

  Catches:             Catches:             Catches:
  Typos, wrong         Missing interface    Missing platform
  field names,         declarations,        bindings, wrong
  type mismatches      unknown interfaces   values, incomplete
                                            provisioning
```

#### CUE Validation Example

```cue
// This is valid — `host` exists on #Postgres.#shape
spec: container: env: {
    DB_HOST: requires.db.host     // [x] string field on #Postgres
}

// This is INVALID — `hostname` does not exist on #Postgres.#shape
spec: container: env: {
    DB_HOST: requires.db.hostname // [ ] CUE error: field not found
}

// This is INVALID — port is uint, not string
spec: container: env: {
    DB_PORT: requires.db.port     // [x] but if used in string context,
                                  //   CUE catches the type mismatch
}
```

### Relationship to Existing Definition Types

```text
┌────────────┬──────────────────────────────────────────────────────────────┐
│ Definition │ Relationship to Interface                                    │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Resource   │ Interfaces describe what a Resource communicates.            │
│            │ A Container resource runs the code; interfaces describe      │
│            │ what that code talks to and offers.                          │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Trait      │ Traits and interfaces are independent paths.                │
│            │ Both can produce provider resources (e.g., K8s Services)    │
│            │ but through separate rendering pipelines.                   │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Policy     │ Policies can constrain interfaces.                           │
│            │ Example: "All provides must have visibility: internal"       │
│            │ or "All requires: #Postgres must use sslMode: verify-ca"     │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Blueprint  │ Blueprints can pre-compose interfaces.                       │
│            │ A StatelessWorkload blueprint could include a default        │
│            │ provides: #HttpServer.                                       │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Policy     │ Policies are where provides/requires are resolved.            │
│            │ The platform matches requires to provides within a Policy.   │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Transformer│ Transformers render traits. Interface resolvers render       │
│            │ interfaces. Each path has its own rendering pipeline.       │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Lifecycle  │ Lifecycle steps can reference interfaces.                    │
│            │ Example: "Run migration on requires.db before upgrade"       │
├────────────┼──────────────────────────────────────────────────────────────┤
│ Status     │ Status can report interface fulfillment state.               │
│            │ Example: healthy: allRequiresFulfilled(requires)             │
└────────────┴──────────────────────────────────────────────────────────────┘
```

## Scenarios

### Scenario A: Module with Internal Database (Direct References) [x]

When a module includes its own database, the author knows both components and references them directly. No interfaces are needed for within-module wiring.

```cue
#BlogModule: #Module & {
    metadata: {
        apiVersion: "acme.com/modules/blog@v0"
        name:       "blog"
        version:    "1.0.0"
    }

    #components: {
        // The API server references the database directly by name and port.
        // No interface needed — the author controls both components.
        "api": #Component & {
            metadata: {
                name: "api"
                labels: { "core.opmodel.dev/workload-type": "stateless" }
            }

            spec: container: {
                name:  "blog-api"
                image: "acme/blog-api:1.0.0"
                ports: { http: { targetPort: 8080 } }
                env: {
                    // Direct references — simple, explicit, no abstraction needed
                    DB_HOST: { name: "DB_HOST", value: "database" }
                    DB_PORT: { name: "DB_PORT", value: "5432" }
                    DB_NAME: { name: "DB_NAME", value: "blog" }
                }
            }
        }

        // The database is a sibling component in the same module.
        "database": #Component & {
            metadata: {
                name: "database"
                labels: { "core.opmodel.dev/workload-type": "stateful" }
            }

            spec: container: {
                name:  "postgres"
                image: "postgres:16"
                ports: { postgres: { targetPort: 5432 } }
            }
        }
    }
}
```

Result: Direct string references. Simple. Explicit. No platform involvement needed. [x]

### Scenario B: Module with External Dependencies (Interfaces) [x]

When a module depends on services it does not control — databases managed by the platform, APIs from other teams, messaging infrastructure — it uses interfaces.

```cue
import (
    interfaces_net  "opmodel.dev/interfaces/network@v0"
    interfaces_data "opmodel.dev/interfaces/data@v0"
    interfaces_msg  "opmodel.dev/interfaces/messaging@v0"
)

#UserServiceModule: #Module & {
    metadata: {
        apiVersion: "acme.com/modules/user@v0"
        name:       "user-service"
        version:    "1.0.0"
    }

    #components: {
        "user-api": #Component & {
            metadata: {
                name: "user-api"
                labels: { "core.opmodel.dev/workload-type": "stateless" }
            }

            // WHAT THIS COMPONENT PROVIDES TO THE OUTSIDE WORLD
            provides: {
                "http-api": interfaces_net.#HttpServer & {
                    port:       8080
                    paths:      [{ path: "/api/v1/users" }]
                    visibility: "public"
                }
                "grpc-api": interfaces_net.#GrpcServer & {
                    port:     9090
                    services: ["user.v1.UserService", "user.v1.AdminService"]
                }
            }

            // WHAT THIS COMPONENT REQUIRES FROM OUTSIDE THE MODULE
            // The module author does NOT control these — the platform fulfills them.
            requires: {
                "db":     interfaces_data.#Postgres
                "cache":  interfaces_data.#Redis
                "events": interfaces_msg.#KafkaTopic
            }

            // SPEC REFERENCES WELL-KNOWN INTERFACE FIELDS
            spec: container: {
                name:  "user-api"
                image: "acme/user-service:1.0.0"
                ports: {
                    http: { targetPort: 8080 }
                    grpc: { targetPort: 9090 }
                }
                env: {
                    // These references are type-safe — CUE validates them
                    DATABASE_HOST:     { name: "DATABASE_HOST",     value: requires.db.host }
                    DATABASE_PORT:     { name: "DATABASE_PORT",     value: "\(requires.db.port)" }
                    DATABASE_NAME:     { name: "DATABASE_NAME",     value: requires.db.dbName }
                    DATABASE_USER:     { name: "DATABASE_USER",     value: requires.db.username }
                    DATABASE_PASSWORD: { name: "DATABASE_PASSWORD", value: requires.db.password }
                    REDIS_HOST:        { name: "REDIS_HOST",        value: requires.cache.host }
                    REDIS_PORT:        { name: "REDIS_PORT",        value: "\(requires.cache.port)" }
                    KAFKA_BROKERS:     { name: "KAFKA_BROKERS",     value: requires.events.brokers[0] }
                    KAFKA_TOPIC:       { name: "KAFKA_TOPIC",       value: requires.events.topic }
                }
            }
        }
    }
}
```

Result: Typed interface references. Platform fills in the values. Could be RDS, Cloud SQL, or another module's component. [x]

### Scenario C: The Contrast [x]

```text
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  Scenario A (Blog):                 Scenario B (User Service):      │
│                                                                     │
│  Module brings its own DB.         Module depends on external DB.   │
│  Author knows both sides.          Author doesn't control the DB.   │
│                                                                     │
│  DB_HOST: "database"               DB_HOST: requires.db.host        │
│  DB_PORT: "5432"                   DB_PORT: requires.db.port        │
│        ↑                                  ↑                         │
│  Direct string reference.          Typed interface reference.       │
│  Simple. Explicit. No platform     Platform fills in the value.     │
│  involvement needed.               Could be RDS, Cloud SQL, or      │
│                                    another module's component.      │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Scenario D: Platform Fulfillment [x]

```cue
policy: #Policy & {
    metadata: { name: "production" }

    #modules: {
        "user-service": #UserServiceModule
    }

    // Platform bindings — fulfilling the requires contracts
    bindings: {
        "user-service": {
            "user-api": {
                requires: {
                    // Platform fulfills #Postgres via DaaS (AWS RDS)
                    "db": {
                        host:     "prod-users-db.rds.amazonaws.com"
                        port:     5432
                        dbName:   "users"
                        username: "app"
                        password: "{{secret:prod/users-db/password}}"
                    }
                    // Platform fulfills #Redis via another module in scope
                    // (auto-wired — a redis module in the same scope provides #Redis)
                    //
                    // Platform fulfills #KafkaTopic via managed Kafka
                    "events": {
                        brokers: ["kafka-1.prod:9092", "kafka-2.prod:9092"]
                        topic:   "user-events"
                    }
                }
            }
        }
    }
}
```

Result: Module author wrote no infrastructure code. Platform provided concrete bindings. CUE validates that all shapes are satisfied. [x]

## Trade-offs

**Advantages:**

- Type-safe dependencies — `requires.db.host` is validated by CUE at definition
  time. Typos and type mismatches caught before deployment.
- Platform portability — module says `requires: #Postgres`, platform decides HOW
  (RDS, Cloud SQL, self-hosted). Module unchanged.
- Dependency graph — platform can build and validate the full service dependency
  graph. Detect cycles, missing dependencies, version conflicts.
- DaaS / managed services — platform can provision infrastructure to fulfill
  interfaces. `requires: #Postgres` leads to the platform spinning up RDS.
- Auto-wiring — when provider and consumer are in the same Policy, platform can
  auto-connect them without manual configuration.
- Documentation as code — `provides` is machine-readable documentation of what a
  service offers. Service catalogs become automatic.
- Incremental adoption — traits remain a fully independent path. Teams can use
  interfaces for new services without changing existing trait-based definitions.
- Extensible — platform operators define custom interfaces for their
  organization's services.
- Contract testing — because interfaces have typed shapes, contract tests can be
  generated automatically.

**Disadvantages:**

- New core concept — adds cognitive load. Developers must learn
  provides/requires in addition to resources/traits.
- Resolution complexity — the platform binding/resolution logic is non-trivial.
  Auto-wiring, DaaS provisioning, and external binding are three different code   paths.
- CUE late-binding challenge — `requires` fields are types (not values) at
  author time. Making CUE resolve these at deploy time requires careful design   of the unification pipeline.
- Interface versioning — as interfaces evolve (e.g., #Postgres adds
  `connectionPoolSize`), backward compatibility must be managed. Breaking shape   changes affect all consumers.
- Over-abstraction risk — for simple services (one container, one port),
  interfaces add ceremony over a simple Expose trait. Path A is the right choice   there.
- Platform burden — platforms must implement fulfillment logic: matching,
  provisioning, binding. This is significant engineering.
- Standard library maintenance — OPM must maintain and evolve the well-known
  interface catalog. Community governance needed.

**Risks:**

```text
┌──────────────────────────────────────────┬──────────┬────────────┬────────────────────────────────┐
│ Risk                                     │ Severity │ Likelihood │ Mitigation                     │
├──────────────────────────────────────────┼──────────┼────────────┼────────────────────────────────┤
│ CUE cannot express late-binding cleanly  │ High     │ Medium     │ Prototype the CUE unification  │
│                                          │          │            │ pipeline before committing.     │
│                                          │          │            │ Spike required.                │
│                                          │          │            │                                │
│ Interface catalog becomes too            │ Medium   │ Medium     │ Start small (10-15 interfaces).│
│ large/ungovernable                       │          │            │ Community governance model.     │
│                                          │          │            │ SemVer on interfaces.          │
│                                          │          │            │                                │
│ Developers avoid interfaces due          │ Medium   │ Low        │ Path A (traits) remains        │
│ to complexity                            │          │            │ available. Good docs.           │
│                                          │          │            │ Blueprint integration hides     │
│                                          │          │            │ complexity.                    │
│                                          │          │            │                                │
│ Platform implementations diverge on      │ High     │ Medium     │ Strict specification of        │
│ fulfillment semantics                    │          │            │ fulfillment contract.           │
│                                          │          │            │ Conformance tests.             │
│                                          │          │            │                                │
│ Performance impact of resolution step    │ Low      │ Low        │ Resolution is a compile-time   │
│                                          │          │            │ step, not runtime. CUE is      │
│                                          │          │            │ fast for this.                 │
└──────────────────────────────────────────┴──────────┴────────────┴────────────────────────────────┘
```

## Open Questions

### Q1: CUE Late-Binding Mechanism

**Question**: How exactly does `requires.db.host` go from `string` (type) to
`"pg.svc.cluster.local"` (value) in the CUE evaluation pipeline?

**Options**:

- A. Platform injects values into `requires` during ModuleRelease rendering
  (CUE unification)
- B. `requires` fields generate a parallel config structure that gets merged
- C. A pre-processing step rewrites `requires.X.field` references to concrete
  value paths

**Impact**: This is the most critical technical question. It determines whether
the interface model works at all in CUE.

**Recommendation**: Spike / proof-of-concept before committing to the
architecture.

### Q2: Interface Versioning Strategy

**Question**: How do interface shapes evolve over time?

**Options**:

- A. SemVer on the interface module (`opmodel.dev/interfaces/data@v0`, `@v1`)
- B. Individual interface versioning (`#Postgres` v1, v2)
- C. Additive-only changes (new optional fields never break existing consumers)

**Recommendation**: Option C with Option A as the major version escape hatch.
New fields are always optional with defaults, so existing consumers are unaffected.

### Q3: Multiple Providers for Same Interface

**Question**: What happens when two components in a Policy both
`provides: #Postgres`?

**Options**:

- A. Ambiguity error — consumer must specify which provider via `ref`
- B. Platform selects based on naming convention or labels
- C. Explicit binding configuration at Policy level

**Recommendation**: Option A with Option C as override. Ambiguity should be an
error, not silently resolved.

### Q4: Circular Dependencies

**Question**: What if Component A requires an interface that Component B
provides, and B requires an interface that A provides?

**Answer**: This is valid and common (mutual communication between services).
The dependency graph must be a DAG for startup ordering, but not for communication. The platform must distinguish between "needs to exist" (startup dependency) and "needs to communicate with" (runtime dependency).

### Q5: Relationship Between provides and Container ports

**Question**: When a component declares
`provides: { "api": #HttpServer & { port: 8080 } }`, must the Container also declare `ports: { http: { targetPort: 8080 } }`?

**Options**:

- A. Yes, both must align (interface doesn't replace Container ports)
- B. Interface generates Container port automatically (less duplication)
- C. Interface validates against Container ports (must exist)

**Recommendation**: This needs design. Duplication is undesirable but implicit
generation is surprising.

### Q6: Scope of the Well-Known Library

**Question**: How many interfaces should OPM ship in v1?

**Recommendation**: Start minimal, grow based on demand:

- **v0**: HttpServer, GrpcServer, TcpServer, Postgres, Redis, Mysql (6
  interfaces)
- **v1**: Add Mongodb, S3, KafkaTopic, NatsStream, Amqp, OidcProvider (~12
  interfaces)
- **Community**: Platform operators publish their own

## Deferred Work

### Incremental Adoption Path

#### Phase 1: Foundation (Current + Near-term)

- Path A traits: Expose, HttpRoute, GrpcRoute, TcpRoute (in progress via
  `add-more-traits` and `add-transformers` changes)
- These are a complete, standalone path for networking — not a prerequisite for
  interfaces

#### Phase 2: Core Interface System

- Add `#Interface` to `core/interface.cue`
- Add `provides` and `requires` fields to `#Component`
- Publish initial well-known interfaces: HttpServer, GrpcServer, TcpServer,
  Postgres, Redis
- Implement Interface Resolver rendering pipeline (independent of trait
  transformers)
- CUE late-binding spike to validate `requires.X.field` pattern

#### Phase 3: Platform Fulfillment

- Define fulfillment contract (how platforms advertise capabilities)
- Implement in-scope auto-wiring (match requires to provides)
- Implement Policy-level binding configuration
- Define DaaS provisioning interface

#### Phase 4: Ecosystem

- Expand well-known interface catalog (Kafka, NATS, S3, MongoDB, etc.)
- Community-contributed interfaces
- Interface conformance testing
- Service catalog generation from provides declarations
- Contract test generation

### Sensitive Data Integration

[RFC-0002](0002-sensitive-data-model.md) introduces `#Secret` as a first-class type for sensitive data. Interface shapes include fields like `#Postgres.password` — currently typed as `string`. When RFC-0002 is implemented, sensitive shape fields will be upgraded to `#Secret`, enabling the toolchain to redact, encrypt, and dispatch secrets through platform-appropriate resources without the module author managing any of that machinery.

## References

- [RFC-0002: Sensitive Data Model](0002-sensitive-data-model.md) — First-class `#Secret` type for OPM, interacts with interface shapes
- [External Secrets Operator](https://external-secrets.io/) — ExternalSecret CRD and ClusterSecretStore
- [Kubernetes Gateway API](https://gateway-api.sigs.k8s.io/) — Protocol-aware routing (comparable to Path A traits)
- [Crossplane Compositions](https://docs.crossplane.io/latest/concepts/compositions/) — Claims/Compositions model closest to provides/requires
- [Radius Recipes](https://docs.radapp.io/guides/recipes/overview/) — Similar platform fulfillment philosophy
- [Score Specification](https://score.dev/) — Workload spec with abstract resource dependencies
