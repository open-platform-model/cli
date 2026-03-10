# mc_router

Stateless hostname router for Minecraft traffic based on `itzg/mc-router`.

## What it does

- Routes player connections by requested hostname
- Supports a default backend and explicit hostname mappings
- Can expose an optional REST API and Prometheus metrics backend

## Files

- `module.cue` - metadata, schema, and `debugValues`
- `components.cue` - router deployment, RBAC, and service exposure

## Example release

See `examples/releases/mc_router/` for a concrete `release.cue` and default `values.cue`.
