# Delta: provider-loading (cli-kernel-adoption)

Provider loading from config is retired (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2"). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Load a named provider from config

**Reason**: `config.providers` is deleted; there is nothing to load providers from (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2").
**Migration**: `platform-resolution` — catalog subscriptions in `~/.opm/platform.cue` / the cluster `Platform` CR.

### Requirement: Parse transformer definitions from provider CUE value

**Reason**: Transformers come from materialized catalogs via the kernel, not from provider values.
**Migration**: Kernel `Materialize` → `#composedTransformers` (library contract).
