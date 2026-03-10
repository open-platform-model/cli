# rcon_web_admin

Stateless web console for administering Minecraft servers over RCON using `itzg/rcon`.

## What it does

- Exposes a browser-based admin UI
- Connects to a target Minecraft server over RCON
- Supports optional Gateway API HTTPRoute configuration

## Files

- `module.cue` - metadata, schema, and `debugValues`
- `components.cue` - web admin deployment and network exposure

## Example release

See `examples/releases/rcon_web_admin/` for a concrete `release.cue` and default `values.cue`.
