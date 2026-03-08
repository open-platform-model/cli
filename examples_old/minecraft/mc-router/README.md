# mc-router Module

A TCP hostname router for Minecraft servers using [itzg/mc-router](https://github.com/itzg/mc-router). Routes incoming Minecraft connections to backend servers based on the hostname the client used to connect.

## Overview

mc-router sits in front of multiple Minecraft servers and inspects the initial handshake packet to determine which backend server should receive the connection. This allows hosting multiple Minecraft servers behind a single IP address on port 25565.

```text
┌──────────────────────────────────────────────────────────┐
│                     Clients                              │
│                                                          │
│   smp.example.com    creative.example.com    ...         │
│         │                    │                            │
│         └────────┬───────────┘                            │
│                  │                                        │
│            ┌─────▼──────┐                                │
│            │  mc-router  │  :25565                        │
│            │  (this mod) │                                │
│            └──┬──────┬───┘                                │
│               │      │                                    │
│       ┌───────▼┐  ┌──▼───────┐                           │
│       │ SMP    │  │ Creative │  (minecraft-java modules)  │
│       │ Server │  │ Server   │                            │
│       └────────┘  └──────────┘                            │
└──────────────────────────────────────────────────────────┘
```

## Server Mappings

Define static hostname-to-server mappings in your values:

```cue
values: {
    router: {
        mappings: [
            {
                externalHostname: "smp.example.com"
                host:             "minecraft-smp.minecraft.svc.cluster.local"
                port:             25565
            },
            {
                externalHostname: "creative.example.com"
                host:             "minecraft-creative.minecraft.svc.cluster.local"
                port:             25565
            },
        ]
    }
}
```

Each mapping translates to a `--mapping=hostname=host:port` flag on the mc-router container.

You can also set a default server for unmatched hostnames:

```cue
values: {
    router: {
        defaultServer: {
            host: "minecraft-lobby.minecraft.svc.cluster.local"
            port: 25565
        }
    }
}
```

## Auto-Scale (Wake/Sleep)

mc-router can automatically scale Minecraft StatefulSets up when a player connects and back down after a period of inactivity. This is useful for saving resources when servers are idle.

```cue
values: {
    router: {
        autoScale: {
            up: enabled:   true
            down: {
                enabled: true
                after:   "10m"
            }
        }
    }
}
```

## RBAC

The module includes a `rbac` component that produces a **ClusterRole + ClusterRoleBinding** binding the `mc-router` ServiceAccount to the following permissions:

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| `""` (core) | `services` | `watch`, `list` |
| `apps` | `statefulsets`, `statefulsets/scale` | `watch`, `list`, `get`, `update`, `patch` |

Both rules are always present. mc-router uses the `services` rule to discover routing targets. It only invokes the StatefulSet scale APIs when `AUTO_SCALE_UP` or `AUTO_SCALE_DOWN` env vars are set (via `router.autoScale` config), so the StatefulSet permission is dormant when auto-scale is disabled.

The `router` component sets `automountToken: true` on the workload identity so the Kubernetes API credentials are available to the container.

## Metrics

mc-router supports several metrics backends:

| Backend      | Description                        |
|-------------|-------------------------------------|
| `discard`    | No metrics (default)               |
| `expvar`     | Go expvar (available at `/debug/vars`) |
| `influxdb`   | Push metrics to InfluxDB           |
| `prometheus` | Expose `/metrics` endpoint          |

```cue
values: {
    router: {
        metrics: backend: "prometheus"
        api: {
            enabled: true
            port:    8080
        }
    }
}
```

## REST API

Enable the REST API to manage routes dynamically at runtime:

```cue
values: {
    router: {
        api: {
            enabled: true
            port:    8080
        }
    }
}
```

The API port is exposed as a second service port alongside the Minecraft port.

## Configuration Reference

| Field                          | Type     | Default    | Description                           |
|-------------------------------|----------|------------|---------------------------------------|
| `router.image`                | Image    | itzg/mc-router:latest | Container image              |
| `router.connectionRateLimit`  | int      | 1          | Max connections per second            |
| `router.debug`                | bool     | false      | Enable debug logging                  |
| `router.simplifySrv`          | bool     | -          | Simplify SRV record lookup            |
| `router.useProxyProtocol`     | bool     | -          | Enable PROXY protocol                 |
| `router.defaultServer`        | struct   | -          | Default server {host, port}           |
| `router.mappings`             | list     | -          | Static hostname-to-server mappings    |
| `router.autoScale`            | struct   | -          | Auto-scale up/down configuration      |
| `router.metrics.backend`      | string   | -          | Metrics backend                       |
| `router.api`                  | struct   | -          | REST API {enabled, port}              |
| `port`                        | int      | 25565      | Minecraft listening port              |
| `serviceType`                 | string   | NodePort   | Service type                          |
| `resources`                   | struct   | -          | CPU/memory requests and limits        |

## Related Resources

- **itzg/mc-router**: <https://github.com/itzg/mc-router>
- **mc-router Docker Hub**: <https://hub.docker.com/r/itzg/mc-router>
- **minecraft-java module**: `../minecraft-java/` (backend server module)
