# Minecraft Proxy Module

OPM module for deploying a Minecraft proxy server using the [itzg/bungeecord](https://hub.docker.com/r/itzg/bungeecord) container image.

## Overview

This module deploys a Minecraft proxy that sits in front of one or more backend Minecraft Java Edition servers. It supports multiple proxy implementations and handles player routing, load distribution, and cross-server communication.

## Proxy Types

| Type | Description |
|------|-------------|
| `BUNGEECORD` | Original BungeeCord proxy (default) |
| `WATERFALL` | PaperMC's fork of BungeeCord with performance improvements |
| `VELOCITY` | Modern, high-performance proxy by PaperMC (recommended for new setups) |
| `CUSTOM` | Bring your own proxy JAR via `jarUrl` or `jarFile` |

Set the proxy type via `proxy.type` in values:

```cue
values: proxy: type: "VELOCITY"
```

## Plugin Support

Plugins can be installed from URLs at startup:

```cue
values: proxy: plugins: [
    "https://example.com/plugin-a.jar",
    "https://example.com/plugin-b.jar",
]
```

## Inline Config

Provide proxy configuration inline instead of mounting config files:

```cue
values: proxy: {
    configFilePath: "/server/config.yml"
    configContent: """
        server_connect_timeout: 5000
        listeners:
          - host: 0.0.0.0:25577
            motd: 'My Proxy Network'
        servers:
          lobby:
            address: minecraft-lobby:25565
          survival:
            address: minecraft-survival:25565
        """
}
```

For Velocity, use TOML format in `configContent`.

## RCON

Optional RCON access for remote administration:

```cue
values: proxy: rcon: {
    enabled:  true
    port:     25575
    password: "change-me"
}
```

## Scaling

The proxy defaults to a single replica. Unlike game servers, proxies can potentially be scaled behind a shared service, though this depends on the proxy implementation and plugin compatibility.

## Storage

Proxy data (config, plugins, logs) is persisted to `/server`. Configure storage via `storage.data`:

```cue
values: storage: data: {
    type: "pvc"
    size: "1Gi"
}
```

## Defaults

- Image: `itzg/bungeecord:latest`
- Proxy type: `BUNGEECORD`
- Port: `25577`
- Memory: `512M`
- Service type: `ClusterIP`
- Storage: 1Gi PVC
