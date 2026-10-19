# Delta: pkg-types (cli-kernel-adoption)

The provider type and typed component accessors leave `pkg/`; `pkg/core.Resource` and the metadata types remain the CLI's exported surface.

## REMOVED Requirements

### Requirement: ModuleInstance has typed component accessors

**Reason**: The CLI `Instance` type is deleted with kernel adoption (enhancement 0006, "The CLI replaces its own render/match pipeline entirely with the `library` kernel").
**Migration**: Kernel `module.Instance`.

### Requirement: Provider is a thin CUE wrapper

**Reason**: Providers are retired (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2"); `pkg/provider` is deleted.
**Migration**: `platform-resolution` capability.
