# mc_monitor

Stateless Minecraft status exporter based on `itzg/mc-monitor`.

## What it does

- Polls one or more Java and optional Bedrock servers
- Exposes Prometheus metrics or pushes to an OpenTelemetry collector
- Works well as a standalone diagnostics module or as part of a bundle

## Files

- `module.cue` - metadata, schema, and `debugValues`
- `components.cue` - deployment and optional service exposure

## Example release

See `examples/releases/mc_monitor/` for a concrete `release.cue` and default `values.cue`.
