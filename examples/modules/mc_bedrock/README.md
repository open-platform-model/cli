# mc_bedrock

Stateful Minecraft Bedrock Edition server example based on `itzg/minecraft-bedrock-server`.

## What it does

- Runs a single Bedrock server with persistent world data
- Exposes UDP port `19132`
- Demonstrates a stateful workload without RCON or backup sidecars

## Files

- `module.cue` - metadata, schema, and `debugValues`
- `components.cue` - server workload, storage, and network exposure

## Example release

See `examples/releases/mc_bedrock/` for a concrete `release.cue` and default `values.cue`.
