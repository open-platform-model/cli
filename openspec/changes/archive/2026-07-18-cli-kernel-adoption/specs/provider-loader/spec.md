# Delta: provider-loader (cli-kernel-adoption)

The provider concept is retired — core has no `#Provider`; platforms + catalog subscriptions replace it (enhancement 0006 ("`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2"), enhancement 0001). Capability retired wholesale.

## REMOVED Requirements

### Requirement: LoadProvider returns a fully-populated core.Provider

**Reason**: `pkg/loader.LoadProvider` and `pkg/provider` are deleted; there are no providers to load (enhancement 0006, "`~/.opm` becomes two import-free data files: `config.cue` drops `providers`/`cacheDir`, D21's local default platform is the sibling `~/.opm/platform.cue`; folded into C2").
**Migration**: Platform resolution + kernel materialization (`platform-resolution` capability).

### Requirement: Extract all provider metadata fields from CUE value

**Reason**: Provider metadata no longer exists.
**Migration**: Catalog/transformer metadata surfaces through the kernel's materialized platform.
