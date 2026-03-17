# mc_java_fleet

Dynamic Minecraft Java server fleet with a single shared mc-router for
hostname-based TCP routing.

## Overview

Define N servers in the `servers` map — each server gets its own StatefulSet
and Kubernetes Service. A single mc-router (typically a `LoadBalancer`) routes
incoming player connections by hostname to the correct backend server.

```text
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
- Injects the shared `rconPassword` into the server container
- Adds a `--mapping` entry to the mc-router

## Features

| Feature | Field | Default |
|---|---|---|
| Dynamic server fleet | `servers` | — |
| Enable/disable per server | `enabled` | `true` |
| Bootstrap from archive | `bootstrap.url` | — |
| 12 server types | `vanilla`, `paper`, `fabric`, ... | — |
| Modrinth modpack/projects | `modrinth` | — |
| CurseForge modpack | `autoCurseForge` | — |
| Spiget plugin download | `paper.plugins.spigetResources` | — |
| Extra container ports | `extraPorts` | — |
| Expose extra ports in Service | `extraPorts[].expose` | `false` |
| Router hostname aliases per server | `aliases` | — |
| Backup sidecar | `backup.enabled` | `false` |
| Prometheus monitor sidecar | `monitor.enabled` | `true` |
| VS Code in browser | `codeServer.enabled` | — |
| Restic snapshot browser | `resticGui.enabled` | — |
| Router auto-scale | `router.autoScale` | — |
| NFS / CIFS / hostPath / PVC | `storage.data.type` | `"pvc"` |

## Top-level configuration

| Field | Required | Description |
|---|---|---|
| `releaseName` | [x] | Must match `ModuleRelease.metadata.name` — used for Service DNS names |
| `domain` | [x] | Base domain, e.g. `mc.example.com` |
| `namespace` | [ ] | K8s namespace (default: `"default"`) |
| `servers` | [x] | Map of server name → per-server config |
| `router` | [x] | Router image, port, serviceType, defaultServer, autoScale, etc. |
| `rconPassword` | [x] | Shared RCON password — K8s Secret reference |
| `codeServer` | [ ] | Optional VS Code-in-browser Deployment |
| `resticGui` | [ ] | Optional Backrest web UI for restic snapshots |

## Per-server configuration (`servers.<name>`)

### Basic

| Field | Default | Description |
|---|---|---|
| `enabled` | `true` | `false` sets replicas to 0 (server stopped, data preserved) |
| `image` | `itzg/minecraft-server:java21` | Override to pin a specific image digest |
| `version` | `"LATEST"` | Minecraft version, e.g. `"1.21.1"` |
| `port` | `25565` | Container port, also used by mc-router mapping |
| `serviceType` | `"ClusterIP"` | `"ClusterIP"` \| `"LoadBalancer"` \| `"NodePort"` |
| `aliases` | — | Extra hostnames the router maps to this server |
| `extraPorts` | — | Extra container ports (e.g. BlueMap web UI) |

### Server types

Set exactly one of the following. Defaults to vanilla when none is set.

| Field | Server software |
|---|---|
| `vanilla` | Unmodified Mojang server |
| `paper` | Paper (high-performance Spigot fork) |
| `fabric` | Fabric modded server |
| `forge` | Minecraft Forge and NeoForge modded server |
| `spigot` | Spigot server |
| `bukkit` | CraftBukkit server |
| `purpur` | Purpur server (Paper fork) |
| `magma` | Magma hybrid (Forge + Bukkit API) |
| `sponge` | SpongeVanilla server |
| `ftba` | Feed The Beast modpack server |
| `autoCurseForge` | CurseForge modpack (requires API key) |
| `modrinth` | Modrinth modpack (`TYPE=MODRINTH`) |

### Paper-specific fields

| Field | Env var | Description |
|---|---|---|
| `paper.build` | `PAPER_BUILD` | Pin a specific Paper build number |
| `paper.channel` | `PAPER_CHANNEL` | `"experimental"` to unlock experimental builds |
| `paper.downloadUrl` | `PAPER_DOWNLOAD_URL` | Override download URL for self-hosted Paper |
| `paper.configRepo` | `PAPER_CONFIG_REPO` | URL to a repo of optimised config files (`bukkit.yml`, `paper-global.yml`, etc.) |
| `paper.skipDownloadDefaults` | `SKIP_DOWNLOAD_DEFAULTS` | Skip downloading default Paper/Bukkit/Spigot config files |

### Mod and plugin auto-download

Mods and plugins can be auto-downloaded from registries at server startup.

#### Modrinth projects

Use version IDs (not version numbers) for precise pinning. Modrinth version IDs
are the short alphanumeric strings shown on the version page (e.g. `Oa9ZDzZq`).

```cue
// Modrinth modpack (sets TYPE=MODRINTH)
modrinth: {
    modpack:              "https://modrinth.com/modpack/my-pack"
    version:              "abc123"        // omit for latest release
    projects:             ["lithium", "starlight:versionId"]
    downloadDependencies: "required"      // "none" | "required" | "optional"
}

// Paper — pin plugins by Modrinth version ID
paper: {
    plugins: {
        modrinth: {
            projects: [
                "essentialsx:Oa9ZDzZq",  // 2.21.2
                "luckperms:OrIs0S6b",     // v5.5.17
                "bluemap:Vb2ZE8bR",       // 5.16-paper
            ]
            downloadDependencies: "required"
        }
        removeOldMods: true   // clear stale jars before downloading
    }
}

// Modrinth projects on top of Fabric
fabric: {
    loaderVersion: "0.15.11"
    mods: modrinth: projects: ["lithium", "starlight"]
}
```

#### Spiget and direct URLs

```cue
// Spiget (SpigotMC resource IDs)
paper: plugins: spigetResources: [28140, 34315]

// Direct jar URLs
paper: plugins: urls: ["https://example.com/MyPlugin.jar"]

// Zip modpack of plugin jars
paper: plugins: modpackUrl: "https://example.com/plugins.zip"
```

#### Bootstrap vs registry

Bootstrap and registry-based downloads compose cleanly and serve different purposes:

- **Bootstrap** provides *state*: worlds, plugin config directories
- **Modrinth / SPIGET / URLs** provide *code*: the jar files

Do not put plugin jars in the bootstrap archive. If a jar is present in both the
bootstrap archive and the Modrinth list, both will land in `/data/plugins` — the
Modrinth version overwrites the bootstrap version (Modrinth downloads last). Use
`removeOldMods: true` to clear any stale jars before fresh downloads.

### Extra ports

Add container ports for plugins that open their own HTTP/TCP listeners (e.g. BlueMap):

```cue
extraPorts: [{
    name:          "bluemap"
    containerPort: 8100
    protocol:      "TCP"   // default

    // expose: true also adds this port to the Kubernetes Service
    expose:        true

    // exposedPort: override the Service-side port (defaults to containerPort)
    // exposedPort: 18100
}]
```

By default (`expose: false`) the port is added to the container spec only —
useful for internal plugin listeners that don't need external access.

### Router hostname aliases

Each server can declare additional hostnames the router should map to it,
in addition to the auto-generated `{serverName}.{domain}` mapping:

```cue
"vanilla": {
    // ...
    // Players connecting to vanilla.example.com OR vanilla.mc.example.com
    // both land on this server.
    aliases: ["vanilla.example.com"]
}
```

Aliases use the same backend DNS as the primary mapping:
`{releaseName}-server-{serverName}.{namespace}.svc:{port}`

### Bootstrap

Bootstrap a new server from a tar archive containing worlds, plugins,
mods, and/or config files:

```cue
bootstrap: {
    url:                    "https://nas.example.com/backups/my-server.tar.xz"
    force:                  false  // true to overwrite existing worlds (disaster recovery)
    skipNewerInDestination: true   // true = server-modified files are preserved
}
```

#### Mandatory archive layout

Directories must be at the **root of the archive** with no wrapper directory.
All directories are optional — include only what you need.

```
my-server.tar.xz        ← root of the archive
├── worlds/
│   ├── world/          (contains level.dat)
│   ├── world_nether/
│   └── world_the_end/
├── plugins/            staged → itzg syncs to /data/plugins on every start
├── mods/               staged → itzg syncs to /data/mods on every start
└── config/             staged → itzg syncs to /data/config on every start
```

#### Creating the archive

Run `tar` **from inside** your server data directory so that `worlds/`,
`plugins/`, etc. appear at the archive root — not wrapped in a subdirectory:

```sh
# xz (recommended — best compression)
cd /path/to/server-data
tar -cJf my-server.tar.xz worlds/ plugins/ mods/ config/

# gzip (faster, larger)
tar -czf my-server.tar.gz worlds/ plugins/ mods/ config/

# Include only what you need — all dirs are optional:
tar -cJf worlds-only.tar.xz worlds/
tar -cJf plugins-only.tar.xz plugins/
```

**Wrong** — do not wrap in a subdirectory:
```sh
tar -cJf bootstrap.tar.xz my-server/   # WRONG: produces my-server/worlds/... inside archive
```

Supported formats: `.tar.gz`, `.tar.xz`, `.tar.bz2`, `.tar.zst`

#### Behavior

- The init container runs on **every pod start** (staging emptyDirs are ephemeral).
  Plugins/mods/config are always re-staged so they stay current across pod recreations.
- Worlds are only copied if they **don't already exist** in `/data`. Existing player
  progress is always preserved unless `force: true` is set.
- `skipNewerInDestination: true` (default) means files already modified by the server
  (e.g. plugin configs tuned in-game) are not overwritten by the archive on subsequent starts.

### Backup sidecar

```cue
backup: {
    enabled:          true
    method:           "restic"          // "tar" | "rsync" | "restic" | "rclone"
    interval:         "1h"
    initialDelay:     "5m"
    pruneBackupsDays: 14
    pauseIfNoPlayers: true
    excludes:         ["./bluemap/*"]

    restic: {
        repository: "s3:https://s3.example.com/my-bucket/server"
        password:   { value: "my-restic-password" }
        accessKey:  { value: "ACCESS_KEY_ID" }
        secretKey:  { value: "SECRET_ACCESS_KEY" }
        retention:  "--keep-within 20d"
    }
}
```

### Monitor sidecar

Enabled by default. Exposes Prometheus metrics via `itzg/mc-monitor` on port 8080:

```cue
monitor: {
    enabled: true   // default
    port:    8080
}
```

### Storage

```cue
storage: data: {
    // PVC (default)
    type:         "pvc"
    size:         "10Gi"
    storageClass: "local-path"

    // OR: hostPath (recommended with codeServer for shared file access)
    type:         "hostPath"
    path:         "/var/data/minecraft/survival"
    hostPathType: "DirectoryOrCreate"

    // OR: NFS
    type:      "nfs"
    nfsServer: "10.10.0.2"
    nfsPath:   "/mnt/data/minecraft"

    // OR: CIFS/SMB (requires smb.csi.k8s.io driver)
    type:          "cifs"
    cifsSource:    "//10.10.0.2/minecraft"
    cifsSecretRef: "cifs-credentials"
}
```

## Code Server

Optional VS Code-in-browser Deployment that mounts all server data volumes
at `/servers/{name}` for direct file access (edit configs, inspect worlds, etc.).

```cue
codeServer: {
    enabled:     true
    port:        8080
    serviceType: "ClusterIP"
    password:    { value: "my-password" }
    storage: home: {
        type:         "pvc"    // persists extensions and settings
        size:         "5Gi"
        storageClass: "local-path"
    }
}
```

**hostPath vs PVC:** With `hostPath` storage on server pods, code-server shares
the same host paths without PVC ownership conflicts — full read/write access to
live server data. With `pvc` storage, code-server gets a separate (empty) PVC per
server and **cannot** access the running server's data.

## Restic GUI (Backrest)

Optional [Backrest](https://github.com/garethgeorge/backrest) web UI for browsing
and restoring restic snapshots created by the backup sidecars.

```cue
resticGui: {
    enabled:     true
    port:        9898
    serviceType: "ClusterIP"
    username:    "admin"
    password:    value: "my-password"
    storage: data: {
        type:         "pvc"
        size:         "5Gi"
        storageClass: "local-path"
    }
}
```

On first deploy, an init container writes `/data/config.json` with one restic repo
pre-configured per server that has `backup.method == "restic"`. Click
"Index Snapshots" in the Backrest UI to populate the snapshot list.

Prune and check schedules are disabled in the pre-configured repos — the
`itzg/mc-backup` sidecar owns the backup schedule. Backrest is used purely for
browsing and restoring.

## Router

The shared `itzg/mc-router` routes TCP connections by SNI hostname. All servers
in the fleet are auto-wired as `--mapping` args. Per-server `aliases` add
additional hostnames pointing to the same backend.

The router identity (ServiceAccount, ClusterRole, ClusterRoleBinding) is named
`{releaseName}-router`, making it safe to run multiple fleet releases in the
same cluster without ownership conflicts.

```cue
router: {
    port:                25565
    serviceType:         "LoadBalancer"
    connectionRateLimit: 1
    defaultServer: {
        host: "{releaseName}-server-lobby.{namespace}.svc"
        port: 25565
    }

    // Wake/sleep StatefulSets when players connect/disconnect
    autoScale: {
        up:   { enabled: true }
        down: { enabled: true, after: "10m" }
    }

    metrics: { backend: "prometheus" }

    api: {
        enabled: true
        port:    8080
    }
}
```

## Example release

```cue
package main

import (
    m     "opmodel.dev/core/modulerelease@v1"
    fleet "example.com/modules/mc_java_fleet@v0.1.0"
)

m.#ModuleRelease

metadata: {
    name:      "my-fleet"
    namespace: "minecraft"
}

#module: fleet

values: {
    releaseName: "my-fleet"
    domain:      "mc.example.com"
    namespace:   "minecraft"

    rconPassword: value: "changeme"

    servers: {
        lobby: {
            enabled: true
            server: {
                motd:       "Welcome!"
                maxPlayers: 50
                mode:       "adventure"
                pvp:        false
                difficulty: "peaceful"
            }
            paper: {
                plugins: {
                    modrinth: {
                        projects: [
                            "essentialsx:Oa9ZDzZq",
                            "luckperms:OrIs0S6b",
                        ]
                        downloadDependencies: "required"
                    }
                    removeOldMods: true
                }
            }
            jvm: memory: "1G"

            // Expose BlueMap web UI on port 8100
            extraPorts: [{
                name:          "bluemap"
                containerPort: 8100
                expose:        true
            }]

            // Also reachable at lobby.example.com (in addition to lobby.mc.example.com)
            aliases: ["lobby.example.com"]
        }

        survival: {
            server: {
                maxPlayers: 20
                difficulty: "hard"
            }
            modrinth: {
                modpack:              "https://modrinth.com/modpack/create-ultimate-selection-2"
                version:              "Mun9yNz5"
                downloadDependencies: "required"
            }
            jvm: { initMemory: "2G", maxMemory: "6G", useAikarFlags: true }

            // Seed world from backup archive on first deploy
            bootstrap: {
                url: "https://nas.example.com/backups/survival.tar.gz"
            }

            backup: {
                enabled:      true
                method:       "restic"
                interval:     "1h"
                initialDelay: "5m"
                restic: {
                    repository: "s3:https://s3.example.com/mc/survival"
                    password:   { value: "restic-pass" }
                    accessKey:  { value: "ACCESS" }
                    secretKey:  { value: "SECRET" }
                }
            }

            storage: data: {
                type:         "hostPath"
                path:         "/data/minecraft/survival"
                hostPathType: "DirectoryOrCreate"
            }
        }
    }

    router: {
        port:        25565
        serviceType: "LoadBalancer"
        defaultServer: {
            host: "my-fleet-server-lobby.minecraft.svc"
            port: 25565
        }
    }

    codeServer: {
        enabled:     true
        port:        8080
        serviceType: "ClusterIP"
        password:    { value: "changeme" }
        storage: home: { type: "pvc", size: "5Gi" }
    }

    resticGui: {
        enabled:     true
        port:        9898
        serviceType: "ClusterIP"
        username:    "admin"
        password:    { value: "changeme" }
        storage: data: { type: "pvc", size: "5Gi" }
    }
}
```

## Kubernetes resources produced

For a release named `my-fleet` with servers `lobby` and `survival`, plus
`codeServer` and `resticGui` enabled:

```text
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

Deployment/my-fleet-code-server
Service/my-fleet-code-server           (ClusterIP)

Deployment/my-fleet-restic-gui
Service/my-fleet-restic-gui            (ClusterIP)
PersistentVolumeClaim/my-fleet-restic-gui-data
```

## Service DNS convention

```text
releaseName = "my-fleet"
namespace   = "minecraft"

server-lobby    →  my-fleet-server-lobby.minecraft.svc.cluster.local
server-survival →  my-fleet-server-survival.minecraft.svc.cluster.local
router          →  my-fleet-router.minecraft.svc.cluster.local
code-server     →  my-fleet-code-server.minecraft.svc.cluster.local
restic-gui      →  my-fleet-restic-gui.minecraft.svc.cluster.local
```

## Differences from the gamestack bundle

| Feature | mc_java_fleet | gamestack bundle |
|---|---|---|
| Velocity proxy | [ ] | [x] |
| mc-monitor | [x] (per-server sidecar) | [x] (per-server sidecar) |
| Deployment model | Single ModuleRelease | Multiple ModuleReleases |
| Proxied network mode | [ ] | [x] |
| Standalone servers | [x] | [x] |
| Bootstrap init container | [x] | [ ] |
| Code Server | [x] | [ ] |
| Restic GUI (Backrest) | [x] | [ ] |
