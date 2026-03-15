# mc_java_fleet

Dynamic Minecraft Java server fleet with a single shared mc-router for
hostname-based TCP routing.

## Overview

Define N servers in the `servers` map — each server gets its own StatefulSet
and Kubernetes Service. A single mc-router (typically a `LoadBalancer`) routes
incoming player connections by hostname to the correct backend server.

```
players
  │
  ▼
mc-router (LoadBalancer :25565)
  │
  ├── lobby.mc.example.com    →  {releaseName}-server-lobby.{ns}.svc:25565
  ├── survival.mc.example.com →  {releaseName}-server-survival.{ns}.svc:25565
  └── creative.mc.example.com →  {releaseName}-server-creative.{ns}.svc:25565
```

Adding a server to `servers` automatically:
- Creates a StatefulSet + PVC + Service for that server
- Injects the shared `rconPassword` into the server
- Adds a `--mapping` entry to the mc-router

## Configuration

| Field | Required | Description |
|---|---|---|
| `releaseName` | [x] | Must match `ModuleRelease.metadata.name` — used for Service DNS names |
| `domain` | [x] | Base domain, e.g. `mc.example.com` |
| `namespace` | [ ] | K8s namespace (default: `"default"`) |
| `servers` | [x] | Map of server name → per-server config |
| `router` | [x] | Router image, port, serviceType, etc. |
| `rconPassword` | [x] | Shared RCON K8s Secret reference |

### Per-server config (`servers.<name>`)

Each server supports the full `itzg/minecraft-server` feature set:

- **Server types**: vanilla, paper, forge, fabric, spigot, bukkit, sponge, purpur, magma, ftba, autoCurseForge
- **JVM**: memory, opts, xxOpts, Aikar's GC flags
- **Server properties**: maxPlayers, difficulty, mode, pvp, motd, seed, ...
- **Storage**: PVC, hostPath, or emptyDir for data and backups
- **Backup sidecar** (`backup.enabled: true`): tar, rsync, restic, or rclone
- **Monitor sidecar** (`monitor.enabled: true`, default): Prometheus metrics via mc-monitor

## Example release

```cue
package main

import (
    m "opmodel.dev/core/modulerelease@v1"
    fleet "example.com/modules/mc_java_fleet@v0.1.0"
)

m.#ModuleRelease

metadata: {
    name:      "my-fleet"
    namespace: "minecraft"
}

module: fleet

config: {
    releaseName: "my-fleet"
    domain:      "mc.example.com"
    namespace:   "minecraft"

    servers: {
        lobby: {
            server: {
                motd:       "Welcome!"
                maxPlayers: 50
                mode:       "adventure"
                pvp:        false
                difficulty: "peaceful"
            }
            paper: {}
            jvm: memory: "1G"
        }
        survival: {
            server: {
                maxPlayers: 20
                difficulty: "hard"
            }
            fabric: {
                loaderVersion: "0.15.11"
                mods: modrinth: projects: ["lithium", "starlight"]
            }
            jvm: memory: "4G"
        }
    }

    router: {
        port:        25565
        serviceType: "LoadBalancer"
    }

    rconPassword: {
        $secretName: "mc-secrets"
        $dataKey:    "rcon-password"
    }
}
```

## Kubernetes resources produced

For a release named `my-fleet` with servers `lobby` and `survival`:

```
StatefulSet/my-fleet-server-lobby
Service/my-fleet-server-lobby          (ClusterIP)
PersistentVolumeClaim/my-fleet-server-lobby-data

StatefulSet/my-fleet-server-survival
Service/my-fleet-server-survival       (ClusterIP)
PersistentVolumeClaim/my-fleet-server-survival-data

Deployment/my-fleet-router
Service/my-fleet-router                (LoadBalancer)
ServiceAccount/my-fleet-router
ClusterRole/my-fleet-router
ClusterRoleBinding/my-fleet-router
```

## Service DNS convention

```
releaseName = "my-fleet"
namespace   = "minecraft"

server-lobby   →  my-fleet-server-lobby.minecraft.svc.cluster.local
server-survival →  my-fleet-server-survival.minecraft.svc.cluster.local
router         →  my-fleet-router.minecraft.svc.cluster.local
```

## Differences from the gamestack bundle

| Feature | mc_java_fleet | gamestack bundle |
|---|---|---|
| Velocity proxy | [ ] | [x] |
| mc-monitor | [x] (per-server sidecar) | [x] (per-server sidecar) |
| Deployment model | Single ModuleRelease | Multiple ModuleReleases |
| Proxied network mode | [ ] | [x] |
| Standalone servers | [x] | [x] |
