# OPM vs KubeVela: A Developer Experience Comparison

| Field       | Value                  |
|-------------|------------------------|
| **Status**  | Draft                  |
| **Created** | 2026-02-12             |
| **Authors** | OPM Contributors       |

## Table of Contents

1. [Introduction](#1-introduction)
2. [Defining an Application](#2-defining-an-application)
3. [Components and Composition](#3-components-and-composition)
4. [Extending the Platform](#4-extending-the-platform)
5. [Blueprints vs ComponentDefinitions](#5-blueprints-vs-componentdefinitions)
6. [Multi-Cluster and Deployment Topology](#6-multi-cluster-and-deployment-topology)
7. [Workflows and Orchestration](#7-workflows-and-orchestration)
8. [The Rendering Pipeline](#8-the-rendering-pipeline)
9. [Summary Table](#9-summary-table)
10. [Conclusions](#10-conclusions)

---

## 1. Introduction

KubeVela and OPM share DNA. Both implement a Component + Trait model inspired by the
Open Application Model (OAM). Both use CUE as their schema and templating language.
Both aim to separate platform concerns from application concerns.

But they diverge sharply in *how* and *where* they do the work.

**KubeVela** is a runtime delivery platform. You install a controller into your
Kubernetes cluster, apply an `Application` Custom Resource, and the controller
continuously reconciles your desired state. It includes a workflow engine, multi-cluster
management, a web UI (VelaUX), and an addon ecosystem. It's a CNCF Incubating project
(since Feb 2023), at v1.10.5, with 7,600+ GitHub stars.

**OPM** is a build-time application model. You define Modules in pure CUE, and the CLI
renders them into platform-specific manifests (currently Kubernetes YAML) before applying.
No controller runs in the cluster. Type safety is enforced at evaluation time by CUE's
unification semantics, not at reconciliation time by a Go controller. OPM is pre-v1,
under heavy development.

This document compares the **developer experience** — how you define applications, compose
capabilities, extend the platform, and deploy — with real code from both projects.

### What This Document Is Not

- Not a feature matrix or marketing comparison
- Not a recommendation of one over the other
- Not a comprehensive API reference for either system

All KubeVela examples are sourced from the official v1.10 documentation at
[kubevela.io](https://kubevela.io/docs). All OPM examples are sourced from the
`catalog/v0/` directory in this repository.

---

## 2. Defining an Application

### KubeVela: The Application CR

In KubeVela, you write a single YAML file describing an `Application` Custom Resource.
It contains components, traits, policies, and an optional workflow. You apply it with
`kubectl apply` or `vela up`, and the in-cluster controller takes it from there.

Here's the canonical "first app" from the KubeVela quick-start guide:

```yaml
# Source: https://kubevela.io/docs/quick-start/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: first-vela-app
spec:
  components:
    - name: express-server
      type: webservice
      properties:
        image: oamdev/hello-world
        ports:
          - port: 8000
            expose: true
      traits:
        - type: scaler
          properties:
            replicas: 1
  policies:
    - name: target-default
      type: topology
      properties:
        clusters: ["local"]
        namespace: "default"
    - name: target-prod
      type: topology
      properties:
        clusters: ["local"]
        namespace: "prod"
    - name: deploy-ha
      type: override
      properties:
        components:
          - type: webservice
            traits:
              - type: scaler
                properties:
                  replicas: 2
  workflow:
    steps:
      - name: deploy2default
        type: deploy
        properties:
          policies: ["target-default"]
      - name: manual-approval
        type: suspend
      - name: deploy2prod
        type: deploy
        properties:
          policies: ["target-prod", "deploy-ha"]
```

Everything lives in one artifact: components, traits, multi-cluster topology, overrides,
and a multi-step workflow with manual approval. The controller parses this, evaluates CUE
templates internally, and renders Kubernetes resources.

### OPM: Module + ModuleRelease

In OPM, the application definition is split into two artifacts:

1. **Module** — the portable blueprint (written by a Module Author)
2. **ModuleRelease** — the concrete deployment instance (written by an End User)

Here's the equivalent from `catalog/v0/examples/modules/basic_module.cue`:

```cue
// Module definition (Module Author writes this)
// Source: catalog/v0/examples/modules/basic_module.cue

package modules

import (
    core "opmodel.dev/core@v0"
    components "opmodel.dev/examples/components@v0"
)

basicModule: core.#Module & {
    metadata: {
        apiVersion: "opmodel.dev@v0"
        name:       "basic-module"
        version:    "0.1.0"

        defaultNamespace: "default"

        labels: {
            "example.com/module-type": "basic"
        }
    }

    #components: {
        web: components.basicComponent & {
            spec: {
                scaling: count:   #config.web.scaling
                container: image: #config.web.image
            }
        }
        db: components.postgresComponent & {
            spec: {
                container: image: #config.db.image
                volumes: "postgres-data": {
                    persistentClaim: size: #config.db.volumeSize
                }
            }
        }
    }

    // Value schema — constraints only, NO defaults
    #config: {
        web: {
            scaling: int
            image:   string
        }
        db: {
            image:      string
            volumeSize: string
        }
    }

    // Sane defaults from the Module Author
    values: {
        web: {
            scaling: 1
            image:   "nginx:1.20.0"
        }
        db: {
            image:      "postgres:14.0"
            volumeSize: "5Gi"
        }
    }
}
```

```cue
// ModuleRelease (End User writes this)
// Source: catalog/v0/examples/modules/basic_module.cue

basicModuleRelease: core.#ModuleRelease & {
    metadata: {
        name:      "basic-module-release"
        namespace: "production"

        labels: {
            "example.com/release-type": "basic"
        }
    }
    #module: basicModule
    values: {
        web: {
            scaling: 3
            image:   "nginx:1.21.6"
        }
        db: {
            image:      "postgres:14.5"
            volumeSize: "10Gi"
        }
    }
}
```

The Module defines *what's configurable* (`#config`) and *what the sane defaults are*
(`values`). The ModuleRelease provides *concrete overrides* for a specific environment.
The CLI evaluates the CUE, unifies config + values, runs transformer matching, and
produces Kubernetes manifests.

### Analysis

| Aspect | KubeVela | OPM |
|--------|----------|-----|
| Artifact count | 1 (Application YAML) | 2 (Module + ModuleRelease in CUE) |
| Language | YAML (with CUE evaluated internally) | Pure CUE throughout |
| Validation | Runtime (controller rejects invalid apps) | Build-time (CUE catches errors before apply) |
| Value system | 2-tier: Definition defaults → Application properties | 3-tier: `#config` schema → Author `values` → Release `values` |
| Controller required | Yes | No |

KubeVela is simpler to get started with — one YAML file, `kubectl apply`, done. The
controller handles everything. The trade-off is that validation happens at runtime. If
you typo a property name, you find out when the controller tries to reconcile.

OPM requires more upfront structure (separate Module and ModuleRelease, CUE imports,
typed schemas) but catches errors earlier. If you provide `scaling: "three"` where an
`int` is expected, CUE rejects it before anything touches the cluster. The three-tier
value system also creates a clearer handoff: the Module Author defines the schema and
safe defaults, the Platform Operator can layer on constraints, and the End User provides
final values.

---

## 3. Components and Composition

### KubeVela: Built-in Component Types

KubeVela ships with opinionated, pre-built component types. You pick a type (e.g.,
`webservice`, `worker`, `task`, `cron-task`, `daemon`) and fill in properties.

```yaml
# Source: https://kubevela.io/docs/tutorials/webservice/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: webservice-app
spec:
  components:
    - name: frontend
      type: webservice
      properties:
        image: oamdev/testapp:v1
        cmd: ["node", "server.js"]
        ports:
          - port: 8080
            expose: true
        exposeType: NodePort
        cpu: "0.5"
        memory: "512Mi"
      traits:
        - type: scaler
          properties:
            replicas: 1
```

```yaml
# Source: https://kubevela.io/docs/end-user/components/references/#cron-task
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: cron-worker
spec:
  components:
    - name: mytask
      type: cron-task
      properties:
        image: perl
        count: 10
        cmd: ["perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"]
        schedule: "*/1 * * * *"
```

The `type: webservice` tells KubeVela to use the `webservice` ComponentDefinition, which
internally uses a CUE template to render a Kubernetes Deployment + Service. You don't see
any of that — you just set properties.

### OPM: Resource + Trait Composition

In OPM, components are composed explicitly from typed building blocks. You import
Resources (the nouns) and Traits (the adjectives), then embed them into a Component.

```cue
// Source: catalog/v0/examples/components/basic_component.cue

package components

import (
    core "opmodel.dev/core@v0"
    workload_resources "opmodel.dev/resources/workload@v0"
    storage_resources "opmodel.dev/resources/storage@v0"
    workload_traits "opmodel.dev/traits/workload@v0"
)

basicComponent: core.#Component & {
    metadata: {
        name: "basic-component"
        labels: {
            "core.opmodel.dev/workload-type": "stateless"
        }
    }

    // Compose resources and traits using helpers
    workload_resources.#Container
    storage_resources.#Volumes
    workload_traits.#Scaling

    spec: {
        scaling: count: int | *1
        container: {
            name:            "nginx-container"
            image:           string | *"nginx:latest"
            imagePullPolicy: "IfNotPresent"
            ports: http: {
                name:       "http"
                targetPort: 80
                protocol:   "TCP"
            }
            env: {
                ENVIRONMENT: {
                    name:  "ENVIRONMENT"
                    value: "production"
                }
            }
            resources: {
                limits: {
                    cpu:    "500m"
                    memory: "256Mi"
                }
                requests: {
                    cpu:    "250m"
                    memory: "128Mi"
                }
            }
        }
        volumes: dbData: {
            name: "dbData"
            persistentClaim: {
                size:         "10Gi"
                accessMode:   "ReadWriteOnce"
                storageClass: "standard"
            }
        }
    }
}
```

A more complex example — a stateful workload with health checks, init containers, update
strategy, and volumes:

```cue
// Source: catalog/v0/examples/components/stateful_workload.cue

statefulWorkload: core.#Component & {
    metadata: {
        name: "stateful-workload"
        labels: {
            "core.opmodel.dev/workload-type": "stateful"
        }
    }

    workload_resources.#Container
    storage_resources.#Volumes
    workload_traits.#Scaling
    workload_traits.#RestartPolicy
    workload_traits.#UpdateStrategy
    workload_traits.#HealthCheck
    workload_traits.#InitContainers

    spec: {
        scaling: count: int | *1
        restartPolicy: "Always"
        updateStrategy: {
            type: "RollingUpdate"
            rollingUpdate: {
                maxUnavailable: 1
                partition:      0
            }
        }
        healthCheck: {
            livenessProbe: {
                exec: command: ["pg_isready", "-U", "admin"]
                initialDelaySeconds: 30
                periodSeconds:       10
                timeoutSeconds:      5
                failureThreshold:    3
            }
            readinessProbe: {
                exec: command: ["pg_isready", "-U", "admin"]
                initialDelaySeconds: 5
                periodSeconds:       10
                timeoutSeconds:      1
                failureThreshold:    3
            }
        }
        initContainers: [{
            name:  "init-db"
            image: string | *"postgres:14"
            env: PGHOST: {
                name:  "PGHOST"
                value: "localhost"
            }
        }]
        container: {
            name:            "postgres"
            image:           string | *"postgres:14"
            imagePullPolicy: "IfNotPresent"
            ports: postgres: {
                name:       "postgres"
                targetPort: 5432
            }
            env: {
                POSTGRES_DB:       { name: "POSTGRES_DB",       value: "myapp" }
                POSTGRES_USER:     { name: "POSTGRES_USER",     value: "admin" }
                POSTGRES_PASSWORD: { name: "POSTGRES_PASSWORD", value: "secretpassword" }
            }
            resources: {
                requests: { cpu: "500m",  memory: "1Gi" }
                limits:   { cpu: "2000m", memory: "4Gi" }
            }
            volumeMounts: data: {
                name:      "data"
                mountPath: "/var/lib/postgresql/data"
            }
        }
        volumes: data: {
            name: "data"
            persistentClaim: size: "10Gi"
        }
    }
}
```

### Analysis

| Aspect | KubeVela | OPM |
|--------|----------|-----|
| How you pick capabilities | `type: webservice` + flat `properties:` | Embed `#Container`, `#Scaling`, `#Expose` etc. |
| What's visible | Properties only — rendering is hidden | Full composition tree — Resources and Traits are explicit |
| Adding a trait | `traits: [{ type: scaler, properties: ... }]` | Embed `workload_traits.#Scaling` + set `spec.scaling:` |
| Type checking | Runtime (controller validates properties against CUE template) | Build-time (CUE unification validates `spec` against all schemas) |
| Verbosity | Lower — opinionated defaults hide complexity | Higher — explicit is the point |

KubeVela optimises for quick onboarding: pick a type, set properties, attach traits by
name. The developer never sees the CUE template that generates the Deployment. This is
great for end users who just want to ship.

OPM optimises for transparency: every Resource and Trait in a component is visible as a
CUE embed. When you write `workload_traits.#Scaling`, you can jump to its definition
and see exactly what schema it contributes to `spec`. Nothing is hidden behind a type
name. This is more work upfront but makes the composition fully auditable.

The label on the component (`"core.opmodel.dev/workload-type": "stateless"`) is how OPM
transformers know *what to generate*. In KubeVela, the `type: webservice` string does
this job. The difference: OPM's labels propagate from Resources and Traits automatically
via CUE unification, so you can't accidentally create an impossible combination.

---

## 4. Extending the Platform

### KubeVela: ComponentDefinition + TraitDefinition

In KubeVela, platform engineers create new component types and traits by writing CUE
templates embedded inside Kubernetes CRDs. Here's a custom `stateless` component that
renders a Deployment:

```cue
// Source: https://kubevela.io/docs/platform-engineers/components/custom-component/
// Applied as a ComponentDefinition CRD to the cluster

stateless: {
    annotations: {}
    attributes: workload: definition: {
        apiVersion: "apps/v1"
        kind:       "Deployment"
    }
    description: ""
    labels: {}
    type: "component"
}
template: {
    output: {
        apiVersion: "apps/v1"
        kind:       "Deployment"
        spec: {
            selector: matchLabels: "app.oam.dev/component": parameter.name
            template: {
                metadata: labels: "app.oam.dev/component": parameter.name
                spec: containers: [{
                    name:  parameter.name
                    image: parameter.image
                }]
            }
        }
    }
    outputs: {}
    parameter: {
        name:  string
        image: string
    }
}
```

And a custom `my-route` trait that composes a Service + Ingress:

```cue
// Source: https://kubevela.io/docs/platform-engineers/traits/customize-trait/
// Applied as a TraitDefinition CRD to the cluster

"my-route": {
    annotations: {}
    attributes: {
        appliesToWorkloads: []
        conflictsWith: []
        podDisruptive:   false
        workloadRefPath: ""
    }
    description: "My ingress route trait."
    labels: {}
    type: "trait"
}
template: {
    parameter: {
        domain: string
        http: [string]: int
    }
    outputs: service: {
        apiVersion: "v1"
        kind:       "Service"
        spec: {
            selector: app: context.name
            ports: [
                for k, v in parameter.http {
                    port:       v
                    targetPort: v
                },
            ]
        }
    }
    outputs: ingress: {
        apiVersion: "networking.k8s.io/v1beta1"
        kind:       "Ingress"
        metadata: name: context.name
        spec: rules: [{
            host: parameter.domain
            http: paths: [
                for k, v in parameter.http {
                    path: k
                    backend: {
                        serviceName: context.name
                        servicePort: v
                    }
                },
            ]
        }]
    }
}
```

Usage:

```yaml
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: testapp
spec:
  components:
    - name: express-server
      type: webservice
      properties:
        cmd: ["node", "server.js"]
        image: oamdev/testapp:v1
        port: 8080
      traits:
        - type: my-route
          properties:
            domain: test.my.domain
            http:
              "/api": 8080
```

The key pattern: `output:` renders the primary resource, `outputs:` renders additional
resources. The controller evaluates the CUE, produces Kubernetes objects, and applies
them.

### OPM: Resources, Traits, and Transformers

In OPM, extending the platform involves three concepts instead of two:

1. **Resource** — defines what a thing *is* (the noun)
2. **Trait** — defines a cross-cutting behavior (the adjective)
3. **Transformer** — defines how to convert a component to a platform resource

Here's the Container resource definition:

```cue
// Source: catalog/v0/resources/workload/container.cue

#ContainerResource: close(core.#Resource & {
    metadata: {
        apiVersion:  "opmodel.dev/resources/workload@v0"
        name:        "container"
        description: "A container definition for workloads"
        labels: {}
    }

    #defaults: #ContainerDefaults
    #spec: container: schemas.#ContainerSchema
})

// Helper for embedding into Components
#Container: close(core.#Component & {
    metadata: labels: {
        "core.opmodel.dev/workload-type"!: "stateless" | "stateful" | "daemon" | "task" | "scheduled-task"
        ...
    }
    #resources: {(#ContainerResource.metadata.fqn): #ContainerResource}
})
```

The Expose trait:

```cue
// Source: catalog/v0/traits/network/expose.cue

#ExposeTrait: close(core.#Trait & {
    metadata: {
        apiVersion:  "opmodel.dev/traits/network@v0"
        name:        "expose"
        description: "A trait to expose a workload via a service"
    }

    appliesTo: [workload_resources.#ContainerResource]

    #defaults: #ExposeDefaults
    #spec: expose: schemas.#ExposeSchema
})

#Expose: close(core.#Component & {
    #traits: {(#ExposeTrait.metadata.fqn): #ExposeTrait}
})

#ExposeDefaults: close(schemas.#ExposeSchema & {
    type: "ClusterIP"
})
```

And the Service transformer — the thing that actually produces Kubernetes output:

```cue
// Source: catalog/v0/providers/kubernetes/transformers/service_transformer.cue

#ServiceTransformer: core.#Transformer & {
    metadata: {
        apiVersion:  "opmodel.dev/providers/kubernetes/transformers@v0"
        name:        "service-transformer"
        description: "Creates Kubernetes Services for components with Expose trait"
    }

    requiredLabels: {}

    requiredResources: {
        "opmodel.dev/resources/workload@v0#Container": workload_resources.#ContainerResource
    }

    requiredTraits: {
        "opmodel.dev/traits/network@v0#Expose": network_traits.#ExposeTrait
    }

    optionalResources: {}
    optionalTraits: {}

    #transform: {
        #component: _
        #context:   core.#TransformerContext

        _container: #component.spec.container
        _expose:    #component.spec.expose

        _ports: [
            for portName, portConfig in _expose.ports {
                {
                    name:       portName
                    port:       portConfig.exposedPort | *portConfig.targetPort
                    targetPort: portConfig.targetPort
                    protocol:   portConfig.protocol | *"TCP"
                    if _expose.type == "NodePort" && portConfig.exposedPort != _|_ {
                        nodePort: portConfig.exposedPort
                    }
                }
            },
        ]

        output: k8scorev1.#Service & {
            apiVersion: "v1"
            kind:       "Service"
            metadata: {
                name:      #component.metadata.name
                namespace: #context.namespace | *"default"
                labels:    #context.labels
            }
            spec: {
                type:     _expose.type
                selector: #context.componentLabels
                ports:    _ports
            }
        }
    }
}
```

### Analysis

| Aspect | KubeVela | OPM |
|--------|----------|-----|
| What you write | CUE template in ComponentDefinition/TraitDefinition CRD | Resource + Trait + Transformer (3 separate definitions) |
| Where it lives | Applied to K8s cluster as a CRD | CUE module in a registry (`opmodel.dev/...`) |
| How it renders | `output:` + `outputs:` in CUE template, evaluated by controller | `#transform.output:` in Transformer, evaluated by CLI |
| Who owns rendering? | The Definition (trait/component template decides what K8s resources to create) | The Transformer (separate from the trait; the trait only defines *what*, the transformer defines *how*) |
| Adding a new platform | Write new ComponentDefinitions (still K8s-only) | Write new Transformers for a new Provider (same Resource/Trait, different output) |

The critical difference: in KubeVela, the `my-route` TraitDefinition *is* the rendering
logic — it directly produces a Service and Ingress. In OPM, the `#ExposeTrait` only
defines the *schema* (what ports, what type). The `#ServiceTransformer` is a separate
artifact that knows how to convert that schema into a Kubernetes Service.

This separation means the same `#ExposeTrait` could be consumed by a Docker Compose
transformer, a Nomad transformer, or any future provider — without changing the trait
definition. KubeVela's approach is simpler (one artifact instead of three) but
inherently Kubernetes-coupled.

---

## 5. Blueprints vs ComponentDefinitions

### KubeVela: ComponentDefinition as the Abstraction Layer

In KubeVela, if you want a "golden path" — a pre-baked pattern that bundles a Deployment
and a Service together — you write a ComponentDefinition that renders both:

```cue
// Source: https://kubevela.io/docs/platform-engineers/components/custom-component/

webserver: {
    annotations: {}
    attributes: workload: definition: {
        apiVersion: "apps/v1"
        kind:       "Deployment"
    }
    description: ""
    labels: {}
    type: "component"
}
template: {
    output: {
        apiVersion: "apps/v1"
        kind:       "Deployment"
        spec: {
            selector: matchLabels: "app.oam.dev/component": context.name
            template: {
                metadata: labels: "app.oam.dev/component": context.name
                spec: containers: [{
                    name:  context.name
                    image: parameter.image
                    if parameter["cmd"] != _|_ {
                        command: parameter.cmd
                    }
                    if parameter["env"] != _|_ {
                        env: parameter.env
                    }
                    if context["config"] != _|_ {
                        env: context.config
                    }
                    ports: [{ containerPort: parameter.port }]
                    if parameter["cpu"] != _|_ {
                        resources: {
                            limits:   cpu: parameter.cpu
                            requests: cpu: parameter.cpu
                        }
                    }
                }]
            }
        }
    }
    outputs: service: {
        apiVersion: "v1"
        kind:       "Service"
        spec: {
            selector: "app.oam.dev/component": context.name
            ports: [{
                port:       parameter.port
                targetPort: parameter.port
            }]
        }
    }
    parameter: {
        image: string
        cmd?: [...string]
        port: *80 | int
        env?: [...{
            name:   string
            value?: string
            valueFrom?: secretKeyRef: {
                name: string
                key:  string
            }
        }]
        cpu?: string
    }
}
```

Usage:

```yaml
# Source: https://kubevela.io/docs/platform-engineers/components/custom-component/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: webserver-demo
  namespace: default
spec:
  components:
    - name: hello-webserver
      type: webserver
      properties:
        image: oamdev/hello-world
        port: 8000
        env:
          - name: "foo"
            value: "bar"
        cpu: "100m"
```

The user says `type: webserver`, sets properties, and gets both a Deployment and a
Service. The rendering logic is opaque.

### OPM: Blueprint as Pre-Composed Patterns

In OPM, a Blueprint bundles Resources + Traits into a reusable pattern. Here's the
`#StatelessWorkload` blueprint:

```cue
// Source: catalog/v0/blueprints/workload/stateless_workload.cue

#StatelessWorkloadBlueprint: close(core.#Blueprint & {
    metadata: {
        apiVersion:  "opmodel.dev/blueprints@v0"
        name:        "stateless-workload"
        description: "A stateless workload with no requirement for stable identity or storage"
        labels: {
            "core.opmodel.dev/category":      "workload"
            "core.opmodel.dev/workload-type": "stateless"
        }
    }

    composedResources: [
        workload_resources.#ContainerResource,
    ]

    composedTraits: [
        workload_traits.#ScalingTrait,
    ]

    #spec: statelessWorkload: schemas.#StatelessWorkloadSchema
})

#StatelessWorkload: close(core.#Component & {
    #blueprints: (#StatelessWorkloadBlueprint.metadata.fqn): #StatelessWorkloadBlueprint

    workload_resources.#Container
    workload_traits.#Scaling
    workload_traits.#RestartPolicy
    workload_traits.#UpdateStrategy
    workload_traits.#HealthCheck
    workload_traits.#SidecarContainers
    workload_traits.#InitContainers

    #spec: {
        statelessWorkload: schemas.#StatelessWorkloadSchema
        container:         statelessWorkload.container
        if statelessWorkload.scaling != _|_ {
            scaling: statelessWorkload.scaling
        }
        if statelessWorkload.restartPolicy != _|_ {
            restartPolicy: statelessWorkload.restartPolicy
        }
        if statelessWorkload.updateStrategy != _|_ {
            updateStrategy: statelessWorkload.updateStrategy
        }
        if statelessWorkload.healthCheck != _|_ {
            healthCheck: statelessWorkload.healthCheck
        }
        if statelessWorkload.sidecarContainers != _|_ {
            sidecarContainers: statelessWorkload.sidecarContainers
        }
        if statelessWorkload.initContainers != _|_ {
            initContainers: statelessWorkload.initContainers
        }
    }
})
```

The blueprint is *transparent*: you can see exactly which Resources and Traits it composes
(`#Container`, `#Scaling`, `#RestartPolicy`, etc.) and how the higher-level
`statelessWorkload` schema maps down to the individual specs.

The blueprint does **not** contain rendering logic. Rendering is still handled by
Transformers (DeploymentTransformer, ServiceTransformer, etc.) which match this
component based on its labels, resources, and traits.

### Analysis

| Aspect | KubeVela ComponentDefinition | OPM Blueprint |
|--------|------------------------------|---------------|
| Contains rendering logic | Yes (`output:` / `outputs:`) | No (delegates to Transformers) |
| Transparency | Opaque — user sees `type: webserver` | Transparent — Resources and Traits are listed |
| Customisation | Override `parameter` values only | Unify with additional Traits, override any `spec` field |
| Adding Kubernetes resources | Add to `outputs:` in the CUE template | Add a new Transformer that matches the component |

Both serve the same purpose — "golden paths" for common patterns. KubeVela bundles
everything (schema + rendering) into one artifact. OPM separates schema (Blueprint) from
rendering (Transformer), which enables portability at the cost of more moving parts.

---

## 6. Multi-Cluster and Deployment Topology

### KubeVela: Topology + Override + Workflow

KubeVela has mature, integrated multi-cluster support. You declare topology policies
(which clusters to target), override policies (per-cluster customisations), and use
workflow steps to orchestrate the rollout.

```yaml
# Source: https://kubevela.io/docs/case-studies/multi-cluster/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: deploy-with-override
  namespace: examples
spec:
  components:
    - name: nginx-with-override
      type: webservice
      properties:
        image: nginx
  policies:
    - name: topology-hangzhou-clusters
      type: topology
      properties:
        clusterLabelSelector:
          region: hangzhou
    - name: topology-local
      type: topology
      properties:
        clusters: ["local"]
        namespace: examples-alternative
    - name: override-nginx-legacy-image
      type: override
      properties:
        components:
          - name: nginx-with-override
            properties:
              image: nginx:1.20
    - name: override-high-availability
      type: override
      properties:
        components:
          - type: webservice
            traits:
              - type: scaler
                properties:
                  replicas: 3
  workflow:
    steps:
      - type: deploy
        name: deploy-local
        properties:
          policies: ["topology-local"]
      - type: deploy
        name: deploy-hangzhou
        properties:
          policies: ["topology-hangzhou-clusters", "override-nginx-legacy-image", "override-high-availability"]
```

This deploys to the local cluster first, then (in a separate step) to hangzhou clusters
with a legacy image and 3 replicas. You can also select clusters by label instead of
name, reuse policies across applications as standalone resources, and run deploys
concurrently.

KubeVela also supports external, reusable policies and workflows:

```yaml
# Source: https://kubevela.io/docs/case-studies/multi-cluster/
apiVersion: core.oam.dev/v1alpha1
kind: Policy
metadata:
  name: topology-hangzhou-clusters
  namespace: examples
type: topology
properties:
  clusterLabelSelector:
    region: hangzhou
---
apiVersion: core.oam.dev/v1alpha1
kind: Workflow
metadata:
  name: make-release-in-hangzhou
  namespace: examples
steps:
  - type: deploy
    name: deploy-hangzhou
    properties:
      auto: false
      policies: ["override-high-availability-webservice", "topology-hangzhou-clusters"]
```

```yaml
# Application referencing the external workflow
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: external-policies-and-workflow
  namespace: examples
spec:
  components:
    - name: nginx-external-policies-and-workflow
      type: webservice
      properties:
        image: nginx
  workflow:
    ref: make-release-in-hangzhou
```

### OPM: ModuleRelease per Target

OPM currently handles multi-environment deployment by creating separate ModuleReleases
that target different namespaces:

```cue
// Deploy to staging
stagingRelease: core.#ModuleRelease & {
    metadata: {
        name:      "my-app-staging"
        namespace: "staging"
    }
    #module: myModule
    values: {
        web: scaling: 1
        web: image:   "myapp:v2.0.0-rc1"
    }
}

// Deploy to production
productionRelease: core.#ModuleRelease & {
    metadata: {
        name:      "my-app-production"
        namespace: "production"
    }
    #module: myModule
    values: {
        web: scaling: 3
        web: image:   "myapp:v2.0.0"
    }
}
```

There's no built-in concept of cluster topology, override policies, or orchestrated
multi-cluster rollouts. Each release is independent, applied separately. Orchestration
across releases is delegated to external CI/CD pipelines.

### Analysis

This is KubeVela's strongest area. Its multi-cluster story is production-proven and
deeply integrated:

- **Cluster Gateway** provides unified API access to managed clusters
- **Topology policies** target clusters by name or label selector
- **Override policies** customise per-cluster (image, replicas, traits)
- **Workflow steps** orchestrate the rollout order with approval gates

OPM's approach is simpler but less capable. A ModuleRelease targets a single namespace,
and orchestrating across environments requires external tooling. This is a deliberate
trade-off — OPM avoids runtime complexity — but it means multi-cluster is currently a
manual exercise.

This is an area where OPM has room to grow. A future `#Bundle` type or environment-aware
deployment model could close the gap, but as of today, if multi-cluster orchestration is
a hard requirement, KubeVela has the answer.

---

## 7. Workflows and Orchestration

### KubeVela: Built-in Workflow Engine

KubeVela includes a CUE-driven workflow engine with support for sequential steps, parallel
execution, suspend/resume, notifications, HTTP requests, conditional logic, and more.

**Manual approval gate:**

```yaml
# Source: https://kubevela.io/docs/end-user/workflow/suspend/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: suspend-demo
  namespace: default
spec:
  components:
    - name: comp1
      type: webservice
      properties:
        image: crccheck/hello-world
        port: 8000
    - name: comp2
      type: webservice
      properties:
        image: crccheck/hello-world
        port: 8000
  workflow:
    steps:
      - name: apply1
        type: apply-component
        properties:
          component: comp1
      - name: suspend
        type: suspend
      - name: apply2
        type: apply-component
        properties:
          component: comp2
```

**Parallel execution with step-group:**

```yaml
# Source: https://kubevela.io/docs/end-user/workflow/built-in-workflow-defs/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: example
  namespace: default
spec:
  components:
    - name: express-server
      type: webservice
      properties:
        image: crccheck/hello-world
        port: 8000
    - name: express-server2
      type: webservice
      properties:
        image: crccheck/hello-world
        port: 8000
  workflow:
    steps:
      - name: step
        type: step-group
        subSteps:
          - name: apply-sub-step1
            type: apply-component
            properties:
              component: express-server
          - name: apply-sub-step2
            type: apply-component
            properties:
              component: express-server2
```

**Notification + approval flow:**

```yaml
# Source: https://kubevela.io/docs/end-user/workflow/built-in-workflow-defs/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: first-vela-workflow
  namespace: default
spec:
  components:
    - name: express-server
      type: webservice
      properties:
        image: oamdev/hello-world
        port: 8000
  workflow:
    steps:
      - name: slack-message
        type: notification
        properties:
          slack:
            url:
              value: <your-slack-url>
            message:
              text: Ready to apply. Ask the admin to approve and resume.
      - name: manual-approval
        type: suspend
      - name: express-server
        type: apply-component
        properties:
          component: express-server
```

**Conditional deployment based on runtime state:**

```yaml
# Source: https://kubevela.io/docs/case-studies/multi-cluster/
apiVersion: core.oam.dev/v1beta1
kind: Application
metadata:
  name: deploy-with-override
spec:
  components:
    - name: mytask
      type: task
      properties:
        image: bash
        count: 1
        cmd: ["echo", "hello world"]
  policies:
    - name: target-default
      type: topology
      properties:
        clusters: ["local"]
        namespace: "default"
    - name: target-prod
      type: topology
      properties:
        clusters: ["local"]
        namespace: "prod"
  workflow:
    steps:
      - type: deploy
        name: deploy-01
        properties:
          policies: ["target-default"]
      - name: read-object
        type: read-object
        outputs:
          - name: ready
            valueFrom: output.value.status["ready"]
        properties:
          apiVersion: batch/v1
          kind: Job
          name: mytask
          namespace: default
          cluster: local
      - type: deploy
        name: deploy-02
        inputs:
          - from: ready
            if: inputs["ready"] == 0
        properties:
          policies: ["target-prod"]
```

Built-in workflow step types include: `deploy`, `apply-component`, `apply-application`,
`apply-object`, `read-object`, `suspend`, `notification`, `request`, `step-group`,
`export2config`, `export2secret`, and `webhook`.

### OPM: No Workflow (By Design)

OPM does not have a workflow concept. The CLI evaluates CUE, produces manifests, and
applies them via server-side apply. That's it.

```text
opm mod apply ./my-module
    │
    ├── Evaluate CUE (Module + ModuleRelease + Provider)
    ├── Match transformers to components
    ├── Run #transform for each match
    ├── Output Kubernetes manifests
    └── kubectl server-side apply
```

If you need:
- Staged rollouts → use your CI/CD pipeline (GitHub Actions, ArgoCD, etc.)
- Approval gates → use your CI/CD's approval mechanism
- Notifications → use your CI/CD's notification integration
- Conditional deployment → use CI/CD conditionals or shell scripting

### Analysis

This is KubeVela's biggest developer experience advantage, full stop. The ability to
declare "deploy to staging, wait for approval, then deploy to production with higher
replicas" in a single declarative artifact is genuinely powerful. Conditional deployment
based on runtime state (`read-object` + `if`) is something that's extremely difficult
to replicate with external tooling.

OPM's trade-off is intentional: no runtime controller means no in-cluster workflow engine.
The upside is operational simplicity — nothing to install, upgrade, or debug in the
cluster. The downside is that orchestration requires external tools and glue code.

For teams that already have a mature CI/CD pipeline, OPM's "render and apply" model fits
cleanly into existing workflows. For teams that want an all-in-one declarative delivery
platform, KubeVela's workflow engine is a compelling reason to use it.

---

## 8. The Rendering Pipeline

### KubeVela: Controller Reconciliation

In KubeVela, rendering happens inside the cluster:

```text
kubectl apply Application CR
         │
         ▼
┌─────────────────────────────┐
│   KubeVela Core Controller  │
│                             │
│  1. Parse Application CR    │
│  2. Resolve ComponentDefs   │
│  3. Evaluate CUE templates  │
│     (output + outputs)      │
│  4. Apply workflow steps    │
│  5. Create K8s resources    │
│  6. Watch & reconcile       │
│                             │
│  context.name = component   │
│  context.namespace = app ns │
│  parameter = properties     │
└─────────────────────────────┘
```

The CUE template has access to `context` (component name, namespace, app info) and
`parameter` (user properties). It renders `output` (primary resource) and `outputs`
(additional resources). The controller continuously reconciles, detecting drift and
re-applying.

### OPM: CLI-Driven CUE Evaluation

In OPM, rendering happens locally:

```text
opm mod apply ./my-module
         │
         ▼
┌──────────────────────────────────────┐
│   CLI (local machine)                │
│                                      │
│  1. Load Module + ModuleRelease      │
│  2. Evaluate CUE (full unification)  │
│  3. #MatchTransformers:              │
│     For each Transformer in Provider │
│       For each Component in Release  │
│         Check requiredLabels         │
│         Check requiredResources      │
│         Check requiredTraits         │
│  4. For each match:                  │
│     Execute #transform               │
│     Inject #TransformerContext        │
│  5. Output K8s manifests             │
│  6. Server-side apply                │
└──────────────────────────────────────┘
```

The matching algorithm is pure CUE — from `catalog/v0/core/transformer.cue`:

```cue
// Source: catalog/v0/core/transformer.cue

#Matches: {
    transformer: #Transformer
    component:   #Component

    // 1. Check Required Labels
    _reqLabels: *transformer.requiredLabels | {}
    _missingLabels: [
        for k, v in _reqLabels
        if len([for lk, lv in component.metadata.labels
                if lk == k && (lv & v) != _|_ {true}]) == 0 {
            k
        },
    ]

    // 2. Check Required Resources
    _reqResources: *transformer.requiredResources | {}
    _missingResources: [
        for k, v in _reqResources
        if len([for rk, rv in component.#resources
                if rk == k && (rv & v) != _|_ {true}]) == 0 {
            k
        },
    ]

    // 3. Check Required Traits
    _reqTraits: *transformer.requiredTraits | {}
    _missingTraits: [
        for k, v in _reqTraits
        if component.#traits == _|_ ||
           len([for tk, tv in component.#traits
                if tk == k && (tv & v) != _|_ {true}]) == 0 {
            k
        },
    ]

    result: len(_missingLabels) == 0 &&
            len(_missingResources) == 0 &&
            len(_missingTraits) == 0
}
```

A single component can match multiple transformers. A component with
`#Container` (stateless) + `#Expose` matches both the `DeploymentTransformer` (because
it has Container + `workload-type: stateless` label) and the `ServiceTransformer`
(because it has Container + Expose trait). Each transformer produces one output resource.

The `#TransformerContext` carries labels and annotations down to the output:

- Module-level labels (from ModuleRelease metadata)
- Component-level labels (from Component metadata, inherited from Resources/Traits)
- Controller labels (`app.kubernetes.io/managed-by`, `app.kubernetes.io/name`, etc.)

### Analysis

| Aspect | KubeVela | OPM |
|--------|----------|-----|
| Where rendering happens | In-cluster (controller) | Locally (CLI) |
| Continuous reconciliation | Yes (controller watches and re-applies) | No (render once, apply, done) |
| Drift detection | Automatic (controller corrects drift) | Manual (re-run `opm mod apply`) |
| Testability | Need a cluster (or envtest) to verify rendering | Pure CUE evaluation — test without a cluster |
| Matching logic | Go code in controller | Pure CUE (`#Matches`) — same semantics everywhere |
| Multi-resource from one component | `output` + `outputs` in a single template | Multiple transformers match one component, each producing one output |

OPM's rendering pipeline has an interesting property: because matching and transformation
are pure CUE, you can verify the entire pipeline — matching, context propagation, output
manifests — without a Kubernetes cluster. Just evaluate the CUE. This makes it possible
to write unit tests for transformers in CUE itself (and OPM does this — each transformer
file includes a `_test*` validation at the bottom).

KubeVela's reconciliation loop is the flip side: you get automatic drift correction and
continuous status reporting, but you need a running cluster to verify that your
ComponentDefinition renders what you expect.

---

## 9. Summary Table

| Dimension | KubeVela | OPM |
|-----------|----------|-----|
| **CNCF status** | Incubating (v1.10.5) | Pre-v1, under heavy development |
| **Runtime** | In-cluster controller (required) | CLI only (no controller) |
| **Language** | YAML + CUE templates | Pure CUE throughout |
| **Validation timing** | Runtime (controller rejects invalid apps) | Build-time (CUE catches errors before apply) |
| **Application artifact** | `Application` CR (single YAML) | `#Module` + `#ModuleRelease` (split CUE) |
| **Value system** | 2-tier (definition defaults → app properties) | 3-tier (schema → author defaults → release values) |
| **Component model** | `type:` name + `properties:` + `traits:` | Embed Resources + Traits + Blueprints into `#Component` |
| **Extending platform** | ComponentDefinition + TraitDefinition (CUE in YAML CRDs) | Resource + Trait + Transformer (pure CUE modules) |
| **Rendering ownership** | Definition owns rendering (`output:`/`outputs:`) | Transformer owns rendering (separate from definitions) |
| **Multi-cluster** | Topology + Override policies, Cluster Gateway | ModuleRelease per target (no orchestration) |
| **Workflow engine** | Built-in (deploy, suspend, notify, step-group, conditional) | None (delegates to CI/CD) |
| **Drift correction** | Automatic (controller reconciles) | Manual (re-run apply) |
| **Web UI** | VelaUX | None |
| **Portability** | Kubernetes only | Provider-agnostic (Transformer per platform) |
| **Testability** | Requires cluster or envtest | Pure CUE evaluation (no cluster needed) |
| **Distribution** | Helm charts, addon system | CUE registry + OCI artifacts |

---

## 10. Conclusions

### They Solve Different Problems

KubeVela answers: *"How do I orchestrate complex application delivery across multiple
Kubernetes clusters with workflows, approvals, and continuous reconciliation?"*

OPM answers: *"How do I define portable, type-safe application blueprints that can be
rendered to any platform without runtime dependencies?"*

KubeVela is a **delivery platform**. OPM is an **application model**. They overlap in
the "define applications with components and traits" space but diverge on everything
else.

### Where KubeVela Excels

1. **Workflow engine** — declarative multi-step deployment with suspend, approval,
   notification, and conditional logic. This is hard to replicate with external tooling.
2. **Multi-cluster maturity** — Cluster Gateway, topology policies, and overrides are
   production-proven and deeply integrated.
3. **Quick onboarding** — pick a type, set properties, `kubectl apply`, done. The
   learning curve is gentle for end users.
4. **Continuous reconciliation** — the controller detects and corrects drift automatically.
5. **Ecosystem** — addons for Terraform, FluxCD, Prometheus, and more.

### Where OPM Excels

1. **Build-time type safety** — CUE catches invalid configurations before anything
   touches a cluster. No more "apply and see what the controller says."
2. **Portable by design** — the Provider/Transformer abstraction cleanly separates
   "what to deploy" from "how to deploy it." Same Module, different Providers.
3. **No runtime dependency** — nothing to install in the cluster. Render locally,
   apply with kubectl. Operational simplicity.
4. **Transparent composition** — every Resource and Trait in a component is visible
   and auditable. No hidden rendering logic.
5. **Testable without a cluster** — pure CUE matching and transformation can be verified
   locally. Transformers include inline tests.
6. **Three-tier value system** — clear handoff from Module Author (schema + defaults) to
   Platform Operator (constraints) to End User (final values).

### Where OPM Can Learn from KubeVela — And Plans To

The gaps identified in this comparison are not permanent. OPM's roadmap includes
adopting these capabilities, adapted to its build-time-first architecture:

- **Workflow orchestration** (planned): Declarative multi-step deployment with staged
  rollouts, approval gates, and conditional execution — defined in CUE and executed by
  the CLI or a lightweight controller, preserving the "render, then apply" philosophy.
- **Multi-cluster topology** (planned): First-class topology and override policies so
  that a single Module or Bundle can target multiple clusters and namespaces without
  duplicating ModuleReleases.
- **Continuous reconciliation** (planned): An optional in-cluster controller that watches
  for drift and re-applies desired state using the same CUE evaluation pipeline as the
  CLI — ensuring identical behaviour locally and in-cluster.
- **Quick-start experience**: the Module + ModuleRelease split is more work than a single
  Application YAML. The `opm mod init` templates help, but there's room to simplify the
  happy path further.

The design goal is to reach feature parity with KubeVela's delivery capabilities while
retaining OPM's core strengths: compile-time type safety, provider-agnostic portability,
and no mandatory runtime controller.

### Complementary, Not Competitive

It's worth noting these systems could work together. KubeVela could consume OPM-rendered
manifests as a delivery backend. OPM could generate Application CRs for KubeVela to
orchestrate. The models aren't mutually exclusive — but they represent fundamentally
different philosophies about where complexity should live.

That said, OPM's ambitions extend beyond what KubeVela addresses. OPM's long-term vision
is to enable a multi-provider ecosystem where different service providers coexist in the
same environment — DNS from one company, Compute from another, Database from a third —
all consumed through standardised interfaces. The architectural decisions that
differentiate OPM from KubeVela today (provider-agnostic definitions, build-time CUE
evaluation, the Transformer abstraction) are foundational to that ecosystem. They're not
incidental design differences — they're load-bearing walls.
