# Minecraft Module Family

OPM modules for the [itzg](https://github.com/itzg) Minecraft Docker/Helm ecosystem. Each module maps to one itzg Helm chart and can be deployed independently or composed together.

## Modules

| Module | Image | Description |
|--------|-------|-------------|
| [minecraft-java](./minecraft-java/) | `itzg/minecraft-server` | Java Edition server with optional mc-backup sidecar |
| [minecraft-bedrock](./minecraft-bedrock/) | `itzg/minecraft-bedrock-server` | Bedrock Edition server (UDP, XUID auth) |
| [minecraft-proxy](./minecraft-proxy/) | `itzg/bungeecord` | BungeeCord / Waterfall / Velocity proxy |
| [mc-router](./mc-router/) | `itzg/mc-router` | TCP hostname router with auto-scale |
| [mc-monitor](./mc-monitor/) | `itzg/mc-monitor` | Prometheus/OTel metrics exporter for server status |
| [rcon-web-admin](./rcon-web-admin/) | `itzg/rcon` | Web UI for RCON console management |

## Composition

Each module is independently deployable. When composed, they connect via network:

```
                    Internet
                       |
                       v
               +--------------+
               |  mc-router   |  :25565 TCP
               |  (hostname   |  Routes by server-address field
               |   routing)   |  in Minecraft protocol
               +------+-------+
                      |
            +---------+-----------+
            v                     v
    +---------------+     +---------------+
    | minecraft-    |     | minecraft-    |
    | proxy         |     | java          |  (standalone)
    | (BungeeCord/  |     |               |
    |  Velocity)    |     +-------+-------+
    +------+--------+             |
           |                      v
      +----+----+         +--------------+
      v         v         | rcon-web-    |
    +-----+  +-----+     | admin        |
    |java |  |java |     | (web UI)     |
    | #1  |  | #2  |     +--------------+
    +-----+  +-----+

    +---------------+
    | minecraft-    |
    | bedrock       |  :19132 UDP (standalone)
    +---------------+
            :                          :
            :  (Minecraft protocol)    :
            v                          v
          +------------------------------+
          |         mc-monitor           |  :8080 /metrics
          |  (status probes via MC       |  (Prometheus)
          |   protocol, not RCON)        |  or push to OTel
          +------------------------------+
```

## Quick Start

### Single Java Server

```bash
opm mod apply --module-dir ./examples/minecraft/minecraft-java

# Check status
opm mod status --module-dir ./examples/minecraft/minecraft-java

# View resource tree
opm mod tree --module-dir ./examples/minecraft/minecraft-java
```

### Java Server + RCON Web Admin

```bash
# Deploy the Minecraft server
opm mod apply --module-dir ./examples/minecraft/minecraft-java

# Deploy RCON Web Admin pointed at the server
# Update rcon-web-admin values to point rconHost at your server service name
opm mod apply --module-dir ./examples/minecraft/rcon-web-admin
```

### Java Server + Monitoring

```bash
# Deploy the Minecraft server
opm mod apply --module-dir ./examples/minecraft/minecraft-java

# Deploy mc-monitor pointed at the server
# Update mc-monitor values to list your server(s) in javaServers
opm mod apply --module-dir ./examples/minecraft/mc-monitor
```

### Multi-Server with Router

```bash
# Deploy mc-router
opm mod apply --module-dir ./examples/minecraft/mc-router

# Deploy multiple Java servers with different values
opm mod apply --module-dir ./examples/minecraft/minecraft-java --values-file values_survival.cue
opm mod apply --module-dir ./examples/minecraft/minecraft-java --values-file values_creative.cue
```

## Related Resources

- [itzg/docker-minecraft-server](https://github.com/itzg/docker-minecraft-server)
- [itzg/docker-mc-backup](https://github.com/itzg/docker-mc-backup)
- [itzg/minecraft-server-charts](https://github.com/itzg/minecraft-server-charts)
- [itzg/mc-router](https://github.com/itzg/mc-router)
- [itzg/mc-monitor](https://github.com/itzg/mc-monitor)
