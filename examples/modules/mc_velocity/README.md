# mc_velocity

Modern Minecraft proxy example focused on Velocity. This is the recommended replacement for the legacy generic `minecraft-proxy` example.

## What it does

- Runs a lightweight Velocity proxy using `itzg/mc-proxy`
- Exposes a single player-facing TCP port
- Supports Velocity forwarding modes and an optional forwarding secret

## Files

- `module.cue` - metadata, schema, and `debugValues`
- `components.cue` - proxy deployment and service exposure

## Example release

See `examples/releases/mc_velocity/` for a concrete `release.cue` and default `values.cue`.
