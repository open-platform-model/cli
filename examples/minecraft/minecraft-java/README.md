# Minecraft Java Edition Module

A production-ready Minecraft Java Edition server with automated backup support using OPM (Open Platform Model). Part of the [minecraft module family](../README.md).

## Features

- **Multiple Server Types**: Vanilla, Paper, Forge, Fabric, Spigot, Bukkit, Purpur, Magma, SpongeVanilla, FTBA, Auto CurseForge
- **Full Helm Chart Parity**: Exposes the complete itzg/minecraft-server Helm chart `values.yaml` surface area
- **Automated Backups**: RCON-coordinated backups with zero data loss (sidecar pattern)
- **Flexible Storage**: PVC (cloud), hostPath (bare-metal), or emptyDir (testing)
- **Multiple Backup Methods**: tar, rsync, restic (cloud), rclone (remote)
- **Mod/Plugin Support**: Modrinth, CurseForge, Spigot, direct URL, VanillaTweaks
- **JVM Tuning**: Memory allocation, JVM opts, -XX flags
- **Production Ready**: Health checks, resource limits, security context, graceful shutdown, Recreate update strategy

## Quick Start

### 1. Basic Deployment (Paper Server)

```bash
# Apply the module with defaults
opm mod apply . --release-name mc-java --namespace default
```

This deploys:

- **Paper server** (latest version, auto-downloaded)
- **10 max players**, normal difficulty, survival mode
- **emptyDir** storage (ephemeral — for testing)
- **Backups disabled** (no data to back up with emptyDir)
- **ClusterIP service** on port 25565

### 2. Check Status

```bash
# View release status and health
opm mod status --release-name mc-java --namespace default
```

### 3. Connect to Server

```bash
# Port-forward for local access
kubectl port-forward svc/server 25565:25565

# Connect from Minecraft client: localhost:25565
```

## Server Type

Set exactly one type struct in your values to select the server software. The `matchN(1, [...])` constraint in the schema enforces that only one is active.

```
  values.cue                    module.cue
  ┌──────────────────┐          ┌──────────────────────────────────────┐
  │ paper: {}         │  ─────> │ paper?: { downloadUrl?: string }     │
  │                   │         │                                      │
  │ // OR             │         │ matchN(1, [                          │
  │ forge: {          │         │   {vanilla!: _},                     │
  │   version: "47.2" │         │   {paper!: _},                      │
  │ }                 │         │   {forge!: _},                       │
  │                   │         │   ...11 options                      │
  │ // OR             │         │ ])                                   │
  │ vanilla: {}       │         │                                      │
  └──────────────────┘          └──────────────────────────────────────┘
```

### Available Types

| Struct               | itzg TYPE value   | Required fields              | Notes |
|----------------------|-------------------|------------------------------|-------|
| `vanilla: {}`        | VANILLA           | none                         | Unmodified Mojang server |
| `paper: {}`          | PAPER             | `downloadUrl?` (optional)    | High-performance Spigot fork |
| `forge: {...}`       | FORGE             | `version` (required)         | Forge modded server |
| `fabric: {...}`      | FABRIC            | `loaderVersion` (required)   | Fabric modded server |
| `spigot: {}`         | SPIGOT            | `downloadUrl?` (optional)    | Spigot server |
| `bukkit: {}`         | BUKKIT            | `downloadUrl?` (optional)    | CraftBukkit server |
| `sponge: {...}`      | SPONGEVANILLA     | `version` (required)         | SpongeVanilla server |
| `purpur: {}`         | PURPUR            | none                         | Paper fork with extras |
| `magma: {}`          | MAGMA             | none                         | Forge + Bukkit/Spigot API |
| `ftba: {}`           | FTBA              | none                         | Feed The Beast modpack |
| `autoCurseForge: {}` | AUTO_CURSEFORGE   | `apiKey` (required)          | CurseForge modpack |

## Configuration Reference

The `#config` schema is organized into clear sections. All fields are available in your `values.cue` under the `values:` key.

### Top-Level Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `image` | `#Image` | `itzg/minecraft-server:latest` | Container image |
| `version` | string | `"LATEST"` | Minecraft version (e.g., "1.20.4", "LATEST", "SNAPSHOT") |
| `eula` | bool | `true` | EULA acceptance (required by Mojang) |
| `port` | uint | *(required)* | Minecraft server port |
| `serviceType` | enum | *(required)* | ClusterIP, LoadBalancer, NodePort |

### JVM Configuration (`jvm`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `jvm.memory` | string | `"2G"` | JVM heap size (e.g., "2G", "4096M") |
| `jvm.opts` | string | optional | General JVM options |
| `jvm.xxOpts` | string | optional | JVM -XX options (applied before general opts) |

### Server Properties (`server`)

Fields that map directly to Minecraft `server.properties`:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `server.motd` | string | `"OPM Minecraft Java Server"` | Message of the day |
| `server.maxPlayers` | uint | `10` | Max concurrent players (1-1000) |
| `server.difficulty` | enum | `"normal"` | peaceful, easy, normal, hard |
| `server.mode` | enum | `"survival"` | survival, creative, adventure, spectator |
| `server.pvp` | bool | `true` | Enable PvP combat |
| `server.enableCommandBlock` | bool | `false` | Enable command blocks |
| `server.ops` | [...string] | optional | Server operator usernames |
| `server.blocklist` | [...string] | optional | Blocked usernames |
| `server.seed` | string | optional | World generation seed |
| `server.maxWorldSize` | uint | optional | Max world radius in blocks |
| `server.viewDistance` | uint | `10` | View distance in chunks (1-32) |
| `server.allowNether` | bool | `true` | Allow Nether travel |
| `server.announcePlayerAchievements` | bool | `true` | Announce achievements |
| `server.forceGameMode` | bool | `false` | Force default gameMode on join |
| `server.generateStructures` | bool | `true` | Generate villages, temples, etc. |
| `server.hardcore` | bool | `false` | Spectator mode on death |
| `server.maxBuildHeight` | uint | `256` | Maximum building height |
| `server.maxTickTime` | int | `60000` | Max tick duration in ms (-1 to disable) |
| `server.spawnAnimals` | bool | `true` | Allow animal spawning |
| `server.spawnMonsters` | bool | `true` | Allow monster spawning |
| `server.spawnNPCs` | bool | `true` | Allow villager spawning |
| `server.spawnProtection` | int | `0` | Non-op edit protection radius (0 = disabled) |
| `server.levelType` | enum | `"DEFAULT"` | DEFAULT, FLAT, LARGEBIOMES, AMPLIFIED, CUSTOMIZED |
| `server.worldSaveName` | string | `"world"` | World save directory name |
| `server.onlineMode` | bool | `true` | Verify accounts against Mojang |
| `server.enforceSecureProfile` | bool | `true` | Require Mojang-signed public key |
| `server.overrideServerProperties` | bool | `true` | Override server.properties on restart |
| `server.resourcePackUrl` | string | optional | Resource pack URL |
| `server.resourcePackSha` | string | optional | Resource pack SHA hash |
| `server.resourcePackEnforce` | bool | optional | Require clients to use resource pack |

### RCON Configuration (`rcon`)

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `rcon.enabled` | bool | `true` | Enable RCON (required for backup) |
| `rcon.password` | `#Secret` | *(required)* | RCON password (stored as K8s Secret) |
| `rcon.port` | uint | *(required)* | RCON port |

### Query Port (`query`)

| Field | Type | Description |
|-------|------|-------------|
| `query.enabled` | bool | Enable query protocol |
| `query.port` | uint | Query port |

### Mods & Plugins (`mods`)

| Field | Type | Description |
|-------|------|-------------|
| `mods.modUrls` | [...string] | URLs to mod jar files |
| `mods.pluginUrls` | [...string] | URLs to plugin jar files |
| `mods.spigetResources` | [...int] | Spigot resource/plugin IDs |
| `mods.modrinth.projects` | [...string] | Modrinth project slugs |
| `mods.modrinth.downloadDependencies` | enum | none, required, optional |
| `mods.modrinth.allowedVersionType` | enum | release, beta, alpha |
| `mods.vanillaTweaksShareCodes` | [...string] | VanillaTweaks share codes |
| `mods.downloadWorldUrl` | string | URL to download a world on startup |
| `mods.downloadModpackUrl` | string | URL to download a modpack on startup |
| `mods.removeOldMods` | bool | Replace old mods with new modpack mods |

### Networking

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | uint | *(required)* | Minecraft server port |
| `serviceType` | enum | *(required)* | ClusterIP, LoadBalancer, NodePort |
| `extraPorts` | [...{name, containerPort, protocol}] | optional | Additional ports (dynmap, bluemap, etc.) |

### Resources

Uses the catalog-standard `#ResourceRequirementsSchema` shape:

```cue
resources: {
    requests: {
        cpu:    "1000m"
        memory: "2Gi"
    }
    limits: {
        cpu:    "4000m"
        memory: "8Gi"
    }
}
```

### Security Context

Optional `securityContext` field using catalog `#SecurityContextSchema`. Default traits apply `runAsUser: 1000`, `runAsGroup: 3000`.

### Traits (Applied Automatically)

- **UpdateStrategy**: `Recreate` (Minecraft cannot run two instances on same data)
- **SecurityContext**: Non-root user (1000:3000)
- **GracefulShutdown**: 60s termination grace period (allows world save)

## Configuration Examples

### Example 1: Modded Forge Server

```cue
values: {
    forge: { version: "47.2.0" }       // Server type
    version: "1.20.1"

    jvm: {
        memory:  "8G"
        opts:    "-XX:+UseG1GC -XX:+ParallelRefProcEnabled"
    }

    server: maxPlayers: 50

    rcon: {
        enabled:  true
        password: value: "CHANGE-THIS-PASSWORD"
        port:     25575
    }

    storage: {
        data: {
            type:         "hostPath"
            path:         "/mnt/minecraft/modded-server"
            hostPathType: "DirectoryOrCreate"
        }
        backups: {
            type:         "hostPath"
            path:         "/mnt/minecraft/backups"
            hostPathType: "DirectoryOrCreate"
        }
    }

    resources: {
        requests: { cpu: "2000m", memory: "8Gi" }
        limits:   { cpu: "4000m", memory: "12Gi" }
    }

    port:        25565
    serviceType: "NodePort"
}
```

### Example 2: Fabric with Modrinth Mods

```cue
values: {
    fabric: { loaderVersion: "0.15.0" }    // Server type
    version: "1.20.1"

    jvm: memory: "6G"

    server: motd: "Modrinth Modpack Server"

    mods: {
        modrinth: {
            projects:             ["sodium", "lithium", "starlight"]
            downloadDependencies: "required"
            allowedVersionType:   "release"
        }
    }

    rcon: {
        enabled:  true
        password: value: "CHANGE-ME"
        port:     25575
    }

    storage: {
        data:    { type: "pvc", size: "20Gi" }
        backups: { type: "pvc", size: "10Gi" }
    }

    resources: {
        requests: { cpu: "2000m", memory: "6Gi" }
        limits:   { cpu: "4000m", memory: "8Gi" }
    }

    port:        25565
    serviceType: "LoadBalancer"
}
```

### Example 3: Production Paper with Restic Cloud Backups

```cue
values: {
    paper: {}                               // Server type
    version: "1.20.4"

    server: {
        motd:       "Production SMP Server"
        maxPlayers: 100
        ops:        ["player1", "player2"]
    }

    rcon: {
        enabled:  true
        password: value: "USE-A-SECURE-PASSWORD-HERE"
        port:     25575
    }

    storage: {
        data:    { type: "pvc", size: "50Gi", storageClass: "fast-ssd" }
        backups: { type: "pvc", size: "10Gi" }
    }

    backup: {
        enabled:      true
        method:       "restic"
        interval:     "6h"
        initialDelay: "10m"

        restic: {
            repository: "s3:s3.amazonaws.com/my-minecraft-backups"
            password:   value: "restic-repo-password"
            hostname:   "production-smp"
            retention:  "--keep-daily 7 --keep-weekly 4 --keep-monthly 6"
        }
    }

    resources: {
        requests: { cpu: "2000m", memory: "4Gi" }
        limits:   { cpu: "8000m", memory: "16Gi" }
    }

    port:        25565
    serviceType: "LoadBalancer"
}
```

### Example 4: Testing/Development (Ephemeral)

```cue
values: {
    paper: {}                               // Server type

    rcon: {
        enabled:  true
        password: value: "test"
        port:     25575
    }

    storage: {
        data:    type: "emptyDir"           // Data deleted when pod restarts
        backups: type: "emptyDir"
    }

    backup: enabled: false

    resources: {
        requests: { cpu: "500m", memory: "1Gi" }
        limits:   { cpu: "2000m", memory: "4Gi" }
    }

    port:        25565
    serviceType: "ClusterIP"
}
```

## Backup Methods

### Tar (Default) - Simple and Reliable

Creates compressed archives in `/backups`. Easy to inspect and restore manually.

### Restic - Cloud Backups with Deduplication

Encrypted, deduplicated, incremental backups. Supports S3, B2, Azure, GCS, SFTP.

### Rsync - Incremental Local Backups

Only copies changed files. Fast incremental backups. Works well with NFS/CIFS.

### Rclone - Remote Cloud Storage

Supports 40+ cloud storage providers via rclone configuration.

## Performance Tuning

### Memory Allocation

| Players | Recommended Memory | CPU Cores |
|---------|--------------------|-----------|
| 1-10    | 2-4 GB             | 1-2       |
| 10-20   | 4-6 GB             | 2-3       |
| 20-50   | 6-10 GB            | 3-4       |
| 50-100  | 10-16 GB           | 4-8       |
| 100+    | 16-32 GB           | 8+        |

### JVM Tuning

```cue
values: {
    jvm: {
        memory:  "8G"
        opts:    "-Dusing.aikars.flags=https://mcflags.emc.gs -Daikars.new.flags=true"
        xxOpts:  "+UseG1GC +ParallelRefProcEnabled +UnlockExperimentalVMOptions"
    }
}
```

## Resource Outputs

When you build this module, OPM generates:

- **StatefulSet** (`server`): 1 replica with server + optional backup sidecar
- **Service** (`server`): Exposes port 25565 (LoadBalancer/NodePort/ClusterIP)
- **Secret** (`server-secrets`): RCON password (when RCON enabled)

## Related Modules

- [minecraft-bedrock](../minecraft-bedrock/) -- Bedrock Edition server (UDP, XUID auth)
- [minecraft-proxy](../minecraft-proxy/) -- BungeeCord/Waterfall/Velocity proxy
- [mc-router](../mc-router/) -- Hostname-based Minecraft routing
- [rcon-web-admin](../rcon-web-admin/) -- Web-based RCON administration

## Related Resources

- **itzg/minecraft-server**: <https://github.com/itzg/docker-minecraft-server>
- **itzg/mc-backup**: <https://github.com/itzg/docker-mc-backup>
- **Minecraft Server Docs**: <https://docker-minecraft-server.readthedocs.io/>
- **Restic Documentation**: <https://restic.readthedocs.io/>
