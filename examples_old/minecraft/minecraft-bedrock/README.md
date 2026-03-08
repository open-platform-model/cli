# Minecraft Bedrock Edition - OPM Module

OPM module for deploying a Minecraft Bedrock Edition dedicated server using the [itzg/minecraft-bedrock-server](https://github.com/itzg/docker-minecraft-bedrock-server) container image.

## Overview

- **Image**: `itzg/minecraft-bedrock-server`
- **Port**: UDP 19132 (Bedrock protocol)
- **Workload**: StatefulSet (single replica, Recreate update strategy)
- **Storage**: Persistent volume for world data at `/data`

## Key Differences from Java Edition

- Uses **UDP** port 19132 (not TCP 25565)
- **No RCON** support - Bedrock server does not expose an RCON interface
- **No backup sidecar** - the `itzg/mc-backup` container requires RCON and is not compatible
- **XUID-based authentication** - operators, members, and visitors are identified by Xbox User ID (comma-separated strings), not usernames
- Permission levels: `visitor`, `member`, `operator` (not the Java op-level system)

## Quick Start

```cue
values: {
    server: {
        image: {
            repository: "itzg/minecraft-bedrock-server"
            tag:        "latest"
            digest:     ""
        }
        version:           "LATEST"
        eula:              true
        difficulty:        "normal"
        gameMode:          "survival"
        maxPlayers:        10
        defaultPermission: "member"
        serverName:        "My Bedrock Server"
        onlineMode:        true
        serverPort:        19132
    }
    storage: data: {
        type: "pvc"
        size: "5Gi"
    }
    serviceType: "LoadBalancer"
}
```

## Configuration

All server settings are configured via environment variables passed to the container. See `module.cue` for the full `#config` schema. Key configuration areas:

- **Server**: version, difficulty, game mode, max players, permissions
- **World**: level type, level name, level seed, view/tick distance
- **Access Control**: whitelist, ops/members/visitors (XUID-based)
- **Storage**: PVC, hostPath, or emptyDir for world data
- **Resources**: CPU/memory requests and limits
