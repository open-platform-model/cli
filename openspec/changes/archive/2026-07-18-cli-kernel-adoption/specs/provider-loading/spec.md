# Delta: provider-loading (cli-kernel-adoption)

Provider loading from config is retired (0006 D39). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Load a named provider from config

**Reason**: `config.providers` is deleted; there is nothing to load providers from (0006 D39).
**Migration**: `platform-resolution` — catalog subscriptions in `~/.opm/platform.cue` / the cluster `Platform` CR.

### Requirement: Parse transformer definitions from provider CUE value

**Reason**: Transformers come from materialized catalogs via the kernel, not from provider values.
**Migration**: Kernel `Materialize` → `#composedTransformers` (library contract).
