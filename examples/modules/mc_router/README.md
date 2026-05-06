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

## Render without a release.cue

For quick iteration, render manifests straight from this module using its
`debugValues`:

```bash
opm module build ./examples/modules/mc_router
```

Override the synthetic name or supply your own values:

```bash
opm module build ./examples/modules/mc_router --name mc-router-debug
opm module build ./examples/modules/mc_router -f my-overrides.cue
```
