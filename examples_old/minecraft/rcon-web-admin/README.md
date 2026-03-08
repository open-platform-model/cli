# RCON Web Admin Module

A web-based RCON admin console for managing game servers from a browser, using OPM (Open Platform Model).

## Overview

[RCON Web Admin](https://github.com/rcon-web-admin/rcon-web-admin) (itzg/rcon) provides a browser UI for sending RCON commands to game servers. This module deploys it as a stateless workload with dual-port networking:

- **HTTP port** (default 80) serves the web UI
- **WebSocket port** (default 4327) provides real-time RCON communication

## Connecting to a Minecraft Server

Point `rconHost` at your Minecraft server's Service name and ensure `rconPort` / `rconPassword` match the server's RCON configuration:

```cue
values: {
    admin: {
        rconHost:     "minecraft-java.minecraft.svc.cluster.local"
        rconPort:     25575
        rconPassword: "your-rcon-password"
        game:         "minecraft"
    }
}
```

When deploying alongside the `minecraft-java` module, use the Kubernetes Service DNS name so the web admin can reach the server within the cluster.

## HTTPRoute for Web Access

To expose the web UI via Gateway API, configure an HTTPRoute:

```cue
values: {
    httpRoute: {
        hostnames: ["rcon.example.com"]
        gatewayRef: {
            name:      "my-gateway"
            namespace: "gateway-system"
        }
    }
}
```

This creates an HTTPRoute targeting the HTTP port with a `/` prefix match.

## Admin User Configuration

The module supports a single admin user for the web UI:

- `admin.isAdmin` -- grants full admin privileges in the console
- `admin.username` / `admin.password` -- login credentials (change the defaults!)
- `admin.restrictCommands` -- limit which RCON commands are available
- `admin.restrictWidgets` -- limit which dashboard widgets are shown
- `admin.immutableWidgetOptions` -- prevent users from changing widget settings

## Supported Games

The `admin.game` field controls which command palette and widgets are loaded. Common values: `minecraft`, `rust`, `csgo`. See the [itzg/rcon documentation](https://github.com/rcon-web-admin/rcon-web-admin) for the full list of supported games.

## Related Resources

- **itzg/rcon image**: <https://hub.docker.com/r/itzg/rcon>
- **RCON Web Admin**: <https://github.com/rcon-web-admin/rcon-web-admin>
- **minecraft-java module**: `../minecraft-java/` (companion game server)
