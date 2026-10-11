# Delta: core-provider (cli-kernel-adoption)

The CLI's `Provider` type is retired with the provider concept (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2"). Capability retired wholesale.

## REMOVED Requirements

### Requirement: Provider type location and interface

**Reason**: `pkg/provider` (and the former `pkg/core` provider type) is deleted; core defines no `#Provider` (enhancement 0001, enhancement 0006 ("`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2")).
**Migration**: Platforms + catalog subscriptions (`platform-resolution`); transformers via the kernel's materialized platform.
